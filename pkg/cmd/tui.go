package cmd

import (
	"github.com/ricoberger/kubectl-issues/pkg/pods"
	"github.com/ricoberger/kubectl-issues/pkg/tui"

	"github.com/spf13/cobra"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

func newTUICommand(_ cmdutil.Factory, options IssuesOptions) *cobra.Command {
	var contexts []string
	podsOptions := pods.DefaultOptions()

	cmd := &cobra.Command{
		Use:          "tui",
		Short:        "Show all unhealthy Pods across one or more contexts in a TUI",
		SilenceUsage: true,
		RunE: func(c *cobra.Command, args []string) error {
			return tui.Start(contexts, options.ConfigFlags, podsOptions)
		},
	}

	cmd.Flags().StringArrayVar(&contexts, "context", nil, "The name of the kubeconfig context to use. Can be specified multiple times to show unhealthy Pods from multiple clusters.")
	cmd.Flags().IntVar(&podsOptions.RestartThreshold, "restart-threshold", podsOptions.RestartThreshold, "Minimum restart count before a Ready Pod with a recent crash is reported. 0 only reports Pods which are not Ready, 1 reports any recent crash.")
	cmd.Flags().DurationVar(&podsOptions.RestartWindow, "restart-window", podsOptions.RestartWindow, "How recent the last crash of a container must be for a Ready Pod to be reported.")

	return cmd
}
