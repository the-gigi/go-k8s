package client

import (
	"context"
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/the-gigi/go-k8s/pkg/kind"
	"github.com/the-gigi/kugo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"os"
	"path"
	"time"
)

const (
	clusterName = "go-k8s-client-test"
	testImage   = "gcr.io/google_containers/pause"
)

var kubeConfigFile = path.Join(os.TempDir(), clusterName+"-kubeconfig")

var _ = Describe("Client Tests", Ordered, func() {
	var err error
	var dynamicClient DynamicClient
	var clientset kubernetes.Interface

	var podsGVR = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
	var pods []unstructured.Unstructured

	BeforeAll(func() {
		c, err := kind.New(clusterName, kind.Options{TakeOver: true, KubeConfigFile: kubeConfigFile})
		Ω(err).Should(BeNil())

		err = c.Clear()
		Ω(err).Should(BeNil())

		// Create namespace ns-1 and deploy 3 replicas of the pausecontainer
		cmd := fmt.Sprintf("create ns ns-1 --kubeconfig %s --context %s", kubeConfigFile, c.GetKubeContext())
		_, err = kugo.Run(cmd)
		Ω(err).Should(BeNil())

		cmd = fmt.Sprintf(`create deployment test-deployment 
                                 --image %s --replicas 3 -n ns-1 
                                 --kubeconfig %s --context %s`, testImage, kubeConfigFile, c.GetKubeContext())
		_, err = kugo.Run(cmd)
		Ω(err).Should(BeNil())

		// wait for deployment to be ready
		cmd = fmt.Sprintf(`wait deployment test-deployment --for condition=Available=True --timeout 60s
                                   -n ns-1 
                                   --kubeconfig %s 
                                   --context %s`, kubeConfigFile, c.GetKubeContext())
		for i := 0; i < 5; i++ {
			_, err = kugo.Run(cmd)
			if err == nil {
				break
			}
			time.Sleep(time.Second)
		}
		Ω(err).Should(BeNil())
	})

	BeforeEach(func() {
		dynamicClient, err = NewDynamicClient(kubeConfigFile)
		Ω(err).Should(BeNil())
		Ω(dynamicClient).ShouldNot(BeNil())

		clientset, err = NewClientset(kubeConfigFile)
		Ω(err).Should(BeNil())
		Ω(clientset).ShouldNot(BeNil())
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
