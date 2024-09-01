package v2

import (
	"k8s.io/apimachinery/pkg/runtime"
)

// 注册资源类型、转换函数、默认值函数
var (
	SchemeBuilder      runtime.SchemeBuilder
	localSchemeBuilder = &SchemeBuilder
	AddToScheme        = localSchemeBuilder.AddToScheme
)

func init() {
	localSchemeBuilder.Register(addKnownTypes)
}
