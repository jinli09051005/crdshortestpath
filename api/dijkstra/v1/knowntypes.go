package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	Group   = "dijkstra.jinli.io"
	Version = "v1"
)

var SchemeGroupVersion = schema.GroupVersion{
	Group:   Group,
	Version: Version,
}

func addKnownTypes(scheme *runtime.Scheme) error {
	// 外部资源类型注册到scheme
	scheme.AddKnownTypes(SchemeGroupVersion,
		&KnownNodes{},
		&KnownNodesList{},
		&Display{},
		&DisplayList{},
	)
	//将自定义的API组版本添加到全局metav1的注册表，让它们可以被Kubernetes系统识别和使用
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}

// Resource takes an unqualified resource and returns a LocalGroup qualified
// GroupResource
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}
