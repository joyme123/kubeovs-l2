package ipam

import (
	"net"
	"sort"
)

// IP 类似于 172.10.0.1
type IP string

// Add 添加计算得到ip
func (ip IP) Add(n uint32) IP {
	num := IPToUint32(ip)
	return Uint32ToIP(num + n)
}

// Sub 减少计算得到ip
func (ip IP) Sub(n uint32) IP {
	num := IPToUint32(ip)
	return Uint32ToIP(num - n)
}

// LessThan 当前ip比b ip小，返回true
func (ip IP) LessThan(b IP) bool {
	aIP := net.ParseIP(string(ip))
	bIP := net.ParseIP(string(b))

	anum := BytesToUint32([]byte(aIP.To4()))
	bnum := BytesToUint32([]byte(bIP.To4()))

	return anum < bnum
}

// GreaterThan 当前ip比b ip大，返回true
func (ip IP) GreaterThan(b IP) bool {
	aIP := net.ParseIP(string(ip))
	bIP := net.ParseIP(string(b))

	anum := BytesToUint32([]byte(aIP.To4()))
	bnum := BytesToUint32([]byte(bIP.To4()))

	return anum > bnum
}

func (ip IP) Equal(b IP) bool {
	aIP := net.ParseIP(string(ip))
	bIP := net.ParseIP(string(b))

	anum := BytesToUint32([]byte(aIP.To4()))
	bnum := BytesToUint32([]byte(bIP.To4()))

	return anum == bnum
}

// IPList ip 列表
type IPList []IP

// Less 比较大小
func (ips IPList) Less(i, j int) bool {
	return ips[i].LessThan(ips[j])
}

// Swap 交换
func (ips IPList) Swap(i, j int) {
	ips[i], ips[j] = ips[j], ips[i]
}

// Len 长度
func (ips IPList) Len() int {
	return len(ips)
}

// IPRange ip地址范围
type IPRange struct {
	Start IP // 包含左边
	End   IP // 包含右边
}

// Contains 是否包含
func (r *IPRange) Contains(ip IP) bool {
	return !r.Start.GreaterThan(ip) && !ip.GreaterThan(r.End)
}

// IPRangeList 范围列表
type IPRangeList []*IPRange

// Contains 是否包含
func (iprl IPRangeList) Contains(ip IP) bool {
	for _, ipr := range iprl {
		if ipr.Contains(ip) {
			return true
		}
	}

	return false
}

// GetIPRangeList 将 IP 数组转换成 IPRange 列表
func GetIPRangeList(ips []IP) []*IPRange {
	list := make(IPRangeList, 0)

	if len(ips) == 0 {
		return list
	}

	sort.Sort(IPList(ips))

	var pre IP
	var rg *IPRange
	rg = &IPRange{}

	for _, ip := range ips {
		if pre == "" {
			rg.Start = ip
		} else {
			if pre.Add(1) != ip {
				// 不连续的
				rg.End = pre
				list = append(list, rg)
				rg = &IPRange{}
				rg.Start = ip
			}
		}

		pre = ip
	}

	rg.End = pre
	list = append(list, rg)

	return list
}
