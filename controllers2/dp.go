package controllers2

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	dijkstrav2 "jinli.io/crdshortestpath/api/dijkstra/v2"
	dijkstraclient "jinli.io/crdshortestpath/generated/external/clientset/versioned"
	dijkstrainformers "jinli.io/crdshortestpath/generated/external/informers/externalversions"
	dijkstralister "jinli.io/crdshortestpath/generated/external/listers/dijkstra/v2"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type DpController struct {
	dpLister   dijkstralister.DisplayLister
	dpInformer cache.SharedIndexInformer
	queue      workqueue.RateLimitingInterface
	client     *dijkstraclient.Clientset
	Scheme     *runtime.Scheme
}

func NewDpController(scheme *runtime.Scheme, client *dijkstraclient.Clientset, dijkstraFactory dijkstrainformers.SharedInformerFactory) *DpController {
	dpInformer := dijkstraFactory.Dijkstra().V2().Displays()

	dc := &DpController{
		dpLister:   dpInformer.Lister(),
		dpInformer: dpInformer.Informer(),
		queue:      workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		client:     client,
		Scheme:     scheme,
	}

	predicates := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			dp := obj.(*dijkstrav2.Display)
			fmt.Printf("dp added: %s\n", dp.Name)
			dc.enqueue(obj)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldDp := oldObj.(*dijkstrav2.Display)
			newDp := newObj.(*dijkstrav2.Display)
			if dc.needUpdate(oldDp, newDp) {
				fmt.Printf("dp updated: %s\n", newDp.Name)
				annotations := make(map[string]string)
				oldStartNode, err := json.Marshal(oldDp.Spec.StartNode)
				if err != nil {
					klog.Error(err)
					dc.enqueue(newObj)
				}
				annotations["oldStartNode"] = string(oldStartNode)
				newDp.Annotations = annotations
				dc.enqueue(newObj)
			}
		},
		DeleteFunc: func(obj interface{}) {
			dp := obj.(*dijkstrav2.Display)
			fmt.Printf("dp deleted: %s\n", dp.Name)
			// 删除事件不入队列
			// dc.enqueue(obj)
		},
	}

	dpInformer.Informer().AddEventHandler(predicates)

	return dc
}

func (dc *DpController) Run(threads int, stopCh <-chan struct{}) {
	defer dc.queue.ShutDown()

	fmt.Println("Starting Dc controller")
	defer fmt.Println("Shutting down Dc controller")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for i := 0; i < threads; i++ {
		go wait.Until(dc.runWorker, time.Second, ctx.Done())
	}

	<-stopCh
}

func (dc *DpController) runWorker() {
	for dc.processNextWorkItem() {
	}
}

func (dc *DpController) processNextWorkItem() bool {
	obj, shutdown := dc.queue.Get()
	if shutdown {
		return false
	}

	err := dc.syncHandler(obj.(string))
	dc.queue.Done(obj)
	if err != nil {
		dc.queue.AddRateLimited(obj)
		utilruntime.HandleError(fmt.Errorf("error syncing kn: %v", err))
		return false
	}
	return true
}

func (dc *DpController) enqueue(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	dc.queue.Add(key)
}

func (dc *DpController) needUpdate(oldDp, newDp *dijkstrav2.Display) bool {
	return newDp.Spec.StartNode.ID != oldDp.Spec.StartNode.ID
}

func (dc *DpController) syncHandler(key string) error {
	ctx := context.TODO()
	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}
	dp, err := dc.dpLister.Displays(ns).Get(name)
	if errors.IsNotFound(err) {
		utilruntime.HandleError(fmt.Errorf("dp %v has been deleted", key))
		dc.enqueue(dp)
		return nil
	}
	if err != nil {
		return err
	}

	// 这里是调谐逻辑
	//更新逻辑
	klog.Info("Begin execution of " + ns + "/" + name + " update logic")
	if err := dc.update(ctx, dp); err != nil {
		if errors.IsConflict(err) {
			// 处理冲突，例如通过重新获取资源并重试更新
			klog.Info("Update conflict, retrying", " namespace:"+ns, " name:"+name)
			return nil
		}
		return err
	}

	return nil
}

func (dc *DpController) update(ctx context.Context, dp *dijkstrav2.Display) error {
	labelSelector := labels.Set(map[string]string{"nodeIdentity": dp.Labels["nodeIdentity"]}).AsSelector().String()
	knList, err := dc.client.DijkstraV2().KnownNodeses(dp.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		klog.Error(err)
		return err
	}

	for i := 0; i < 5; i++ {
		dp, err := dc.dpLister.Displays(dp.Namespace).Get(dp.Name)
		if err != nil {
			klog.Error(err)
			return err
		}

		if len(dp.OwnerReferences) == 0 {
			err = controllerutil.SetOwnerReference(&knList.Items[0], dp, dc.Scheme)
			if err != nil {
				klog.Error(err)
				return err
			}
		}
		// 更新资源
		_, err = dc.client.DijkstraV2().Displays(dp.Namespace).Update(ctx, dp, metav1.UpdateOptions{})
		if err != nil {
			if errors.IsConflict(err) {
				continue
			}
			klog.Error(err)
			return err
		}
	}

	// 需要更新资源列表
	oldTargetNode := dp.Status.TargetNodes
	// 根据指定算法计算最短路径
	ComputeShortestPath(&knList.Items[0], dp)
	newTargetNode := dp.Status.TargetNodes
	status := dp.Status

	if !TargetNodesEqual(newTargetNode, oldTargetNode) {
		// 更新子资源列表
		for i := 0; i < 5; i++ {
			oldDp, err := dc.dpLister.Displays(dp.Namespace).Get(dp.Name)
			if err != nil {
				klog.Error(err)
				return err
			}

			newDp := oldDp.DeepCopy()
			newDp.Status = status

			_, err = dc.client.DijkstraV2().Displays(newDp.Namespace).UpdateStatus(ctx, newDp, metav1.UpdateOptions{})
			if err != nil {
				if errors.IsConflict(err) {
					continue
				}
				klog.Error(err)
				return err
			}
			break
		}
		return nil
	}

	return nil
}
