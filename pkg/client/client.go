package client

import (
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type DynamicClient dynamic.Interface
type Clientset kubernetes.Interface


func getKubeConfig(kubeconfigPath string) (config *rest.Config, err error) {
	if kubeconfigPath != "" {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	} else {
		config, err = rest.InClusterConfig()
	}
	return
}

func NewDynamicClient(kubeConfigPath string) (client dynamic.Interface, err error) {
	kubeConfig, err := getKubeConfig(kubeConfigPath)
	client, err = dynamic.NewForConfig(kubeConfig)

	return
}

func NewClientset(kubeConfigPath string) (client kubernetes.Interface, err error) {
	kubeConfig, err := getKubeConfig(kubeConfigPath)
	client, err = kubernetes.NewForConfig(kubeConfig)

	return
}
