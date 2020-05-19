package daemon

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"syscall"

	"github.com/containernetworking/cni/pkg/types/current"
	"github.com/joyme123/kubeovs-l2/pkg/ipametcd/backend/allocator"
	etcd "github.com/joyme123/kubeovs-l2/pkg/ipametcd/backend/etcd"
	log "github.com/sirupsen/logrus"
	"go.etcd.io/etcd/clientv3"
)

const (
	// DefaultKubeOVSDirectory 默认目录地址
	DefaultKubeOVSDirectory string = "/var/run/kubeovs/"
)

// IpamServer 服务
type IpamServer struct {
	allocs   []*allocator.IPAllocator
	ipamConf *allocator.IPAMConfig
	store    *etcd.Store
}

// NewIpamServer 新建 ipam server
func NewIpamServer(ipamConf *allocator.IPAMConfig) (*IpamServer, error) {
	conf := clientv3.Config{
		Endpoints:   ipamConf.EtcdServer,
		DialTimeout: 5,
	}
	store, err := etcd.New(conf, ipamConf.Name, ipamConf.PrefixKey)
	if err != nil {
		return nil, err
	}

	// 每个 range 都有自己的 allocator
	allocs := []*allocator.IPAllocator{}
	for idx, rangeset := range ipamConf.Ranges {
		allocator := allocator.NewIPAllocator(&rangeset, store, idx)
		allocs = append(allocs, allocator)
	}

	return &IpamServer{allocs: allocs, ipamConf: ipamConf, store: store}, nil
}

// Run 运行 ipam server
func (s *IpamServer) Run() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/add", s.Add)
	mux.HandleFunc("/del", s.Del)
	mux.HandleFunc("/check", s.Check)

	sockfile := DefaultKubeOVSDirectory + "kubeovs.sock"

	syscall.Unlink(sockfile)

	err := http.ListenAndServe(sockfile, mux)
	if err != nil {
		return fmt.Errorf("start ipam server error: %v", err)
	}

	return nil
}

// Add 从 ipam server 分配 ip 地址
func (s *IpamServer) Add(w http.ResponseWriter, req *http.Request) {
	reqIPs := req.Form.Get("ips")
	contID := req.Form.Get("containerID")
	ifName := req.Form.Get("ifName")

	ips := strings.Split(reqIPs, ",")
	requestedIPs := map[string]net.IP{}
	allocatedIPs := make([]*current.IPConfig, 0)

	for _, ipStr := range ips {
		ip := net.ParseIP(ipStr)
		requestedIPs[ipStr] = ip
	}

	for i := range s.allocs {
		var requestedIP net.IP
		for k, ip := range requestedIPs {
			if s.allocs[i].Rangeset.Contains(ip) {
				requestedIP = ip
				delete(requestedIPs, k)
				break
			}
		}

		ipConf, err := s.allocs[i].Get(contID, ifName, requestedIP)
		if err != nil {
			for _, alloc := range s.allocs {
				_ = alloc.Release(contID, ifName)
			}
			errInfo := fmt.Sprintf("failed to allocate for range %d: %v", i, err)
			log.Errorf(errInfo)
			writeErrorResponse(w, http.StatusBadRequest, "errInfo")
			return
		}
		allocatedIPs = append(allocatedIPs, ipConf)
	}

	// if an IP was requested that wasn't fullfilled, fail
	if len(requestedIPs) != 0 {
		for _, alloc := range s.allocs {
			_ = alloc.Release(contID, ifName)
		}
		errstr := "failed to allocate all requested IPs:"
		for _, ip := range requestedIPs {
			errstr = errstr + " " + ip.String()
		}
		log.Errorf(errstr)
		writeErrorResponse(w, http.StatusBadRequest, "errInfo")
		return
	}

	result := &current.Result{}
	if s.ipamConf.ResolvConf != "" {
		dns, err := parseResolvConf(s.ipamConf.ResolvConf)
		if err != nil {
			log.Errorf("resolve ipam config error:", err)
			return
		}
		result.DNS = *dns
	}
	result.IPs = allocatedIPs
	result.Routes = s.ipamConf.Routes

	data := NewResponseData()
	data["result"] = result
	writeOkResponse(w, data)
}

// Del 从 ipam server 释放 ip
func (s *IpamServer) Del(w http.ResponseWriter, req *http.Request) {

	contID := req.Form.Get("containerID")
	ifName := req.Form.Get("ifName")

	var errors []string
	for i := range s.allocs {
		err := s.allocs[i].Release(contID, ifName)
		if err != nil {
			errors = append(errors, err.Error())
		}
	}

	if errors != nil {
		log.Errorf(strings.Join(errors, ";"))
		writeErrorResponse(w, http.StatusBadRequest, "release ip failed")
		return
	}

	data := NewResponseData()
	writeOkResponse(w, data)
}

// Check 检查 ip
func (s *IpamServer) Check(w http.ResponseWriter, req *http.Request) {
	contID := req.Form.Get("containerID")
	ifName := req.Form.Get("ifName")
	containerIPFound := s.store.FindByID(contID, ifName)
	if !containerIPFound {
		errInfo := fmt.Sprintf("host-local: Failed to find address added by container %v", contID)
		log.Errorf(errInfo)
		writeErrorResponse(w, http.StatusBadRequest, errInfo)
		return
	}
	data := NewResponseData()
	writeOkResponse(w, data)
}
