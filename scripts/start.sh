#!/bin/bash

cp /app/kubeovs-l2 /etc/cni/net.d/

/app/kubeovsd -c "$@"