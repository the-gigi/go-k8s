package client

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	memory "k8s.io/client-go/discovery/cached"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/util/flowcontrol"
)

type DynamicClient interface {
	dynamic.Interface
	rest.Interface
	GroupVersionResourceFor(kind schema.GroupVersionKind) (gvr schema.GroupVersionResource, err error)
}

type dynamicClient struct {
	discoveryClient *discovery.DiscoveryClient
	dynamicClient   dynamic.Interface
}

func (d *dynamicClient) GetRateLimiter() flowcontrol.RateLimiter {
	return d.discoveryClient.RESTClient().GetRateLimiter()
}

func (d *dynamicClient) Verb(verb string) *rest.Request {
	return d.discoveryClient.RESTClient().Verb(verb)
}

func (d *dynamicClient) Post() *rest.Request {
	return d.discoveryClient.RESTClient().Post()
}

func (d *dynamicClient) Put() *rest.Request {
	return d.discoveryClient.RESTClient().Put()
}

func (d *dynamicClient) Patch(pt types.PatchType) *rest.Request {
	return d.discoveryClient.RESTClient().Patch(pt)
}

func (d *dynamicClient) Get() *rest.Request {
	return d.discoveryClient.RESTClient().Get()
}

func (d *dynamicClient) Delete() *rest.Request {
	return d.discoveryClient.RESTClient().Delete()
}

func (d *dynamicClient) APIVersion() schema.GroupVersion {
	return d.discoveryClient.RESTClient().APIVersion()
}

func (d *dynamicClient) Resource(resource schema.GroupVersionResource) dynamic.NamespaceableResourceInterface {
	return d.dynamicClient.Resource(resource)
}

func (d *dynamicClient) GroupVersionResourceFor(gvk schema.GroupVersionKind) (gvr schema.GroupVersionResource, err error) {
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(d.discoveryClient))
	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return
	}
	gvr = mapping.Resource
	return
}

func NewDynamicClient(kubeConfigPath string, kubeContext string) (client DynamicClient, err error) {
	kubeConfig, err := getKubeConfig(kubeConfigPath, kubeContext)
	if err != nil {
		return
	}

	dynCli, err := dynamic.NewForConfig(kubeConfig)
	if err != nil {
		return
	}

	discCli, err := discovery.NewDiscoveryClientForConfig(kubeConfig)
	if err != nil {
		return
	}

	client = &dynamicClient{
		discoveryClient: discCli,
		dynamicClient:   dynCli,
	}

	return
}
