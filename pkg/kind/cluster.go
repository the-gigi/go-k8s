package kind

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/the-gigi/go-k8s/pkg/client"
	"github.com/the-gigi/kugo"
	"io/ioutil"
	"os"
	"strings"
)

var (
	defaultKubeConfig = os.ExpandEnv("${HOME}/.kube/config")
	builtinNamespaces = map[string]bool{
		"default":            true,
		"kube-node-lease":    true,
		"kube-public":        true,
		"kube-system":        true,
		"local-path-storage": true,
	}
)

type Cluster struct {
	name           string
	kubeConfigFile string
}

// Delete - deletes a cluster by name
func (c *Cluster) Delete() (err error) {
	output, err := run("delete", "cluster", "--name", c.name)
	if err != nil {
		err = errors.Wrapf(err, "stderr: %s", output)
	}
	return
}

// GetNodes returns the list of nodes in the cluster
func (c *Cluster) GetNodes() (nodes []string, err error) {
	output, err := run("get", "nodes", "--name", c.name)
	if err != nil {
		return
	}

	for _, node := range strings.Split(output, "\n") {
		if node != "" {
			nodes = append(nodes, node)
		}
	}
	return
}

func (c *Cluster) Exists() (exists bool, err error) {
	clusters, err := getClusters()
	if err != nil {
		return
	}

	for _, cluster := range clusters {
		if cluster == c.name {
			exists = true
			return
		}
	}
	return
}

func (c *Cluster) GetKubeContext() string {
	return "kind-" + c.name
}

func (c *Cluster) GetKubeConfig() string {
	return c.kubeConfigFile
}

// Clear - delete all namespaces except the built-in namespaces
func (c *Cluster) Clear() (err error) {
	output, err := kugo.Get(kugo.GetRequest{
		BaseRequest: kugo.BaseRequest{
			KubeConfigFile: c.kubeConfigFile,
			KubeContext:    c.GetKubeContext(),
		},
		Kind:   "ns",
		Output: "name",
	})
	if err != nil {
		return
	}

	output = strings.Replace(output, "namespace/", "", -1)
	namespaces := strings.Split(output, "\n")
	for _, ns := range namespaces {
		if !builtinNamespaces[ns] && ns != "" {
			cmd := fmt.Sprintf("delete ns %s --kubeconfig %s --context %s", ns, c.kubeConfigFile, c.GetKubeContext())
			output, err = kugo.Run(cmd)
			if err != nil {
				err = errors.Wrap(err, output)
				return
			}
		}
	}
	return
}

// GetDynamicClient - return a DynamicClient to interact with the cluster
//
// It returns an error if the cluster uses the default kube config
func (c *Cluster) GetDynamicClient() (cli client.DynamicClient, err error) {
	kubeConfig := c.GetKubeConfig()
	if kubeConfig == "" {
		err = errors.New("can't get dynamic client without a dedicated kube config file")
	}

	cli, err = client.NewDynamicClient(kubeConfig)
	return
}

// Options determines what to do if a cluster with the same name already exists
//
// At most one of `TakeOver` and `Recreate` can be true
// If both are false then New() wil fail
//
// If KubeConfigFile is a path to an existing file it will be overwritten.
// New() will NOT overwrite the default ~/.kube/config.
//
// Note that the cluster will always be added to the active kubeconfig file.
type Options struct {
	TakeOver       bool   // if true, take over an existing cluster with same name
	Recreate       bool   // if true, delete existing cluster and create a new one
	KubeConfigFile string // if not empty, save cluster's kubeconfig to a file
}

// New - create a new cluster
//
// name: the name of the new cluster
// options: create options that determine the behavior if the cluster already exists
//
// If the cluster doesn't exist yet, then it is created.
// If it exists and all options are false it will return an error
// If it exists and TakeOver is true it will just use the existing cluster
// If it exists and Recreate is selected it will delete the existing cluster and create it from scratch.
func New(name string, options Options) (cluster *Cluster, err error) {
	// Validate the options
	if options.Recreate && options.TakeOver {
		err = errors.New("invalid options. At most one option can be true")
		return
	}

	if options.KubeConfigFile == defaultKubeConfig {
		err = errors.New("can't overwite default kubeconfig")
		return
	}

	cluster = &Cluster{name: name, kubeConfigFile: options.KubeConfigFile}
	exists, err := cluster.Exists()
	if err != nil {
		return
	}

	var output string
	// Delete existing cluster if options.Recreate is true and sets exists to false
	if exists && options.Recreate {
		output, err = run("delete", "cluster", "--name", name)
		if err != nil {
			err = errors.Wrap(err, output)
			return
		}
		exists = false
	}

	// At the end, if the cluster was created successfully write its kubeconfig to a file if needed
	defer func() {
		if err != nil || options.KubeConfigFile == "" {
			return
		}

		kubeConfig, err := run("get", "kubeconfig", "--name", name)
		if err != nil {
			return
		}
		err = ioutil.WriteFile(options.KubeConfigFile, []byte(kubeConfig), 0644)
	}()

	// Create a new cluster if no cluster with the same name exists and return
	if !exists {
		args := []string{"create", "cluster", "--name", name}
		if options.KubeConfigFile != "" {
			args = append(args, "--kubeconfig", options.KubeConfigFile)
		}
		output, err = run(args...)
		if err != nil {
			err = errors.Wrap(err, output)
		}
		return
	}

	// Fail if another cluster with the same name exists and the caller doesn't want to take over it
	if !options.TakeOver {
		err = errors.Errorf("cluster named '%s' already exists", cluster.name)
		return
	}

	// If we get here it means we take over the existing cluster. Nothing to do :-)
	return
}
