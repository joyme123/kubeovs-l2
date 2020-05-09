package main

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	"github.com/containernetworking/cni/pkg/types/current"
	"github.com/containernetworking/plugins/pkg/ip"
	"github.com/containernetworking/plugins/pkg/ipam"
	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/j-keck/arping"
	"github.com/vishvananda/netlink"
	"golang.org/x/exp/rand"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/klog"
)

const (
	defaultBridgeName string = "kubeovs-br"
	podMacAddr               = "aa:bb:cc:dd:ee:ff"
)

type NetConf struct {
	types.NetConf

	BridgeName string `json:"bridge"`

	// copied from bridge plugin, clean up later
	IsGW         bool `json:"isGateway`
	IsDefaultGW  bool `json:"isDefaultGateway"`
	ForceAddress bool `json:"forceAddress"`
	IPMasq       bool `json:"ipMasq"`
	MTU          int  `json:"mtu"`
	HairpinMode  bool `json:"hairpinMode"`
	PromiscMode  bool `json:"promiscMode"`
}

type gwInfo struct {
	gws               []net.IPNet
	family            int
	defaultRouteFound bool
}

func init() {
	// this ensures that main runs only on main thread (thread group leader).
	// since namespace ops (unshare, setns) are done for a single thread, we
	// must ensure that the goroutine does not jump from OS thread to thread
	runtime.LockOSThread()
}

func loadNetConf(bytes []byte) (*NetConf, string, error) {
	n := &NetConf{
		BridgeName: defaultBridgeName,
	}

	if err := json.Unmarshal(bytes, n); err != nil {
		return nil, "", fmt.Errorf("failed to load netconf: %v", err)
	}
	return n, n.CNIVersion, nil
}

func bridgeByName(name string) (netlink.Link, error) {
	br, err := netlink.LinkByName(name)
	if err != nil {
		return nil, fmt.Errorf("could not lookup %q: %v", name, err)
	}

	return br, nil
}

func getBridgeInterface() (*current.Interface, error) {
	br, err := bridgeByName(defaultBridgeName)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch bridge %q: %v", defaultBridgeName, err)
	}

	return &current.Interface{
		Name: br.Attrs().Name,
		Mac:  br.Attrs().HardwareAddr.String(),
	}, nil
}

