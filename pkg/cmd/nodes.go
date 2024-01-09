package cmd

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/ricoberger/kubectl-issues/pkg/cmd/utils"
	"github.com/ricoberger/kubectl-issues/pkg/writer"

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

func newNodesCommand(factory cmdutil.Factory, options IssuesOptions) *cobra.Command {
	o := newNodesOptions(options)

	cmd := &cobra.Command{
		Use:          "nodes",
		Aliases:      []string{"node", "no"},
		Short:        "List issues with Nodes",
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

func (o *NodesOptions) Run(ctx context.Context, noHeader bool) error {
	client, err := o.GetClient()
	if err != nil {
		return err
	}

	nodes, err := client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	var matrix [][]string

	for _, node := range nodes.Items {
		for _, condition := range node.Status.Conditions {
			if (condition.Type == corev1.NodeReady && condition.Status != corev1.ConditionTrue) || (condition.Type != corev1.NodeReady && condition.Status == corev1.ConditionTrue) {
				row := []string{node.Name, getNodeStatus(node.Status.Conditions), node.Labels["kubernetes.io/role"], utils.GetAge(node.CreationTimestamp), node.Status.NodeInfo.KubeletVersion}
				matrix = append(matrix, row)
			}
		}
	}

	headers := []string{"NAME", "Status", "ROLES", "AGE", "VERSION"}

	buf := bytes.NewBuffer(nil)
	writer.WriteResults(buf, headers, matrix, noHeader)
	fmt.Printf("%s", buf.String())

	return nil
}

func getNodeStatus(conditions []corev1.NodeCondition) string {
	var statuses []string

	for _, condition := range conditions {
		if condition.Status == corev1.ConditionTrue {
			statuses = append(statuses, string(condition.Type))
		}
	}

	if len(statuses) == 0 {
		return "NotReady"
	}

	return strings.Join(statuses, ", ")
}
