package client

import (
	"os"
	"path/filepath"
	"testing"

	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func writeKubeconfig(t *testing.T) string {
	t.Helper()

	kubeconfig := `apiVersion: v1
kind: Config
current-context: staging
clusters:
  - name: cluster1
    cluster:
      server: https://staging.example.com
  - name: cluster2
    cluster:
      server: https://production.example.com
contexts:
  - name: staging
    context:
      cluster: cluster1
      namespace: staging-namespace
  - name: production
    context:
      cluster: cluster2
users: []
`

	path := filepath.Join(t.TempDir(), "kubeconfig")
	if err := os.WriteFile(path, []byte(kubeconfig), 0o600); err != nil {
		t.Fatalf("failed to write kubeconfig: %v", err)
	}
	return path
}

func newConfigFlags(kubeconfig, namespace string) *genericclioptions.ConfigFlags {
	configFlags := genericclioptions.NewConfigFlags(false)
	configFlags.KubeConfig = &kubeconfig
	configFlags.Context = nil
	if namespace != "" {
		configFlags.Namespace = &namespace
	}
	return configFlags
}

func TestNew(t *testing.T) {
	kubeconfig := writeKubeconfig(t)

	tests := []struct {
		name          string
		contextName   string
		namespace     string
		allNamespaces bool
		wantName      string
		wantNamespace string
	}{
		{
			name:          "empty context resolves current context and its namespace",
			contextName:   "",
			wantName:      "staging",
			wantNamespace: "staging-namespace",
		},
		{
			name:          "explicit context uses its default namespace",
			contextName:   "production",
			wantName:      "production",
			wantNamespace: "default",
		},
		{
			name:          "namespace flag overrides context namespace",
			contextName:   "staging",
			namespace:     "override",
			wantName:      "staging",
			wantNamespace: "override",
		},
		{
			name:          "all namespaces yields empty namespace",
			contextName:   "staging",
			allNamespaces: true,
			wantName:      "staging",
			wantNamespace: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cc, err := New(newConfigFlags(kubeconfig, tt.namespace), tt.contextName, tt.allNamespaces)
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}
			if cc.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", cc.Name, tt.wantName)
			}
			if cc.Namespace != tt.wantNamespace {
				t.Errorf("Namespace = %q, want %q", cc.Namespace, tt.wantNamespace)
			}
			if cc.Client == nil {
				t.Error("Client is nil")
			}
		})
	}
}

func TestNewUnknownContext(t *testing.T) {
	kubeconfig := writeKubeconfig(t)

	if _, err := New(newConfigFlags(kubeconfig, ""), "unknown", false); err == nil {
		t.Fatal("New() expected error for unknown context, got nil")
	}
}

func TestNewRestoresContext(t *testing.T) {
	kubeconfig := writeKubeconfig(t)
	configFlags := newConfigFlags(kubeconfig, "")

	if _, err := New(configFlags, "production", false); err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if configFlags.Context != nil {
		t.Errorf("Context = %q, want nil", *configFlags.Context)
	}
}
