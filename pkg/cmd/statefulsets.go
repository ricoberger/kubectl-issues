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

type StatefulSetsOptions struct {
	IssuesOptions
}

func newStatefulSetsOptions(options IssuesOptions) *StatefulSetsOptions {
	return &StatefulSetsOptions{
		IssuesOptions: options,
	}
}

func newStatefulSetsCommand(factory cmdutil.Factory, options IssuesOptions) *cobra.Command {
	o := newStatefulSetsOptions(options)

	cmd := &cobra.Command{
		Use:          "statefulsets",
		Short:        "List issues with StatefulSets",
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

func (o *StatefulSetsOptions) Run(ctx context.Context, noHeader bool) error {
	client, err := o.GetClient()
	if err != nil {
		return err
	}

	statefulSets, err := client.AppsV1().StatefulSets(o.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	var matrix [][]string

	for _, sts := range statefulSets.Items {
		if sts.Status.Replicas != sts.Status.ReadyReplicas || sts.Status.ReadyReplicas != sts.Status.UpdatedReplicas || sts.Status.Replicas != sts.Status.AvailableReplicas {
			row := []string{sts.Namespace, sts.Name, fmt.Sprintf("%d/%d", sts.Status.ReadyReplicas, sts.Status.Replicas), fmt.Sprintf("%d", sts.Status.UpdatedReplicas), fmt.Sprintf("%d", sts.Status.AvailableReplicas), utils.GetAge(sts.CreationTimestamp)}
			matrix = append(matrix, row)
		}
	}

	headers := []string{"NAMESPACE", "NAME", "READY", "UP-TO-DATE", "AVAILABLE", "AGE"}

	buf := bytes.NewBuffer(nil)
	writer.WriteResults(buf, headers, matrix, noHeader)
	fmt.Printf("%s", buf.String())

	return nil
}
