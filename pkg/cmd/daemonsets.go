package cmd

import (
	"context"
	"fmt"

	"github.com/ricoberger/kubectl-issues/pkg/client"
	"github.com/ricoberger/kubectl-issues/pkg/cmd/utils"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

type DaemonSetsOptions struct {
	IssuesOptions
}

func newDaemonSetsOptions(options IssuesOptions) *DaemonSetsOptions {
	return &DaemonSetsOptions{
		IssuesOptions: options,
	}
}

func newDaemonSetsCommand(_ cmdutil.Factory, options IssuesOptions) *cobra.Command {
	o := newDaemonSetsOptions(options)

	cmd := &cobra.Command{
		Use:          "daemonsets",
		Aliases:      []string{"daemonset", "ds"},
		Short:        "List issues with DaemonSets",
		SilenceUsage: true,
		RunE: func(c *cobra.Command, args []string) error {
			if err := o.Complete(c); err != nil {
				return err
			}

			ctx := context.Background()
			noHeader := c.Flag("no-headers").Changed
			headers := []string{"NAMESPACE", "NAME", "DESIRED", "CURRENT", "READY", "UP-TO-DATE", "AVAILABLE", "AGE"}
			if err := o.RunAcrossContexts(ctx, noHeader, headers, o.rows); err != nil {
				fmt.Fprintln(options.Streams.ErrOut, err.Error())
				return nil
			}
			return nil
		},
	}

	o.ResourceBuilderFlags.AddFlags(cmd.Flags())

	return cmd
}

func (o *DaemonSetsOptions) rows(ctx context.Context, contextClient client.ContextClient) ([][]string, error) {
	daemonSets, err := contextClient.Client.AppsV1().DaemonSets(contextClient.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var matrix [][]string

	for _, ds := range daemonSets.Items {
		if ds.Status.DesiredNumberScheduled != ds.Status.CurrentNumberScheduled || ds.Status.DesiredNumberScheduled != ds.Status.NumberReady || ds.Status.DesiredNumberScheduled != ds.Status.UpdatedNumberScheduled || ds.Status.DesiredNumberScheduled != ds.Status.NumberAvailable || ds.Status.NumberMisscheduled > 0 {
			row := []string{ds.Namespace, ds.Name, fmt.Sprintf("%d", ds.Status.DesiredNumberScheduled), fmt.Sprintf("%d", ds.Status.CurrentNumberScheduled), fmt.Sprintf("%d", ds.Status.NumberReady), fmt.Sprintf("%d", ds.Status.UpdatedNumberScheduled), fmt.Sprintf("%d", ds.Status.NumberAvailable), utils.GetAge(ds.CreationTimestamp)}
			matrix = append(matrix, row)
		}
	}

	return matrix, nil
}
