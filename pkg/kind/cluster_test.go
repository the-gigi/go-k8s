package kind

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/the-gigi/kugo"
	"os"
	"path"
	//"os"
	//"path"
)

var _ = Describe("Cluster tests", Ordered, func() {
	var (
		cluster          *Cluster
		clusterName      string
		err              error
		existingClusters map[string]bool
	)

	BeforeAll(func() {
		existingClusters, err = getClusters()
		Ω(err).To(BeNil())

		suffix, err := uuid.NewRandom()
		Ω(err).To(BeNil())
		clusterName = "test-" + suffix.String()
	})

	AfterAll(func() {
		// delete all clusters created by the tests
		clusters, err := getClusters()
		Ω(err).To(BeNil())
		// delete clusters that are not members of existingClusters
		for cluster := range clusters {
			if !existingClusters[cluster] {
				_, err := run("delete cluster --name " + cluster)
				Ω(err).To(BeNil())
			}
		}
	})

	Context("Cluster creation and deletion tests", Ordered, func() {
		AfterAll(func() {
			_, err := kugo.Run("version --context kind-" + clusterName)
			if err == nil {
				err = cluster.Delete()
				Ω(err).To(BeNil())
			}
		})
		It("should create a new cluster", func() {
			cluster, err = New(clusterName, Options{})
			Ω(err).To(BeNil())
			Ω(cluster).ToNot(BeNil())

			nodes, err := cluster.GetNodes()
			Ω(err).To(BeNil())

			Ω(nodes).To(HaveLen(1))
			Ω(nodes[0]).To(Equal(clusterName + "-control-plane"))
		})

		It("should take over an existing cluster", func() {
			cluster, err = New(clusterName, Options{TakeOver: true})
			Ω(err).To(BeNil())
			Ω(cluster).ToNot(BeNil())
		})
		It("should fail to create an existing cluster with no options", func() {
			cluster, err = New(clusterName, Options{})
			Ω(err).ToNot(BeNil())
		})

		It("should recreate an existing cluster", func() {
			var err error
			cluster, err = New(clusterName, Options{Recreate: true})
			Ω(err).To(BeNil())
			// Verify the cluster is up and running and has nodes
			var nodes []string
			nodes, err = cluster.GetNodes()
			Ω(err).To(BeNil())
			Ω(nodes).Should(HaveLen(1))
			Ω(nodes[0]).Should(Equal(clusterName + "-control-plane"))
		})

		It("should delete existing cluster successfully", func() {
			// Verify the cluster exists before deleting it
			exists, err := cluster.Exists()
			Ω(err).To(BeNil())
			Ω(exists).To(BeTrue())

			err = cluster.Delete()
			Ω(err).To(BeNil())

			// Verify the cluster is gone
			exists, err = cluster.Exists()
			Ω(err).To(BeNil())
			Ω(exists).To(BeFalse())
		})

		It("should succeed to delete non-existent cluster", func() {
			cluster := &Cluster{name: "no-such-cluster"}
			// Verify the cluster doesn't exist
			exists, err := cluster.Exists()
			Ω(err).To(BeNil())
			Ω(exists).To(BeFalse())

			// Try to delete and verify it results in an error
			err = cluster.Delete()
			Ω(err).To(BeNil())

			// Verify the cluster still doesn't exist
			exists, err = cluster.Exists()
			Ω(err).To(BeNil())
			Ω(exists).To(BeFalse())
		})
	})

	Context("Write KubeConfig File", func() {
		It("should succeed to write to another kubeconfig file", func() {
			filename := path.Join(os.TempDir(), "go-k8s-client-test-kubeconfig")
			// Remove previous file if exists
			_, err := os.Stat(filename)
			if err == nil {
				err = os.Remove(filename)
				Ω(err).To(BeNil())
			} else {
				Ω(os.IsNotExist(err)).Should(BeTrue())
			}
			cluster, err = New(clusterName, Options{TakeOver: true, KubeConfigFile: filename})
			Ω(err).To(BeNil())

			// Verify the file was created
			_, err = os.Stat(filename)
			Ω(err).To(BeNil())
		})

		It("should fail to write the default kubeconfig file", func() {
			filename := defaultKubeConfig
			// Verify the default kubeconfig exists
			_, err := os.Stat(filename)
			Ω(err).To(BeNil())

			// Get its content
			var origKubeConfig []byte
			origKubeConfig, err = os.ReadFile(filename)
			Ω(err).To(BeNil())

			cluster, err = New(clusterName, Options{TakeOver: true, KubeConfigFile: filename})
			Ω(err).ToNot(BeNil())

			// Verify the original file is still there with the same content
			var kubeConfig []byte
			kubeConfig, err = os.ReadFile(filename)

			Ω(err).To(BeNil())
			Ω(kubeConfig).To(Equal(origKubeConfig))
		})
	})
})

/*




func (s *ClusterTestSuite) TestDeleteNonExistingCluster() {
    cluster := &Cluster{name: "no-such-cluster"}
    exists, err := cluster.Exists()
    s.Require().Nil(err)
    s.Require().False(exists)

    err = s.cluster.Delete()
    s.Require().Nil(err)

    // Verify the cluster still doesn't exist
    exists, err = cluster.Exists()
    s.Require().Nil(err)
    s.Require().False(exists)
}
*/
