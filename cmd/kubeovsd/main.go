package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"

	"github.com/containernetworking/cni/pkg/types"
	"github.com/joyme123/kubeovs-l2/pkg/daemon"
	"github.com/joyme123/kubeovs-l2/pkg/ipametcd/backend/allocator"
	"github.com/vishvananda/netlink"
	"k8s.io/klog"
)

const (
	cniConfigPath = "/etc/cni/net.d/10-kubeovsl2.conf"
	bridgeName    = "kubeovs-br"
)

var (
	defaultNIC              = "enp0s8"
	defaultControllerTarget = "tcp:127.0.0.1:6653" // 暂时没用
	defaultClusterCIDR      = "192.168.50.0/24"
)

func setupOVSBridgeIfNotExists() error {
	command := []string{
		"--may-exist", "add-br", bridgeName,
	}

	out, err := exec.Command("ovs-vsctl", command...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to setup OVS bridge %q, err: %v, output: %q",
			bridgeName, err, string(out))
	}

	br, err := netlink.LinkByName(bridgeName)
	if err != nil {
		return fmt.Errorf("could not lookup %q: %v", bridgeName, err)
	}

	if err := netlink.LinkSetUp(br); err != nil {
		return fmt.Errorf("failed to bring bridge %q up: %v", bridgeName, err)
	}

	return nil
}

func setupPhysicalPortToBr() error {
	command := []string{
		"--may-exist", "add-port", bridgeName, defaultNIC,
	}

	out, err := exec.Command("ovs-vsctl", command...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to setup cluster-wide port, err: %v, output: %q", err, out)
	}

	physicalPort, err := netlink.LinkByName(defaultNIC)
	if err != nil {
		return fmt.Errorf("could not lookup %q: %v", defaultNIC, err)
	}

	if err := netlink.LinkSetUp(physicalPort); err != nil {
		return fmt.Errorf("failed to bring bridge %q up: %v", physicalPort, err)
	}

	return nil
}

func changeRoute(clusterCIDR string, link netlink.Link) error {
	_, clusterIPNet, err := net.ParseCIDR(clusterCIDR)
	if err != nil {
		return err
	}

	r := &netlink.Route{
		LinkIndex: link.Attrs().Index,
		Scope:     netlink.SCOPE_LINK,
		Dst:       clusterIPNet,
	}

	return netlink.RouteReplace(r)
}

func installCNIConf() error {
	conf := `{
	"name": "kubeovs-l2",
	"type": "kubeovs-l2",
	"bridge": "kubeovs-br",
	"isGateway": false,
	"isDefaultGateway": false
}`

	return ioutil.WriteFile(cniConfigPath, []byte(conf), 0644)
}

func main() {
	// 在启动的时候负责启动节点上的 ovs，创建 br， 修改 route 指向
	klog.InitFlags(flag.CommandLine)
	klog.Info("starting kubeovs")

	var configPath string
	flag.StringVar(&configPath, "c", "/etc/kubeovs-config", "specify config file path")
	flag.Parse()

	f, err := os.Open(configPath)
	if err != nil {
		klog.Errorf("open config file %v error: %v", configPath, err)
		os.Exit(1)
	}

	data, err := ioutil.ReadAll(f)
	if err != nil {
		klog.Errorf("get config data error: %v", err)
		os.Exit(1)
	}

	var conf KubeOVSDConfig

	err = json.Unmarshal(data, &conf)
	if err != nil {
		klog.Errorf("unmarshal json error: %v", err)
		os.Exit(1)
	}

	// 从配置文件中读取配置
	defaultNIC = conf.NIC
	defaultClusterCIDR = conf.ClusterCIDR

	// 写入 cni 配置文件
	err = installCNIConf()
	if err != nil {
		klog.Errorf("failed to intall CNI: %v", err)
		os.Exit(1)
	}

	// 设置 ovs bridge
	err = setupOVSBridgeIfNotExists()
	if err != nil {
		klog.Errorf("failed to setup OVS bridge: %v", err)
		os.Exit(1)
	}

	klog.Infof("ovs bridge setup successfully")

	// 将物理网卡和 bridge 连接
	err = setupPhysicalPortToBr()
	if err != nil {
		klog.Errorf("failed to add physical NIC to OVS bridge: %v", err)
		os.Exit(1)
	}

	klog.Infof("add physical NIC to OVS bridge successfully")

	bridgeLink, err := netlink.LinkByName(bridgeName)
	if err != nil {
		klog.Errorf("failed to get bridge %q, err: %v", bridgeName, err)
		os.Exit(1)
	}

	// 获取物理网卡的 ip，赋值到 bridge 上
	physicalLink, err := netlink.LinkByName(defaultNIC)
	if err != nil {
		klog.Errorf("failed to get physical nic %q, err: %v", defaultNIC, err)
		os.Exit(1)
	}

	addrs, err := netlink.AddrList(physicalLink, netlink.FAMILY_V4)
	if err != nil {
		klog.Warningf("failed to get physical nic %q ip addr, err: %v", defaultNIC, err)
	}

	// 只有在拿到了物理网卡上的 ip 地址后,才会执行分配到 ovs bridge 的操作
	// 否则默认该操作已经执行过了
	if len(addrs) > 0 {
		// TODO: 先假设只有一个
		err = netlink.AddrDel(physicalLink, &addrs[0])
		if err != nil {
			klog.Errorf("failed to del physical nic %q ip addr, err: %v", defaultNIC, err)
			os.Exit(1)
		}

		err = netlink.AddrAdd(bridgeLink, &netlink.Addr{IPNet: addrs[0].IPNet})
		if err != nil {
			klog.Errorf("failed to add bridge %q ip addr, err: %v", bridgeName, err)
			os.Exit(1)
		}
	}

	// 将 bridge 启动
	err = netlink.LinkSetUp(bridgeLink)
	if err != nil {
		klog.Errorf("failed to set bridge %q up, err: %v", bridgeName, err)
		os.Exit(1)
	}

	klog.Infof("set ovs bridge up successfully")

	// 将指向 physical nic 的路由修改到 bridge
	err = changeRoute(defaultClusterCIDR, bridgeLink)
	if err != nil {
		klog.Errorf("failed to change route, cidr: %v, link: %v", defaultClusterCIDR, bridgeLink)
		os.Exit(1)
	}

	klog.Infof("change ip route successfully")

	// 启动 ipam server
	ipamConf := allocator.IPAMConfig{
		Name:       "kubeovs-net",
		Type:       "kubeovs-l2",
		EtcdServer: conf.EtcdServer,
		Ranges: []allocator.RangeSet{
			[]allocator.Range{
				{
					RangeStart: net.ParseIP(conf.IPAM.RangeStart).To4(),
					RangeEnd:   net.ParseIP(conf.IPAM.RangeEnd).To4(),
					Subnet: types.IPNet(net.IPNet{
						IP:   net.ParseIP(conf.IPAM.SubnetIP).To4(),
						Mask: net.IPMask(net.ParseIP(conf.IPAM.SubnetMask).To4()),
					}),
					Gateway: net.ParseIP(conf.IPAM.Gateway).To4(),
				},
			},
		},
	}

	ipamServer, err := daemon.NewIpamServer(&ipamConf)
	if err != nil {
		klog.Errorf("new ipam server error: %v", err)
		os.Exit(1)
	}

	klog.Infof("run ipam server")

	err = ipamServer.Run()
	if err != nil {
		klog.Errorf("run ipam server error: %v", err)
		os.Exit(1)
	}
}
