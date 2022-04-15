package informerset

import (
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
)

type informer struct {
	sharedIndexInformer cache.SharedIndexInformer
	lister              cache.GenericLister
}

func (in *informer) AddEventHandler(handler cache.ResourceEventHandler) {
	in.sharedIndexInformer.AddEventHandler(handler)
}

func (in *informer) List(selector labels.Selector, namespace string) (ret []runtime.Object, err error) {
	if namespace == "" {
		ret, err = in.lister.List(selector)
		return
	}
	ret, err = in.lister.ByNamespace(namespace).List(selector)
	return
}

func (in *informer) Get(name string) (runtime.Object, error) {
	return in.lister.Get(name)
}

func newInformer(gi informers.GenericInformer) (in *informer, err error) {
	if gi == nil {
		err = errors.New("generic informer can't be nil")
		return
	}
	in = &informer{
		sharedIndexInformer: gi.Informer(),
		lister:              gi.Lister(),
	}
	return
}
