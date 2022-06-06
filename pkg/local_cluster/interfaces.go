package local_cluster

import (
	"context"
	"github.com/the-gigi/go-k8s/pkg/client"
)

type Cluster interface {
	Delete() (err error)
	Exists() (exists bool, err error)
	GetKubeContext() (context string)
	GetKubeConfig() (config string)
	Clear() (err error)
	GetDynamicClient() (cli client.DynamicClient, err error)
}

type ClusterCLI interface {
	IsInstalled() (ok bool)
	Install() (err error)
	GetClusters() (clusters []string, err error)
	Delete(clusterName string) (err error)
	Create(ctx context.Context, clusterName string, kubeConfigFile string) (err error)
	WriteKubeConfig(clusterName string, filename string) (err error)
}
