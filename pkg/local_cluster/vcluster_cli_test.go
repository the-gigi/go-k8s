package local_cluster

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/the-gigi/kugo"
	"os"
	"path"
	"strings"
)

var _ = Describe("VCluster CLI Tests", Ordered, Serial, func() {
	var clusterName string
	var cli ClusterCLI
	var kubeConfigFile string

	BeforeAll(func() {
		hostClusterKubeConfig := path.Join(os.Getenv("HOME"), ".kube", "config")
		hostClusterContext := os.Getenv("HOME_CLUSTER_CONTEXT")
		cli = getVClusterCLI(hostClusterKubeConfig, hostClusterContext)
		// delete all virtual clusters
		clusters, err := cli.GetClusters()
		Ω(err).Should(BeNil())
		for _, c := range clusters {
			err = cli.Delete(c)
			Ω(err).Should(BeNil())
		}
		clusters, err = cli.GetClusters()
		Ω(err).Should(BeNil())
		Ω(clusters).Should(HaveLen(0))

		suffix, err := uuid.NewRandom()
		Ω(err).Should(BeNil())
		clusterName = "test-" + suffix.String()
		kubeConfigFile = path.Join(os.TempDir(), "kubeconfig-"+clusterName)
	})

	AfterAll(func() {
		clusters, err := cli.GetClusters()
		Ω(err).Should(BeNil())
		for _, cl := range clusters {
			if cl != clusterName {
				continue
			}
			err = cli.Delete(cl)
			Ω(err).Should(BeNil())
			break
		}
	})

	It("should create cluster successfully", func() {
		clusters, err := cli.GetClusters()
		Ω(err).Should(BeNil())
		Ω(clusters).Should(HaveLen(0))

		ctx, cancelFunc := context.WithCancel(context.Background())
		err = cli.Create(ctx, clusterName, kubeConfigFile)
		if err != nil {
			fmt.Println(err.Error())
		}
		defer cancelFunc()
		Ω(err).Should(BeNil())

		clusters, err = cli.GetClusters()
		Ω(err).Should(BeNil())
		Ω(clusters).Should(HaveLen(1))
		Ω(clusters[0]).Should(Equal(clusterName))

		args := strings.Split("cluster-info --kubeconfig "+kubeConfigFile, " ")
		output, err := kugo.Run(args...)
		Ω(output).Should(MatchRegexp(".*Kubernetes control plane.*is running at"))
	})
})
