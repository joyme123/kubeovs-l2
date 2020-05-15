package ipam

import (
	"bytes"
	"crypto/rand"
	"math/big"
	mathrand "math/rand"
	"net"
	"time"
)

const (
	DefaultMacPrefix = "02:42:AC"
)

var hexMap map[byte]byte

func init() {
	hexMap = make(map[byte]byte)
	for i := 0; i <= 9; i++ {
		hexMap[byte(i)] = byte('1' - 1 + i)
	}
	hexMap[10] = 'A'
	hexMap[11] = 'B'
	hexMap[12] = 'C'
	hexMap[13] = 'D'
	hexMap[14] = 'E'
	hexMap[15] = 'F'
}

// BytesToUint32 4字节 byte 数组转 Uint32
func BytesToUint32(bs []byte) uint32 {
	if len(bs) != 4 {
		return 0
	}
	return uint32(bs[0])<<24 | uint32(bs[1])<<16 | uint32(bs[2])<<8 | uint32(bs[3])
}

// Uint32ToBytes uint32 转换为 byte 数组
func Uint32ToBytes(n uint32) []byte {
	return []byte{byte(n >> 24), byte(n >> 16), byte(n >> 8), byte(n)}
}

// IPToUint32 IP 转成 Uint32
func IPToUint32(ipStr IP) uint32 {
	ip := net.ParseIP(string(ipStr))

	return BytesToUint32(ip.To4())
}

// Uint32ToIP uint32 转成字符串 ip
func Uint32ToIP(num uint32) IP {
	return IP(net.IP(Uint32ToBytes(num)).String())
}

// GetFirstIP 获取 cidr 中的第一个 ip 地址
func GetFirstIP(cidr string) string {
	ip, ipnet, _ := net.ParseCIDR(cidr)

	// TODO: 暂时只考虑 ipv4
	ones, bits := ipnet.Mask.Size()

	start := BytesToUint32(ip.To4()) & ((^uint32(0)) << (bits - ones))

	bs := Uint32ToBytes(start + 1)
	first := net.IP(bs)
	return first.String()
}

// GetLastIP 获取 cidr 中的第一个 ip 地址
func GetLastIP(cidr string) string {
	// 192.168.33.0/25 => 11000000 10101000 00100001 00000000 & 11111111 11111111 11111111 10000000
	// => 11000000 10101000 00100001 0(xxxxxxx)
	ip, ipnet, _ := net.ParseCIDR(cidr)

	// TODO: 暂时只考虑 ipv4
	ones, bits := ipnet.Mask.Size()

	end := BytesToUint32(ip.To4())&((^uint32(0))<<(bits-ones)) | (^uint32(0) >> ones)

	bs := Uint32ToBytes(end - 1)
	lastIP := net.IP(bs)
	return lastIP.String()
}

func splitRange(a, b *IPRange) IPRangeList {
	if b.End.LessThan(a.Start) || b.Start.GreaterThan(a.End) {
		return IPRangeList{a}
	}

	if (a.Start.Equal(b.Start) || a.Start.GreaterThan(b.Start)) &&
		(a.End.Equal(b.End) || a.End.LessThan(b.End)) {
		return nil
	}

	if (a.Start.Equal(b.Start) || a.Start.GreaterThan(b.Start)) &&
		a.End.GreaterThan(b.End) {
		ipr := IPRange{Start: b.End.Add(1), End: a.End}
		return IPRangeList{&ipr}
	}

	if (a.End.Equal(b.End) || a.End.LessThan(b.End)) && a.Start.LessThan(b.Start) {
		ipr := IPRange{Start: a.Start, End: b.Start.Add(1)}
		return IPRangeList{&ipr}
	}

	ipr1 := IPRange{Start: a.Start, End: b.Start.Sub(1)}
	ipr2 := IPRange{Start: b.End.Add(1), End: a.End}
	return IPRangeList{&ipr1, &ipr2}
}

func SplitIPRangeList(iprl IPRangeList, ip IP) (bool, IPRangeList) {
	newIPRangeList := []*IPRange{}
	split := false
	for _, ipr := range iprl {
		if split {
			newIPRangeList = append(newIPRangeList, ipr)
			continue
		}

		if ipr.Start.Equal(ipr.End) && ipr.Start.Equal(ip) {
			split = true
			continue
		}

		if ipr.Start.Equal(ip) {
			newIPRangeList = append(newIPRangeList, &IPRange{Start: ip.Add(1), End: ipr.End})
			split = true
			continue
		}

		if ipr.Contains(ip) {
			newIpr1 := IPRange{Start: ipr.Start, End: ip.Sub(1)}
			newIpr2 := IPRange{Start: ip.Add(1), End: ipr.End}
			newIPRangeList = append(newIPRangeList, &newIpr1, &newIpr2)
			split = true
			continue
		}

		newIPRangeList = append(newIPRangeList, ipr)
	}

	return split, newIPRangeList
}

// MergeIPRangeList 合并
func MergeIPRangeList(iprl IPRangeList, ip IP) (bool, IPRangeList) {
	insertIPRangeList := []*IPRange{}
	inserted := false
	if iprl.Contains(ip) {
		return false, nil
	}

	for _, ipr := range iprl {
		if inserted || ipr.Start.LessThan(ip) {
			insertIPRangeList = append(insertIPRangeList, ipr)
			continue
		}

		if ipr.Start.GreaterThan(ip) {
			insertIPRangeList = append(insertIPRangeList, &IPRange{Start: ip, End: ip}, ipr)
			inserted = true
			continue
		}
	}

	if !inserted {
		newIpr := IPRange{Start: ip, End: ip}
		insertIPRangeList = append(insertIPRangeList, &newIpr)
	}

	mergedIPRangeList := []*IPRange{}
	for _, ipr := range insertIPRangeList {
		if len(mergedIPRangeList) == 0 {
			mergedIPRangeList = append(mergedIPRangeList, ipr)
			continue
		}

		if mergedIPRangeList[len(mergedIPRangeList)-1].End.Add(1).Equal(ipr.Start) {
			mergedIPRangeList[len(mergedIPRangeList)-1].End = ipr.End
		} else {
			mergedIPRangeList = append(mergedIPRangeList, ipr)
		}
	}

	return true, mergedIPRangeList
}

func RandInt64(max int64) int64 {
	n, err := rand.Int(rand.Reader, new(big.Int).SetInt64(max))
	if err != nil {
		// 出错，使用 math/rand 生成
		mathrand.Seed(time.Now().Unix())
		return mathrand.Int63() % max
	}

	return n.Int64()
}

func byteToHexBytes(b byte) []byte {
	bs := make([]byte, 2)
	bs[0] = hexMap[b>>4]
	bs[1] = hexMap[b&0x0f]
	return bs
}

// GenerateMac 生成 mac 地址
// mac 地址共 48 位，第1Bit为广播地址(0)/群播地址(1)，第2Bit为广域地址(0)/区域地址(1)。前3~24位由IEEE决定如何分配给每一家制造商，且不重复
// 比如: 02:42:AC:11:00:03
func GenerateMac() string {
	n1 := byte(RandInt64(256))
	n2 := byte(RandInt64(256))
	n3 := byte(RandInt64(256))

	buf := bytes.NewBuffer([]byte{})

	buf.WriteString(DefaultMacPrefix)
	buf.WriteByte(':')
	buf.Write(byteToHexBytes(n1))
	buf.WriteByte(':')
	buf.Write(byteToHexBytes(n2))
	buf.WriteByte(':')
	buf.Write(byteToHexBytes(n3))

	return buf.String()
}