func setupVeth(netns ns.NetNS, ifName string) (*current.Interface, *current.Interface, error) {
	contIface := &current.Interface{}
	hostIface := &current.Interface{}

	err := netns.Do(func(hostNs ns.NetNS) error {
		// create the veth pair in the container and move host end into host netns
		hostVeth, containerVeth, err := ip.SetupVeth(ifName, 1500, hostNs)
		if err != nil {
			return err
		}
		contIface.Name = containerVeth.Name
		contIface.Mac = containerVeth.HardwareAddr.String()
		contIface.Sandbox = netns.Path()
		hostIface.Name = hostVeth.Name
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	// need to lookup hostVeth again as its index has changed during ns move
	hostVeth, err := netlink.LinkByName(hostIface.Name)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to lookup %q: %v", hostIface.Name, err)
	}
	hostIface.Mac = hostVeth.Attrs().HardwareAddr.String()
	return hostIface, contIface, nil
}

func getPodInfo() (string, string, error) {
	cniArgs := os.Getenv("CNI_ARGS")
	if cniArgs == "" {
		return "", "", errors.New("env var CNI_ARGS was empty")
	}

	var podNamespace, podName string
	args := strings.Split(cniArgs, ";")
	for _, arg := range args {
		if strings.Contains(arg, "K8S_POD_NAMESPACE") {
			podNamespace = strings.TrimPrefix(arg, "K8S_POD_NAMESPACE=")
			continue
		}

		if strings.Contains(arg, "K8S_POD_NAME") {
			podName = strings.TrimPrefix(arg, "K8S_POD_NAME=")
			continue
		}
	}

	return podNamespace, podName, nil
}

func normalizedNetNS(netns string) string {
	return strings.Replace(netns, "/", "", -1)
}

func addPort(bridgeName, port, containerMac, netNs, podNamespace, podName string) error {
	commands := []string{
		"--may-exist", "add-port", bridgeName, port, "--", "set", "port", port,
		fmt.Sprintf("external-ids:netns=%s", normalizedNetNS(netNs)),
		fmt.Sprintf("external-ids:k8s_pod_namespace=%s", podNamespace),
		fmt.Sprintf("external-ids:k8s_pod_name=%s", podName),
		"--", "set", "interface", port,
		fmt.Sprintf("external-ids:netns=%s", normalizedNetNS(netNs)),
		fmt.Sprintf("external-ids:k8s_pod_namespace=%s", podNamespace),
		fmt.Sprintf("external-ids:k8s_pod_name=%s", podName),
		"--", "set", "port", port, fmt.Sprintf("mac=\"%s\"", containerMac),
	}
	output, err := exec.Command("ovs-vsctl", commands...).CombinedOutput()
	if err != nil {
		klog.Infoln("ovs addport cmd:", strings.Join(commands, " "))
		return fmt.Errorf("failed to add OVS port %q to bridge %q, err: %v, %v", port, bridgeName, string(output), err)
	}

	return nil
}

func getAvailableIP() (*current.Result, error) {
	// TODO: 暂时随机一个
	// n1 := rand.Int() % 256
	rand.Seed(uint64(time.Now().UnixNano()))
	n1 := 50
	n2 := rand.Int() % 256

	// TODO(jiang): 暂时模式是第0个
	index := 2

	return &current.Result{
		IPs: []*current.IPConfig{
			{
				Version:   "4",
				Interface: &index,
				Address: net.IPNet{
					IP:   net.IP([]byte{192, 168, uint8(n1), uint8(n2)}),
					Mask: net.IPMask([]byte{255, 255, 255, 0}),
				},
				Gateway: net.IP([]byte{192, 168, 50, 1}),
			},
		},
		// TODO: routes 暂时留空
	}, nil
}

func calcGatewayIP(ipn *net.IPNet) net.IP {
	nid := ipn.IP.Mask(ipn.Mask)
	return ip.NextIP(nid)
}

// calcGateways processes the results from the IPAM plugin and does the
// following for each IP family:
//    - Calculates and compiles a list of gateway addresses
//    - Adds a default route if needed
func calcGateways(result *current.Result, n *NetConf) (*gwInfo, *gwInfo, error) {
	gwsV4 := &gwInfo{}
	gwsV6 := &gwInfo{}

	for _, ipc := range result.IPs {
		var gws *gwInfo
		defaultNet := &net.IPNet{}
		switch {
		case ipc.Address.IP.To4() != nil:
			gws = gwsV4
			gws.family = netlink.FAMILY_V4
			defaultNet.IP = net.IPv4zero
		case len(ipc.Address.IP) == net.IPv6len:
			gws = gwsV6
			gws.family = netlink.FAMILY_V6
			defaultNet.IP = net.IPv6zero
		default:
			return nil, nil, fmt.Errorf("Unknown IP object: %v", ipc)
		}
		defaultNet.Mask = net.IPMask(defaultNet.IP)

		// All IPs currently refer to the container interface
		ipc.Interface = current.Int(2)

		// If not provided, calculate the gateway address corresponding
		// to the selected IP address
		if ipc.Gateway == nil && n.IsGW {
			ipc.Gateway = calcGatewayIP(&ipc.Address)
		}

		// Add a default route for this family using the current gateway address if necessary
		// TODO(jiang): 这里可能也是我要添加默认路由的地方
		if n.IsDefaultGW && !gws.defaultRouteFound {
			for _, route := range result.Routes {
				if route.GW != nil && defaultNet.String() == route.Dst.String() {
					gws.defaultRouteFound = true
					break
				}
			}
			if !gws.defaultRouteFound {
				result.Routes = append(
					result.Routes,
					&types.Route{Dst: *defaultNet, GW: ipc.Gateway},
				)
				gws.defaultRouteFound = true
			}
		}

		// Append this gateway address to the list of gateways
		if n.IsGW {
			gw := net.IPNet{
				IP:   ipc.Gateway,
				Mask: ipc.Address.Mask,
			}
			gws.gws = append(gws.gws, gw)
		}
	}

	return gwsV4, gwsV6, nil
}

func findPort(bridge, netNS, podNamespace, podName string) (string, error) {
	commands := []string{
		"--format=json", "--column=name", "find", "port",
		fmt.Sprintf("external-ids:netns=%s", normalizedNetNS(netNS)),
		fmt.Sprintf("external-ids:k8s_pod_namespace=%s", podNamespace),
		fmt.Sprintf("external-ids:k8s_pod_name=%s", podName),
	}

	out, err := exec.Command("ovs-vsctl", commands...).Output()
	if err != nil {
		return "", fmt.Errorf("failed to get OVS port with net namespace %q from bridge %q, err: %v",
			netNS, bridge, err)
	}

	dbData := struct {
		Data [][]string
	}{}
	if err = json.Unmarshal(out, &dbData); err != nil {
		return "", err
	}

	if len(dbData.Data) == 0 {
		// TODO: might make more sense to not return an error since CNI delete can be called multiple times
		return "", fmt.Errorf("OVS port for %s/%s was not found, OVS DB data: %v, output: %q",
			podNamespace, podName, dbData.Data, string(out))
	}

	portName := dbData.Data[0][0]
	return portName, nil
}

func delPort(bridge, port string) error {
	commands := []string{
		"--if-exists", "del-port", bridge, port,
	}

	out, err := exec.Command("ovs-vsctl", commands...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to delete OVS port %q on bridge %q, err: %v, output: %q",
			port, bridge, err, string(out))
	}

	return nil
}

