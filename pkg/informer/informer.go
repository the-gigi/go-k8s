package informer

import (
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type informer[T any] struct {
    base  baseInformer
    sinks []EventHandler[T]
}

func (in *informer[T]) OnAdd(obj interface{}, isInInitialList bool) {
    u := obj.(*unstructured.Unstructured)

    var object T
    err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &object)
    if err != nil {
        return
    }

    for _, sink := range in.sinks {
        sink.OnAdd(object)
    }
}

func (in *informer[T]) OnUpdate(oldObj interface{}, newObj interface{}) {
    u := oldObj.(*unstructured.Unstructured)

    var oldObject T
    err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &oldObject)
    if err != nil {
        return
    }

    u = newObj.(*unstructured.Unstructured)

    var newObject T
    err = runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &newObject)
    if err != nil {
        return
    }

    for _, sink := range in.sinks {
        sink.OnUpdate(oldObject, newObject)
    }
}

func (in *informer[T]) OnDelete(obj interface{}) {
    u := obj.(*unstructured.Unstructured)

    var object T
    err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &object)
    if err != nil {
        return
    }

    for _, sink := range in.sinks {
        sink.OnAdd(object)
    }
}

func (in *informer[T]) List(selector labels.Selector, namespace string) (objects []T, err error) {
    var runtimeObjects []runtime.Object
    if namespace == "" {
        runtimeObjects, err = in.base.lister.List(selector)
    } else {
        runtimeObjects, err = in.base.lister.ByNamespace(namespace).List(selector)
    }

    if err != nil {
        return
    }

    // convert runtime objects to typed objects
    for _, ro := range runtimeObjects {
        var object T
        err = runtime.DefaultUnstructuredConverter.FromUnstructured(ro.(*unstructured.Unstructured).Object, &object)
        if err != nil {
            return
        }
        objects = append(objects, object)
    }

    return
}

func (in *informer[T]) Get(name string, obj *T) (err error) {
    var ro runtime.Object
    ro, err = in.base.lister.Get(name)
    if err != nil {
        return
    }

    err = runtime.DefaultUnstructuredConverter.FromUnstructured(ro.(*unstructured.Unstructured).Object, &obj)
    return
}

func (in *informer[T]) AddEventHandler(handler EventHandler[T]) (err error) {
    if handler == nil {
        err = errors.New("handler can't be nil")
    }
    in.sinks = append(in.sinks, handler)
    return
}

func NewInformer[T any](f Factory, gvr schema.GroupVersionResource) (in Informer[T], err error) {
    if f == nil {
        err = errors.New("generic informer can't be nil")
        return
    }

    var bii BaseInformer
    bii, err = f.GetBaseInformer(gvr)
    if err != nil {
        return
    }

    bi, ok := bii.(*baseInformer)
    if !ok {
        err = errors.New("invalid base informer")
        return
    }

    in = &informer[T]{
        base: *bi,
    }

    bi.AddEventHandler(in.(*informer[T]))

    return
}
