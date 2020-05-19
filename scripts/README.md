## 测试 cni 插件

```
$ CNI_PATH=$(pwd)/../bin
$ sudo CNI_PATH=$CNI_PATH ./docker-run.sh --rm busybox:latest sh -c "ifconfig && ping -c 3 192.168.50.56"
```