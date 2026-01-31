package k8sclient

import (
	"flag"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// NewClient creates a new Kubernetes client
func NewClient(kubeconfigPath string) (*kubernetes.Clientset, error) {
	var kubeconfig *string
	
	if kubeconfigPath != "" {
		kubeconfig = &kubeconfigPath
	} else {
		if home := homedir.HomeDir(); home != "" {
			kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
		} else {
			kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
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

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return clientset, nil
}