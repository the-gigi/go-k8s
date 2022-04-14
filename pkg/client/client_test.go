package client

import (
	"context"
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/the-gigi/kugo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"github.com/the-gigi/go-k8s/pkg/kind_cluster"
	"strings"

	//"k8s.io/client-go/dynamic"
	//"k8s.io/client-go/kubernetes"

)

const (
	clusterName = "go-k8s-client-test"
	kubeContext = "kind-" + clusterName
	kubeConfigFile = "/tmp/" + clusterName + "-kubeconfig"
)

var (
	builtinNamespaces = map[string]bool{
		"default": true,
		"kube-node-lease": true,
		"kube-public": true,
		"kube-system": true,
		"local-path-storage": true,
	}
)

// Delete all namespaces except the built-in namespaces
func resetCluster() (err error) {
	output, err := kugo.Get(kugo.GetRequest{
		BaseRequest:    kugo.BaseRequest{
			KubeContext: kubeContext,
		},
		Kind: "ns",
		Output: "name",
	})
	if err != nil {
		return
	}

	output = strings.Replace(output, "namespace/", "", -1)
	namespaces := strings.Split(output, "\n")
	for _, ns := range namespaces {
		if !builtinNamespaces[ns] {
			cmd := fmt.Sprintf("delete ns %s --context %s", ns, kubeContext)
			_, err = kugo.Run(cmd)
			if err != nil {
				return
			}
		}
	}
	return
}

var _ = Describe("Client Tests", Ordered, func() {
	var err error
	var dynamicClient DynamicClient
	//var kubernetes.Interface clientset

	var podsGVR schema.GroupVersionResource
	var pods *unstructured.UnstructuredList

	BeforeAll(func() {
		_, err = kind_cluster.New(clusterName, kind_cluster.Options{TakeOver: true, KubeConfigFile: kubeConfigFile})
		Ω(err).Should(BeNil())

		err = resetCluster()
		Ω(err).Should(BeNil())
	})

	BeforeEach(func() {
		dynamicClient, err = NewDynamicClient(kubeConfigFile)
		Ω(err).Should(BeNil())
		Ω(dynamicClient).ShouldNot(BeNil())

		podsGVR = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
	})

	It("should get pods successfully", func() {
		pods, err = dynamicClient.Resource(podsGVR).List(context.Background(), metav1.ListOptions{})
		Ω(err).Should(BeNil())
		Ω(pods).ShouldNot(BeNil())
	})
})
