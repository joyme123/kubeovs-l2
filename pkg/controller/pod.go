package controller

import (
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog"
)

const (
	AllocatedAnnotation  = "kubeovs.io/allocated"
	RoutedAnnotation     = "kubeovs.io/routed"
	MacAddressAnnotation = "kubeovs.io/mac_address"
	IPAddressAnnotation  = "kubeovs.io/ip_address"
	CidrAnnotation       = "kubeovs.io/cidr"
	GateWayAnnotation    = "kubeovs.io/gateway"
	IPPoolAnnotation     = "kubeovs.io/ip_pool"
)

func isPodAlive(p *v1.Pod) bool {
	if p.Status.Phase == v1.PodSucceeded && p.Spec.RestartPolicy != v1.RestartPolicyAlways {
		return false
	}

	if p.Status.Phase == v1.PodFailed && p.Spec.RestartPolicy == v1.RestartPolicyNever {
		return false
	}

	if p.Status.Phase == v1.PodFailed && p.Status.Reason == "Evicted" {
		return false
	}

	return true
}

func (c *Controller) enqueueAddPod(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}

	p := obj.(*v1.Pod)
	if !isPodAlive(p) {
		c.deletePodQueue.Add(key)
		return
	}

	klog.V(3).Infof("enqueue add pod %s", key)
	c.addPodQueue.Add(key)
}

func (c *Controller) enqueueDeletePod(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}

	c.deletePodQueue.Add(key)
}

func (c *Controller) enqueueUpdatePod(oldObj, newObj interface{}) {
	oldPod := oldObj.(*v1.Pod)
	newPod := newObj.(*v1.Pod)
	// update pod 触发的机制是什么？为什么会出现新旧资源版本号一样的问题
	if oldPod.ResourceVersion == newPod.ResourceVersion {
		return
	}

	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(newObj); err != nil {
		utilruntime.HandleError(err)
		return
	}

	if !isPodAlive(newPod) {
		c.deletePodQueue.Add(key)
		return
	}

	// pod assigned an ip
	if newPod.Annotations[AllocatedAnnotation] == "true" &&
		newPod.Spec.NodeName != "" {
		// nodeName 表示该pod已经调度到节点上了
		klog.V(3).Infof("enqueue update pod %s", key)
		c.updatePodQueue.Add(key)
	}
}

func (c *Controller) runAddPodWorker() {

}

func (c *Controller) runDeletePodWorker() {

}

func (c *Controller) runUpdatePodWorker() {

}

func (c *Controller) processNextAddPodWorkItem() bool {
	obj, shutdown := c.addPodQueue.Get()

	if shutdown {
		return false
	}
	now := time.Now()

	err := func(obj interface{}) error {
		defer c.addPodQueue.Done(obj)
		var key string
		var ok bool
		if key, ok = obj.(string); !ok {
			c.addPodQueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}
		if err := c.handleAddPod(key); err != nil {
			c.addPodQueue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}
		c.addPodQueue.Forget(obj)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	last := time.Since(now)
	klog.Infof("take %d ms to deal with add pod", last.Microseconds())
	return true
}

func (c *Controller) processNextDeletePodWorkItem() bool {
	obj, shutdown := c.deletePodQueue.Get()
	if shutdown {
		return false
	}

	now := time.Now()
	err := func(obj interface{}) error {
		defer c.deletePodQueue.Done(obj)
		var key string
		var ok bool
		if key, ok = obj.(string); !ok {
			c.deletePodQueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}
		if err := c.handleDeletePod(key); err != nil {
			c.deletePodQueue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}
		c.deletePodQueue.Forget(obj)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	last := time.Since(now)
	klog.Infof("take %d ms to deal with delete pod", last.Milliseconds())
	return true
}

func (c *Controller) processNextUpdatePodWorkItem() bool {
	obj, shutdown := c.updatePodQueue.Get()
	if shutdown {
		return false
	}

	err := func(obj interface{}) error {
		defer c.updatePodQueue.Done(obj)
		var key string
		var ok bool
		if key, ok = obj.(string); !ok {
			c.updatePodQueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}
		if err := c.handleUpdatePod(key); err != nil {
			c.updatePodQueue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}
		c.updatePodQueue.Forget(obj)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

func (c *Controller) handleAddPod(key string) error {
	return nil
}

func (c *Controller) handleDeletePod(key string) error {
	return nil
}

func (c *Controller) handleUpdatePod(key string) error {
	return nil
}
