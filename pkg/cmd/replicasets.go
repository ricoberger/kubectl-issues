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

type ReplicaSetsOptions struct {
	IssuesOptions
}

func newReplicaSetsOptions(options IssuesOptions) *ReplicaSetsOptions {
	return &ReplicaSetsOptions{
		IssuesOptions: options,
	}
}

func newReplicaSetsCommand(_ cmdutil.Factory, options IssuesOptions) *cobra.Command {
	o := newReplicaSetsOptions(options)

	cmd := &cobra.Command{
		Use:          "replicasets",
		Aliases:      []string{"replicaset", "rs"},
		Short:        "List issues with ReplicaSets",
		SilenceUsage: true,
		RunE: func(c *cobra.Command, args []string) error {
			if err := o.Complete(c); err != nil {
				return err
			}

			ctx := context.Background()
			noHeader := c.Flag("no-headers").Changed
			headers := []string{"NAMESPACE", "NAME", "DESIRED", "CURRENT", "READY", "AGE"}
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

func (o *ReplicaSetsOptions) rows(ctx context.Context, contextClient client.ContextClient) ([][]string, error) {
	replicaSets, err := contextClient.Client.AppsV1().ReplicaSets(contextClient.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var matrix [][]string

	for _, rs := range replicaSets.Items {
		if rs.Status.Replicas != rs.Status.AvailableReplicas || rs.Status.Replicas != rs.Status.ReadyReplicas {
			row := []string{rs.Namespace, rs.Name, fmt.Sprintf("%d", rs.Status.Replicas), fmt.Sprintf("%d", rs.Status.AvailableReplicas), fmt.Sprintf("%d", rs.Status.ReadyReplicas), utils.GetAge(rs.CreationTimestamp)}
			matrix = append(matrix, row)
		}
	}

	return matrix, nil
}
