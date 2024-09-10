package controllers2

import (
	"context"
	"fmt"
	"strconv"
	"time"

	dijkstrav2 "jinli.io/crdshortestpath/api/dijkstra/v2"
	dijkstraclient "jinli.io/crdshortestpath/generated/external/clientset/versioned"
	dijkstrainformers "jinli.io/crdshortestpath/generated/external/informers/externalversions"
	dijkstralister "jinli.io/crdshortestpath/generated/external/listers/dijkstra/v2"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type KnController struct {
	knLister   dijkstralister.KnownNodesLister
	knInformer cache.SharedIndexInformer
	queue      workqueue.RateLimitingInterface
	client     *dijkstraclient.Clientset
}

func NewKnController(client *dijkstraclient.Clientset, dijkstraFactory dijkstrainformers.SharedInformerFactory) *KnController {
	knInformer := dijkstraFactory.Dijkstra().V2().KnownNodeses()

	kc := &KnController{
		knLister:   knInformer.Lister(),
		knInformer: knInformer.Informer(),
		queue:      workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		client:     client,
	}

	knInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			kn := obj.(*dijkstrav2.KnownNodes)
			fmt.Printf("kn added: %s\n", kn.Name)
			kc.enqueue(obj)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldKn := oldObj.(*dijkstrav2.KnownNodes)
			newKn := newObj.(*dijkstrav2.KnownNodes)
			if kc.needUpdate(oldKn, newKn) {
				fmt.Printf("kn updated: %s\n", newKn.Name)
				kc.enqueue(newObj)
			}
		},
		DeleteFunc: func(obj interface{}) {
			kn := obj.(*dijkstrav2.KnownNodes)
			fmt.Printf("kn deleted: %s\n", kn.Name)
			// 删除事件不入队列
			// kc.enqueue(obj)
		},
	})

	return kc
}

func (kc *KnController) Run(threads int, stopCh <-chan struct{}) {
	defer kc.queue.ShutDown()

	fmt.Println("Starting Kn controller")
	defer fmt.Println("Shutting down Kn controller")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for i := 0; i < threads; i++ {
		go wait.Until(kc.runWorker, time.Second, ctx.Done())
	}

	<-stopCh
}

func (kc *KnController) runWorker() {
	for kc.processNextWorkItem() {
	}
}

func (kc *KnController) processNextWorkItem() bool {
	obj, shutdown := kc.queue.Get()
	if shutdown {
		return false
	}

	err := kc.syncHandler(obj.(string))
	kc.queue.Done(obj)
	if err != nil {
		kc.queue.AddRateLimited(obj)
		utilruntime.HandleError(fmt.Errorf("error syncing kn: %v", err))
		return false
	}
	return true
}

func (kc *KnController) enqueue(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	kc.queue.Add(key)
}

func (kc *KnController) needUpdate(oldKn, newKn *dijkstrav2.KnownNodes) bool {
	if newKn.DeletionTimestamp != nil {
		return true
	}
	return !NodesEqual(newKn.Spec.Nodes, oldKn.Spec.Nodes)
}

func (kc *KnController) syncHandler(key string) error {
	ctx := context.TODO()
	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}
	kn, err := kc.knLister.KnownNodeses(ns).Get(name)
	if errors.IsNotFound(err) {
		utilruntime.HandleError(fmt.Errorf("kn %v has been deleted", key))
		return nil
	}
	if err != nil {
		return err
	}

	// 这里是调谐逻辑
	// 删除逻辑
	if kn.DeletionTimestamp != nil {
		klog.Info("开始执行" + ns + "/" + name + " 删除逻辑")
		if err := kc.clean(ctx, kn); err != nil {
			return err
		}
		return nil
	}

	//更新逻辑
	klog.Info("开始执行" + ns + "/" + name + " 更新逻辑")
	if err := kc.update(ctx, kn); err != nil {
		if errors.IsConflict(err) {
			// 处理冲突，例如通过重新获取资源并重试更新
			klog.Info("Update conflict, retrying", " namespace:"+ns, " name:"+name)
			return nil
		}
		return err
	}

	return nil
}

