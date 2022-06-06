package local_cluster

import (
	"context"
	"github.com/pkg/errors"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
)

const (
	kindVersion = "v0.12.0"
)

type kindCLI struct {
}

func (c *kindCLI) run(args ...string) (combinedOutput string, err error) {
	if len(args) == 0 {
		err = errors.New("Run() requires at least one argument")
		return
	}

	if len(args) == 1 {
		args = strings.Split(args[0], " ")
	}

	bytes, err := exec.Command("kind", args...).CombinedOutput()
	combinedOutput = string(bytes)
	if err != nil {
		err = errors.Wrap(err, combinedOutput)
	}
	return
}

func (c *kindCLI) IsInstalled() (ok bool) {
	return isToolInPath("kind")
}

func (c *kindCLI) Install() (err error) {
	kindModule := "sigs.k8s.io/kind@" + kindVersion
	err = exec.Command("go", "install", kindModule).Wait()
	return
}

func (c *kindCLI) GetClusters() (clusters []string, err error) {
	output, err := c.run("get", "clusters")
	if err != nil {
		return
	}

	for _, cluster := range strings.Split(output, "\n") {
		if cluster != "" {
			clusters = append(clusters, cluster)
		}
	}
	return
}

func (c *kindCLI) Create(ctx context.Context, clusterName string, kubeConfigFile string) (err error) {
	args := []string{"create", "cluster", "--name", clusterName}
	if kubeConfigFile != "" {
		args = append(args, "--kubeconfig", kubeConfigFile)
	}
	output, err := c.run(args...)
	if err != nil {
		err = errors.Wrap(err, output)
		return
	}

	if kubeConfigFile == "" {
		return
	}

	// Ensure kubeconfig if kubeconfig file is not default
	args = []string{"get", "kubeconfig", "--name", clusterName}
	kubeconfig, err := c.run(args...)
	if err != nil {
		err = errors.Wrap(err, output)
	}

	currKubeConfig, err := ioutil.ReadFile(kubeConfigFile)
	if err != nil && !os.IsNotExist(err) {
		return
	}

	if kubeconfig != string(currKubeConfig) {
		err = ioutil.WriteFile(kubeConfigFile, []byte(kubeconfig), 0644)
	}

	return
}

func (c *kindCLI) Delete(clusterName string) (err error) {
	output, err := c.run("delete", "cluster", "--name", clusterName)
	if err != nil {
		err = errors.Wrap(err, output)
	}
	return
}

func (c *kindCLI) WriteKubeConfig(clusterName string, filename string) (err error) {
	output, err := c.run("get", "kubeconfig", "--name", clusterName)
	if err != nil {
		err = errors.Wrap(err, output)
	}

	err = ioutil.WriteFile(filename, []byte(output), 0644)
	return
}
