package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/ricoberger/kubectl-issues/pkg/client"
	"github.com/ricoberger/kubectl-issues/pkg/cmd/utils"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

type NodesOptions struct {
	IssuesOptions
}

func newNodesOptions(options IssuesOptions) *NodesOptions {
	return &NodesOptions{
		IssuesOptions: options,
	}
}

func newNodesCommand(_ cmdutil.Factory, options IssuesOptions) *cobra.Command {
	o := newNodesOptions(options)

	cmd := &cobra.Command{
		Use:          "nodes",
		Aliases:      []string{"node", "no"},
		Short:        "List issues with Nodes",
		SilenceUsage: true,
		RunE: func(c *cobra.Command, args []string) error {
			if err := o.Complete(c); err != nil {
				return err
			}

			ctx := context.Background()
			noHeader := c.Flag("no-headers").Changed
			headers := []string{"NAME", "STATUS", "ROLES", "AGE", "VERSION"}
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

func (o *NodesOptions) rows(ctx context.Context, contextClient client.ContextClient) ([][]string, error) {
	nodes, err := contextClient.Client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var matrix [][]string

	for _, node := range nodes.Items {
		if !isNodeReady(node.Status.Conditions) || node.Spec.Unschedulable {
			row := []string{node.Name, getNodeStatus(node), node.Labels["kubernetes.io/role"], utils.GetAge(node.CreationTimestamp), node.Status.NodeInfo.KubeletVersion}
			matrix = append(matrix, row)
		}
	}

	return matrix, nil
}

func isNodeReady(conditions []corev1.NodeCondition) bool {
	for _, condition := range conditions {
		if condition.Type == corev1.NodeReady && condition.Status == corev1.ConditionTrue {
			return true
		}
	}

	return false
}

func getNodeStatus(node corev1.Node) string {
	var statuses []string

	for _, condition := range node.Status.Conditions {
		if condition.Status == corev1.ConditionTrue {
			statuses = append(statuses, string(condition.Type))
		}
	}

	if len(statuses) == 0 {
		if node.Spec.Unschedulable {
			return "NotReady,SchedulingDisabled"
		}

		return "NotReady"
	}

	if node.Spec.Unschedulable {
		statuses = append(statuses, "SchedulingDisabled")
	}

	return strings.Join(statuses, ",")
}
