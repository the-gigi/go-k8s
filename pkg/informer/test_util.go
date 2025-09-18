package informer

import (
	"fmt"
	. "github.com/onsi/gomega"
	"github.com/the-gigi/kugo"
	"k8s.io/apimachinery/pkg/util/wait"
	"strings"
	"time"
	"sync"

	"os"
)

const (
	clusterName = "go-k8s-client-test"
	testImage   = "registry.k8s.io/pause:3.9"
)

var kubeConfigFile = os.TempDir() + clusterName + "-kubeconfig"

var (
	testCounter int64
	testMutex   sync.Mutex
)

// getUniqueDeploymentName generates a unique deployment name for each test
func getUniqueDeploymentName() string {
	testMutex.Lock()
	defer testMutex.Unlock()
	testCounter++
	return fmt.Sprintf("test-deployment-%d-%d", time.Now().UnixNano(), testCounter)
}

// createDeployment deploy 3 replicas of the pause container and waits for deployment to be ready
func createDeployment() string {
	deploymentName := getUniqueDeploymentName()
	cmd := fmt.Sprintf("create deployment %s --image %s --replicas 3 -n ns-1 --kubeconfig %s", deploymentName, testImage, kubeConfigFile)
	_, err := kugo.Run(cmd)
	Ω(err).Should(BeNil())

	// wait for the deployment to exist (otherwise the subsequent wait command might fail)
	cmd = fmt.Sprintf("get deployment %s -n ns-1 --kubeconfig %s", deploymentName, kubeConfigFile)
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
	cmd = fmt.Sprintf("wait deployment %s --for condition=Available=True --timeout 60s -n ns-1 --kubeconfig %s", deploymentName, kubeConfigFile)
	_, err = kugo.Run(cmd)
	Ω(err).Should(BeNil())

	return deploymentName
}
