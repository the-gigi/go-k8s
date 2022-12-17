package client

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/suite"
	"github.com/the-gigi/go-k8s/pkg/kind"
	"github.com/the-gigi/kugo"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
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

	podsGVR schema.GroupVersionResource
	pods    []unstructured.Unstructured
}

func (s *ClientTestSuite) SetupSuite() {
	fmt.Println("Setup suite starting...")
	s.podsGVR = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
	fmt.Println("Creating kind cluster...")
	c, err := kind.New(clusterName, kind.Options{TakeOver: true, KubeConfigFile: kubeConfigFile})
	s.Require().Nil(err)

	err = c.Clear()
	s.Require().Nil(err)
	fmt.Println("Creating namespace and deploying pause container...")
	// Create namespace ns-1 and deploy 3 replicas of the pausecontainer
	cmd := fmt.Sprintf("create ns ns-1 --kubeconfig %s --context %s", kubeConfigFile, c.GetKubeContext())
	_, err = kugo.Run(cmd)
	s.Require().Nil(err)

	cmd = fmt.Sprintf(`create deployment test-deployment
	                           --image %s --replicas 3 -n ns-1
	                           --kubeconfig %s --context %s`, testImage, kubeConfigFile, c.GetKubeContext())
	_, err = kugo.Run(cmd)
	s.Require().Nil(err)

	fmt.Println("Waiting for deployment to be ready...")
	// wait for deployment to be ready
	cmd = fmt.Sprintf(`wait deployment test-deployment --for condition=Available=True --timeout 60s
	                             -n ns-1
	                             --kubeconfig %s
	                             --context %s`, kubeConfigFile, c.GetKubeContext())
	_, err = kugo.Run(cmd)
	s.Require().Nil(err)
	fmt.Println("Setup suite is done.")
}

func (s *ClientTestSuite) TestGetPodsWithDynamicClient() {
	dynamicClient, err := NewDynamicClient(kubeConfigFile, "")
	s.Require().Nil(err)
	s.Require().NotNil(dynamicClient)

	podList, err := dynamicClient.Resource(s.podsGVR).Namespace("ns-1").List(context.Background(), metav1.ListOptions{})
	s.Require().Nil(err)
	s.Require().NotNil(podList)

	pods := podList.Items
	s.Require().NotNil(pods)
	s.Require().Len(pods, 3)

	pod := pods[0].Object
	metadata := pod["metadata"].(map[string]interface{})
	labels := metadata["labels"].(map[string]interface{})
	s.Require().Contains(labels, "app")
	app := labels["app"].(string)
	s.Require().Equal(app, "test-deployment")

	var p corev1.Pod
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(pod, &p)
	s.Require().Nil(err)
	app = p.ObjectMeta.Labels["app"]
	s.Require().Equal(app, "test-deployment")
}

func (s *ClientTestSuite) TestGetPodsWithClientset() {
	clientset, err := NewClientset(kubeConfigFile, "")
	s.Require().Nil(err)
	s.Require().NotNil(clientset)

	podList, err := clientset.CoreV1().Pods("ns-1").List(context.Background(), metav1.ListOptions{})
	s.Require().Nil(err)
	s.Require().NotNil(podList)

	pods := podList.Items
	s.Require().NotNil(pods)
	s.Require().Len(pods, 3)
	app, ok := pods[0].ObjectMeta.Labels["app"]
	s.Require().True(ok)
	s.Require().Equal(app, "test-deployment")
	s.Require().Equal(pods[0].Spec.Containers[0].Image, testImage)
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestClientTestSuite(t *testing.T) {
	suite.Run(t, new(ClientTestSuite))
}
