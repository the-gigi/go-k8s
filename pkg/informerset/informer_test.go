package informerset

import (
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/the-gigi/go-k8s/pkg/client"
	"github.com/the-gigi/go-k8s/pkg/kind"
	"github.com/the-gigi/kugo"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"os"
)

const (
	clusterName = "go-k8s-client-test"
	testImage   = "k8s.gcr.io/pause:3.1"
)

var kubeConfigFile = os.TempDir() + clusterName + "-kubeconfig"

// createDeployment deploy 3 replicas of the pause container and waits for deployment to be ready
func createDeployment() {
	cmd := fmt.Sprintf("create deployment test-deployment --image %s --replicas 3 -n ns-1 --kubeconfig %s", testImage, kubeConfigFile)
	_, err := kugo.Run(cmd)
	Ω(err).Should(BeNil())

	// wait for the deployment to exist (otherwise the subsequent wait command might fail)
	cmd = "get deployment test-deployment -n ns-1 --kubeconfig " + kubeConfigFile
	var done = make(chan struct{})
	var output string
	wait.Until(func() {
		output, err = kugo.Run(cmd)
		if err != nil || strings.Contains(output, "not found") {
			return
		}
		close(done)
	}, time.Second, done)
	// wait for deployment to be ready
	cmd = "wait deployment test-deployment --for condition=Available=True --timeout 60s -n ns-1 --kubeconfig " + kubeConfigFile
	_, err = kugo.Run(cmd)
	Ω(err).Should(BeNil())
}

var _ = Describe("Informer Tests", Ordered, func() {
	var err error

	var podsGVR = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
	var deploymentsGVR = schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
	var dynamicClient dynamic.Interface
	var ins Informerset
	var cluster *kind.Cluster
	var options Options

	BeforeAll(func() {
		cluster, err = kind.New(clusterName, kind.Options{TakeOver: true, KubeConfigFile: kubeConfigFile})
		Ω(err).Should(BeNil())

		dynamicClient, err = client.NewDynamicClient(kubeConfigFile)
		Ω(err).Should(BeNil())

		gvrs := []schema.GroupVersionResource{
			podsGVR,
			deploymentsGVR,
		}
		options = Options{
			Gvrs:   gvrs,
			Client: dynamicClient,
		}
	})

	BeforeEach(func() {
		err = cluster.Clear()
		Ω(err).Should(BeNil())

		// Create namespace ns-1
		_, err = kugo.Run("create ns ns-1 --kubeconfig " + cluster.GetKubeConfig())
		Ω(err).Should(BeNil())

		ins, err = NewInformerset(options)
		Ω(err).Should(BeNil())
	})

	It("should successfully watch for ns-1 pods and deployments", func() {
		pods := []string{}
		deployments := []string{}
		go func() {
			err = ins.AddEventHandler(podsGVR, cache.ResourceEventHandlerFuncs{
				AddFunc: func(obj interface{}) {
					u := obj.(*unstructured.Unstructured)
					if u.GetNamespace() != "ns-1" {
						return
					}
					pods = append(pods, u.GetName())
				},
			})
			Ω(err).Should(BeNil())
			err = ins.AddEventHandler(deploymentsGVR, cache.ResourceEventHandlerFuncs{
				AddFunc: func(obj interface{}) {
					u := obj.(*unstructured.Unstructured)
					if u.GetNamespace() != "ns-1" {
						return
					}
					deployments = append(deployments, u.GetName())
				},
			})
			Ω(err).Should(BeNil())
		}()

		// Start listening
		ins.Start()

		// Create a deployment with 3 pods, which will generate events
		createDeployment()

		// Wait for all 3 pods of the deployment to be created or until 5 seconds have passed
		for i := 0; i < 5; i++ {
			if len(pods) == 3 {
				break
			}
			time.Sleep(time.Second)
		}

		Ω(deployments).Should(HaveLen(1))
		Ω(deployments[0]).Should(Equal("test-deployment"))
		Ω(pods).Should(HaveLen(3))
		for _, pod := range pods {
			Ω(pod).Should(MatchRegexp("test-deployment-.*"))
		}

		// Stop the informer
		ins.Stop()
	})

	It("should successfully list ns-1 pods and deployments", func() {
		pods := []runtime.Object{}
		deployments := []runtime.Object{}

		// Start listening
		ins.Start()

		// Create a deployment with 3 pods, which will generate events
		createDeployment()

		// Wait for all 3 pods of the deployment to be created or until 5 seconds have passed
		selector := labels.NewSelector()
		ok := false
		var objectMap map[schema.GroupVersionResource][]runtime.Object
		for i := 0; i < 5; i++ {
			objectMap, err = ins.List(selector, "ns-1")
			Ω(err).Should(BeNil())
			pods, ok = objectMap[podsGVR]
			if ok && len(pods) == 3 {
				break
			}
			time.Sleep(time.Second)
		}

		deployments = objectMap[deploymentsGVR]
		Ω(deployments).Should(HaveLen(1))
		deployment, err := runtime.DefaultUnstructuredConverter.ToUnstructured(deployments[0])
		Ω(err).Should(BeNil())
		metadata := deployment["metadata"].(map[string]interface{})
		name := metadata["name"].(string)
		Ω(name).Should(Equal("test-deployment"))
		Ω(pods).Should(HaveLen(3))
		for i := range pods {
			pod, err := runtime.DefaultUnstructuredConverter.ToUnstructured(pods[i])
			Ω(err).Should(BeNil())
			metadata := pod["metadata"].(map[string]interface{})
			name = metadata["name"].(string)
			Ω(name).Should(MatchRegexp("test-deployment-.*"))
		}

		// Stop the informer
		ins.Stop()
	})
})
