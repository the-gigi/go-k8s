package kind

import (
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/the-gigi/kugo"
	"testing"
)

type ClusterTestSuite struct {
	suite.Suite

	clusterName string
	cluster     *Cluster
}

// assert operator similar to Gomega'a å
var å *assert.Assertions

func (s *ClusterTestSuite) SetupSuite() {
	å = s.Assert()

	suffix, err := uuid.NewRandom()
	å.Nil(err)
	s.clusterName = "test-" + suffix.String()
}

func (s *ClusterTestSuite) TeardownSuite() {
	_, err := kugo.Run("get version --context kind-" + s.clusterName)
	if err == nil {
		err = s.cluster.Delete()
		å.Nil(err)
	}
}

func (s *ClusterTestSuite) TestCreateCluster() {
	var err error

	// Create new cluster
	s.cluster, err = New(s.clusterName, Options{})
	å.Nil(err)
	å.NotNil(s.cluster)

	// Verify the cluster is up and running and has nodes
	nodes, err := s.cluster.GetNodes()
	å.Nil(err)

	å.Len(nodes, 1)
	å.Equal(nodes[0], s.clusterName+"-control-plane")

	// should take over existinbg cluster successfully
	s.cluster, err = New(s.clusterName, Options{TakeOver: true})
	å.Nil(err)
	å.NotNil(s.cluster)

	// should fail to create existing cluster with no options
	s.cluster, err = New(s.clusterName, Options{})
	å.NotNil(err)

}

//
//	It("should write kubeconfig to a file successfully", func() {
//		filename := path.Join(os.TempDir(), "go-k8s-client-test-kubeconfig")
//		// Remove previous file if exists
//		_, err := os.Stat(filename)
//		if err == nil {
//			err = os.Remove(filename)
//			å.Nil(err)
//		} else {
//			å(errors.Is(err, os.ErrNotExist)).Should(BeTrue())
//		}
//		cluster, err = New(clusterName, Options{TakeOver: true, KubeConfigFile: filename})
//		å.Nil(err)
//
//		// Verify the file was created
//		_, err = os.Stat(filename)
//		å.Nil(err)
//	})
//
//	It("should fail to ovewrite the default kubeconfig", func() {
//		filename := defaultKubeConfig
//
//		// Verify the default kubeconfig exists
//		_, err := os.Stat(filename)
//		å.Nil(err)
//
//		// Get its content
//		origKubeConfig, err := os.ReadFile(filename)
//		å.Nil(err)
//
//		cluster, err = New(clusterName, Options{TakeOver: true, KubeConfigFile: filename})
//		å.NotNil(err)
//
//		// Verify the original file is still there with the same content
//		kubeConfig, err := os.ReadFile(filename)
//		å.Nil(err)
//		å(bytes.Equal(kubeConfig, origKubeConfig)).Should(BeTrue())
//	})
//
//	It("should re-create an existing cluster successfully", func() {
//		cluster, err = New(clusterName, Options{Recreate: true})
//		å.Nil(err)
//		// Verify the cluster is up and running and has nodes
//		nodes, err := cluster.GetNodes()
//		å.Nil(err)
//
//		å(nodes).Should(HaveLen(1))
//		å(nodes[0]).Should(Equal(clusterName + "-control-plane"))
//	})
//
//	It("should delete cluster successfully", func() {
//		// Verify the cluster exists before deleting it
//		exists, err := cluster.Exists()
//		å.Nil(err)
//		å(exists).Should(BeTrue())
//
//		err = cluster.Delete()
//		å.Nil(err)
//
//		// Verify the cluster is gone
//		exists, err = cluster.Exists()
//		å.Nil(err)
//		å(exists).Should(BeFalse())
//	})
//
//	It("should delete non-existing cluster auccessfully", func() {
//		// Verify the cluster doesn't exist
//		cluster = &Cluster{name: "no-such-cluster"}
//		exists, err := cluster.Exists()
//		å.Nil(err)
//		å(exists).Should(BeFalse())
//
//		err = cluster.Delete()
//		å.Nil(err)
//	})
//})

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestClusterTestSuite(t *testing.T) {
	suite.Run(t, new(ClusterTestSuite))
}

