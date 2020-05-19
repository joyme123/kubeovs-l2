package daemon

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/containernetworking/cni/pkg/types/current"
	"github.com/joyme123/kubeovs-l2/pkg/ipametcd/backend/allocator"
	etcd "github.com/joyme123/kubeovs-l2/pkg/ipametcd/backend/etcd"
	"go.etcd.io/etcd/clientv3"
	"k8s.io/klog"
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
		DialTimeout: 5 * time.Second,
	}
	store, err := etcd.New(conf, ipamConf.Name, ipamConf.PrefixKey)
	if err != nil {
		return nil, fmt.Errorf("new etcd store error: %v", err)
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

	_, err := os.Stat(DefaultKubeOVSDirectory)
	if os.IsNotExist(err) {
		// 不存在
		err = os.MkdirAll(DefaultKubeOVSDirectory, os.ModePerm)
		if err != nil {
			return fmt.Errorf("mkdir %v error: %v", DefaultKubeOVSDirectory, err)
		}
	}

	sockfile := GenerateSocketPath()

	syscall.Unlink(sockfile)

	server := http.Server{
		Handler: mux,
	}

	unixListener, err := net.Listen("unix", sockfile)
	if err != nil {
		return fmt.Errorf("listen unix socket %v error: %v", sockfile, err)
	}

	err = server.Serve(unixListener)
	if err != nil {
		return fmt.Errorf("start ipam server error: %v", err)
	}

	return nil
}

// Add 从 ipam server 分配 ip 地址
func (s *IpamServer) Add(w http.ResponseWriter, req *http.Request) {
	req.ParseForm()
	reqIPs := req.Form.Get("ips")
	contID := req.Form.Get("containerID")
	ifName := req.Form.Get("ifName")

	klog.Infof("request ips: %v, containerID: %v, ifName: %v", reqIPs, contID, ifName)

	ips := strings.Split(reqIPs, ",")
	requestedIPs := map[string]net.IP{}
	allocatedIPs := make([]*current.IPConfig, 0)

	for _, ipStr := range ips {
		ip := net.ParseIP(ipStr)
		klog.Infof("request ip: %v", ipStr)
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
		klog.Infof("allocs requestedIP: %v", requestedIP)
		ipConf, err := s.allocs[i].Get(contID, ifName, requestedIP)
		if err != nil {
			for _, alloc := range s.allocs {
				_ = alloc.Release(contID, ifName)
			}
			errInfo := fmt.Sprintf("failed to allocate for range %d: %v", i, err)
			klog.Errorf(errInfo)
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
		klog.Errorf(errstr)
		writeErrorResponse(w, http.StatusBadRequest, "errInfo")
		return
	}

	result := &current.Result{}
	if s.ipamConf.ResolvConf != "" {
		dns, err := parseResolvConf(s.ipamConf.ResolvConf)
		if err != nil {
			klog.Errorf("resolve ipam config error:", err)
			return
		}
		result.DNS = *dns
	}
	result.IPs = allocatedIPs
	result.Routes = s.ipamConf.Routes

	writeOkResponse(w, result)
}

// Del 从 ipam server 释放 ip
func (s *IpamServer) Del(w http.ResponseWriter, req *http.Request) {
	req.ParseForm()
	contID := req.Form.Get("containerID")
	ifName := req.Form.Get("ifName")

	var errors []string
	for i := range s.allocs {
		err := s.allocs[i].Release(contID, ifName)
		klog.Infof("release for container: %v, interface: %v", contID, ifName)
		if err != nil {
			errors = append(errors, err.Error())
		}
	}

	if errors != nil {
		klog.Errorf(strings.Join(errors, ";"))
		writeErrorResponse(w, http.StatusBadRequest, "release ip failed")
		return
	}

	data := NewResponseData()
	writeOkResponse(w, data)
}

// Check 检查 ip
func (s *IpamServer) Check(w http.ResponseWriter, req *http.Request) {
	req.ParseForm()
	contID := req.Form.Get("containerID")
	ifName := req.Form.Get("ifName")
	containerIPFound := s.store.FindByID(contID, ifName)
	if !containerIPFound {
		errInfo := fmt.Sprintf("host-local: Failed to find address added by container %v", contID)
		klog.Errorf(errInfo)
		writeErrorResponse(w, http.StatusBadRequest, errInfo)
		return
	}
	data := NewResponseData()
	writeOkResponse(w, data)
}
