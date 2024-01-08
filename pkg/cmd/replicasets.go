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

type ReplicaSetsOptions struct {
	IssuesOptions
}

func newReplicaSetsOptions(options IssuesOptions) *ReplicaSetsOptions {
	return &ReplicaSetsOptions{
		IssuesOptions: options,
	}
}

func newReplicaSetsCommand(factory cmdutil.Factory, options IssuesOptions) *cobra.Command {
	o := newReplicaSetsOptions(options)

	cmd := &cobra.Command{
		Use:          "replicasets",
		Short:        "List issues with ReplicaSets",
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

func (o *ReplicaSetsOptions) Run(ctx context.Context, noHeader bool) error {
	client, err := o.GetClient()
	if err != nil {
		return err
	}

	replicaSets, err := client.AppsV1().ReplicaSets(o.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	var matrix [][]string

	for _, rs := range replicaSets.Items {
		if rs.Status.Replicas != rs.Status.AvailableReplicas || rs.Status.Replicas != rs.Status.ReadyReplicas {
			row := []string{rs.Namespace, rs.Name, fmt.Sprintf("%d", rs.Status.Replicas), fmt.Sprintf("%d", rs.Status.AvailableReplicas), fmt.Sprintf("%d", rs.Status.ReadyReplicas), utils.GetAge(rs.CreationTimestamp)}
			matrix = append(matrix, row)
		}
	}

	headers := []string{"NAMESPACE", "NAME", "DESIRED", "CURRENT", "READY", "AGE"}

	buf := bytes.NewBuffer(nil)
	writer.WriteResults(buf, headers, matrix, noHeader)
	fmt.Printf("%s", buf.String())

	return nil
}
