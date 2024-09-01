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
var displaylog = logf.Log.WithName("display-resource")

func (dp *Display) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(dp).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-dijkstra-jinli-io-v1-display,mutating=true,failurePolicy=fail,sideEffects=None,groups=dijkstra.jinli.io,resources=displays,verbs=create;update,versions=v1,name=mdisplay.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &Display{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (dp *Display) Default() {
	displaylog.Info("default", "name", dp.Name)

	//默认值设置
	if dp.Spec.NodeIdentity == "" {
		dp.Spec.NodeIdentity = "jinli-default-nodeid"
	}

	// 添加标签
	labels := make(map[string]string)
	labels["nodeIdentity"] = dp.Spec.NodeIdentity
	dp.Labels = labels

	// 添加默认状态
	dp.Status.LastUpdate = metav1.Time{}
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-dijkstra-jinli-io-v1-display,mutating=false,failurePolicy=fail,sideEffects=None,groups=dijkstra.jinli.io,resources=displays,verbs=create;update,versions=v1,name=vdisplay.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &Display{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (dp *Display) ValidateCreate() (admission.Warnings, error) {
	var warnning admission.Warnings
	displaylog.Info("validate create", "name", dp.Name)

	warnning = append(warnning, "KnownNodes with the same nodeIdentity need to be created before creating Display!")
	warnning = append(warnning, "Display with the same nodeIdentity and startNode.ID not allow!")
	warnning = append(warnning, "The total number of Display cannot exceed the number of Nodes of KnownNodes with the same nodeIdentity!")

	return warnning, ValidateDisplayCreate(dp)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (dp *Display) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	var warnning admission.Warnings
	displaylog.Info("validate update", "name", dp.Name)
	oldDisplay := old.(*Display)

	warnning = append(warnning, "Field .Spec.NodeIdentity cannot be modified")
	return warnning, ValidateDisplayUpdate(dp, oldDisplay)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (dp *Display) ValidateDelete() (admission.Warnings, error) {
	// TODO
	return nil, nil
}
