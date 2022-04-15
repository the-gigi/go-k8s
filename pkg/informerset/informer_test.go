package informerset

import (
	"context"
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/the-gigi/go-k8s/pkg/kind_cluster"
	"github.com/the-gigi/kugo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"strings"
)

// Delete all namespaces except the built-in namespaces
func resetCluster() (err error) {
	output, err := kugo.Get(kugo.GetRequest{
		BaseRequest: kugo.BaseRequest{
			KubeContext: kubeContext,
		},
		Kind:   "ns",
		Output: "name",
	})
	if err != nil {
		return
	}

	output = strings.Replace(output, "namespace/", "", -1)
	namespaces := strings.Split(output, "\n")
	for _, ns := range namespaces {
		if !builtinNamespaces[ns] && ns != "" {
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
	var clientset kubernetes.Interface

	var podsGVR schema.GroupVersionResource
	var pods []unstructured.Unstructured

	BeforeAll(func() {
		_, err = kind_cluster.New(clusterName, kind_cluster.Options{TakeOver: true, KubeConfigFile: kubeConfigFile})
		Ω(err).Should(BeNil())

		err = resetCluster()
		Ω(err).Should(BeNil())

		// Create namespace ns-1 and deploy 3 replicas of the pausecontainer
		_, err = kugo.Run("create ns ns-1 --context " + kubeContext)
		Ω(err).Should(BeNil())

		cmd := fmt.Sprintf("create deployment test-deployment --image %s --replicas 3 -n ns-1 --context %s", testImage, kubeContext)
		_, err = kugo.Run(cmd)
		Ω(err).Should(BeNil())

		// wait for deployment to be ready
		cmd = "wait deployment test-deployment --for condition=Available=True --timeout 60s"
		_, err = kugo.Run(cmd)
		Ω(err).Should(BeNil())
	})

	BeforeEach(func() {
		dynamicClient, err = NewDynamicClient(kubeConfigFile)
		Ω(err).Should(BeNil())
		Ω(dynamicClient).ShouldNot(BeNil())

		clientset, err = NewClientset(kubeConfigFile)
		Ω(err).Should(BeNil())
		Ω(clientset).ShouldNot(BeNil())

		podsGVR = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
	})

	It("should get pods successfully with the dynamic client", func() {
		podList, err := dynamicClient.Resource(podsGVR).Namespace("ns-1").List(context.Background(), metav1.ListOptions{})
		Ω(err).Should(BeNil())

		pods = podList.Items
		Ω(pods).ShouldNot(BeNil())
		Ω(pods).Should(HaveLen(3))
	})

	It("should get pods successfully with the clientset", func() {
		podList, err := clientset.CoreV1().Pods("ns-1").List(context.Background(), metav1.ListOptions{})
		Ω(err).Should(BeNil())

		pods := podList.Items
		Ω(pods).ShouldNot(BeNil())
		Ω(pods).Should(HaveLen(3))

		Ω(pods[0].Spec.Containers[0].Image).Should(Equal(testImage))
	})
})
