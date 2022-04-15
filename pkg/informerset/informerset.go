package informerset

import (
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
	"time"
)

type informerset struct {
	Informers   map[schema.GroupVersionResource]*informer
	StopChannel chan struct{}
}

func (is *informerset) Stop() {
	is.StopChannel <- struct{}{}
}

func (is *informerset) AddEventHandler(gvr schema.GroupVersionResource, handlers cache.ResourceEventHandlerFuncs) (err error) {
	in, ok := is.Informers[gvr]
	if !ok {
		err = errors.Errorf("informerset doesn't support GVR: '%v'", gvr)
		return
	}
	in.AddEventHandler(handlers)
	return
}

type Options struct {
	Gvrs          []schema.GroupVersionResource
	Client        dynamic.Interface
	Namespace     string
	DefaultResync time.Duration
	WaitForSync   bool
}

func NewInformerset(o Options) (ins Informerset, err error) {
	f := dynamicinformer.NewFilteredDynamicSharedInformerFactory
	namespace := o.Namespace
	if namespace == "" {
		namespace = corev1.NamespaceAll
	}

	informerMap := map[schema.GroupVersionResource]*informer{}
	factory := f(o.Client, o.DefaultResync, o.Namespace, nil)
	for _, gvr := range o.Gvrs {
		in, err := newInformer(factory.ForResource(gvr))
		if err != nil {
			return
		}
		informerMap[gvr] = in
	}

	var stopCh chan struct{}
	factory.Start(stopCh)
	factory.WaitForCacheSync(stopCh)

	ins = &informerset{Informers: informerMap}
	return
}
