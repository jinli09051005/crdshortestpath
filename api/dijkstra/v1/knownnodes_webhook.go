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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var knownnodeslog = logf.Log.WithName("knownnodes-resource")

func (kn *KnownNodes) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(kn).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-dijkstra-jinli-io-v1-knownnodes,mutating=true,failurePolicy=fail,sideEffects=None,groups=dijkstra.jinli.io,resources=knownnodeses,verbs=create;update,versions=v1,name=mknownnodes.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &KnownNodes{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (kn *KnownNodes) Default() {
	knownnodeslog.Info("default", "name", kn.Name)

	//默认值设置
	if kn.Spec.NodeIdentity == "" {
		kn.Spec.NodeIdentity = "jinli-default-nodeid"
	}

	// 添加标签
	labels := make(map[string]string)
	labels["nodeIdentity"] = kn.Spec.NodeIdentity
	kn.Labels = labels

	// 添加默认状态
	kn.Status.LastUpdate = metav1.Time{}
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-dijkstra-jinli-io-v1-knownnodes,mutating=false,failurePolicy=fail,sideEffects=None,groups=dijkstra.jinli.io,resources=knownnodeses,verbs=create;update,versions=v1,name=vknownnodes.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &KnownNodes{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (kn *KnownNodes) ValidateCreate() (admission.Warnings, error) {
	var warnning admission.Warnings
	knownnodeslog.Info("validate create", "name", kn.Name)

	warnning = append(warnning, "KnownNodes with the same nodeIdentity not allow!")

	return warnning, ValidateKnownNodesCreate(kn)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (kn *KnownNodes) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	var warnning admission.Warnings
	knownnodeslog.Info("validate update", "name", kn.Name)
	oldKnownNodes := old.(*KnownNodes)

	warnning = append(warnning, "Field .Spec.NodeIdentity cannot be modified")
	return warnning, ValidateKnownNodesUpdate(kn, oldKnownNodes)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (kn *KnownNodes) ValidateDelete() (admission.Warnings, error) {
	// TODO
	return nil, nil
}
