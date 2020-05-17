## 测试 cni 插件

```
$ CNI_PATH=$(pwd)
$ sudo CNI_PATH=$CNI_PATH ./docker-run.sh --rm busybox:latest ifconfig
```