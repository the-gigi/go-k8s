package local_cluster

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/the-gigi/go-k8s/pkg/client"
	"github.com/the-gigi/kugo"
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
	clusterCLIMap = map[string]ClusterCLI{
		"kind":     &kindCLI{},
		"vcluster": &vclusterCLI{},
	}
)

type cluster struct {
	name           string
	kubeConfigFile string
	cli            ClusterCLI
	ctx            context.Context
	cancelFunc     context.CancelFunc
}

// Delete - deletes a cluster by name
func (c *cluster) Delete() (err error) {

	return c.cli.Delete(c.name)
}

func (c *cluster) Exists() (exists bool, err error) {
	clusters, err := c.cli.GetClusters()
	if err != nil {
		return
	}

	exists = false
	for _, item := range clusters {
		if item == c.name {
			exists = true
			return
		}
	}

	return
}

func (c *cluster) GetKubeContext() string {
	return "kind-" + c.name
}

func (c *cluster) GetKubeConfig() string {
	return c.kubeConfigFile
}

// Clear - delete all namespaces except the built-in namespaces
func (c *cluster) Clear() (err error) {
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
func (c *cluster) GetDynamicClient() (cli client.DynamicClient, err error) {
	kubeConfig := c.GetKubeConfig()
	if kubeConfig == "" {
		err = errors.New("can't get dynamic client without a dedicated kube config file")
	}

	cli, err = client.NewDynamicClient(kubeConfig, "")
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
// clusterType: Kind or VCluster
// name: the name of the new cluster
// options: create options that determine the behavior if the cluster already exists
//
// If the cluster doesn't exist yet, then it is created.
// If it exists and all options are false it will return an error
// If it exists and TakeOver is true it will just use the existing cluster
// If it exists and Recreate is selected it will delete the existing cluster and create it from scratch.
func newCluster(clusterType string, name string, options Options) (cl *cluster, err error) {
	// Validate the options
	if options.Recreate && options.TakeOver {
		err = errors.New("invalid options. At most one option can be true")
		return
	}

	if options.KubeConfigFile == defaultKubeConfig {
		err = errors.New("can't overwrite default kubeconfig")
		return
	}

	cli, ok := clusterCLIMap[clusterType]
	if !ok {
		err = errors.New("unsupported cluster type: " + clusterType)
		return
	}

	ctx, cancelFunc := context.WithCancel(context.Background())

	cl = &cluster{
		cli:            cli,
		name:           name,
		kubeConfigFile: options.KubeConfigFile,
		ctx:            ctx, cancelFunc: cancelFunc,
	}
	exists, err := cl.Exists()
	if err != nil {
		return
	}

	// Delete existing cluster if options.Recreate is true and sets exists to false
	if exists && options.Recreate {
		err = cl.Delete()
		if err != nil {
			return
		}
		exists = false
	}

	// Create a new cluster if no cluster with the same name exists and return
	if !exists {
		err = cl.cli.Create(ctx, name, options.KubeConfigFile)
		if err != nil {
			return
		}

		if options.KubeConfigFile != "" {
			err = cl.cli.WriteKubeConfig(name, options.KubeConfigFile)
			if err != nil {
				return
			}
		}

		return
	}

	// Fail if another cluster with the same name exists and the caller doesn't want to take over it
	if !options.TakeOver {
		err = errors.Errorf("cluster named '%s' already exists", name)
		return
	}

	return
}
