package cmd

import (
	"github.com/spf13/cobra"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

var cmdExample = `  # List issues with Pods
  kubectl issues pods
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
	o.ConfigFlags.AddFlags(flags)

	matchVersionFlags := cmdutil.NewMatchVersionFlags(o.ConfigFlags)
	matchVersionFlags.AddFlags(flags)

	f := cmdutil.NewFactory(matchVersionFlags)

	cmd.AddCommand(newDeploysCommand(f, o))
	cmd.AddCommand(newDSsCommand(f, o))
	cmd.AddCommand(newJobsCommand(f, o))
	cmd.AddCommand(newNodesCommand(f, o))
	cmd.AddCommand(newPodsCommand(f, o))
	cmd.AddCommand(newPVCsCommand(f, o))
	cmd.AddCommand(newPVsCommand(f, o))
	cmd.AddCommand(newStatefulSetsCommand(f, o))

	return cmd
}
