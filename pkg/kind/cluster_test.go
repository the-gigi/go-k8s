package kind

import (
    "github.com/google/uuid"
    "github.com/stretchr/testify/suite"
    "github.com/the-gigi/kugo"
    "os"
    "path"
    "testing"
    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"
)


func (s *ClusterTestSuite) SetupSuite() {
    suffix, err := uuid.NewRandom()
    Ω(err).Should(BeNil())
    s.clusterName = "test-" + suffix.String()
}

func (s *ClusterTestSuite) TeardownSuite() {
    _, err := kugo.Run("get version --context kind-" + s.clusterName)
    if err == nil {
        err = s.cluster.Delete()
        Ω(err).Should(BeNil())
    }
}

func (s *ClusterTestSuite) TestCreateCluster() {
    var err error

    // Create new cluster
    s.cluster, err = New(s.clusterName, Options{})
    Ω(err).Should(BeNil())
    Ω(s.cluster).ShouldNot(BeNil())

    // Verify the cluster is up and running and has nodes
    nodes, err := s.cluster.GetNodes()
    Ω(err).Should(BeNil())

    Ω(nodes).Should(HaveLen(1))
    Ω(nodes[0]).Should(Equal(s.clusterName + "-control-plane"))

    // should take over existing cluster successfully
    s.cluster, err = New(s.clusterName, Options{TakeOver: true})
    Ω(err).Should(BeNil())
    Ω(s.cluster).ShouldNot(BeNil())

    // should fail to create existing cluster with no options
    s.cluster, err = New(s.clusterName, Options{})
    Ω(err).ShouldNot(BeNil())
}

func (s *ClusterTestSuite) TestWriteKubeConfigFile() {
    filename := path.Join(os.TempDir(), "go-k8s-client-test-kubeconfig")
    // Remove previous file if exists
    _, err := os.Stat(filename)
    if err == nil {
        err = os.Remove(filename)
        Ω(err).Should(BeNil())
    } else {
        Ω(err).Should(Equal(os.ErrNotExist))
    }
    s.cluster, err = New(s.clusterName, Options{TakeOver: true, KubeConfigFile: filename})
    Ω(err).Should(BeNil())

    // Verify the file was created
    _, err = os.Stat(filename)
    Ω(err).Should(BeNil())
}

func (s *ClusterTestSuite) TestWriteDefaultKubeConfigFile_Fail() {
    filename := defaultKubeConfig

    // Verify the default kubeconfig exists
    _, err := os.Stat(filename)
    Ω(err).Should(BeNil())

    // Get its content
    origKubeConfig, err := os.ReadFile(filename)
    Ω