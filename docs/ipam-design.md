# ipam 设计

因为设计的是一个 underlay 的网络，那么就必须有一个中心的 ip 地址分配的机制，来保证在不同的 Node 上的 Pod 分配的 IP 不会冲突。

这里我先整理了一些 ipam 的实现方案或思路：

- host-local: host-local 是一个本地的 ipam 管理方案，它使用本地磁盘来存储已分配的ip地址。这里如果我们把它的存储改成 etcd 的话，应该可以实现一个集中的 ipam 分配。
- kube-ovn: kube-ovn 也实现了一个集中的 ipam 分配方案，当然，它还支持 pod 固定 ip, mac 等等。它的方案比较独特，通过在 node, pod 的 annotation 中存储分配的 ip 信息，然后在 ipam 初始化的时候，通过 list pod/node，来恢复已分配的 ip 信息。