package ipam

import (
	"errors"
	"sync"
)

var (
	// ErrOutOfRange 分配的ip超出了范围
	ErrOutOfRange = errors.New("AddressOutOfRange")
	// ErrConflict 分配的ip冲突
	ErrConflict = errors.New("ConflictError")
	// ErrNoAvailable 不可用错误
	ErrNoAvailable = errors.New("NoAvailableAddress")
	// ErrInvalidCIDR 无效的 CIDR
	ErrInvalidCIDR = errors.New("CIDRInvalid")
)

// IPAM 负责 ip 地址分配
type IPAM struct {
	Mutex   sync.RWMutex
	Subnets map[string]*Subnet
}
