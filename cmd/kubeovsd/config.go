package main

type KubeOVSDConfig struct {
	NIC         string             `yaml:"nic"`         // 要桥接的物理网卡
	ClusterCIDR string             `yaml:"clusterCIDR"` // 集群的 CIDR
	EtcdServer  []string           `yaml:"etcdServer"`  // etcd 服务的 endpoints
	IPAM        KubeOVSDIPAMConfig `yaml:"ipam"`        // ipam 配置
}

type KubeOVSDIPAMConfig struct {
	RangeStart string `yaml:"rangeStart"`
	RangeEnd   string `yaml:"rangeEnd"`
	SubnetIP   string `yaml:"subnetIP"`
	SubnetMask string `yaml:"subnetMask"`
	Gateway    string `yaml:"gateway"`
}
