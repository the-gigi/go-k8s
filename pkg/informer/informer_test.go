package informer

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/the-gigi/go-k8s/pkg/client"
	"github.com/the-gigi/go-k8s/pkg/kind"
	"github.com/the-gigi/kugo"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"time"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

type podEventHandler struct {
	Added   []corev1.Pod
	Updated []corev1.Pod // storing old, new one after the other for each update
	Deleted []corev1.Pod
}

func (h *podEventHandler) OnAdd(obj corev1.Pod) {
	if obj.Namespace == "ns-1" {
		h.Added = append(h.Added, obj)
	}
}

func (h *podEventHandler) OnUpdate(oldObj corev1.Pod, newObj corev1.Pod) {
	if oldObj.Namespace == "ns-1" {
		h.Updated = append(h.Updated, oldObj, newObj)
	}
}

func (h *podEventHandler) OnDelete(obj corev1.Pod) {
	if obj.GetNamespace() == "ns-1" {
		h.Deleted = append(h.Deleted, obj)
	}
}

type deploymentEventHandler struct {
	Added   []appsv1.Deployment
	Updated []appsv1.Deployment // storing old, new one after the other for each update
	Deleted []appsv1.Deployment
}

func (h *deploymentEventHandler) OnAdd(obj appsv1.Deployment) {
	if obj.Namespace == "ns-1" {
		h.Added = append(h.Added, obj)
	}
}

func (h *deploymentEventHandler) OnUpdate(oldObj appsv1.Deployment, newObj appsv1.Deployment) {
	if oldObj.Namespace == "ns-1" {
		h.Updated = append(h.Updated, oldObj, newObj)
	}
}

func (h *deploymentEventHandler) OnDelete(obj appsv1.Deployment) {
	if obj.GetNamespace() == "ns-1" {
		h.Deleted = append(h.Deleted, obj)
	}
}

var _ = Describe("Informer Tests", Ordered, func() {
	var err error

	var podsGVR = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
	var deploymentsGVR = schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
	var dynamicClient dynamic.Interface
	var inf *informerFactory
	var cluster *kind.Cluster
	var options Options
	var podInformer Informer[corev1.Pod]
	var deploymentInformer Informer[appsv1.Deployment]

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

	BeforeEach(func() {
		err = cluster.Clear()
		Ω(err).Should(BeNil())

		// Create namespace ns-1
		_, err = kugo.Run("create ns ns-1 --kubeconfig " + cluster.GetKubeConfig())
		Ω(err).Should(BeNil())

		var f Factory
		f, err = NewInformerFactory(options)
		Ω(err).Should(BeNil())

		var ok bool
		inf, ok = f.(*informerFactory)
		Ω(ok).Should(BeTrue())
	})

	It("should successfully watch for ns-1 pods and deployments", func() {
		podInformer, err = NewInformer[corev1.Pod](inf, podsGVR)
		Ω(err).Should(BeNil())

		peh := &podEventHandler{}
		err = podInformer.AddEventHandler(peh)
		Ω(err).Should(BeNil())

		deploymentInformer, err = NewInformer[appsv1.Deployment](inf, deploymentsGVR)
		Ω(err).Should(BeNil())

		deh := &deploymentEventHandler{}
		err = deploymentInformer.AddEventHandler(deh)
		Ω(err).Should(BeNil())

		// Start listening
		inf.Start()

		// Create a deployment with 3 pods, which will generate events
		createDeployment()

		// Wait for all 3 pods of the deployment to be created or until 5 seconds have passed
		for i := 0; i < 5; i++ {
			if len(peh.Added) == 3 {
				break
			}
			time.Sleep(time.Second)
		}

		Ω(deh.Added).Should(HaveLen(1))
		Ω(deh.Added[0].Name).Should(Equal("test-deployment"))
		Ω(peh.Added).Should(HaveLen(3))
		for _, pod := range peh.Added {
			Ω(pod.Name).Should(MatchRegexp("test-deployment-.*"))
		}
		// Stop the informer
		inf.Stop()
	})

	It("should successfully list ns-1 pods and deployments", func() {
		var podInformer Informer[corev1.Pod]
		podInformer, err = NewInformer[corev1.Pod](inf, podsGVR)
		Ω(err).Should(BeNil())

		var deploymentInformer Informer[appsv1.Deployment]
		deploymentInformer, err = NewInformer[appsv1.Deployment](inf, deploymentsGVR)
		Ω(err).Should(BeNil())

		// Start listening
		inf.Start()

		// Create a deployment with 3 pods, which will generate events
		createDeployment()

		var pods []corev1.Pod
		var deployments []appsv1.Deployment

		// Wait for all 3 pods of the deployment to be created or until 5 seconds have passed
		for i := 0; i < 5; i++ {
			pods, err = podInformer.List(labels.NewSelector(), "ns-1")
			Ω(err).Should(BeNil())
			if len(pods) == 3 {
				break
			}

			// List the deployments
			deployments, err = deploymentInformer.List(labels.NewSelector(), "ns-1")
			Ω(err).Should(BeNil())

			time.Sleep(time.Second)
		}

		Ω(pods).Should(HaveLen(3))
		for _, pod := range pods {
			Ω(pod.Name).Should(MatchRegexp("test-deployment-.*"))
		}

		// List the deployments
		deployments, err = deploymentInformer.List(labels.NewSelector(), "ns-1")
		Ω(err).Should(BeNil())
		found := false
		for _, deployment := range deployments {
			if deployment.Name == "test-deployment" {
				found = true
				break
			}
		}

		Ω(found).Should(BeTrue())

		// Stop the informer
		inf.Stop()
	})
})
