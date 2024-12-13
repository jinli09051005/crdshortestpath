package webhook

import (
	"context"

	dijkstrav2 "jinli.io/crdshortestpath/api/dijkstra/v2"
	dijkstraclient "jinli.io/crdshortestpath/generated/external/clientset/versioned/typed/dijkstra/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	restclient "k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type CustomDpWebhook struct {
	ClientConfig *restclient.Config
}

//+kubebuilder:webhook:path=/mutate-dijkstra-jinli-io-v2-display,mutating=true,failurePolicy=fail,sideEffects=None,groups=dijkstra.jinli.io,resources=displays,verbs=create;update,versions=v2,name=m2display.kb.io,admissionReviewVersions=v1

var _ webhook.CustomDefaulter = &CustomDpWebhook{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (w *CustomDpWebhook) Default(ctx context.Context, obj runtime.Object) error {
	dp := obj.(*dijkstrav2.Display)
	//默认值设置
	if dp.Spec.NodeIdentity == "" {
		dp.Spec.NodeIdentity = "jinli-default-nodeid"
	}

	if dp.Spec.Algorithm == "" {
		dp.Spec.Algorithm = "dijkstra"
	}

	// 添加标签
	labels := make(map[string]string)
	labels["nodeIdentity"] = dp.Spec.NodeIdentity
	dp.Labels = labels

	// 添加默认状态
	if dp.Status.ComputeStatus == "" {
		dp.Status.ComputeStatus = "Wait"
	}

	return nil
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-dijkstra-jinli-io-v2-display,mutating=false,failurePolicy=fail,sideEffects=None,groups=dijkstra.jinli.io,resources=displays,verbs=create;update,versions=v2,name=v2display.kb.io,admissionReviewVersions=v1

var _ webhook.CustomValidator = &CustomDpWebhook{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (w *CustomDpWebhook) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	dp := obj.(*dijkstrav2.Display)
	var warnning admission.Warnings
	var fielderr *field.Error
	warnning = append(warnning, "Display with the same nodeIdentity not allow!")
	// 使用validation,字段验证在前
	errList := dijkstrav2.ValidateDisplayCreate(dp)

	// 设置要查询的字段选择器
	labelSelector := labels.Set(map[string]string{"nodeIdentity": dp.Labels["nodeIdentity"]}).AsSelector().String()
	dijkstraClient := dijkstraclient.NewForConfigOrDie(w.ClientConfig)

	//创建Display前相同NodeIdentity的KnownNodes需要创建
	kns, err := dijkstraClient.KnownNodeses(dp.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		klog.Error(err)
		fielderr = field.Invalid(field.NewPath("spec", "nodeIdentity"), dp.Spec.NodeIdentity, "KnownNodes with the same nodeIdentity could not be found!")
		errList = append(errList, fielderr)
		return warnning, errList.ToAggregate()
	} else if len(kns.Items) == 0 {
		fielderr = field.Invalid(field.NewPath("spec", "nodeIdentity"), dp.Spec.NodeIdentity, "KnownNodes with the same nodeIdentity need to be created before creating Display!")
		errList = append(errList, fielderr)
		return warnning, errList.ToAggregate()
	}

	// 相同NodeIdentity的StartNode不允许相同
	dps, err := dijkstraClient.Displays(dp.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		klog.Error(err)
		fielderr = field.Invalid(field.NewPath("spec", "nodeIdentity"), dp.Spec.NodeIdentity, "Display with the same nodeIdentity could not be found!")
		errList = append(errList, fielderr)
		return warnning, errList.ToAggregate()
	} else if len(dps.Items) != 0 {
		for i := range dps.Items {
			if dps.Items[i].Spec.StartNode.ID != dp.Spec.StartNode.ID {
				continue
			}
			fielderr = field.Invalid(field.NewPath("spec", "startNode"), dp.Spec.StartNode, "Display with the same nodeIdentity and startNode.ID not allow!")
			errList = append(errList, fielderr)
			return warnning, errList.ToAggregate()
		}
	}

	// 创建dp对象时校验总数有没有超过对应kn的nodes数量
	if len(dps.Items)+1 > len(kns.Items[0].Spec.Nodes) {
		fielderr = field.Invalid(field.NewPath("spec", "nodeIdentity"), dp.Spec.NodeIdentity, "the number of dps is greater than kn nodes for "+dp.Namespace+"/"+dp.Spec.NodeIdentity)
		errList = append(errList, fielderr)
		return warnning, errList.ToAggregate()
	}

	return warnning, errList.ToAggregate()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (w *CustomDpWebhook) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	oldDisplay := oldObj.(*dijkstrav2.Display)
	newDisplay := newObj.(*dijkstrav2.Display)
	var warnning admission.Warnings
	warnning = append(warnning, "Field .Spec.NodeIdentity cannot be modified")

	return warnning, dijkstrav2.ValidateDisplayUpdate(newDisplay, oldDisplay)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (w *CustomDpWebhook) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	// TODO
	return nil, nil
}
