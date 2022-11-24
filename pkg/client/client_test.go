package client

import (
    "context"
    "fmt"
    "github.com/stretchr/testify/suite"
    "github.com/the-gigi/go-k8s/pkg/kind"
    "github.com/the-gigi/kugo"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
    "k8s.io/apimachinery/pkg/runtime/schema"
    "os"
    "path"
    "testing"
)

const (
    clusterName = "go-k8s-client-test"
    testImage   = "registry.k8s.io/pause:3.8"
)

var kubeConfigFile = path.Join(os.TempDir(), clusterName+"-kubeconfig")

type ClientTestSuite struct {
    suite.Suite

    //dynamicClient DynamicClient
    //clientset     kubernetes.Interface
    podsGVR schema.GroupVersionResource
    pods    []unstructured.Unstructured
}

func (s *ClientTestSuite) SetupSuite() {
    s.podsGVR = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
    c, err := kind.New(clusterName, kind.Options{TakeOver: true, KubeConfigFile: kubeConfigFile})
    s.Require().Nil(err)

    err = c.Clear()
    s.Require().Nil(err)
    // Create namespace ns-1 and deploy 3 replicas of the pausecontainer
    cmd := fmt.Sprintf("create ns ns-1 --kubeconfig %s --context %s", kubeConfigFile, c.GetKubeContext())
    _, err = kugo.Run(cmd)
    s.Require().Nil(err)

    cmd = fmt.Sprintf(`create deployment test-deployment
	                           --image %s --replicas 3 -n ns-1
	                           --kubeconfig %s --context %s`, testImage, kubeConfigFile, c.GetKubeContext())
    _, err = kugo.Run(cmd)
    s.Require().Nil(err)

    // wait for deployment to be ready
    cmd = fmt.Sprintf(`wait deployment test-deployment --for condition=Available=True --timeout 60s
	                             -n ns-1
	                             --kubeconfig %s
	                             --context %s`, kubeConfigFile, c.GetKubeContext())
    _, err = kugo.Run(cmd)
    s.Require().Nil(err)
}

func (s *ClientTestSuite) TestGetPodsWithDynamicClient() {
    dynamicClient, err := NewDynamicClient(kubeConfigFile)
    s.Require().Nil(err)
    s.Require().NotNil(dynamicClient)

    podList, err := dynamicClient.Resource(s.podsGVR).Namespace("ns-1").List(context.Background(), metav1.ListOptions{})
    s.Require().Nil(err)
    s.Require().NotNil(podList)

    pods := podList.Items
    s.Require().NotNil(pods)
    s.Require().Len(pods, 3)
}

func (s *ClientTestSuite) TestGetPodsWithClientset() {
    clientset, err := NewClientset(kubeConfigFile)
    s.Require().Nil(err)
    s.Require().NotNil(clientset)

    podList, err := clientset.CoreV1().Pods("ns-1").List(context.Background(), metav1.ListOptions{})
    s.Require().Nil(err)
    s.Require().NotNil(podList)

    pods := podList.Items
    s.Require().NotNil(pods)
    s.Require().Len(pods, 3)
    s.Require().Equal(pods[0].Spec.Containers[0].Image, testImage)
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestClientTestSuite(t *testing.T) {
    suite.Run(t, new(ClientTestSuite))
}
