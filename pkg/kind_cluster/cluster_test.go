package kind_cluster

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/the-gigi/kugo"
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
