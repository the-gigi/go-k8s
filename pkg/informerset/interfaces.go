package informerset

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"
)

type Informerset interface {
	Start()
	Stop()
	AddEventHandler(gvr schema.GroupVersionResource, handlers cache.ResourceEventHandlerFuncs) error
}
