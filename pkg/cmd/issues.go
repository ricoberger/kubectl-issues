package cmd

import (
	"github.com/spf13/cobra"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

var cmdExample = `  # List issues with Pods
  kubectl issues pods

  # List issues with Pods across multiple contexts
  kubectl issues pods --all-namespaces --context staging --context production

  # Show all unhealthy Pods across multiple contexts in a TUI
  kubectl issues tui --all-namespaces --context staging --context production
`

func NewIssuesCommand() *cobra.Command {
	o := NewIssuesOptions()

	cmd := &cobra.Command{
		Use:          "issues",
		Example:      cmdExample,
		Short:        "Find issues with your Kubernetes objects",
		SilenceUsage: true,
	}

	cmd.PersistentFlags().Bool("no-headers", false, "Don't print headers (default print headers).")

	flags := cmd.PersistentFlags()

	// The context flag of the config flags is replaced with our own repeatable
	// flag, so that issues can be listed across multiple contexts at once.
	o.ConfigFlags.Context = nil
	o.ConfigFlags.AddFlags(flags)
	flags.StringArray("context", nil, "The name of the kubeconfig context to use. Can be specified multiple times to list issues from multiple clusters.")

	matchVersionFlags := cmdutil.NewMatchVersionFlags(o.ConfigFlags)
	matchVersionFlags.AddFlags(flags)

	f := cmdutil.NewFactory(matchVersionFlags)

	cmd.AddCommand(newDaemonSetsCommand(f, o))
	cmd.AddCommand(newDeploymentsCommand(f, o))
	cmd.AddCommand(newJobsCommand(f, o))
	cmd.AddCommand(newNodesCommand(f, o))
	cmd.AddCommand(newPersistentVolumeClaimsCommand(f, o))
	cmd.AddCommand(newPersistentVolumesCommand(f, o))
	cmd.AddCommand(newPodsCommand(f, o))
	cmd.AddCommand(newReplicaSetsCommand(f, o))
	cmd.AddCommand(newStatefulSetsCommand(f, o))
	cmd.AddCommand(newTUICommand(f, o))

	return cmd
}
