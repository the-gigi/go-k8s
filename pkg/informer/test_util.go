package informer

import (
	"fmt"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/wait"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
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
	cmd := exec.Command("kubectl", "create", "deployment", deploymentName, "--image", testImage, "--replicas", "3", "-n", "ns-1", "--kubeconfig", kubeConfigFile)
	err := cmd.Run()
	Ω(err).Should(BeNil())

	// wait for the deployment to exist (otherwise the subsequent wait command might fail)
	var done = make(chan struct{})
	wait.Until(func() {
		cmd := exec.Command("kubectl", "get", "deployment", deploymentName, "-n", "ns-1", "--kubeconfig", kubeConfigFile)
		outputBytes, err := cmd.CombinedOutput()
		output := string(outputBytes)
		if err != nil || strings.Contains(output, "not found") {
			return
		}
		close(done)
	}, time.Second, done)
	// wait for deployment to be ready
	cmd = exec.Command("kubectl", "wait", "deployment", deploymentName, "--for", "condition=Available=True", "--timeout", "60s", "-n", "ns-1", "--kubeconfig", kubeConfigFile)
	err = cmd.Run()
	Ω(err).Should(BeNil())

	return deploymentName
}
