package k8sclient

import (
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// GetConfig returns a Kubernetes REST config
func GetConfig(kubeconfigPath string) (*rest.Config, error) {
	var kubeconfig *string

	if kubeconfigPath != "" {
		kubeconfig = &kubeconfigPath
	} else {
		if home := homedir.HomeDir(); home != "" {
			defaultPath := filepath.Join(home, ".kube", "config")
			kubeconfig = &defaultPath
		} else {
			defaultPath := ""
			kubeconfig = &defaultPath
		}
	}

	// Try to build config from cluster first (for in-cluster usage)
	config, err := rest.InClusterConfig()
	if err != nil {
		// If not in cluster, use kubeconfig file
		config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
		if err != nil {
			return nil, err
		}
	}

	return config, nil
}

// NewClient creates a new Kubernetes client
func NewClient(kubeconfigPath string) (*kubernetes.Clientset, error) {
	config, err := GetConfig(kubeconfigPath)
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return clientset, nil
}
