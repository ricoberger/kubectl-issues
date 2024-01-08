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

type JobsOptions struct {
	IssuesOptions
}

func newJobsOptions(options IssuesOptions) *JobsOptions {
	return &JobsOptions{
		IssuesOptions: options,
	}
}

func newJobsCommand(factory cmdutil.Factory, options IssuesOptions) *cobra.Command {
	o := newJobsOptions(options)

	cmd := &cobra.Command{
		Use:          "jobs",
		Aliases:      []string{"job"},
		Short:        "List issues with Jobs",
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

func (o *JobsOptions) Run(ctx context.Context, noHeader bool) error {
	client, err := o.GetClient()
	if err != nil {
		return err
	}

	jobs, err := client.BatchV1().Jobs(o.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	var matrix [][]string

	for _, job := range jobs.Items {
		for _, c := range job.Status.Conditions {
			if c.Reason == "BackoffLimitExceeded" || c.Reason == "DeadlineExceeded" {
				row := []string{job.Namespace, job.Name, c.Reason, c.Message, utils.GetAge(job.CreationTimestamp)}
				matrix = append(matrix, row)
			}
		}
	}

	headers := []string{"NAMESPACE", "NAME", "REASON", "MESSAGE", "AGE"}

	buf := bytes.NewBuffer(nil)
	writer.WriteResults(buf, headers, matrix, noHeader)
	fmt.Printf("%s", buf.String())

	return nil
}
