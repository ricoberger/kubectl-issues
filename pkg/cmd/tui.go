package cmd

import (
	"github.com/ricoberger/kubectl-issues/pkg/tui"

	"github.com/spf13/cobra"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

func newTUICommand(_ cmdutil.Factory, options IssuesOptions) *cobra.Command {
	var contexts []string

	cmd := &cobra.Command{
		Use:          "tui",
		Short:        "Show all unhealthy Pods across one or more contexts in a TUI",
		SilenceUsage: true,
		RunE: func(c *cobra.Command, args []string) error {
			return tui.Start(contexts, options.ConfigFlags)
		},
	}

	cmd.Flags().StringArrayVar(&contexts, "context", nil, "The name of the kubeconfig context to use. Can be specified multiple times to show unhealthy Pods from multiple clusters.")

	return cmd
}
