package v1

import (
	"fmt"
	"reflect"

	"k8s.io/apimachinery/pkg/util/validation/field"
)

func ValidateDisplayCreate(dp *Display) error {
	errors := field.ErrorList{}

	if fmt.Sprintf("%v", reflect.TypeOf(dp.Spec.NodeIdentity)) != "string" {
		errors = append(errors, field.Invalid(field.NewPath("spec", "nodeIdentity"), dp.Spec.NodeIdentity, "must be string"))
	}

	if fmt.Sprintf("%v", reflect.TypeOf(dp.Spec.StartNode.Name)) != "string" {
		errors = append(errors, field.Invalid(field.NewPath("spec", "name"), dp.Spec.StartNode.Name, "must be string"))
	}

	if fmt.Sprintf("%v", reflect.TypeOf(dp.Spec.StartNode.ID)) != "int32" {
		errors = append(errors, field.Invalid(field.NewPath("spec", "startNode", "id"), dp.Spec.StartNode.ID, "must be int32"))
	} else if dp.Spec.StartNode.ID <= 0 {
		errors = append(errors, field.Invalid(field.NewPath("spec", "startNode", "id"), dp.Spec.StartNode.ID, "must be large 0"))
	}

	return errors.ToAggregate()
}

func ValidateDisplayUpdate(new, old *Display) error {
	errors := field.ErrorList{}
	// 不允许修改nodeIdentity
	if new.Spec.NodeIdentity != old.Spec.NodeIdentity {
		errors = append(errors, field.Invalid(field.NewPath("spec", "nodeIdentity"), old.Spec.NodeIdentity, "No modifications allowed"))
	}

	if fmt.Sprintf("%v", reflect.TypeOf(new.Spec.StartNode.ID)) != "int32" {
		errors = append(errors, field.Invalid(field.NewPath("spec", "startNode", "id"), new.Spec.StartNode.ID, "must be int32"))
	} else if new.Spec.StartNode.ID <= 0 {
		errors = append(errors, field.Invalid(field.NewPath("spec", "startNode", "id"), new.Spec.StartNode.ID, "must be large 0"))
	}

	return errors.ToAggregate()
}

func ValidateKnownNodesCreate(kn *KnownNodes) error {
	errors := field.ErrorList{}

	if fmt.Sprintf("%v", reflect.TypeOf(kn.Spec.NodeIdentity)) != "string" {
		errors = append(errors, field.Invalid(field.NewPath("spec", "nodeIdentity"), kn.Spec.NodeIdentity, "must be string"))
	}
	return errors.ToAggregate()
}

func ValidateKnownNodesUpdate(new, old *KnownNodes) error {
	errors := field.ErrorList{}
	// 不允许修改nodeIdentity
	if new.Spec.NodeIdentity != old.Spec.NodeIdentity {
		errors = append(errors, field.Invalid(field.NewPath("spec", "nodeIdentity"), old.Spec.NodeIdentity, "No modifications allowed"))
	}

	return errors.ToAggregate()
}
