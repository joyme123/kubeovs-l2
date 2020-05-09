# ipam 设计

因为设计的是一个 underlay 的网络，那么就必须有一个中心的 ip 地址分配的机制，来保证在不同的 Node 上的 Pod 分配的 IP 不会冲突。