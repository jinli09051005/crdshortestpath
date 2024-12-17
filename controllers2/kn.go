package controllers2

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	dijkstrav2 "jinli.io/crdshortestpath/api/dijkstra/v2"
	dijkstraclient "jinli.io/crdshortestpath/generated/external/clientset/versioned"
	dijkstrainformers "jinli.io/crdshortestpath/generated/external/informers/externalversions"
	dijkstralister "jinli.io/crdshortestpath/generated/external/listers/dijkstra/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type KnController struct {
	knLister    dijkstralister.KnownNodesLister
	knInformer  cache.SharedIndexInformer
	podInformer cache.SharedIndexInformer
	queue       workqueue.RateLimitingInterface
	client      *dijkstraclient.Clientset
	k8sClient   *kubernetes.Clientset
	scheme      *runtime.Scheme
	schema.GroupVersionKind
}

func NewKnController(scheme *runtime.Scheme, k8sclient *kubernetes.Clientset, client *dijkstraclient.Clientset, k8sFactory informers.SharedInformerFactory, dijkstraFactory dijkstrainformers.SharedInformerFactory) *KnController {
	knInformer := dijkstraFactory.Dijkstra().V2().KnownNodeses()
	podInformer := k8sFactory.Core().V1().Pods()

	kc := &KnController{
		knLister:         knInformer.Lister(),
		knInformer:       knInformer.Informer(),
		podInformer:      podInformer.Informer(),
		queue:            workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		client:           client,
		k8sClient:        k8sclient,
		scheme:           scheme,
		GroupVersionKind: dijkstrav2.SchemeGroupVersion.WithKind("KnownNodes"),
	}

	knPredicates := cache.ResourceEventHandlerFuncs{
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
	}

	podPredicates := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			pod := obj.(*corev1.Pod)
			fmt.Printf("pod added: %s\n", pod.Name)
			// 获取kn
			if controllerRef := metav1.GetControllerOf(pod); controllerRef != nil {
				kn := kc.resolveControllerRef(pod.Namespace, controllerRef)
				if kn != nil {
					kc.enqueue(kn)
				}
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			newPod := newObj.(*corev1.Pod)
			fmt.Printf("pod updated: %s\n", newPod.Name)
			if controllerRef := metav1.GetControllerOf(newPod); controllerRef != nil {
				kn := kc.resolveControllerRef(newPod.Namespace, controllerRef)
				if kn != nil {
					kc.enqueue(kn)
				}
			}
		},
		DeleteFunc: func(obj interface{}) {
			pod := obj.(*corev1.Pod)
			fmt.Printf("pod deleted: %s\n", pod.Name)
			if controllerRef := metav1.GetControllerOf(pod); controllerRef != nil {
				kn := kc.resolveControllerRef(pod.Namespace, controllerRef)
				if kn != nil {
					kc.enqueue(kn)
				}
			}
		},
	}
	kc.knInformer.AddEventHandler(knPredicates)
	kc.podInformer.AddEventHandler(podPredicates)

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
	if !NodesEqual(newKn.Spec.Nodes, oldKn.Spec.Nodes) {
		oldNodes, err := json.Marshal(oldKn.Spec.Nodes)
		if err != nil {
			klog.Error(err)
			return true
		}
		newKn.Annotations["oldNodes"] = string(oldNodes)
		return true
	}
	return false
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
		klog.Info("Begin execution of " + ns + "/" + name + " deletion logic")
		if err := kc.clean(ctx, kn); err != nil {
			return err
		}
		return nil
	}

	//更新逻辑
	klog.Info("Begin execution of " + ns + "/" + name + " update logic")
	if err := kc.update(ctx, kn); err != nil {
		if errors.IsConflict(err) {
			// 处理冲突，例如通过重新获取资源并重试更新
			klog.Info("Update conflict, retrying", " namespace:"+ns, " name:"+name)
			kc.enqueue(kn)
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
	// DP状态更新标志
	updateDPFlag := false
	labelSelector := labels.Set(map[string]string{"nodeIdentity": kn.Labels["nodeIdentity"]}).AsSelector().String()
	// 更新资源
	for i := 0; i < 5; i++ {
		kn, err := kc.knLister.KnownNodeses(kn.Namespace).Get(kn.Name)
		if err != nil {
			klog.Error(err)
			return err
		}
		knCopy := kn.DeepCopy()
		if len(knCopy.Finalizers) == 0 && knCopy.Annotations == nil {
			// 更新Finalizers标签
			controllerutil.AddFinalizer(knCopy, "alldpstatus/computestatus")
			// 更新Annotation nodes标签
			annotations := make(map[string]string)
			annotations["nodes"] = strconv.Itoa(len(knCopy.Spec.Nodes))
			knCopy.Annotations = annotations
			err := kc.addPods(ctx, knCopy)
			if err != nil {
				klog.Error(err)
				return err
			}
		} else {
			updateDPFlag = true
			if knCopy.Annotations["nodes"] != strconv.Itoa(len(knCopy.Spec.Nodes)) {
				// 更新Annotation nodes标签
				knCopy.Annotations["nodes"] = strconv.Itoa(len(knCopy.Spec.Nodes))
			}
		}

		_, err = kc.client.DijkstraV2().KnownNodeses(knCopy.Namespace).Update(ctx, knCopy, metav1.UpdateOptions{})
		if err != nil {
			if errors.IsConflict(err) {
				continue
			}
			klog.Error(err)
			return err
		}
		break

	}

	if updateDPFlag {
		// 更新KN状态
		for i := 0; i < 5; i++ {
			oldKn, err := kc.knLister.KnownNodeses(kn.Namespace).Get(kn.Name)
			if err != nil {
				klog.Error(err)
				return err
			}

			newKn := oldKn.DeepCopy()
			newKn.Status.LastUpdate = metav1.NewTime(time.Now())
			_, err = kc.client.DijkstraV2().KnownNodeses(newKn.Namespace).UpdateStatus(ctx, newKn, metav1.UpdateOptions{})
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

	if updateDPFlag {
		// 更新pod
		//获取Nodes变化的节点
		var oldNodes []dijkstrav2.Node
		if oldNodesString, ok := kn.Annotations["oldNodes"]; ok {
			err := json.Unmarshal([]byte(oldNodesString), &oldNodes)
			if err != nil {
				klog.Info(err)
				return err
			}
			delNodes := DifferenceNodes(oldNodes, kn.Spec.Nodes)
			addNodes := DifferenceNodes(kn.Spec.Nodes, oldNodes)

			for i := range delNodes {
				name := fmt.Sprintf("%s-%d", kn.Name, delNodes[i].ID)
				err := kc.delPod(ctx, name, kn.Namespace)
				if err != nil {
					klog.Info(err)
					return err
				}
			}

			for i := range addNodes {
				name := fmt.Sprintf("%s-%d", kn.Name, addNodes[i].ID)
				err := kc.addPod(ctx, name, kn)
				if err != nil {
					klog.Info(err)
					return err
				}
			}
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

func (kc *KnController) resolveControllerRef(namespace string, controllerRef *metav1.OwnerReference) *dijkstrav2.KnownNodes {
	// We can't look up by UID, so look up by Name and then verify UID.
	// Don't even try to look up by Name if it's the wrong Kind.
	if controllerRef.Kind != kc.Kind {
		return nil
	}
	kn, err := kc.knLister.KnownNodeses(namespace).Get(controllerRef.Name)
	if err != nil {
		return nil
	}
	if kn.UID != controllerRef.UID {
		// The controller we found with this Name is not the same one that the
		// ControllerRef points to.
		return nil
	}
	return kn
}

func (kc *KnController) addPods(ctx context.Context, kn *dijkstrav2.KnownNodes) error {
	for i := range kn.Spec.Nodes {
		name := fmt.Sprintf("%s-%d", kn.Name, kn.Spec.Nodes[i].ID)
		err := kc.addPod(ctx, name, kn)
		if err != nil {
			return err
		}
	}
	return nil
}

func (kc *KnController) addPod(ctx context.Context, name string, kn *dijkstrav2.KnownNodes) error {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: kn.Namespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "busybox",
					Image: "registry.cn-hangzhou.aliyuncs.com/jinli09051005/tools:busybox-latest",
					Command: []string{
						"tail",
						"-f",
						"/dev/null",
					},
				},
			},
		},
	}

	if err := ctrl.SetControllerReference(kn, pod, kc.scheme); err != nil {
		klog.Info(err)
		return err
	}

	_, err := kc.k8sClient.CoreV1().Pods(kn.Namespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			fmt.Printf("Pod  %s already exist in namespace %s\n", name, kn.Namespace)
			return nil
		}
		klog.Info(err)
		return err
	}
	return nil
}

func (kc *KnController) delPod(ctx context.Context, name, ns string) error {
	err := kc.k8sClient.CoreV1().Pods(ns).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			fmt.Printf("Pod  %s does not exist in namespace %s\n", name, ns)
			return nil
		}
		klog.Info(err)
		return err
	}
	return nil
}
