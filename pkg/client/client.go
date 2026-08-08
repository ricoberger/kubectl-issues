package client

import (
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
)

// ContextClient is a Kubernetes client for a single kubeconfig context. The
// Name is the (resolved) name of the context and the Namespace is the
// namespace to use for the context, where an empty string means all
// namespaces.
type ContextClient struct {
	Name      string
	Namespace string
	Client    kubernetes.Interface
}

// New builds a Kubernetes client for the given kubeconfig context, using the
// connection settings from the given config flags. If the context name is
// empty, the current context of the kubeconfig is used. The namespace is
// resolved from the --namespace flag if set, otherwise from the context's
// kubeconfig entry. If allNamespaces is true, the namespace is empty so that
// all namespaces are used.
//
// The config flags must be created with usePersistentConfig set to false, so
// that a cached loader from a previous call for another context is not reused.
func New(configFlags *genericclioptions.ConfigFlags, contextName string, allNamespaces bool) (ContextClient, error) {
	originalContext := configFlags.Context
	if contextName != "" {
		configFlags.Context = &contextName
	}
	defer func() { configFlags.Context = originalContext }()

	loader := configFlags.ToRawKubeConfigLoader()

	resolvedName := contextName
	if resolvedName == "" {
		if rawConfig, err := loader.RawConfig(); err == nil {
			resolvedName = rawConfig.CurrentContext
		}
	}

	namespace := ""
	if !allNamespaces {
		ns, _, err := loader.Namespace()
		if err != nil {
			return ContextClient{}, err
		}
		namespace = ns
	}

	restConfig, err := loader.ClientConfig()
	if err != nil {
		return ContextClient{}, err
	}

	client, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return ContextClient{}, err
	}

	return ContextClient{Name: resolvedName, Namespace: namespace, Client: client}, nil
}
