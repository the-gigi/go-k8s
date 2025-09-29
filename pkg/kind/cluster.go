package kind

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	//"github.com/the-gigi/go-k8s/pkg/client"
	"github.com/the-gigi/kugo"
	"os"
	"strings"
	"time"
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
	return c.DeleteWithContext(context.Background())
}

// DeleteWithContext - deletes a cluster by name with context support
func (c *Cluster) DeleteWithContext(ctx context.Context) (err error) {
	output, err := runWithContext(ctx, "delete", "cluster", "--name", c.name)
	if err != nil {
		err = errors.Wrapf(err, "stderr: %s", output)
	}
	return
}

// GetNodes returns the list of nodes in the cluster
func (c *Cluster) GetNodes() (nodes []string, err error) {
	return c.GetNodesWithContext(context.Background())
}

// GetNodesWithContext returns the list of nodes in the cluster with context support
func (c *Cluster) GetNodesWithContext(ctx context.Context) (nodes []string, err error) {
	output, err := runWithContext(ctx, "get", "nodes", "--name", c.name)
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

	if clusters[c.name] {
		exists = true
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

//// GetDynamicClient - return a DynamicClient to interact with the cluster
////
//// It returns an error if the cluster uses the default kube config
//func (c *Cluster) GetDynamicClient() (cli client.DynamicClient, err error) {
//	kubeConfig := c.GetKubeConfig()
//	if kubeConfig == "" {
//		err = errors.New("can't get dynamic client without a dedicated kube config file")
//	}
//
//	cli, err = client.NewDynamicClient(kubeConfig)
//	return
//}

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
	ConfigFile     string // if not empty, use this Kind config file for cluster creation
	Wait           time.Duration // timeout for cluster creation (default: 0 = no timeout)
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
	return NewWithContext(context.Background(), name, options)
}

// NewWithContext - create a new cluster with context support
func NewWithContext(ctx context.Context, name string, options Options) (cluster *Cluster, err error) {
	// Validate the options
	if options.Recreate && options.TakeOver {
		err = errors.New("invalid options. At most one option can be true")
		return
	}

	if options.KubeConfigFile == defaultKubeConfig {
		err = errors.New("can't overwrite default kubeconfig")
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
		output, err = runWithContext(ctx, "delete", "cluster", "--name", name)
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

		kubeConfig, err := runWithContext(ctx, "get", "kubeconfig", "--name", name)
		if err != nil {
			return
		}
		err = os.WriteFile(options.KubeConfigFile, []byte(kubeConfig), 0600)
	}()

	// Create a new cluster if no cluster with the same name exists and return
	if !exists {
		args := []string{"create", "cluster", "--name", name}
		if options.KubeConfigFile != "" {
			args = append(args, "--kubeconfig", options.KubeConfigFile)
		}
		if options.ConfigFile != "" {
			args = append(args, "--config", options.ConfigFile)
		}
		if options.Wait > 0 {
			args = append(args, "--wait", options.Wait.String())
		}
		output, err = runWithContext(ctx, args...)
		if err != nil {
			err = errors.Wrap(err, output)
			return
		}
		
		// Ensure CNI is properly installed and working
		err = cluster.ensureCNI(ctx)
		if err != nil {
			return
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

// LoadDockerImage loads a Docker image into the cluster
func (c *Cluster) LoadDockerImage(imageName string) error {
	return c.LoadDockerImageWithContext(context.Background(), imageName)
}

// LoadDockerImageWithContext loads a Docker image into the cluster with context support
func (c *Cluster) LoadDockerImageWithContext(ctx context.Context, imageName string) error {
	output, err := runWithContext(ctx, "load", "docker-image", imageName, "--name", c.name)
	if err != nil {
		return errors.Wrapf(err, "failed to load image %s: %s", imageName, output)
	}
	return nil
}

// LoadDockerImages loads multiple Docker images into the cluster
func (c *Cluster) LoadDockerImages(imageNames []string) error {
	return c.LoadDockerImagesWithContext(context.Background(), imageNames)
}

// LoadDockerImagesWithContext loads multiple Docker images into the cluster with context support
func (c *Cluster) LoadDockerImagesWithContext(ctx context.Context, imageNames []string) error {
	for _, imageName := range imageNames {
		if err := c.LoadDockerImageWithContext(ctx, imageName); err != nil {
			return err
		}
	}
	return nil
}

// LoadArchive loads a Docker image from a tar archive into the cluster
func (c *Cluster) LoadArchive(archivePath string) error {
	return c.LoadArchiveWithContext(context.Background(), archivePath)
}

// LoadArchiveWithContext loads a Docker image from a tar archive into the cluster with context support
func (c *Cluster) LoadArchiveWithContext(ctx context.Context, archivePath string) error {
	output, err := runWithContext(ctx, "load", "image-archive", archivePath, "--name", c.name)
	if err != nil {
		return errors.Wrapf(err, "failed to load archive %s: %s", archivePath, output)
	}
	return nil
}

// ensureCNI installs and validates that CNI is working properly
func (c *Cluster) ensureCNI(ctx context.Context) error {
	// First check if CNI is already working by checking node status
	output, err := kugo.Run("get", "nodes", "--kubeconfig", c.kubeConfigFile, "--context", c.GetKubeContext())
	if err == nil && strings.Contains(output, "Ready") {
		return nil // CNI is already working
	}
	
	// Install Calico CNI (more reliable with Kind)
	_, err = kugo.Run("apply", "-f", "https://raw.githubusercontent.com/projectcalico/calico/v3.28.1/manifests/tigera-operator.yaml",
		"--kubeconfig", c.kubeConfigFile, "--context", c.GetKubeContext())
	if err != nil {
		return errors.Wrap(err, "failed to install Calico operator")
	}
	
	// Apply the Calico custom resources
	_, err = kugo.Run("apply", "-f", "https://raw.githubusercontent.com/projectcalico/calico/v3.28.1/manifests/custom-resources.yaml",
		"--kubeconfig", c.kubeConfigFile, "--context", c.GetKubeContext())
	if err != nil {
		return errors.Wrap(err, "failed to install Calico")
	}
	
	// Wait for CNI to be ready and nodes to become Ready
	_, err = kugo.Run("wait", "--for=condition=ready", "node", "--all", "--timeout=120s",
		"--kubeconfig", c.kubeConfigFile, "--context", c.GetKubeContext())
	if err != nil {
		return errors.Wrap(err, "nodes did not become ready after CNI installation")
	}
	
	return nil
}
