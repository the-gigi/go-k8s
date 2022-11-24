package kind

import (
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"github.com/the-gigi/kugo"
	"os"
	"path"
	"testing"
)

type ClusterTestSuite struct {
    suite.Suite

    clusterName string
    cluster     *Cluster
}

func (s *ClusterTestSuite) SetupSuite() {
    suffix, err := uuid.NewRandom()
    s.Require().Nil(err)
    s.clusterName = "test-" + suffix.String()
}

func (s *ClusterTestSuite) TeardownSuite() {
    _, err := kugo.Run("get version --context kind-" + s.clusterName)
    if err == nil {
        err = s.cluster.Delete()
        s.Require().Nil(err)
    }
}

func (s *ClusterTestSuite) TestCreateCluster() {
    var err error

    // Create new cluster
    s.cluster, err = New(s.clusterName, Options{})
    s.Require().Nil(err)
    s.Require().NotNil(s.cluster)

    // Verify the cluster is up and running and has nodes
    nodes, err := s.cluster.GetNodes()
    s.Require().Nil(err)

    s.Require().Len(nodes, 1)
    s.Require().Equal(nodes[0], s.clusterName+"-control-plane")

    // should take over existinbg cluster successfully
    s.cluster, err = New(s.clusterName, Options{TakeOver: true})
    s.Require().Nil(err)
    s.Require().NotNil(s.cluster)

    // should fail to create existing cluster with no options
    s.cluster, err = New(s.clusterName, Options{})
    s.Require().NotNil(err)

}

func (s *ClusterTestSuite) TestWriteKubeConfigFile() {
    filename := path.Join(os.TempDir(), "go-k8s-client-test-kubeconfig")
    // Remove previous file if exists
    _, err := os.Stat(filename)
    if err == nil {
        err = os.Remove(filename)
        s.Require().Nil(err)
    } else {
        s.Require().ErrorIs(err, os.ErrNotExist)
    }
    s.cluster, err = New(s.clusterName, Options{TakeOver: true, KubeConfigFile: filename})
    s.Require().Nil(err)

    // Verify the file was created
    _, err = os.Stat(filename)
    s.Require().Nil(err)
}

func (s *ClusterTestSuite) TestWriteDefaultKubeConfigFile_Fail() {
    filename := defaultKubeConfig

    // Verify the default kubeconfig exists
    _, err := os.Stat(filename)
    s.Require().Nil(err)

    // Get its content
    origKubeConfig, err := os.ReadFile(filename)
    s.Require().Nil(err)

    s.cluster, err = New(s.clusterName, Options{TakeOver: true, KubeConfigFile: filename})
    s.Require().NotNil(err)

    // Verify the original file is still there with the same content
    kubeConfig, err := os.ReadFile(filename)
    s.Require().Nil(err)
    s.Require().Equal(kubeConfig, origKubeConfig)
}

func (s *ClusterTestSuite) TestRecreateCluster() {
    var err error
    s.cluster, err = New(s.clusterName, Options{Recreate: true})
    s.Require().Nil(err)
    // Verify the cluster is up and running and has nodes
    nodes, err := s.cluster.GetNodes()
    s.Require().Nil(err)
    s.Require().Len(nodes, 1)
    s.Require().Equal(nodes[0], s.clusterName+"-control-plane")
}

func (s *ClusterTestSuite) TestDeleteExisitngCluster() {
    // Verify the cluster exists before deleting it
    exists, err := s.cluster.Exists()
    s.Require().Nil(err)
    s.Require().True(exists)

    err = s.cluster.Delete()
    s.Require().Nil(err)

    // Verify the cluster is gone
    exists, err = s.cluster.Exists()
    s.Require().Nil(err)
    s.Require().False(exists)
}

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

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestClusterTestSuite(t *testing.T) {
    suite.Run(t, new(ClusterTestSuite))
}
