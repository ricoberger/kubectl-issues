package tui

import (
	"github.com/ricoberger/kubectl-issues/pkg/client"
	"github.com/ricoberger/kubectl-issues/pkg/pods"

	tea "charm.land/bubbletea/v2"
)

// Start starts the TUI showing the unhealthy Pods from all given clients. The
// podsOptions tune which Pods are considered unhealthy.
func Start(clients []client.ContextClient, podsOptions pods.Options) error {
	model := NewModel(clients, podsOptions)

	p := tea.NewProgram(model)
	if _, err := p.Run(); err != nil {
		return err
	}

	return nil
}
