package informerset

import (
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"
)

type Informerset interface {
	Start()
	Stop()
	AddEventHandler(gvr schema.GroupVersionResource, handlers cache.ResourceEventHandlerFuncs) error
	List(selector labels.Selector, namespace string) (objects map[schema.GroupVersionResource][]runtime.Object, err error)
}
