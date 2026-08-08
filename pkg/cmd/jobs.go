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

type JobsOptions struct {
	IssuesOptions
}

func newJobsOptions(options IssuesOptions) *JobsOptions {
	return &JobsOptions{
		IssuesOptions: options,
	}
}

func newJobsCommand(_ cmdutil.Factory, options IssuesOptions) *cobra.Command {
	o := newJobsOptions(options)

	cmd := &cobra.Command{
		Use:          "jobs",
		Aliases:      []string{"job"},
		Short:        "List issues with Jobs",
		SilenceUsage: true,
		RunE: func(c *cobra.Command, args []string) error {
			if err := o.Complete(c); err != nil {
				return err
			}

			ctx := context.Background()
			noHeader := c.Flag("no-headers").Changed
			headers := []string{"NAMESPACE", "NAME", "REASON", "MESSAGE", "AGE"}
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

func (o *JobsOptions) rows(ctx context.Context, contextClient client.ContextClient) ([][]string, error) {
	jobs, err := contextClient.Client.BatchV1().Jobs(contextClient.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
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

	return matrix, nil
}
