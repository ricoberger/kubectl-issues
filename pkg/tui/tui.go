package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// ContextClient is a Kubernetes client for a single kubeconfig context. The
// Name is the (resolved) name of the context and is shown in the table.
type ContextClient struct {
	Name   string
	Client kubernetes.Interface
}

// Start builds a Kubernetes client for every given context and starts the TUI.
// If no contexts are given, the current context of the kubeconfig is used.
func Start(contexts []string, configFlags *genericclioptions.ConfigFlags) error {
	kubeconfig := ""
	if configFlags != nil && configFlags.KubeConfig != nil {
		kubeconfig = *configFlags.KubeConfig
	}

	if len(contexts) == 0 {
		contexts = []string{""}
	}

	var clients []ContextClient
	for _, name := range contexts {
		client, resolvedName, err := newClient(kubeconfig, name)
		if err != nil {
			return fmt.Errorf("failed to create client for context %q: %w", name, err)
		}

		clients = append(clients, ContextClient{Name: resolvedName, Client: client})
	}

	model := NewModel(clients)

	p := tea.NewProgram(model)
	if _, err := p.Run(); err != nil {
		return err
	}

	return nil
}

// newClient creates a Kubernetes client for the given kubeconfig context. If
// the context name is empty, the current context of the kubeconfig is used. The
// resolved context name is returned as the second return value.
func newClient(kubeconfig, contextName string) (kubernetes.Interface, string, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	if kubeconfig != "" {
		loadingRules.ExplicitPath = kubeconfig
	}

	overrides := &clientcmd.ConfigOverrides{}
	if contextName != "" {
		overrides.CurrentContext = contextName
	}

	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, overrides)

	resolvedName := contextName
	if resolvedName == "" {
		if rawConfig, err := clientConfig.RawConfig(); err == nil {
			resolvedName = rawConfig.CurrentContext
		}
	}

	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, resolvedName, err
	}

	client, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, resolvedName, err
	}

	return client, resolvedName, nil
}