//var _ = Describe("Kind Cluster Tests", Ordered, Serial, func() {
//	var err error
//	var clusterName string
//	var cluster *Cluster
//
//	BeforeAll(func() {
//		suffix, err := uuid.NewRandom()
//		å.Nil(err)
//		clusterName = "test-" + suffix.String()
//	})
//
//	AfterAll(func() {
//		_, err := kugo.Run("get version --context kind-" + clusterName)
//		if err == nil {
//			err = cluster.Delete()
//			å.Nil(err)
//		}
//	})
//
//	It("should create cluster successfully", func() {
//		cluster, err = New(clusterName, Options{})
//		å.Nil(err)
//		å(cluster).ShouldNot(BeNil())
//
//		// Verify the cluster is up and running and has nodes
//		nodes, err := cluster.GetNodes()
//		å.Nil(err)
//
//		å(nodes).Should(HaveLen(1))
//		å(nodes[0]).Should(Equal(clusterName + "-control-plane"))
//	})
//
//	It("should fail to create existing cluster with no options", func() {
//		cluster, err = New(clusterName, Options{})
//		å.NotNil(err)
//	})
//
//	It("should take over an existing cluster successfully", func() {
//		cluster, err = New(clusterName, Options{TakeOver: true})
//		å.Nil(err)
//	})
//
//	It("should write kubeconfig to a file successfully", func() {
//		filename := path.Join(os.TempDir(), "go-k8s-client-test-kubeconfig")
//		// Remove previous file if exists
//		_, err := os.Stat(filename)
//		if err == nil {
//			err = os.Remove(filename)
//			å.Nil(err)
//		} else {
//			å(errors.Is(err, os.ErrNotExist)).Should(BeTrue())
//		}
//		cluster, err = New(clusterName, Options{TakeOver: true, KubeConfigFile: filename})
//		å.Nil(err)
//
//		// Verify the file was created
//		_, err = os.Stat(filename)
//		å.Nil(err)
//	})
//
//	It("should fail to ovewrite the default kubeconfig", func() {
//		filename := defaultKubeConfig
//
//		// Verify the default kubeconfig exists
//		_, err := os.Stat(filename)
//		å.Nil(err)
//
//		// Get its content
//		origKubeConfig, err := os.ReadFile(filename)
//		å.Nil(err)
//
//		cluster, err = New(clusterName, Options{TakeOver: true, KubeConfigFile: filename})
//		å.NotNil(err)
//
//		// Verify the original file is still there with the same content
//		kubeConfig, err := os.ReadFile(filename)
//		å.Nil(err)
//		å(bytes.Equal(kubeConfig, origKubeConfig)).Should(BeTrue())
//	})
//
//	It("should re-create an existing cluster successfully", func() {
//		cluster, err = New(clusterName, Options{Recreate: true})
//		å.Nil(err)
//		// Verify the cluster is up and running and has nodes
//		nodes, err := cluster.GetNodes()
//		å.Nil(err)
//
//		å(nodes).Should(HaveLen(1))
//		å(nodes[0]).Should(Equal(clusterName + "-control-plane"))
//	})
//
//	It("should delete cluster successfully", func() {
//		// Verify the cluster exists before deleting it
//		exists, err := cluster.Exists()
//		å.Nil(err)
//		å(exists).Should(BeTrue())
//
//		err = cluster.Delete()
//		å.Nil(err)
//
//		// Verify the cluster is gone
//		exists, err = cluster.Exists()
//		å.Nil(err)
//		å(exists).Should(BeFalse())
//	})
//
//	It("should delete non-existing cluster auccessfully", func() {
//		// Verify the cluster doesn't exist
//		cluster = &Cluster{name: "no-such-cluster"}
//		exists, err := cluster.Exists()
//		å.Nil(err)
//		å(exists).Should(BeFalse())
//
//		err = cluster.Delete()
//		å.Nil(err)
//	})
//})
