package client

import (
    "context"
    "fmt"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/suite"
    "os"
    "path"
    "testing"
    //"time"
    //"context"
    "github.com/the-gigi/go-k8s/pkg/kind"
    "github.com/the-gigi/kugo"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
    "k8s.io/apimachinery/pkg/runtime/schema"
    "k8s.io/client-go/kubernetes"
)

const (
    clusterName = "go-k8s-client-test"
    testImage   = "registry.k8s.io/pause:3.8"
)

var kubeConfigFile = path.Join(os.TempDir(), clusterName+"-kubeconfig")

type ClientTestSuite struct {
    suite.Suite

    dynamicClient DynamicClient
    clientset     kubernetes.Interface
    podsGVR       schema.GroupVersionResource
    pods          []unstructured.Unstructured
}

// assert operator similar to Gomega'a Ω
var å *assert.Assertions

func (s *ClientTestSuite) SetupSuite() {
    å = s.Assert()
    s.podsGVR = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
    c, err := kind.New(clusterName, kind.Options{TakeOver: true, KubeConfigFile: kubeConfigFile})
    å.Nil(err)

    err = c.Clear()
    å.Nil(err)
    // Create namespace ns-1 and deploy 3 replicas of the pausecontainer
    cmd := fmt.Sprintf("create ns ns-1 --kubeconfig %s --context %s", kubeConfigFile, c.GetKubeContext())
    _, err = kugo.Run(cmd)
    å.Nil(err)

    cmd = fmt.Sprintf(`create deployment test-deployment
	                           --image %s --replicas 3 -n ns-1
	                           --kubeconfig %s --context %s`, testImage, kubeConfigFile, c.GetKubeContext())
    _, err = kugo.Run(cmd)
    å.Nil(err)

    // wait for deployment to be ready
    cmd = fmt.Sprintf(`wait deployment test-deployment --for condition=Available=True --timeout 60s
	                             -n ns-1
	                             --kubeconfig %s
	                             --context %s`, kubeConfigFile, c.GetKubeContext())
    _, err = kugo.Run(cmd)
    å.Nil(err)
}

func (s *ClientTestSuite) SetupTest() {
    var err error
    s.dynamicClient, err = NewDynamicClient(kubeConfigFile)
    å.Nil(err)
    å.NotNil(s.dynamicClient)

    s.clientset, err = NewClientset(kubeConfigFile)
    å.Nil(err)
    å.NotNil(s.clientset)
}

func (s *ClientTestSuite) TestGetPodsWithDynamicClient() {
    podList, err := s.dynamicClient.Resource(s.podsGVR).Namespace("ns-1").List(context.Background(), metav1.ListOptions{})
    å.Nil(err)
    å.NotNil(podList)

    pods := podList.Items
    å.NotNil(pods)
    å.Len(pods, 3)
}

func (s *ClientTestSuite) TestGetPodsWithClientset() {
    podList, err := s.clientset.CoreV1().Pods("ns-1").List(context.Background(), metav1.ListOptions{})
    å.Nil(err)
    å.NotNil(podList)

    pods := podList.Items
    å.NotNil(pods)
    å.Len(pods, 3)
    å.Equal(pods[0].Spec.Containers[0].Image, testImage)
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestClientTestSuite(t *testing.T) {
    suite.Run(t, new(ClientTestSuite))
}
