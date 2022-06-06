package local_cluster

import (
	"bytes"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"github.com/the-gigi/kugo"
	"io/ioutil"
	"os"
	"path"
	"strings"
)

var _ = FDescribe("Kind Cluster Tests", Ordered, Serial, func() {
	var err error
	var clusterName string
	var theCluster Cluster
	var kubeConfigFile string

	BeforeAll(func() {
		suffix, err := uuid.NewRandom()
		Ω(err).Should(BeNil())
		clusterName = "test-" + suffix.String()
		kubeConfigFile = path.Join(os.TempDir(), "kubeconfig-"+clusterName)
	})

	AfterAll(func() {
		_, err := kugo.Run("get version --context kind-" + clusterName)
		if err == nil {
			err = theCluster.Delete()
			Ω(err).Should(BeNil())
		}
	})

	It("should create cluster successfully", func() {
		theCluster, err = newCluster("kind", clusterName, Options{
			KubeConfigFile: kubeConfigFile,
		})
		Ω(err).Should(BeNil())
		Ω(theCluster).ShouldNot(BeNil())

		// Verify the theCluster is up and running
		args := strings.Split("cluster-info --kubeconfig "+kubeConfigFile, " ")
		output, err := kugo.Run(args...)
		Ω(err).Should(BeNil())
		Ω(output).Should(MatchRegexp(".*Kubernetes control plane.*is running at"))
	})

	It("should fail to create existing theCluster with no options", func() {
		theCluster, err = newCluster("kind", clusterName, Options{})
		Ω(err).ShouldNot(BeNil())
	})

	It("should take over an existing theCluster successfully", func() {
		theCluster, err = newCluster("kind", clusterName, Options{TakeOver: true})
		Ω(err).Should(BeNil())
	})

	It("should write kubeconfig to a file successfully", func() {
		filename := path.Join(os.TempDir(), "go-k8s-cluster-test-kubeconfig")
		// Remove previous file if exists
		_, err := os.Stat(filename)
		if err == nil {
			err = os.Remove(filename)
			Ω(err).Should(BeNil())
		} else {
			Ω(errors.Is(err, os.ErrNotExist)).Should(BeTrue())
		}
		theCluster, err = newCluster("kind", clusterName, Options{TakeOver: true, KubeConfigFile: filename})
		Ω(err).Should(BeNil())

		// Verify the file was created
		output, err := os.Stat(filename)
		Ω(err).Should(BeNil())
		Ω(output).ShouldNot(BeEmpty())
	})

	It("should fail to overwrite the default kubeconfig", func() {
		filename := defaultKubeConfig

		// Verify the default kubeconfig exists
		_, err := os.Stat(filename)
		Ω(err).Should(BeNil())

		// Get its content
		origKubeConfig, err := ioutil.ReadFile(filename)
		Ω(err).Should(BeNil())

		theCluster, err = newCluster("kind", clusterName, Options{TakeOver: true, KubeConfigFile: filename})
		Ω(err).ShouldNot(BeNil())

		// Verify the original file is still there with the same content
		kubeConfig, err := ioutil.ReadFile(filename)
		Ω(err).Should(BeNil())
		Ω(bytes.Equal(kubeConfig, origKubeConfig)).Should(BeTrue())
	})

	It("should re-create an existing cluster successfully", func() {
		theCluster, err = newCluster("kind", clusterName, Options{Recreate: true})
		Ω(err).Should(BeNil())
		// Verify the theCluster is up and running and has nodes
		args := strings.Split("cluster-info --kubeconfig "+kubeConfigFile, " ")
		output, err := kugo.Run(args...)
		Ω(err).Should(BeNil())
		Ω(strings.Contains(output, "Kubernetes control plane is running at")).Should(BeTrue())
	})

	It("should delete a cluster successfully", func() {
		// Verify the theCluster exists before deleting it
		exists, err := theCluster.Exists()
		Ω(err).Should(BeNil())
		Ω(exists).Should(BeTrue())

		err = theCluster.Delete()
		Ω(err).Should(BeNil())

		// Verify the theCluster is gone
		exists, err = theCluster.Exists()
		Ω(err).Should(BeNil())
		Ω(exists).Should(BeFalse())
	})

	It("should delete non-existing cluster auccessfully", func() {
		// Verify the theCluster doesn't exist
		cl := &cluster{
			name: "no-such-cluster",
		}
		exists, err := cl.Exists()
		Ω(err).Should(BeNil())
		Ω(exists).Should(BeFalse())

		err = cl.Delete()
		Ω(err).Should(BeNil())
	})
})
