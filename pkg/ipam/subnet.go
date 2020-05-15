package ipam

import (
	"net"
	"sync"
)

// Subnet 子网
type Subnet struct {
	Name           string
	mutex          sync.RWMutex
	CIDR           *net.IPNet
	FreeIPList     IPRangeList // 自由分配的 ip 池
	ReservedIPList IPRangeList // 固定分配的 ip 池
	PodToIP        map[string]IP
	IPToPod        map[IP]string
	PodToMac       map[string]string
	MacToPod       map[string]string
}

// NewSubnet 构造 subnet
func NewSubnet(name, cidrStr string, excludeIps []IP) (*Subnet, error) {
	_, cidr, err := net.ParseCIDR(cidrStr)
	if err != nil {
		return nil, ErrInvalidCIDR
	}

	firstIP := GetFirstIP(cidrStr)
	lastIP := GetLastIP(cidrStr)

	subnet := &Subnet{
		Name:           name,
		mutex:          sync.RWMutex{},
		CIDR:           cidr,
		FreeIPList:     []*IPRange{{Start: IP(firstIP), End: IP(lastIP)}},
		ReservedIPList: GetIPRangeList(excludeIps),
		PodToIP:        make(map[string]IP),
		IPToPod:        make(map[IP]string),
		PodToMac:       make(map[string]string),
		MacToPod:       make(map[string]string),
	}

	// 从 FreeIPList 中去除 ReservedIPList，形成实际可分配的 ip pool
	subnet.splitFreeWithReserve()
	return subnet, nil
}

// splitFreeWithReserve 根据保留ip的范围，对 free ip 进行分割
func (subnet *Subnet) splitFreeWithReserve() {
	for _, reserve := range subnet.ReservedIPList {
		freeList := IPRangeList{}
		for _, free := range subnet.FreeIPList {
			if iprl := splitRange(free, reserve); iprl != nil {
				freeList = append(freeList, iprl...)
			}
		}
		subnet.FreeIPList = freeList
	}
}

// GetRandomMac 获取随机 mac 地址，如果当前 pod 分配过，则会分配之前的 mac
func (subnet *Subnet) GetRandomMac(pod string) string {
	if mac, ok := subnet.PodToMac[pod]; ok {
		return mac
	}

	mac := GenerateMac()
	subnet.PodToMac[pod] = mac
	subnet.MacToPod[mac] = pod

	return mac
}

// GetStaticMac 获取静态 mac,如果该 mac 已使用，则会返回错误
func (subnet *Subnet) GetStaticMac(pod, mac string) error {
	if _, ok := subnet.MacToPod[mac]; ok {
		return ErrConflict
	}
	subnet.PodToMac[pod] = mac
	subnet.MacToPod[mac] = pod
	return nil
}

// GetRandomAddress 随机获取IP地址，如果该pod分配过ip，则返回之前分配的ip
func (subnet *Subnet) GetRandomAddress(pod string) (IP, string, error) {
	subnet.mutex.Lock()
	defer subnet.mutex.Unlock()
	if ip, ok := subnet.PodToIP[pod]; ok {
		return ip, subnet.PodToMac[pod], nil
	}

	if len(subnet.FreeIPList) == 0 {
		return "", "", ErrNoAvailable
	}

	// 从 freelist 中取出最前面的 ip 分配
	freeRange := subnet.FreeIPList[0]
	ip := freeRange.Start
	mac := subnet.GetRandomMac(pod)
	if freeRange.Start == freeRange.End {
		subnet.FreeIPList = subnet.FreeIPList[1:]
	} else {
		subnet.FreeIPList[0].Start = subnet.FreeIPList[0].Start.Add(1)
	}

	subnet.IPToPod[ip] = pod
	subnet.PodToIP[pod] = ip

	return ip, mac, nil
}

// GetStaticAddress 获取静态的IP地址
func (subnet *Subnet) GetStaticAddress(pod string, ip IP, mac string) (IP, string, error) {
	subnet.mutex.Lock()
	defer subnet.mutex.Unlock()

	if !subnet.CIDR.Contains(net.ParseIP(string(ip))) {
		return ip, mac, ErrOutOfRange
	}

	if mac == "" {
		if m, ok := subnet.PodToMac[mac]; ok {
			mac = m
		} else {
			mac = subnet.GetRandomMac(pod)
		}
	} else {
		if err := subnet.GetStaticMac(pod, mac); err != nil {
			return ip, mac, err
		}
	}

	if existPod, ok := subnet.IPToPod[ip]; ok {
		if existPod != pod {
			return ip, mac, ErrConflict
		}

		return ip, mac, nil
	}

	if subnet.ReservedIPList.Contains(ip) {
		subnet.PodToIP[pod] = ip
		subnet.IPToPod[ip] = pod
		return ip, mac, nil
	}

	if split, newFreeList := SplitIPRangeList(subnet.FreeIPList, ip); split {
		subnet.FreeIPList = newFreeList
		subnet.PodToIP[pod] = ip
		subnet.IPToPod[ip] = pod
		return ip, mac, nil
	} else {
		return ip, mac, ErrNoAvailable
	}
}

// ReleaseAddress 释放地址
func (subnet *Subnet) ReleaseAddress(pod string) (IP, string) {
	subnet.mutex.Lock()
	defer subnet.mutex.Unlock()
	ip, mac := IP(""), ""
	var ok bool
	if ip, ok = subnet.PodToIP[pod]; ok {
		delete(subnet.PodToIP, pod)
		delete(subnet.IPToPod, ip)
		if mac, ok = subnet.PodToMac[pod]; ok {
			delete(subnet.PodToMac, pod)
			delete(subnet.MacToPod, mac)
		}

		if !subnet.CIDR.Contains(net.ParseIP(string(ip))) {
			return ip, mac
		}

		if subnet.ReservedIPList.Contains(ip) {
			return ip, mac
		}

		if merged, newFreeList := MergeIPRangeList(subnet.FreeIPList, ip); merged {
			subnet.FreeIPList = newFreeList
			return ip, mac
		}
	}
	return ip, mac
}

// ContainAddress 检查IP是否已分配给Pod
func (subnet *Subnet) ContainAddress(address IP) bool {
	subnet.mutex.RLock()
	defer subnet.mutex.RUnlock()
	if _, ok := subnet.IPToPod[address]; ok {
		return true
	}
	return false
}

// GetPodAddress 获取Pod地址
func (subnet *Subnet) GetPodAddress(podName string) (IP, string, bool) {
	ip, mac := subnet.PodToIP[podName], subnet.PodToMac[podName]
	return ip, mac, (ip != "" && mac != "")
}
