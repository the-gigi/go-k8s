package kind

import (
    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"
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

func (s *ClusterTestSuite) TestWriteKubeConfigFile() {
    filename := path.Join(os.TempDir(), "go-k8s-client-test-kubeconfig")
    // Remove previous file if exists
    _, err := os.Stat(filename)
    if err == nil {
        err = os.Remove(filename)
        å.Nil(err)
    } else {
        å.ErrorIs(err, os.ErrNotExist)
    }
    s.cluster, err = New(s.clusterName, Options{TakeOver: true, KubeConfigFile: filename})
    å.Nil(err)

    // Verify the file was created
    _, err = os.Stat(filename)
    å.Nil(err)
}

func (s *ClusterTestSuite) TestWriteDefaultKubeConfigFile_Fail() {
    filename := defaultKubeConfig

    // Verify the default kubeconfig exists
    _, err := os.Stat(filename)
    å.Nil(err)

    // Get its content
    origKubeConfig, err := os.ReadFile(filename)
    å.Nil(err)

    s.cluster, err = New(s.clusterName, Options{TakeOver: true, KubeConfigFile: filename})
    å.NotNil(err)

    // Verify the original file is still there with the same content
    kubeConfig, err := os.ReadFile(filename)
    å.Nil(err)
    å.Equal(kubeConfig, origKubeConfig)
}

func (s *ClusterTestSuite) TestRecreateCluster() {
    var err error
    s.cluster, err = New(s.clusterName, Options{Recreate: true})
    å.Nil(err)
    // Verify the cluster is up and running and has nodes
    nodes, err := s.cluster.GetNodes()
    å.Nil(err)
    å.Len(nodes, 1)
    å.Equal(nodes[0], s.clusterName+"-control-plane")
}

func (s *ClusterTestSuite) TestDeleteExisitngCluster() {
    // Verify the cluster exists before deleting it
    exists, err := s.cluster.Exists()
    å.Nil(err)
    å.True(exists)

    err = s.cluster.Delete()
    å.Nil(err)

    // Verify the cluster is gone
    exists, err = s.cluster.Exists()
    å.Nil(err)
    å.False(exists)
}

func (s *ClusterTestSuite) TestDeleteNonExistingCluster() {
    cluster := &Cluster{name: "no-such-cluster"}
    exists, err := cluster.Exists()
    å.Nil(err)
    å.False(exists)

    err = s.cluster.Delete()
    å.Nil(err)

    // Verify the cluster still doesn't exist
    exists, err = cluster.Exists()
    å.Nil(err)
    å.False(exists)
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestClusterTestSuite(t *testing.T) {
    suite.Run(t, new(ClusterTestSuite))
}
