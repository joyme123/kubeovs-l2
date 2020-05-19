module github.com/joyme123/kubeovs-l2

go 1.13

replace (
	github.com/coreos/go-systemd => github.com/coreos/go-systemd/v22 v22.0.0
	google.golang.org/grpc => google.golang.org/grpc v1.26.0
)

require (
	github.com/alexflint/go-filemutex v1.1.0 // indirect
	github.com/argoproj/argo v2.5.2+incompatible // indirect
	github.com/containernetworking/cni v0.7.1
	github.com/containernetworking/plugins v0.8.6
	github.com/coreos/etcd v3.3.20+incompatible // indirect
	github.com/go-kit/kit v0.10.0 // indirect
	github.com/j-keck/arping v0.0.0-20160618110441-2cf9dc699c56
	github.com/onsi/ginkgo v1.12.0
	github.com/onsi/gomega v1.9.0
	github.com/pkg/errors v0.9.1 // indirect
	github.com/projectcalico/libcalico-go v1.7.3 // indirect
	github.com/sirupsen/logrus v1.6.0
	github.com/vishvananda/netlink v1.1.0
	go.etcd.io/etcd v0.0.0-20191023171146-3cf2f69b5738
	golang.org/x/exp v0.0.0-20200331195152-e8c3332aa8e5
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d // indirect
	golang.org/x/time v0.0.0-20200416051211-89c76fbcd5d1 // indirect
	google.golang.org/genproto v0.0.0-20200507105951-43844f6eee31 // indirect
	k8s.io/api v0.18.2
	k8s.io/apimachinery v0.18.2
	k8s.io/client-go v11.0.0+incompatible
	k8s.io/klog v1.0.0
	k8s.io/utils v0.0.0-20200414100711-2df71ebbae66 // indirect
)
