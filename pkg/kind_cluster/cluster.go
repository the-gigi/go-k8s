package kind_cluster

import (
	"github.com/pkg/errors"
	"strings"
)

type Cluster struct {
	name string
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

// Options determines what to do if a cluster with the same name already exists
//
// At most one of `TakeOver` and `Recreate` can be true
// If both are false then New() wil fail
type Options struct {
	TakeOver bool // if true, take over an existing cluster with same name
	Recreate bool // if true, delete existing cluster and create a new one
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

	cluster = &Cluster{name: name}
	exists, err := cluster.Exists()
	if err != nil {
		return
	}

	// Delete existing cluster if options.Recreate is true and sets exists to false
	if exists && options.Recreate {
		_, err = run("delete", "cluster", "--name", name)
		if err != nil {
			return
		}
		exists = false
	}

	// Create a new cluster if no cluster with the same name exists and return
	if !exists {
		_, err = run("create", "cluster", "--name", name)
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
