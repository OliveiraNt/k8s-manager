package deployments

import (
	"context"
	"github.com/OliveiraNt/k8s-manager/internal/kubernetes"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"k8s.io/apimachinery/pkg/watch"
)

type Model struct {
	Namespace   string
	Deployments table.Model
	Help        help.Model
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
		m.Deployments.SetWidth(msg.Width)

	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "enter":
			// No action for enter in deployments view
		default:
			m.Deployments, cmd = m.Deployments.Update(msg)
		}
	case ChangeMsg:
		RefreshDeployments(&m, false)
	}
	return m, cmd
}

func (m Model) View() string {
	return m.Deployments.View() + helpStyle.Render(m.Help.View(keys))
}

func RefreshDeployments(m *Model, goTop bool) {
	var rows []table.Row
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	deps, err := kubernetes.GetDeployments(ctx, m.Namespace)
	if err != nil {
		panic(err)
	}
	for _, d := range deps {
		row := table.Row{
			d.Name,
			kubernetes.ColumnHelperReplicas(d.Status),
			string(d.Status.Conditions[0].Type),
			kubernetes.ColumnHelperAge(d.CreationTimestamp),
		}
		rows = append(rows, row)
	}
	m.Deployments.SetRows(rows)
	if goTop {
		m.Deployments.GotoTop()
	}
}

func New(namespace string) Model {
	columns := []table.Column{
		{Title: "NAME", Width: 50},
		{Title: "READY", Width: 10},
		{Title: "STATUS", Width: 15},
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
		Namespace:   namespace,
		Deployments: t,
		Help:        help.New(),
	}
	RefreshDeployments(&m, true)
	return m
}
