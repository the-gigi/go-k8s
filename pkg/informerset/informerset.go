package informerset

import (
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
	"sync"
	"time"
)

type informerset struct {
	informers   map[schema.GroupVersionResource]*informer
	stopChannel chan struct{}
	waitForSync bool
	factory     dynamicinformer.DynamicSharedInformerFactory
	m           sync.Mutex
}

// Start - start all
func (is *informerset) Start() {
	is.m.Lock()
	defer is.m.Unlock()
	is.factory.Start(is.stopChannel)
	if is.waitForSync {
		// Ignore returned map of synced informers.
		// It's relevant only if Stop() was called before Start() channel was closed and then all bets are off
		is.factory.WaitForCacheSync(is.stopChannel)
	}
}

// Stop - stops all informers from listening and waiting for sync
func (is *informerset) Stop() {
	is.stopChannel <- struct{}{}
}

func (is *informerset) AddEventHandler(gvr schema.GroupVersionResource, handlers cache.ResourceEventHandlerFuncs) (err error) {
	is.m.Lock()
	defer is.m.Unlock()

	in, ok := is.informers[gvr]
	if !ok {
		err = errors.Errorf("informerset doesn't support GVR: '%v'", gvr)
		return
	}
	in.AddEventHandler(handlers)
	return
}

func (is *informerset) List(selector labels.Selector, namespace string) (objects map[schema.GroupVersionResource][]runtime.Object, err error) {
	objects = map[schema.GroupVersionResource][]runtime.Object{}
	var list []runtime.Object
	for gvr, inf := range is.informers {
		list, err = inf.List(selector, namespace)
		if err != nil {
			return
		}
		objects[gvr] = list
	}
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
	var in *informer
	for _, gvr := range o.Gvrs {
		in, err = newInformer(factory.ForResource(gvr))
		if err != nil {
			return
		}
		informerMap[gvr] = in
	}

	ins = &informerset{
		informers:   informerMap,
		stopChannel: make(chan struct{}),
		waitForSync: o.WaitForSync,
		factory:     factory,
	}
	return
}
