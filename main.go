package main

import (
	"os"

	"github.com/ricoberger/kubectl-issues/pkg/cmd"

	"github.com/spf13/pflag"
)

func main() {
	flags := pflag.NewFlagSet("kubectl-issues", pflag.ExitOnError)
	pflag.CommandLine = flags

	root := cmd.NewIssuesCommand()
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
