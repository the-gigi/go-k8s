package client

import (
	"context"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	//"k8s.io/client-go/dynamic"
	//"k8s.io/client-go/kubernetes"

	"os"
)


var _ = Describe("Client Tests", func() {
	var err error
	var dynamicClient DynamicClient
	//var kubernetes.Interface clientset

	var podsGVR schema.GroupVersionResource
	var pods *unstructured.UnstructuredList

	BeforeEach(func() {
		kubeConfigPath := os.ExpandEnv("$HOME/.kube/config")
		dynamicClient, err = NewDynamicClient(kubeConfigPath)
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
