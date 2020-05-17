package etcd

type EtcdConfig struct {
	Endpoints []string
	DialTimeout int
}