package client

import (
	"k8s.io/client-go/kubernetes"
)

type Clientset kubernetes.Interface

func NewClientset(kubeConfigPath string, kubeContext string) (client Clientset, err error) {
	kubeConfig, err := getKubeConfig(kubeConfigPath, kubeContext)
	if err != nil {
		return
	}
	client, err = kubernetes.NewForConfig(kubeConfig)
	return
}
