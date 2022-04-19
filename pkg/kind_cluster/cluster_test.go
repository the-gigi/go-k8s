package kind_cluster

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
)

var _ = Describe("Kind Cluster Tests", Ordered, Serial, func() {
	var err error
	var clusterName string
	var cluster *Cluster

	BeforeAll(func() {
		suffix, err := uuid.NewRandom()
		Ω(err).Should(BeNil())
		clusterName = "test-" + suffix.String()
	})

	AfterAll(func() {
		_, err := kugo.Run("get version --context kind-" + clusterName)
		if err == nil {
			err = cluster.Delete()
			Ω(err).Should(BeNil())
		}
	})

	It("should create cluster successfully", func() {
		cluster, err = New(clusterName, Options{})
		Ω(err).Should(BeNil())
		Ω(cluster).ShouldNot(BeNil())

		// Verify the cluster is up and running and has nodes
		nodes, err := cluster.GetNodes()
		Ω(err).Should(BeNil())

		Ω(nodes).Should(HaveLen(1))
		Ω(nodes[0]).Should(Equal(clusterName + "-control-plane"))
	})

	It("should fail to create existing cluster with no options", func() {
		cluster, err = New(clusterName, Options{})
		Ω(err).ShouldNot(BeNil())
	})

	It("should take over an existing cluster successfully", func() {
		cluster, err = New(clusterName, Options{TakeOver: true})
		Ω(err).Should(BeNil())
	})

	It("should write kubeconfig to a file successfully", func() {
		filename :=  path.Join(os.TempDir(), "go-k8s-client-test-kubeconfig")
		// Remove previous file if exists
		_, err := os.Stat(filename)
		if err == nil {
			err = os.Remove(filename)
			Ω(err).Should(BeNil())
		} else {
			Ω(errors.Is(err, os.ErrNotExist)).Should(BeTrue())
		}
		cluster, err = New(clusterName, Options{TakeOver: true, KubeConfigFile: filename})
		Ω(err).Should(BeNil())

		// Verify the file was created
		_, err = os.Stat(filename)
		Ω(err).Should(BeNil())
	})

	It("should fail to ovewrite the default kubeconfig", func() {
		filename := defaultKubeConfig

		// Verify the default kubeconfig exists
		_, err := os.Stat(filename)
		Ω(err).Should(BeNil())

		// Get its content
		origKubeConfig, err := ioutil.ReadFile(filename)
		Ω(err).Should(BeNil())

		cluster, err = New(clusterName, Options{TakeOver: true, KubeConfigFile: filename})
		Ω(err).ShouldNot(BeNil())

		// Verify the original file is still there with the same content
		kubeConfig, err := ioutil.ReadFile(filename)
		Ω(err).Should(BeNil())
		Ω(bytes.Equal(kubeConfig, origKubeConfig)).Should(BeTrue())
	})

	It("should re-create an existing cluster successfully", func() {
		cluster, err = New(clusterName, Options{Recreate: true})
		Ω(err).Should(BeNil())
		// Verify the cluster is up and running and has nodes
		nodes, err := cluster.GetNodes()
		Ω(err).Should(BeNil())

		Ω(nodes).Should(HaveLen(1))
		Ω(nodes[0]).Should(Equal(clusterName + "-control-plane"))
	})

	It("should delete cluster successfully", func() {
		// Verify the cluster exists before deleting it
		exists, err := cluster.Exists()
		Ω(err).Should(BeNil())
		Ω(exists).Should(BeTrue())

		err = cluster.Delete()
		Ω(err).Should(BeNil())

		// Verify the cluster is gone
		exists, err = cluster.Exists()
		Ω(err).Should(BeNil())
		Ω(exists).Should(BeFalse())
	})

	It("should delete non-existing cluster auccessfully", func() {
		// Verify the cluster doesn't exist
		cluster = &Cluster{name: "no-such-cluster"}
		exists, err := cluster.Exists()
		Ω(err).Should(BeNil())
		Ω(exists).Should(BeFalse())

		err = cluster.Delete()
		Ω(err).Should(BeNil())
	})
})