func (kc *KnController) clean(ctx context.Context, kn *dijkstrav2.KnownNodes) error {
	// 检查所有相关dp对象的计算状态
	allDPCom := true
	// 所有DP对象计算完成
	labelSelector := labels.Set(map[string]string{"nodeIdentity": kn.Labels["nodeIdentity"]}).AsSelector().String()

	dpList, err := kc.client.DijkstraV2().Displays(kn.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return err
	}

	for i := range dpList.Items {
		if dpList.Items[i].Status.ComputeStatus == "Succeed" || dpList.Items[i].Status.ComputeStatus == "Failed" {
			continue
		}
		allDPCom = false
		break
	}

	if allDPCom {
		// 删除kn对象finalizer
		controllerutil.RemoveFinalizer(kn, "alldpstatus/computestatus")
		_, err := kc.client.DijkstraV2().KnownNodeses(kn.Namespace).Update(ctx, kn, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("wait for all dp calculations to complete")
	}

	return nil
}

func (kc *KnController) update(ctx context.Context, kn *dijkstrav2.KnownNodes) error {
	// 更新标志
	updateFlag := false
	// DP状态更新标志
	updateDPFlag := false
	labelSelector := labels.Set(map[string]string{"nodeIdentity": kn.Labels["nodeIdentity"]}).AsSelector().String()
	// 更新资源
	for i := 0; i < 5; i++ {
		kn, err := kc.client.DijkstraV2().KnownNodeses(kn.Namespace).Get(ctx, kn.Name, metav1.GetOptions{})
		if err != nil {
			klog.Error(err)
			return err
		}
		knCopy := kn.DeepCopy()
		if len(knCopy.Finalizers) == 0 && knCopy.Annotations == nil {
			updateFlag = true
			// 更新Finalizers标签
			controllerutil.AddFinalizer(knCopy, "alldpstatus/computestatus")
			// 更新Annotation nodes标签
			annotations := make(map[string]string)
			annotations["nodes"] = strconv.Itoa(len(knCopy.Spec.Nodes))
			knCopy.Annotations = annotations
		} else {
			updateDPFlag = true
			if knCopy.Annotations["nodes"] != strconv.Itoa(len(knCopy.Spec.Nodes)) {
				updateFlag = true
				// 更新Annotation nodes标签
				knCopy.Annotations["nodes"] = strconv.Itoa(len(knCopy.Spec.Nodes))
			}
		}

		if updateFlag {
			_, err := kc.client.DijkstraV2().KnownNodeses(kn.Namespace).Update(ctx, knCopy, metav1.UpdateOptions{})
			if err != nil {
				if errors.IsConflict(err) {
					continue
				}
				klog.Error(err)
				return err
			}
			break
		}
	}

	if updateDPFlag {
		// 更新KN状态
		for i := 0; i < 5; i++ {
			kn, err := kc.client.DijkstraV2().KnownNodeses(kn.Namespace).Get(ctx, kn.Name, metav1.GetOptions{})
			if err != nil {
				klog.Error(err)
				return err
			}
			kn.Status.LastUpdate = metav1.NewTime(time.Now())
			_, err = kc.client.DijkstraV2().KnownNodeses(kn.Namespace).UpdateStatus(ctx, kn, metav1.UpdateOptions{})
			if err != nil {
				if errors.IsConflict(err) {
					continue
				}
				klog.Error(err)
				return err
			}
		}
	}

	if updateDPFlag {
		// 更新DP状态
		dpList, err := kc.client.DijkstraV2().Displays(kn.Namespace).List(ctx, metav1.ListOptions{
			LabelSelector: labelSelector,
		})
		if err != nil {
			klog.Error(err)
			return err
		}
		err = kc.handleDependencies(ctx, kn, dpList)
		if err != nil {
			return err
		}
	}
	return nil
}

func (kc *KnController) handleDependencies(ctx context.Context, kn *dijkstrav2.KnownNodes, dpList *dijkstrav2.DisplayList) error {
	labelSelector := labels.Set(map[string]string{"nodeIdentity": kn.Labels["nodeIdentity"]}).AsSelector().String()
	// dijkstraClient := dijkstraclient.NewForConfigOrDie(r.ClientConfig)
	//重新计算所有dp对象最短路径，并更新dp对象
	for i := range dpList.Items {
		dpCopy := dpList.Items[i].DeepCopy()
		oldTargetNode := dpCopy.Status.TargetNodes
		ComputeShortestPath(kn, dpCopy)
		newTargetNode := dpCopy.Status.TargetNodes
		status := dpCopy.Status
		if !TargetNodesEqual(newTargetNode, oldTargetNode) {
			// 更新子资源列表
			for j := 0; j < 5; j++ {
				//创建Display前相同NodeIdentity的KnownNodes需要创建
				oldDp, err := kc.client.DijkstraV2().Displays(dpList.Items[i].Namespace).Get(ctx, dpList.Items[i].Name, metav1.GetOptions{})
				if err != nil {
					klog.Error(err)
					return err
				}

				newDp := oldDp.DeepCopy()
				newDp.Status = status
				_, err = kc.client.DijkstraV2().Displays(newDp.Namespace).UpdateStatus(ctx, newDp, metav1.UpdateOptions{})
				if err != nil {
					if errors.IsConflict(err) {
						continue
					}
					klog.Error(err)
					return err
				}

				break
			}
		}
	}

	// 判断dp对象的startNode是否在kn对象Nodes中,不在就删除
	if len(kn.Spec.Nodes) == 0 {
		err := kc.client.DijkstraV2().Displays(kn.Namespace).DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{
			LabelSelector: labelSelector,
		})

		if err != nil {
			klog.Error(err)
			return err
		}
		return nil
	}

	for i := range dpList.Items {
		var flag int
		for j := range kn.Spec.Nodes {
			if dpList.Items[i].Spec.StartNode.ID != kn.Spec.Nodes[j].ID {
				flag++
				continue
			}
			break
		}
		if flag == len(kn.Spec.Nodes) {
			err := kc.client.DijkstraV2().Displays(kn.Namespace).Delete(ctx, dpList.Items[i].Name, metav1.DeleteOptions{})
			if err != nil {
				klog.Error(err)
				return err
			}
		}
	}

	return nil
}
