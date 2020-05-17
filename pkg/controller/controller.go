package controller

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	v1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
)

type Controller struct {
	podLister  v1.PodLister
	podsSynced cache.InformerSynced

	addPodQueue    workqueue.RateLimitingInterface
	updatePodQueue workqueue.RateLimitingInterface
	deletePodQueue workqueue.RateLimitingInterface

	recorder        record.EventRecorder
	informerFactory informers.SharedInformerFactory
	elector         *leaderelection.LeaderElector
}

func NewController() *Controller {

	config, _ := rest.InClusterConfig()
	kubeCli, _ := kubernetes.NewForConfig(config)

	klog.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeCli.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: "kubeovs-controller"})

	informerFactory := kubeinformers.NewSharedInformerFactoryWithOptions(kubeCli, 0,
		kubeinformers.WithTweakListOptions(func(listOption *metav1.ListOptions) {
			listOption.AllowWatchBookmarks = true
		}))

	podInformer := informerFactory.Core().V1().Pods()

	controller := &Controller{
		podLister:  podInformer.Lister(),
		podsSynced: podInformer.Informer().HasSynced,

		addPodQueue:    workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "AddPod"),
		deletePodQueue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "DeletePod"),
		updatePodQueue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "UpdatePod"),

		recorder:        recorder,
		informerFactory: informerFactory,
	}

	return controller
}
