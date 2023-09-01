package informer

import (
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"
)

type BaseInformer interface {
	AddEventHandler(handler cache.ResourceEventHandler)
	List(selector labels.Selector, namespace string) (objects []runtime.Object, err error)
	Get(name string) (runtime.Object, error)
}

type Factory interface {
	Start()
	Stop()
	GetBaseInformer(gvr schema.GroupVersionResource) (BaseInformer, error)
}

type EventHandler[T any] interface {
	OnAdd(obj T)
	OnUpdate(oldObj T, newObj T)
	OnDelete(obj T)
}

type Informer[T any] interface {
	AddEventHandler(handler EventHandler[T])
	List(selector labels.Selector, namespace string) (objects []T, err error)
	Get(name string, obj *T) error
}
