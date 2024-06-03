package main

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type namespace struct {
	namespaces        list.Model
	selectedNamespace string
}

func UpdateNamespace(m model, msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.namespace.namespaces.SetWidth(msg.Width)
		return m, nil

	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "esc":
			m.currentView = "pod"
			return m, nil
		case "enter":
			i, ok := m.namespace.namespaces.SelectedItem().(item)
			if ok {
				m.namespace.selectedNamespace = i.name
			}

			m.namespaceChange <- m.namespace.selectedNamespace
			m.currentView = "pod"
			return refreshPods(m), nil
		}
	}

	var cmd tea.Cmd
	m.namespace.namespaces, cmd = m.namespace.namespaces.Update(msg)
	return m, cmd
}

func namespaceView(m model) string {
	return "\n" + m.namespace.namespaces.View()
}

func buildNamespacesList() list.Model {
	var items []list.Item
	ns := getNamespaces()
	for _, n := range ns {
		items = append(items, item{name: n.Name})
	}
	l := list.New(items, itemDelegate{}, defaultWidth, listHeight)
	l.Title = "Select Namespace"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle
	return l
}
