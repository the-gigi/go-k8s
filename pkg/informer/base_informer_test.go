package informer

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/the-gigi/go-k8s/pkg/client"
	"github.com/the-gigi/go-k8s/pkg/kind"
	"github.com/the-gigi/kugo"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
	"time"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

var _ = Describe("Informer Tests", Ordered, func() {
	var err error

	var podsGVR = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
	var deploymentsGVR = schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
	var dynamicClient dynamic.Interface
	var inf *informerFactory
	var cluster *kind.Cluster
	var options Options
	var bii BaseInformer

	BeforeAll(func() {
		cluster, err = kind.New(clusterName, kind.Options{TakeOver: true, KubeConfigFile: kubeConfigFile})
		Ω(err).Should(BeNil())

		dynamicClient, err = client.NewDynamicClient(kubeConfigFile, "")
		Ω(err).Should(BeNil())
		options = Options{
			Client:        dynamicClient,
			DefaultResync: 0,
			WaitForSync:   true,
		}
	})

	AfterEach(func() {
		// Stop the informer before cleanup to ensure proper ordering
		if inf != nil {
			inf.Stop()
			// Wait for informer to fully stop and drain any pending events
			time.Sleep(500 * time.Millisecond)
		}

		// Clear any remaining resources in the test namespace
		if cluster != nil {
			// Delete all deployments in ns-1
			_, _ = kugo.Run("delete deployments --all -n ns-1 --kubeconfig " + cluster.GetKubeConfig())
			// Delete all pods in ns-1
			_, _ = kugo.Run("delete pods --all -n ns-1 --kubeconfig " + cluster.GetKubeConfig())
			// Wait for cleanup to complete
			time.Sleep(300 * time.Millisecond)
		}
	})

	BeforeEach(func() {
		err = cluster.Clear()
		Ω(err).Should(BeNil())

		// Wait for cluster to be fully cleared and caches to reset
		time.Sleep(500 * time.Millisecond)

		// Create namespace ns-1
		_, err = kugo.Run("create ns ns-1 --kubeconfig " + cluster.GetKubeConfig())
		Ω(err).Should(BeNil())

		// Create a fresh informer factory for each test to avoid shared state
		var f Factory
		f, err = NewInformerFactory(options)
		Ω(err).Should(BeNil())

		var ok bool
		inf, ok = f.(*informerFactory)
		Ω(ok).Should(BeTrue())

		// Reset any global test state
		bii = nil
	})

	It("should successfully watch for ns-1 pods and deployments", func() {
		var pods []string
		var deployments []string

		bii, err = inf.GetBaseInformer(podsGVR)
		Ω(err).Should(BeNil())

		eh := cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				u := obj.(*unstructured.Unstructured)
				if u.GetNamespace() != "ns-1" {
					return
				}
				pods = append(pods, u.GetName())
			},
		}

		bii.AddEventHandler(eh)
		Ω(err).Should(BeNil())

		bii, err = inf.GetBaseInformer(deploymentsGVR)
		bii.AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				u := obj.(*unstructured.Unstructured)
				if u.GetNamespace() != "ns-1" {
					return
				}
				deployments = append(deployments, u.GetName())
			},
		})
		Ω(err).Should(BeNil())

		// Start listening
		inf.Start()

		// Create a deployment with 3 pods, which will generate events
		deploymentName := createDeployment()

		// Wait for all 3 pods of the deployment to be created or until 5 seconds have passed
		for i := 0; i < 5; i++ {
			if len(pods) == 3 {
				break
			}
			time.Sleep(time.Second)
		}

		Ω(deployments).Should(HaveLen(1))
		Ω(deployments[0]).Should(Equal(deploymentName))
		Ω(pods).Should(HaveLen(3))
		for _, pod := range pods {
			Ω(pod).Should(MatchRegexp(deploymentName + "-.*"))
		}

		objs, _ := bii.List(labels.NewSelector(), "ns-1")
		Ω(objs).Should(HaveLen(1))

		// Stop the informer
		inf.Stop()
	})

	It("should successfully list ns-1 pods and deployments", func() {
		var pods []runtime.Object
		var podsInformer BaseInformer
		podsInformer, err = inf.GetBaseInformer(podsGVR)
		Ω(err).Should(BeNil())

		var deployments []runtime.Object
		var deploymentsInformer BaseInformer
		deploymentsInformer, err = inf.GetBaseInformer(deploymentsGVR)
		Ω(err).Should(BeNil())

		// Start listening
		inf.Start()

		// Create a deployment with 3 pods, which will generate events
		deploymentName := createDeployment()

		// Wait for all 3 pods of the deployment to be created or until 5 seconds have passed
		for i := 0; i < 5; i++ {
			pods, err = podsInformer.List(labels.NewSelector(), "ns-1")
			Ω(err).Should(BeNil())
			if len(pods) == 3 {
				break
			}

			// List the deployments
			deployments, err = deploymentsInformer.List(labels.NewSelector(), "ns-1")
			Ω(err).Should(BeNil())

			time.Sleep(time.Second)
		}

		Ω(pods).Should(HaveLen(3))
		for _, pod := range pods {
			u := pod.(*unstructured.Unstructured)
			if u.GetNamespace() != "ns-1" {
				return
			}
			Ω(u.GetName()).Should(MatchRegexp(deploymentName + "-.*"))
		}

		// List the deployments
		deployments, err = deploymentsInformer.List(labels.NewSelector(), "ns-1")
		Ω(err).Should(BeNil())
		Ω(deployments).Should(HaveLen(1))
		deployment := deployments[0].(*unstructured.Unstructured)
		Ω(deployment.GetName()).Should(Equal(deploymentName))

		// Stop the informer
		inf.Stop()
	})
})
