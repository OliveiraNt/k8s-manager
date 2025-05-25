package pods

import (
	"context"
	"github.com/OliveiraNt/k8s-manager/internal/kubernetes"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"k8s.io/apimachinery/pkg/watch"
	"log"
)

type Model struct {
	Namespace string
	Pods      table.Model
	Help      help.Model
}
type ChangeMsg watch.Event

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Help.Width = msg.Width
		m.Pods.SetWidth(msg.Width)

	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "enter":
		default:
			m.Pods, cmd = m.Pods.Update(msg)
		}
	case ChangeMsg:
		RefreshPods(&m, false)
	}
	return m, cmd

}

func (m Model) View() string {
	return m.Pods.View() + helpStyle.Render(m.Help.View(keys))
}

func RefreshPods(m *Model, goTop bool) {
	var rows []table.Row
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	pds, err := kubernetes.GetPods(ctx, m.Namespace)
	if err != nil {
		// Log the error but continue with empty rows
		log.Printf("[ERROR] Failed to get pods: %v", err)
		rows = []table.Row{{"Error loading pods", "", "", "", ""}}
	} else {
		for _, p := range pds {
			row := table.Row{
				p.Name,
				kubernetes.ColumnHelperReady(p.Status.ContainerStatuses),
				kubernetes.ColumnHelperStatus(p.Status),
				kubernetes.ColumnHelperRestarts(p.Status.ContainerStatuses),
				kubernetes.ColumnHelperAge(p.CreationTimestamp),
			}
			rows = append(rows, row)
		}
	}
	m.Pods.SetRows(rows)
	if goTop {
		m.Pods.GotoTop()
	}
}

func New(namespace string) Model {
	columns := []table.Column{
		{Title: "NAME", Width: 50},
		{Title: "READY", Width: 5},
		{Title: "STATUS", Width: 10},
		{Title: "RESTARTS", Width: 10},
		{Title: "AGE", Width: 5},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("#FF7900")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("#FF7900")).
		Background(lipgloss.Color("#000")).
		Bold(false)
	t.SetStyles(s)

	m := Model{
		Namespace: namespace,
		Pods:      t,
		Help:      help.New(),
	}
	RefreshPods(&m, true)
	return m
}