func cmdAdd(args *skel.CmdArgs) error {

	// load net config
	netConf, cniVersion, err := loadNetConf(args.StdinData)
	if err != nil {
		return err
	}

	klog.Infof("stdin data: %s", string(args.StdinData))
	klog.Infof("args: %s", args.Args)
	klog.Infof("containerID: %s", args.ContainerID)
	klog.Infof("ifName: %s", args.IfName)
	klog.Infof("netns: %s", args.Netns)
	klog.Infof("path: %s", args.Path)

	bridge, err := getBridgeInterface()
	if err != nil {
		return fmt.Errorf("failed to setup bridge: %v", err)
	}

	netns, err := ns.GetNS(args.Netns)
	if err != nil {
		return fmt.Errorf("failed to open netns %q: %v", args.Netns, err)
	}
	defer netns.Close()

	hostInterface, containerInterface, err := setupVeth(netns, args.IfName)
	if err != nil {
		return err
	}

	podNamespace, podName, err := getPodInfo()

	err = addPort(netConf.BridgeName, hostInterface.Name, containerInterface.Mac, args.Netns, podNamespace, podName)
	if err != nil {
		return err
	}

	hostLink, err := netlink.LinkByName(hostInterface.Name)

	if err := netlink.LinkSetUp(hostLink); err != nil {
		return err
	}

	result := &current.Result{CNIVersion: cniVersion, Interfaces: []*current.Interface{bridge, hostInterface, containerInterface}}

	// 从全局拿到应该可分配的 ip addr
	ipamResult, err := getAvailableIP()
	if err != nil {
		return err
	}

	result.IPs = ipamResult.IPs
	result.Routes = ipamResult.Routes

	if len(result.IPs) == 0 {
		return errors.New("no IPs available")
	}

	// Configure the container hardware address and IP address
	if err := netns.Do(func(_ ns.NetNS) error {
		contVeth, err := net.InterfaceByName(args.IfName)
		if err != nil {
			return err
		}

		klog.Infof("ifname: %v, ipam result: %v, interface index: %v", args.IfName, result, *result.IPs[0].Interface)

		// Add the IP to the interface
		if err := ipam.ConfigureIface(args.IfName, result); err != nil {
			return err
		}

		for _, ipc := range result.IPs {
			if ipc.Version == "4" {
				_ = arping.GratuitousArpOverIface(ipc.Address.IP, *contVeth)
			}
		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed to configure container addresses: %v", err)
	}

	// Refetch the bridge since its MAC address may change when the first
	// veth is added or after its IP address is set
	br, err := bridgeByName(netConf.BridgeName)
	if err != nil {
		return err
	}
	bridge.Mac = br.Attrs().HardwareAddr.String()

	result.DNS = netConf.DNS

	return types.PrintResult(result, cniVersion)
}

func cmdDel(args *skel.CmdArgs) error {
	netConf, _, err := loadNetConf(args.StdinData)
	if err != nil {
		return err
	}

	podNamespace, podName, err := getPodInfo()
	if err != nil {
		return err
	}

	if args.Netns != "" {
		portName, err := findPort(netConf.BridgeName, args.Netns, podNamespace, podName)
		if err != nil {
			return err
		}

		err = delPort(netConf.BridgeName, portName)
		if err != nil {
			return err
		}
	}

	if netConf.IPAM.Type != "" {
		if err := ipam.ExecDel(netConf.IPAM.Type, args.StdinData); err != nil {
			return err
		}
	}

	if args.Netns == "" {
		return nil
	}

	// There is a netns so try to clean up. Delete can be called multiple times
	// so don't return an error if the device is already removed.
	// If the device isn't there then don't try to clean up IP masq either.
	err = ns.WithNetNSPath(args.Netns, func(_ ns.NetNS) error {
		var err error
		_, err = ip.DelLinkByNameAddr(args.IfName)
		if err != nil && err == ip.ErrLinkNotFound {
			return nil
		}
		return err
	})
	if err != nil {
		return err
	}
	return nil
}
