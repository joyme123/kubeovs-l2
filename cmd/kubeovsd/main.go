package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"

	"github.com/vishvananda/netlink"
	"k8s.io/klog"
)

const (
	cniConfigPath           = "/etc/cni/net.d/10-kubeovsl2.json"
	bridgeName              = "kubeovs-br"
	defaultNIC              = "enp0s8"
	defaultControllerTarget = "tcp:127.0.0.1:6653"
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

	// 写入 cni 配置文件
	err := installCNIConf()
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

	// 将物理网卡和 bridge 连接
	err = setupPhysicalPortToBr()
	if err != nil {
		klog.Errorf("failed to add physical NIC to OVS bridge: %v", err)
		os.Exit(1)
	}

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
		klog.Errorf("failed to get physical nic %q ip addr, err: %v", defaultNIC, err)
		os.Exit(1)
	}

	if len(addrs) == 0 {
		klog.Errorf("failed to get physical nic %q ip addr, err: %v", defaultNIC, err)
		os.Exit(1)
	}

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

	// 将 bridge 启动
	err = netlink.LinkSetUp(bridgeLink)
	if err != nil {
		klog.Errorf("failed to set bridge %q up, err: %v", bridgeName, err)
		os.Exit(1)
	}

	// 将指向 physical nic 的路由修改到 bridge
	err = changeRoute(defaultClusterCIDR, bridgeLink)

}