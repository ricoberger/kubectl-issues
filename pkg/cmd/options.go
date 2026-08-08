package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/ricoberger/kubectl-issues/pkg/client"
	"github.com/ricoberger/kubectl-issues/pkg/writer"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type IssuesOptions struct {
	Streams              genericclioptions.IOStreams
	ConfigFlags          *genericclioptions.ConfigFlags
	ResourceBuilderFlags *genericclioptions.ResourceBuilderFlags
	contexts             []string
	allNamespaces        bool
}

func NewIssuesOptions() IssuesOptions {
	rbFlags := &genericclioptions.ResourceBuilderFlags{}
	rbFlags.WithAllNamespaces(false)

	return IssuesOptions{
		// The config flags must not use a persistent (cached) config, because
		// GetClients builds one client per requested context and a cached
		// loader would reuse the config of the first context for all others.
		ConfigFlags:          genericclioptions.NewConfigFlags(false),
		ResourceBuilderFlags: rbFlags,
		Streams: genericclioptions.IOStreams{
			In:     os.Stdin,
			Out:    os.Stdout,
			ErrOut: os.Stderr,
		},
	}
}

func (o *IssuesOptions) Complete(cmd *cobra.Command) error {
	if flag := cmd.Flag("all-namespaces"); flag != nil && flag.Changed {
		o.allNamespaces = *o.ResourceBuilderFlags.AllNamespaces
	}

	contexts, err := cmd.Flags().GetStringArray("context")
	if err != nil {
		return err
	}
	o.contexts = contexts

	return nil
}

// GetClients builds a Kubernetes client for every requested context. If no
// contexts were requested, a single client for the current context of the
// kubeconfig is returned.
func (o *IssuesOptions) GetClients() ([]client.ContextClient, error) {
	contexts := o.contexts
	if len(contexts) == 0 {
		contexts = []string{""}
	}

	var clients []client.ContextClient
	for _, name := range contexts {
		contextClient, err := client.New(o.ConfigFlags, name, o.allNamespaces)
		if err != nil {
			return nil, fmt.Errorf("failed to create client for context %q: %w", name, err)
		}
		clients = append(clients, contextClient)
	}

	return clients, nil
}

// rowsFunc collects the table rows for a single context. The returned rows
// must not contain the context name, it is prepended by RunAcrossContexts.
type rowsFunc func(ctx context.Context, contextClient client.ContextClient) ([][]string, error)

// RunAcrossContexts collects the table rows from all requested contexts in
// parallel and prints them as a single table with a leading CONTEXT column.
// The rows are grouped in the order the contexts were specified. An error in
// one context is printed to stderr and does not hide the results of the other
// contexts.
func (o *IssuesOptions) RunAcrossContexts(ctx context.Context, noHeader bool, headers []string, rows rowsFunc) error {
	clients, err := o.GetClients()
	if err != nil {
		return err
	}

	type result struct {
		rows [][]string
		err  error
	}

	results := make([]result, len(clients))

	var wg sync.WaitGroup
	for i, contextClient := range clients {
		wg.Add(1)

		go func(i int, contextClient client.ContextClient) {
			defer wg.Done()

			contextRows, err := rows(ctx, contextClient)
			if err != nil {
				results[i].err = fmt.Errorf("%s: %w", contextClient.Name, err)
				return
			}

			for _, row := range contextRows {
				results[i].rows = append(results[i].rows, append([]string{contextClient.Name}, row...))
			}
		}(i, contextClient)
	}
	wg.Wait()

	var matrix [][]string
	for _, r := range results {
		if r.err != nil {
			fmt.Fprintln(o.Streams.ErrOut, r.err.Error())
			continue
		}
		matrix = append(matrix, r.rows...)
	}

	buf := bytes.NewBuffer(nil)
	writer.WriteResults(buf, append([]string{"CONTEXT"}, headers...), matrix, noHeader)
	fmt.Fprintf(o.Streams.Out, "%s", buf.String())

	return nil
}
