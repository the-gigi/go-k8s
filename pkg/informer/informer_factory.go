package informer

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"sync"
	"time"
)

type informerFactory struct {
	stopChannel chan struct{}
	waitForSync bool
	factory     dynamicinformer.DynamicSharedInformerFactory
	m           sync.Mutex
}

// Start - start all
func (f *informerFactory) Start() {
	f.m.Lock()
	defer f.m.Unlock()
	f.factory.Start(f.stopChannel)
	if f.waitForSync {
		// Ignore returned map of synced informers.
		// It's relevant only if Stop() was called before Start() channel was closed and then all bets are off
		f.factory.WaitForCacheSync(f.stopChannel)
	}
}

// Stop - stops all informers from listening and waiting for sync
func (f *informerFactory) Stop() {
	f.stopChannel <- struct{}{}
}

type Options struct {
	Client        dynamic.Interface
	Namespace     string
	DefaultResync time.Duration
	WaitForSync   bool
}

func (f *informerFactory) GetBaseInformer(gvr schema.GroupVersionResource) (in BaseInformer, err error) {
	return newBaseInformer(f.factory.ForResource(gvr))
}

func NewInformerFactory(o Options) (factory Factory, err error) {
	f := dynamicinformer.NewFilteredDynamicSharedInformerFactory(o.Client, o.DefaultResync, o.Namespace, nil)

	factory = &informerFactory{
		stopChannel: make(chan struct{}),
		waitForSync: o.WaitForSync,
		factory:     f,
	}
	return
}
