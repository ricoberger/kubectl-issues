package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/ricoberger/kubectl-issues/pkg/client"
	"github.com/ricoberger/kubectl-issues/pkg/pods"

	"github.com/spf13/cobra"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

type PodsOptions struct {
	IssuesOptions
	restartThreshold int
	restartWindow    time.Duration
}

func newPodsOptions(options IssuesOptions) *PodsOptions {
	return &PodsOptions{
		IssuesOptions: options,
	}
}

func newPodsCommand(_ cmdutil.Factory, options IssuesOptions) *cobra.Command {
	o := newPodsOptions(options)

	cmd := &cobra.Command{
		Use:          "pods",
		Aliases:      []string{"pod", "po"},
		Short:        "List issues with Pods",
		SilenceUsage: true,
		RunE: func(c *cobra.Command, args []string) error {
			if err := o.Complete(c); err != nil {
				return err
			}

			ctx := context.Background()
			noHeader := c.Flag("no-headers").Changed
			headers := []string{"NAMESPACE", "NAME", "READY", "STATUS", "RESTARTS", "AGE"}
			if err := o.RunAcrossContexts(ctx, noHeader, headers, o.rows); err != nil {
				fmt.Fprintln(options.Streams.ErrOut, err.Error())
				return nil
			}
			return nil
		},
	}

	o.ResourceBuilderFlags.AddFlags(cmd.Flags())
	cmd.Flags().StringVarP(&o.output, "output", "o", "", "Output format. One of: json.")

	defaults := pods.DefaultOptions()
	cmd.Flags().IntVar(&o.restartThreshold, "restart-threshold", defaults.RestartThreshold, "Minimum restart count before a Ready Pod with a recent crash is reported. 0 only reports Pods which are not Ready, 1 reports any recent crash.")
	cmd.Flags().DurationVar(&o.restartWindow, "restart-window", defaults.RestartWindow, "How recent the last crash of a container must be for a Ready Pod to be reported.")

	return cmd
}

func (o *PodsOptions) rows(ctx context.Context, contextClient client.ContextClient) ([][]string, error) {
	unhealthy, err := pods.ListUnhealthy(ctx, contextClient.Client, contextClient.Namespace, "", pods.Options{
		RestartThreshold: o.restartThreshold,
		RestartWindow:    o.restartWindow,
	})
	if err != nil {
		return nil, err
	}

	var matrix [][]string
	for _, pod := range unhealthy {
		matrix = append(matrix, []string{pod.Namespace, pod.Name, pod.Ready, pod.Status, pod.Restarts, pod.Age})
	}

	return matrix, nil
}
