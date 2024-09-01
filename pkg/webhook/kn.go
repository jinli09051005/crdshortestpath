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

type CustomKnWebhook struct {
	ClientConfig *restclient.Config
}

// +kubebuilder:webhook:path=/mutate-dijkstra-jinli-io-v2-knownnodes,mutating=true,failurePolicy=fail,sideEffects=None,groups=dijkstra.jinli.io,resources=knownnodeses,verbs=create;update,versions=v2,name=m2knownnodes.kb.io,admissionReviewVersions=v1

var _ webhook.CustomDefaulter = &CustomKnWebhook{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (w *CustomKnWebhook) Default(ctx context.Context, obj runtime.Object) error {
	kn := obj.(*dijkstrav2.KnownNodes)
	//默认值设置
	if kn.Spec.NodeIdentity == "" {
		kn.Spec.NodeIdentity = "jinli-default-nodeid"
	}

	// 添加标签
	labels := make(map[string]string)
	labels["nodeIdentity"] = kn.Spec.NodeIdentity
	kn.Labels = labels

	return nil
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-dijkstra-jinli-io-v2-knownnodes,mutating=false,failurePolicy=fail,sideEffects=None,groups=dijkstra.jinli.io,resources=knownnodeses,verbs=create;update,versions=v2,name=v2knownnodes.kb.io,admissionReviewVersions=v1

var _ webhook.CustomValidator = &CustomKnWebhook{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (w *CustomKnWebhook) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	kn := obj.(*dijkstrav2.KnownNodes)
	var warnning admission.Warnings
	var fielderr *field.Error
	warnning = append(warnning, "KnownNodes with the same nodeIdentity not allow!")
	// 使用validation,字段验证在前
	errList := dijkstrav2.ValidateKnownNodesCreate(kn)

	// 设置要查询的字段选择器
	labelSelector := labels.Set(map[string]string{"nodeIdentity": kn.Labels["nodeIdentity"]}).AsSelector().String()

	//不允许创建NodeIdentity相同KnownNodes
	dijkstraClient := dijkstraclient.NewForConfigOrDie(w.ClientConfig)
	kns, err := dijkstraClient.KnownNodeses(kn.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		klog.Error(err)
		fielderr = field.Invalid(field.NewPath("spec", "nodeIdentity"), kn.Spec.NodeIdentity, "KnownNodes with the same nodeIdentity could not be found!")
	} else if len(kns.Items) != 0 {
		fielderr = field.Invalid(field.NewPath("spec", "nodeIdentity"), kn.Spec.NodeIdentity, "Exist knownNodes with the same nodeIdentity!")
	}

	if fielderr != nil {
		errList = append(errList, fielderr)
	}

	return warnning, errList.ToAggregate()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (w *CustomKnWebhook) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	oldKnownNodes := oldObj.(*dijkstrav2.KnownNodes)
	newKnownNodes := newObj.(*dijkstrav2.KnownNodes)
	var warnning admission.Warnings
	warnning = append(warnning, "Field .Spec.NodeIdentity cannot be modified")

	return warnning, dijkstrav2.ValidateKnownNodesUpdate(newKnownNodes, oldKnownNodes)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (w *CustomKnWebhook) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	// TODO
	return nil, nil
}
