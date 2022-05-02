package client

import (
	"k8s.io/client-go/kubernetes"
)

type Clientset kubernetes.Interface

func NewClientset(kubeConfigPath string) (client kubernetes.Interface, err error) {
	kubeConfig, err := getKubeConfig(kubeConfigPath)
	client, err = kubernetes.NewForConfig(kubeConfig)

	return
}
