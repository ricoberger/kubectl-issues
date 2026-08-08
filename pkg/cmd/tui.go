package cmd

import (
	"time"

	"github.com/ricoberger/kubectl-issues/pkg/pods"
	"github.com/ricoberger/kubectl-issues/pkg/tui"

	"github.com/spf13/cobra"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

type TUIOptions struct {
	IssuesOptions
	restartThreshold int
	restartWindow    time.Duration
}

func newTUIOptions(options IssuesOptions) *TUIOptions {
	return &TUIOptions{
		IssuesOptions: options,
	}
}

func newTUICommand(_ cmdutil.Factory, options IssuesOptions) *cobra.Command {
	o := newTUIOptions(options)

	cmd := &cobra.Command{
		Use:          "tui",
		Short:        "Show all unhealthy Pods across one or more contexts in a TUI",
		SilenceUsage: true,
		RunE: func(c *cobra.Command, args []string) error {
			if err := o.Complete(c); err != nil {
				return err
			}

			clients, err := o.GetClients()
			if err != nil {
				return err
			}

			return tui.Start(clients, pods.Options{
				RestartThreshold: o.restartThreshold,
				RestartWindow:    o.restartWindow,
			})
		},
	}

	o.ResourceBuilderFlags.AddFlags(cmd.Flags())

	defaults := pods.DefaultOptions()
	cmd.Flags().IntVar(&o.restartThreshold, "restart-threshold", defaults.RestartThreshold, "Minimum restart count before a Ready Pod with a recent crash is reported. 0 only reports Pods which are not Ready, 1 reports any recent crash.")
	cmd.Flags().DurationVar(&o.restartWindow, "restart-window", defaults.RestartWindow, "How recent the last crash of a container must be for a Ready Pod to be reported.")

	return cmd
}
