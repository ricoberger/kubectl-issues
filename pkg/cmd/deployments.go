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

type DeploymentsOptions struct {
	IssuesOptions
}

func newDeploymentsOptions(options IssuesOptions) *DeploymentsOptions {
	return &DeploymentsOptions{
		IssuesOptions: options,
	}
}

func newDeploymentsCommand(factory cmdutil.Factory, options IssuesOptions) *cobra.Command {
	o := newDeploymentsOptions(options)

	cmd := &cobra.Command{
		Use:          "deployments",
		Aliases:      []string{"deployment", "deploy"},
		Short:        "List issues with Deployments",
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

func (o *DeploymentsOptions) Run(ctx context.Context, noHeader bool) error {
	client, err := o.GetClient()
	if err != nil {
		return err
	}

	deployments, err := client.AppsV1().Deployments(o.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	var matrix [][]string

	for _, deploy := range deployments.Items {
		if deploy.Status.Replicas != deploy.Status.ReadyReplicas || deploy.Status.Replicas != deploy.Status.UpdatedReplicas || deploy.Status.Replicas != deploy.Status.AvailableReplicas {
			row := []string{deploy.Namespace, deploy.Name, fmt.Sprintf("%d/%d", deploy.Status.ReadyReplicas, deploy.Status.Replicas), fmt.Sprintf("%d", deploy.Status.UpdatedReplicas), fmt.Sprintf("%d", deploy.Status.AvailableReplicas), utils.GetAge(deploy.CreationTimestamp)}
			matrix = append(matrix, row)
		}
	}

	headers := []string{"NAMESPACE", "NAME", "READY", "UP-TO-DATE", "AVAILABLE", "AGE"}

	buf := bytes.NewBuffer(nil)
	writer.WriteResults(buf, headers, matrix, noHeader)
	fmt.Printf("%s", buf.String())

	return nil
}
