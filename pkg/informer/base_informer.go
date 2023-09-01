package informer

import (
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
)

type baseInformer struct {
	sharedIndexInformer cache.SharedIndexInformer
	lister              cache.GenericLister
}

func (in *baseInformer) AddEventHandler(handler cache.ResourceEventHandler) {
	in.sharedIndexInformer.AddEventHandler(handler)
}

func (in *baseInformer) List(selector labels.Selector, namespace string) (objects []runtime.Object, err error) {
	if namespace == "" {
		objects, err = in.lister.List(selector)
		return
	}
	objects, err = in.lister.ByNamespace(namespace).List(selector)
	return
}

func (in *baseInformer) Get(name string) (runtime.Object, error) {
	return in.lister.Get(name)
}

func newBaseInformer(gi informers.GenericInformer) (in *baseInformer, err error) {
	if gi == nil {
		err = errors.New("generic baseInformer can't be nil")
		return
	}
	in = &baseInformer{
		sharedIndexInformer: gi.Informer(),
		lister:              gi.Lister(),
	}
	return
}
