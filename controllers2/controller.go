package controllers2

import (
	"os"
	"os/signal"
	"syscall"

	dijkstraclient "jinli.io/crdshortestpath/generated/external/clientset/versioned"
	dijkstrainformers "jinli.io/crdshortestpath/generated/external/informers/externalversions"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

func RunController(scheme *runtime.Scheme, restConfig *rest.Config) {
	stopCh := make(chan struct{})
	defer close(stopCh)
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	client := dijkstraclient.NewForConfigOrDie(restConfig)
	dijkstraFactory := dijkstrainformers.NewSharedInformerFactory(client, 0)

	kc := NewKnController(client, dijkstraFactory)
	dc := NewDpController(scheme, client, dijkstraFactory)

	dijkstraFactory.Start(stopCh)
	if !cache.WaitForCacheSync(stopCh, kc.knInformer.HasSynced, dc.dpInformer.HasSynced) {
		panic("Failed to sync cache")
	}

	go kc.Run(5, stopCh)
	go dc.Run(5, stopCh)

	<-signals
}
