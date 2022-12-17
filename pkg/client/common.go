package client

import (
    "errors"
    yaml "gopkg.in/yaml.v3"
    "io/ioutil"
    "k8s.io/client-go/rest"
    "k8s.io/client-go/tools/clientcmd"
)

type Cluster struct {
    Cluster map[string]string
    Name    string
}

type KubeConfig struct {
    ApiVersion string
    Clusters   []Cluster
    Contexts   []interface{}
}

func getServerUrl(kubeconfigPath string, kubeContext string) (url string, err error) {
    data, err := ioutil.ReadFile(kubeconfigPath)
    if err != nil {
        return
    }
    var config KubeConfig
    err = yaml.Unmarshal(data, &config)
    if err != nil {
        return
    }

    for _, cluster := range config.Clusters {
        if cluster.Name == kubeContext {
            url = cluster.Cluster["server"]
            return
        }
    }

    err = errors.New("No such context: " + kubeContext)
    return
}

func getKubeConfig(kubeconfigPath string, kubeContext string) (config *rest.Config, err error) {
    if kubeconfigPath == "" {
        config, err = rest.InClusterConfig()
        return
    }

    // If kubeContxt is provided fetch the server URL from the the kube config file
    var serverUrl string
    if kubeContext != "" {
        serverUrl, err = getServerUrl(kubeconfigPath, kubeContext)
        if err != nil {
            return
        }
    }

    // If we have a server URL we don't need the kubeConfig file anymore
    if serverUrl != "" {
        kubeconfigPath = ""
    }

    // when calling BuildConfigFromFlags we will have either server URL or kube config file
    config, err = clientcmd.BuildConfigFromFlags(serverUrl, kubeconfigPath)
    return
}
