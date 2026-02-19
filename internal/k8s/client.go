package k8s

import (
	"fmt"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Client wraps Kubernetes typed and dynamic clients.
type Client struct {
	Clientset     kubernetes.Interface
	Dynamic       dynamic.Interface
	RestConfig    *rest.Config
	ServerVersion string
}

// NewClient creates a Kubernetes client from the given kubeconfig path.
// If kubeconfigPath is empty, it attempts in-cluster config.
func NewClient(kubeconfigPath string) (*Client, error) {
	var restConfig *rest.Config
	var err error

	if kubeconfigPath != "" {
		restConfig, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			return nil, fmt.Errorf("building kubeconfig from %s: %w", kubeconfigPath, err)
		}
	} else {
		restConfig, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("building in-cluster config: %w", err)
		}
	}

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("creating kubernetes clientset: %w", err)
	}

	dynamicClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("creating dynamic client: %w", err)
	}

	serverVersion := ""
	if versionInfo, err := clientset.Discovery().ServerVersion(); err == nil {
		serverVersion = versionInfo.GitVersion
	}

	return &Client{
		Clientset:     clientset,
		Dynamic:       dynamicClient,
		RestConfig:    restConfig,
		ServerVersion: serverVersion,
	}, nil
}
