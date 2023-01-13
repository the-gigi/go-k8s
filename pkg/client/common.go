package client

import (
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func getKubeConfig(kubeconfigPath string, kubeContext string) (config *rest.Config, err error) {
	if kubeconfigPath == "" {
		config, err = rest.InClusterConfig()
		return
	}

	var conf *clientcmdapi.Config
	conf, err = clientcmd.LoadFromFile(kubeconfigPath)
	if err != nil {
		return
	}

	overrides := &clientcmd.ConfigOverrides{}
	if kubeContext != "" {
		overrides.CurrentContext = kubeContext
	}

	config, err = clientcmd.NewDefaultClientConfig(*conf, overrides).ClientConfig()
	return
}
