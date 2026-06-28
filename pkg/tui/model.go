package tui

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/ricoberger/kubectl-issues/pkg/pods"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/evertras/bubble-table/table"
)

const (
	columnKeyContext   = "context"
	columnKeyNamespace = "namespace"
	columnKeyName      = "name"
	columnKeyReady     = "ready"
	columnKeyStatus    = "status"
	columnKeyRestarts  = "restarts"
	columnKeyAge       = "age"
)

const (
	refreshInterval = 10 * time.Second
	fetchTimeout    = 30 * time.Second
)

var tableBorder = table.Border{
	Top:    "─",
	Left:   "│",
	Right:  "│",
	Bottom: "─",

	TopRight:    "╮",
	TopLeft:     "╭",
	BottomRight: "╯",
	BottomLeft:  "╰",

	TopJunction:    "╥",
	LeftJunction:   "├",
	RightJunction:  "┤",
	BottomJunction: "╨",
	InnerJunction:  "╫",

	InnerDivider: "║",
}

// TickMsg is sent on every refresh interval to trigger a new fetch of the
// unhealthy Pods.
type TickMsg time.Time

func tickEvery() tea.Cmd {
	return tea.Every(refreshInterval, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

// podsMsg is sent once a fetch of the unhealthy Pods has finished.
type podsMsg struct {
	items []pods.Pod
	err   error
}

// fetchCmd fetches the unhealthy Pods from all contexts in parallel.
func fetchCmd(clients []ContextClient) tea.Cmd {
	return func() tea.Msg {
		var (
			wg       sync.WaitGroup
			mu       sync.Mutex
			items    []pods.Pod
			firstErr error
		)

		for _, client := range clients {
			wg.Add(1)

			go func(client ContextClient) {
				defer wg.Done()

				ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
				defer cancel()

				result, err := pods.ListUnhealthy(ctx, client.Client, "", client.Name)

				mu.Lock()
				defer mu.Unlock()

				if err != nil {
					if firstErr == nil {
						firstErr = fmt.Errorf("%s: %w", client.Name, err)
					}
					return
				}

				items = append(items, result...)
			}(client)
		}

		wg.Wait()

		sort.Slice(items, func(i, j int) bool {
			if items[i].Context != items[j].Context {
				return items[i].Context < items[j].Context
			}
			if items[i].Namespace != items[j].Namespace {
				return items[i].Namespace < items[j].Namespace
			}
			return items[i].Name < items[j].Name
		})

		return podsMsg{items: items, err: firstErr}
	}
}

type Model struct {
	ScreenWidth  int
	ScreenHeight int

	Clients []ContextClient
	Table   table.Model

	count       int
	lastErr     error
	lastUpdated time.Time
}

func NewModel(clients []ContextClient) tea.Model {
	return &Model{
		Clients: clients,
		Table: table.New([]table.Column{
			table.NewFlexColumn(columnKeyContext, "Context", 1).WithStyle(lipgloss.NewStyle().Align(lipgloss.Left)),
			table.NewFlexColumn(columnKeyNamespace, "Namespace", 1).WithStyle(lipgloss.NewStyle().Align(lipgloss.Left)),
			table.NewFlexColumn(columnKeyName, "Name", 2).WithStyle(lipgloss.NewStyle().Align(lipgloss.Left)),
			table.NewColumn(columnKeyReady, "Ready", 7),
			table.NewFlexColumn(columnKeyStatus, "Status", 1),
			table.NewColumn(columnKeyRestarts, "Restarts", 18),
			table.NewColumn(columnKeyAge, "Age", 8),
		}).
			Focused(true).
			HeaderStyle(lipgloss.NewStyle().Bold(true)).
			HighlightStyle(lipgloss.NewStyle().Background(lipgloss.Color("12")).Foreground(lipgloss.Color("8"))).
			Border(tableBorder).
			WithPageSize(10).
			WithStaticFooter("Loading..."),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(fetchCmd(m.Clients), tickEvery())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	m.Table, cmd = m.Table.Update(msg)
	cmds = append(cmds, cmd)

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			cmds = append(cmds, tea.Quit)
		case "r":
			cmds = append(cmds, fetchCmd(m.Clients))
		}
	case tea.WindowSizeMsg:
		m.ScreenWidth = msg.Width
		m.ScreenHeight = msg.Height

		pageSize := max(msg.Height-6, 1)

		m.Table = m.Table.WithTargetWidth(msg.Width).WithMinimumHeight(msg.Height).WithPageSize(pageSize)
	case podsMsg:
		m.count = len(msg.items)
		m.lastErr = msg.err
		m.lastUpdated = time.Now()
		m.Table = m.Table.WithRows(generateRows(msg.items))
	case TickMsg:
		cmds = append(cmds, fetchCmd(m.Clients), tickEvery())
	}

	m.Table = m.Table.WithStaticFooter(m.footer())

	return m, tea.Batch(cmds...)
}

func (m Model) View() tea.View {
	var content string

	if m.ScreenWidth == 0 || m.ScreenHeight == 0 {
		content = "Loading..."
	} else {
		content = m.Table.View()
	}

	v := tea.NewView(content)
	v.AltScreen = true

	return v
}

func (m Model) footer() string {
	footer := fmt.Sprintf("Page %d/%d • %d unhealthy Pods", m.Table.CurrentPage(), m.Table.MaxPages(), m.count)

	if !m.lastUpdated.IsZero() {
		footer += fmt.Sprintf(" • Updated %s", m.lastUpdated.Format("15:04:05"))
	}

	if m.lastErr != nil {
		footer += fmt.Sprintf(" • Error: %s", m.lastErr.Error())
	}

	footer += " • r refresh"

	return footer
}

func generateRows(items []pods.Pod) []table.Row {
	var rows []table.Row

	for _, item := range items {
		rows = append(rows, table.NewRow(table.RowData{
			columnKeyContext:   item.Context,
			columnKeyNamespace: item.Namespace,
			columnKeyName:      item.Name,
			columnKeyReady:     item.Ready,
			columnKeyStatus:    table.NewStyledCell(item.Status, statusStyle(item.Status)),
			columnKeyRestarts:  item.Restarts,
			columnKeyAge:       item.Age,
		}))
	}

	return rows
}

// statusStyle returns the style for a status cell. Transient states are colored
// yellow, everything else is colored red as all shown Pods are unhealthy.
func statusStyle(status string) lipgloss.Style {
	color := lipgloss.Color("9")

	switch status {
	case "Running", "Completed":
		color = lipgloss.Color("10")
	case "Pending", "ContainerCreating", "PodInitializing", "Terminating":
		color = lipgloss.Color("11")
	}

	return lipgloss.NewStyle().Foreground(color).Bold(true)
}
