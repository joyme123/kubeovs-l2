package daemon

import (
	"net/http"

	"github.com/joyme123/kubeovs-l2/pkg/ipametcd/backend/allocator"
	etcd "github.com/joyme123/kubeovs-l2/pkg/ipametcd/backend/etcd"
	"go.etcd.io/etcd/clientv3"
)

type IpamServer struct {
	allocs []*allocator.IPAllocator
}

func NewIpamServer(ipamConf *allocator.IPAMConfig) *IpamServer {
	conf := clientv3.Config{
		Endpoints:   ipamConf.EtcdServer,
		DialTimeout: 5,
	}
	store, err := etcd.New(conf, ipamConf.Name, ipamConf.PrefixKey)
	if err != nil {
		return err
	}

	allocs := []*allocator.IPAllocator{}
	for idx, rangeset := range ipamConf.Ranges {
		allocator := allocator.NewIPAllocator(&rangeset, store, idx)
	}

	return &IpamServer{}
}

func (s *IpamServer) Run() {
	mux := http.NewServeMux()
	mux.HandleFunc("/add", s.Add)
	mux.HandleFunc("/del", s.Del)
	mux.HandleFunc("/check", s.Check)
}

func (s *IpamServer) Add(w http.ResponseWriter, req *http.Request) {

}

func (s *IpamServer) Del(w http.ResponseWriter, req *http.Request) {

}

func (s *IpamServer) Check(w http.ResponseWriter, req *http.Request) {

}
