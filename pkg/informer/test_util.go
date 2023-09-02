package informer

import (
	"fmt"
	. "github.com/onsi/gomega"
	"github.com/the-gigi/kugo"
	"k8s.io/apimachinery/pkg/util/wait"
	"strings"
	"time"

	"os"
)

const (
	clusterName = "go-k8s-client-test"
	testImage   = "registry.k8s.io/pause:3.9"
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
