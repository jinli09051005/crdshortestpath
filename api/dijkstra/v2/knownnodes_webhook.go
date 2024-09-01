package v2

import (
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

func (kn *KnownNodes) SetupWebhookWithManager(mgr ctrl.Manager, d webhook.CustomDefaulter, v webhook.CustomValidator) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(kn).
		WithDefaulter(d).
		WithValidator(v).
		Complete()
}
