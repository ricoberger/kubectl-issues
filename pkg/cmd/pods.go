package cmd

import (
	"bytes"
	"context"
	"fmt"

	"github.com/ricoberger/kubectl-issues/pkg/pods"
	"github.com/ricoberger/kubectl-issues/pkg/writer"

	"github.com/spf13/cobra"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

type PodsOptions struct {
	IssuesOptions
}

func newPodsOptions(options IssuesOptions) *PodsOptions {
	return &PodsOptions{
		IssuesOptions: options,
	}
}

func newPodsCommand(factory cmdutil.Factory, options IssuesOptions) *cobra.Command {
	o := newPodsOptions(options)

	cmd := &cobra.Command{
		Use:          "pods",
		Aliases:      []string{"pod", "po"},
		Short:        "List issues with Pods",
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

func (o *PodsOptions) Run(ctx context.Context, noHeader bool) error {
	client, err := o.GetClient()
	if err != nil {
		return err
	}

	unhealthy, err := pods.ListUnhealthy(ctx, client, o.namespace, "")
	if err != nil {
		return err
	}

	var matrix [][]string
	for _, pod := range unhealthy {
		matrix = append(matrix, []string{pod.Namespace, pod.Name, pod.Ready, pod.Status, pod.Restarts, pod.Age})
	}

	headers := []string{"NAMESPACE", "NAME", "READY", "STATUS", "RESTARTS", "AGE"}

	buf := bytes.NewBuffer(nil)
	writer.WriteResults(buf, headers, matrix, noHeader)
	fmt.Printf("%s", buf.String())

	return nil
}
