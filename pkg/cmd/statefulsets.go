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

type StatefulSetsOptions struct {
	IssuesOptions
}

func newStatefulSetsOptions(options IssuesOptions) *StatefulSetsOptions {
	return &StatefulSetsOptions{
		IssuesOptions: options,
	}
}

func newStatefulSetsCommand(_ cmdutil.Factory, options IssuesOptions) *cobra.Command {
	o := newStatefulSetsOptions(options)

	cmd := &cobra.Command{
		Use:          "statefulsets",
		Aliases:      []string{"statefulset", "sts"},
		Short:        "List issues with StatefulSets",
		SilenceUsage: true,
		RunE: func(c *cobra.Command, args []string) error {
			if err := o.Complete(c); err != nil {
				return err
			}

			ctx := context.Background()
			noHeader := c.Flag("no-headers").Changed
			headers := []string{"NAMESPACE", "NAME", "READY", "UP-TO-DATE", "AVAILABLE", "AGE"}
			if err := o.RunAcrossContexts(ctx, noHeader, headers, o.rows); err != nil {
				fmt.Fprintln(options.Streams.ErrOut, err.Error())
				return nil
			}
			return nil
		},
	}

	o.ResourceBuilderFlags.AddFlags(cmd.Flags())
	cmd.Flags().StringVarP(&o.output, "output", "o", "", "Output format. One of: json.")

	return cmd
}

func (o *StatefulSetsOptions) rows(ctx context.Context, contextClient client.ContextClient) ([][]string, error) {
	statefulSets, err := contextClient.Client.AppsV1().StatefulSets(contextClient.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var matrix [][]string

	for _, sts := range statefulSets.Items {
		if sts.Status.Replicas != sts.Status.ReadyReplicas || sts.Status.ReadyReplicas != sts.Status.UpdatedReplicas || sts.Status.Replicas != sts.Status.AvailableReplicas {
			row := []string{sts.Namespace, sts.Name, fmt.Sprintf("%d/%d", sts.Status.ReadyReplicas, sts.Status.Replicas), fmt.Sprintf("%d", sts.Status.UpdatedReplicas), fmt.Sprintf("%d", sts.Status.AvailableReplicas), utils.GetAge(sts.CreationTimestamp)}
			matrix = append(matrix, row)
		}
	}

	return matrix, nil
}
