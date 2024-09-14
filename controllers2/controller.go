package controllers2

import (
	"os"
	"os/signal"
	"syscall"

	dijkstraclient "jinli.io/crdshortestpath/generated/external/clientset/versioned"
	dijkstrainformers "jinli.io/crdshortestpath/generated/external/informers/externalversions"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

func RunController(scheme *runtime.Scheme, restConfig *rest.Config) {
	stopCh := make(chan struct{})
	defer close(stopCh)
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	k8sClient := kubernetes.NewForConfigOrDie(restConfig)
	client := dijkstraclient.NewForConfigOrDie(restConfig)
	dijkstraFactory := dijkstrainformers.NewSharedInformerFactory(client, 0)
	k8sFactory := informers.NewSharedInformerFactory(k8sClient, 0)

	kc := NewKnController(scheme, k8sClient, client, k8sFactory, dijkstraFactory)
	dc := NewDpController(scheme, client, dijkstraFactory)

	dijkstraFactory.Start(stopCh)
	k8sFactory.Start(stopCh)
	if !cache.WaitForCacheSync(stopCh, kc.knInformer.HasSynced, kc.podInformer.HasSynced, dc.dpInformer.HasSynced) {
		panic("Failed to sync cache")
	}

	go kc.Run(5, stopCh)
	go dc.Run(5, stopCh)

	<-signals
}
