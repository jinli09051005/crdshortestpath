package v2

import (
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

var _ conversion.Hub = &KnownNodes{}

// Hub marks that a given type is the hub type for conversion
func (kn *KnownNodes) Hub() {}

var _ conversion.Hub = &Display{}

// Hub marks that a given type is the hub type for conversion
func (dp *Display) Hub() {}
