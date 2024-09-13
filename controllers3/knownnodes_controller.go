/*
Copyright 2024 jinli.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	dijkstrav2 "jinli.io/crdshortestpath/api/dijkstra/v2"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var _ reconcile.Reconciler = &KnownNodesReconciler{}

// KnownNodesReconciler reconciles a KnownNodes object
type KnownNodesReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// SetupWithManager sets up the controller with the Manager.
func (r *KnownNodesReconciler) SetupWithManager(mgr ctrl.Manager) error {
	predicates := predicate.Funcs{
		CreateFunc: func(c event.CreateEvent) bool {
			kn, ok := c.Object.(*dijkstrav2.KnownNodes)
			if ok {
				fmt.Printf("kn added: %s\n", kn.Name)
			}
			return true
		},
		UpdateFunc: func(up event.UpdateEvent) bool {
			oldKn, oldOk := up.ObjectOld.(*dijkstrav2.KnownNodes)
			newKn, newOk := up.ObjectNew.(*dijkstrav2.KnownNodes)
			if oldOk && newOk {
				fmt.Printf("kn updated: %s\n", newKn.Name)
				// 删除重试时判断
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
			return true
		},
		// 删除事件不入队列
		DeleteFunc: func(de event.DeleteEvent) bool {
			kn, ok := de.Object.(*dijkstrav2.KnownNodes)
			if ok {
				fmt.Printf("kn deleted: %s\n", kn.Name)
				return false
			}
			return true
		},
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&dijkstrav2.KnownNodes{}).
		Owns(&corev1.Pod{}).
		WithEventFilter(predicates).
		Complete(r)
}

//+kubebuilder:rbac:groups=dijkstra.jinli.io,resources=knownnodeses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=dijkstra.jinli.io,resources=knownnodeses/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=dijkstra.jinli.io,resources=knownnodeses/finalizers,verbs=update

func (r *KnownNodesReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	rlog := log.FromContext(ctx)
	// 获取kn对象
	var kn dijkstrav2.KnownNodes
	if err := r.get(ctx, req.NamespacedName, &kn); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// 删除逻辑
	if kn.DeletionTimestamp != nil {
		rlog.Info("开始执行" + req.Namespace + "/" + req.Name + " 删除逻辑")
		if err := r.clean(ctx, &kn); err != nil {
			// 如果删除失败，3秒后重新入队
			return ctrl.Result{Requeue: true, RequeueAfter: 3 * time.Second}, err
		}
		return ctrl.Result{}, nil
	}

	//更新逻辑
	rlog.Info("开始执行" + req.Namespace + "/" + req.Name + " 更新逻辑")
	if err := r.update(ctx, req.NamespacedName, &kn); err != nil {
		// 如果创建失败，重新入队
		if errors.IsConflict(err) {
			// 处理冲突，例如通过重新获取资源并重试更新
			klog.Info("Update conflict, retrying", " namespace:"+kn.Namespace, " name:"+kn.Name)
			// 重新入队，不再经过事件过滤
			return reconcile.Result{Requeue: true}, nil
		}
		return ctrl.Result{Requeue: true}, err
	}

	return ctrl.Result{}, nil
}

func (r *KnownNodesReconciler) clean(ctx context.Context, kn *dijkstrav2.KnownNodes) error {
	// 检查所有相关dp对象的计算状态
	allDPCom := true
	// 所有DP对象计算完成
	labelSelector := labels.Set(map[string]string{"nodeIdentity": kn.Labels["nodeIdentity"]}).AsSelector()
	var dpList dijkstrav2.DisplayList
	err := r.List(ctx, &dpList, &client.ListOptions{
		LabelSelector: labelSelector,
		Namespace:     kn.Namespace,
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
		if err := r.Update(ctx, kn); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("wait for all dp calculations to complete")
	}

	return nil
}

func (r *KnownNodesReconciler) get(ctx context.Context, name types.NamespacedName, kn *dijkstrav2.KnownNodes) error {
	// 其中name为workqueue中的key，Get方法就是去cache中获取对应事件对象
	if err := r.Get(ctx, name, kn); err != nil {
		if errors.IsNotFound(err) {
			klog.Info("kn资源已经删除了")
			return err
		}
		klog.Error(err, "无法获取kn资源")
		return err
	}
	return nil
}

func (r *KnownNodesReconciler) update(ctx context.Context, name types.NamespacedName, kn *dijkstrav2.KnownNodes) error {
	// DP状态更新标志
	updateDPFlag := false

	// 更新资源
	for i := 0; i < 5; i++ {
		if err := r.get(ctx, name, kn); err != nil {
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
			err := r.addPods(ctx, knCopy)
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

		if err := r.Update(ctx, knCopy); err != nil {
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
			if err := r.get(ctx, name, kn); err != nil {
				klog.Error(err)
				return err
			}
			kn.Status.LastUpdate = v1.NewTime(time.Now())
			if err := r.Status().Update(ctx, kn); err != nil {
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
		labelSelector := labels.Set(map[string]string{"nodeIdentity": kn.Labels["nodeIdentity"]}).AsSelector()
		var dpList dijkstrav2.DisplayList
		err := r.List(ctx, &dpList, &client.ListOptions{
			LabelSelector: labelSelector,
			Namespace:     kn.Namespace,
		})
		if err != nil {
			klog.Error(err)
			return err
		}
		err = r.handleDependencies(ctx, kn, &dpList)
		if err != nil {
			return err
		}
	}

	if updateDPFlag {
		// 更新pod
		//获取Nodes变化的节点
		var oldNodes []dijkstrav2.Node
		err := json.Unmarshal([]byte(kn.Annotations["oldNodes"]), &oldNodes)
		if err != nil {
			klog.Info(err)
			return err
		}
		delNodes := DifferenceNodes(oldNodes, kn.Spec.Nodes)
		addNodes := DifferenceNodes(kn.Spec.Nodes, oldNodes)

		for i := range delNodes {
			name := fmt.Sprintf("%s-%d", kn.Name, delNodes[i].ID)
			err := r.delPod(ctx, name, kn.Namespace)
			if err != nil {
				klog.Info(err)
				return err
			}
		}

		for i := range addNodes {
			name := fmt.Sprintf("%s-%d", kn.Name, addNodes[i].ID)
			err := r.addPod(ctx, name, kn)
			if err != nil {
				klog.Info(err)
				return err
			}
		}

	}
	return nil
}

func (r *KnownNodesReconciler) handleDependencies(ctx context.Context, kn *dijkstrav2.KnownNodes, dpList *dijkstrav2.DisplayList) error {
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
				name := types.NamespacedName{
					Name:      dpList.Items[i].Name,
					Namespace: dpList.Items[i].Namespace,
				}
				if err := r.Get(ctx, name, dpCopy); err != nil {
					klog.Error(err)
					return err
				}

				oldDp := dpCopy.DeepCopy()
				newDp := dpCopy.DeepCopy()
				newDp.Status = status

				if err := r.Status().Patch(ctx, newDp, client.MergeFrom(oldDp)); err != nil {
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
		// 定义要匹配的标签
		labels := map[string]string{
			"nodeIdentity": kn.Labels["nodeIdentity"],
		}
		opts := []client.DeleteAllOfOption{
			client.InNamespace("default"), // 指定命名空间
			// 可以添加更多的选项，例如：
			client.MatchingLabels(labels),
		}
		err := r.DeleteAllOf(ctx, &dijkstrav2.Display{}, opts...)
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
			err := r.Delete(ctx, &dpList.Items[i])
			if err != nil {
				klog.Error(err)
				return err
			}
		}
	}

	return nil
}

func (r *KnownNodesReconciler) addPods(ctx context.Context, kn *dijkstrav2.KnownNodes) error {
	for i := range kn.Spec.Nodes {
		name := fmt.Sprintf("%s-%d", kn.Name, kn.Spec.Nodes[i].ID)
		err := r.addPod(ctx, name, kn)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *KnownNodesReconciler) addPod(ctx context.Context, name string, kn *dijkstrav2.KnownNodes) error {
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

	if err := ctrl.SetControllerReference(kn, pod, r.Scheme); err != nil {
		klog.Info(err)
		return err
	}

	err := r.Create(ctx, pod)
	if err != nil {
		if errors.IsAlreadyExists(err) {
			klog.Info("Pod", name, " Already Exists!")
			fmt.Printf("Pod  %s already exist in namespace %s\n", name, kn.Namespace)
			return nil
		}
		klog.Info(err)
		return err
	}
	return nil
}

func (r *KnownNodesReconciler) delPod(ctx context.Context, name, ns string) error {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
	}
	err := r.Delete(ctx, pod)
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
