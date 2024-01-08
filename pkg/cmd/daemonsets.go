package cmd

import (
	"bytes"
	"context"
	"fmt"

	"github.com/ricoberger/kubectl-issues/pkg/cmd/utils"
	"github.com/ricoberger/kubectl-issues/pkg/writer"

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

func newDaemonSetsCommand(factory cmdutil.Factory, options IssuesOptions) *cobra.Command {
	o := newDaemonSetsOptions(options)

	cmd := &cobra.Command{
		Use:          "daemonsets",
		Aliases:      []string{"daemonset", "ds"},
		Short:        "List issues with DaemonSets",
		SilenceUsage: true,
		RunE: func(c *cobra.Command, args []string) error {
			if err := o.Complete(factory, c); err != nil {
				return err
			}

			ctx := context.Background()
			noHeader := c.Flag("no-headers").Changed
			if err := o.Run(ctx, noHeader); err != nil {
				fmt.Fprintln(options.Streams.ErrOut, err.Error())
				return nil
			}
			return nil
		},
	}

	o.ResourceBuilderFlags.AddFlags(cmd.Flags())

	return cmd
}

func (o *DaemonSetsOptions) Run(ctx context.Context, noHeader bool) error {
	client, err := o.GetClient()
	if err != nil {
		return err
	}

	daemonSets, err := client.AppsV1().DaemonSets(o.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	var matrix [][]string

	for _, ds := range daemonSets.Items {
		if ds.Status.DesiredNumberScheduled != ds.Status.CurrentNumberScheduled || ds.Status.DesiredNumberScheduled != ds.Status.NumberReady || ds.Status.DesiredNumberScheduled != ds.Status.UpdatedNumberScheduled || ds.Status.DesiredNumberScheduled != ds.Status.NumberAvailable || ds.Status.NumberMisscheduled > 0 {
			row := []string{ds.Namespace, ds.Name, fmt.Sprintf("%d", ds.Status.DesiredNumberScheduled), fmt.Sprintf("%d", ds.Status.CurrentNumberScheduled), fmt.Sprintf("%d", ds.Status.NumberReady), fmt.Sprintf("%d", ds.Status.UpdatedNumberScheduled), fmt.Sprintf("%d", ds.Status.NumberAvailable), utils.GetAge(ds.CreationTimestamp)}
			matrix = append(matrix, row)
		}
	}

	headers := []string{"NAMESPACE", "NAME", "DESIRED", "CURRENT", "READY", "UP-TO-DATE", "AVAILABLE", "AGE"}

	buf := bytes.NewBuffer(nil)
	writer.WriteResults(buf, headers, matrix, noHeader)
	fmt.Printf("%s", buf.String())

	return nil
}
