package local_cluster

import (
	"encoding/json"
	"github.com/pkg/errors"
	"github.com/the-gigi/kugo"
	"golang.org/x/net/context"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

type vclusterCLI struct {
	hostClusterKubeConfig  string
	hostClusterKubeContext string
}

var (
	regex *regexp.Regexp
)

func init() {
	var err error
	regex, err = regexp.Compile(".*Kubernetes control plane.*is running at")
	if err != nil {
		panic(err)
	}
}

func (c vclusterCLI) run(ctx context.Context, args ...string) (combinedOutput string, err error) {
	if len(args) == 0 {
		err = errors.New("run() requires at least one argument")
		return
	}

	if len(args) == 1 {
		args = strings.Split(args[0], " ")
	}

	origKubeConfig := os.Getenv("KUBECONFIG")
	defer os.Setenv("KUBECONFIG", origKubeConfig)
	err = os.Setenv("KUBECONFIG", c.hostClusterKubeConfig)
	if err != nil {
		return
	}

	args = append(args, "--context", c.hostClusterKubeContext)

	bytes, err := exec.CommandContext(ctx, "vcluster", args...).CombinedOutput()
	combinedOutput = string(bytes)
	if err != nil {
		err = errors.Wrap(err, combinedOutput)
	}
	return
}

func (c vclusterCLI) IsInstalled() (ok bool) {
	return isToolInPath("vcluster")
}

func (c vclusterCLI) Install() (err error) {
	err = exec.Command("brew", "install", "vcluster").Wait()
	return
}

func (c vclusterCLI) GetClusters() (clusters []string, err error) {
	ctx := context.Background()
	output, err := c.run(ctx, "list", "--output", "json")
	if err != nil {
		return
	}

	var clusterList []map[string]interface{}
	err = json.Unmarshal([]byte(output), &clusterList)
	if err != nil {
		return
	}

	for _, cl := range clusterList {
		clusters = append(clusters, cl["Name"].(string))
	}
	return
}

func (c vclusterCLI) connect(ctx context.Context, clusterName string, kubeConfigFile string) (err error) {
	go func(e *error) {
		output, ee := c.run(ctx, "connect", clusterName, "-n", clusterName, "--kube-config", kubeConfigFile)
		select {
		case <-ctx.Done(): // connection cancelled. this is fine. do nothing
		default:
			if ee != nil {
				*e = errors.Wrap(ee, output)
			}
		}
	}(&err)

	for {
		if err != nil {
			break
		}
		// check if cluster is running
		output, currErr := kugo.Run("--kubeconfig", kubeConfigFile, "cluster-info")
		if currErr == nil && regex.MatchString(output) {
			break
		}
		time.Sleep(time.Second)
	}
	return
}

func (c vclusterCLI) Create(ctx context.Context, clusterName string, kubeConfigFile string) (err error) {
	_, err = c.run(ctx, "create", clusterName, "-n", clusterName, "--create-namespace")
	if err != nil {
		return
	}
	err = c.connect(ctx, clusterName, kubeConfigFile)
	return
}

func (c vclusterCLI) Delete(clusterName string) (err error) {
	ctx := context.Background()
	_, err = c.run(ctx, "delete", clusterName, "-n", clusterName)
	return
}

func (c vclusterCLI) WriteKubeConfig(clusterName string, filename string) (err error) {
	ctx := context.Background()
	err = c.connect(ctx, clusterName, filename)
	return
}

func getVClusterCLI(hostClusterKubeConfig string, hostClusterContext string) ClusterCLI {
	return &vclusterCLI{
		hostClusterKubeConfig:  hostClusterKubeConfig,
		hostClusterKubeContext: hostClusterContext,
	}
}
