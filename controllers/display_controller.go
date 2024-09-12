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

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	dijkstrav2 "jinli.io/crdshortestpath/api/dijkstra/v2"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ reconcile.Reconciler = &DisplayReconciler{}

// DisplayReconciler reconciles a Display object
type DisplayReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// SetupWithManager sets up the controller with the Manager.
func (r *DisplayReconciler) SetupWithManager(mgr ctrl.Manager) error {
	predicates := predicate.Funcs{
		CreateFunc: func(c event.CreateEvent) bool {
			dp := c.Object.(*dijkstrav2.Display)
			fmt.Printf("dp added: %s\n", dp.Name)
			return true
		},
		UpdateFunc: func(up event.UpdateEvent) bool {
			oldDp := up.ObjectOld.(*dijkstrav2.Display)
			newDp := up.ObjectNew.(*dijkstrav2.Display)
			fmt.Printf("dp updated: %s\n", newDp.Name)
			if newDp.Spec.StartNode.ID != oldDp.Spec.StartNode.ID {
				annotations := make(map[string]string)
				oldStartNode, err := json.Marshal(oldDp.Spec.StartNode)
				if err != nil {
					klog.Error(err)
					return true
				}
				annotations["oldStartNode"] = string(oldStartNode)
				newDp.Annotations = annotations
				return true
			}
			return false
		},
		DeleteFunc: func(de event.DeleteEvent) bool {
			dp := de.Object.(*dijkstrav2.Display)
			fmt.Printf("dp deleted: %s\n", dp.Name)
			return false
		},
		GenericFunc: func(event.GenericEvent) bool {
			return true
		},
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&dijkstrav2.Display{}).
		WithEventFilter(predicates).
		Complete(r)
}

//+kubebuilder:rbac:groups=dijkstra.jinli.io,resources=displays,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=dijkstra.jinli.io,resources=displays/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=dijkstra.jinli.io,resources=displays/finalizers,verbs=update

func (r *DisplayReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	rlog := log.FromContext(ctx)

	// TODO(user): your logic here
	// 获取dp对象
	var dp dijkstrav2.Display
	if err := r.get(ctx, req.NamespacedName, &dp); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, nil
	}

	// 更新逻辑
	rlog.Info("开始调谐" + req.Namespace + "/" + req.Name + " 更新事件")
	if err := r.update(ctx, req.NamespacedName, &dp); err != nil {
		// 如果更新失败，重新入队
		if errors.IsConflict(err) {
			// 处理冲突，例如通过重新获取资源并重试更新
			klog.Info("Update conflict, retrying", " namespace:", dp.Namespace, " name:", dp.Name)
			return reconcile.Result{Requeue: true}, nil
		}
		return ctrl.Result{Requeue: true}, err
	}

	return ctrl.Result{}, nil
}

func (r *DisplayReconciler) get(ctx context.Context, name types.NamespacedName, dp *dijkstrav2.Display) error {
	if err := r.Get(ctx, name, dp); err != nil {
		if errors.IsNotFound(err) {
			klog.Info("dp资源已经删除了")
			return err
		}
		klog.Error(err, "无法获取dp资源")
		return err
	}
	return nil
}

func (r *DisplayReconciler) update(ctx context.Context, name types.NamespacedName, dp *dijkstrav2.Display) error {
	labelSelector := labels.Set(map[string]string{"nodeIdentity": dp.Labels["nodeIdentity"]}).AsSelector()
	var knList dijkstrav2.KnownNodesList
	err := r.List(ctx, &knList, &client.ListOptions{
		LabelSelector: labelSelector,
		Namespace:     dp.Namespace,
	})
	if err != nil {
		klog.Error(err)
		return err
	}

	// 更新dp对象
	for i := 0; i < 5; i++ {
		if err := r.get(ctx, name, dp); err != nil {
			klog.Error(err)
			return err
		}
		if len(dp.OwnerReferences) == 0 {
			err = controllerutil.SetOwnerReference(&knList.Items[0], dp, r.Scheme)
			if err != nil {
				klog.Error(err)
				return err
			}
		}
		// 更新资源
		if err := r.Update(ctx, dp); err != nil {
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
			if err := r.get(ctx, name, dp); err != nil {
				klog.Error(err)
				return err
			}
			oldDp := dp.DeepCopy()
			newDp := dp.DeepCopy()
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
		return nil
	}

	return nil
}
