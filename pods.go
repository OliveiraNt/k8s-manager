package main

import (
	"fmt"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"strings"
)

type pods struct {
	pods table.Model
	help help.Model
	sub  chan struct{}
}

type podsEvents struct{}

func waitForActivity(sub chan struct{}) tea.Cmd {
	return func() tea.Msg {
		return podsEvents(<-sub)
	}
}

func UpdatePod(m model, msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.pod.help.Width = msg.Width
		m.pod.pods.SetWidth(msg.Width)
		return m, nil

	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "n":
			m.currentView = "namespace"
			return m, nil
		case "c":
			m.currentView = "context"
			return m, nil
		}
	case podsEvents:
		m = refreshPods(m)
		return m, waitForActivity(m.pod.sub)
	}

	var cmd tea.Cmd
	m.pod.pods, cmd = m.pod.pods.Update(msg)
	return m, cmd
}

func podsView(m model) string {

	view := m.pod.pods.View()
	hv := m.pod.helpView()
	var b strings.Builder
	fmt.Fprintf(&b, "CONTEXT: %s\n", m.context.selectedContext)
	fmt.Fprintf(&b, "NAMESPACE: %s\n", m.namespace.selectedNamespace)
	return titleStyle.Render(b.String()) + "\n\n" + view + hv
}

func (m pods) helpView() string {
	return helpStyle.Render(m.help.View(keys))
}

func buildPodsTable() table.Model {
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
		table.WithHeight(7),
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

	return t
}

func refreshPods(m model) model {
	var rows []table.Row
	pds := getPods(m.namespace.selectedNamespace)
	for _, p := range pds {
		row := table.Row{
			p.Name,
			columnHelperReady(p.Status.ContainerStatuses),
			columnHelperStatus(p.Status),
			columnHelperRestarts(p.Status.ContainerStatuses),
			columnHelperAge(p.CreationTimestamp),
		}
		rows = append(rows, row)
	}
	m.pod.pods.SetRows(rows)

	return m
}
