package main

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type context struct {
	contexts        list.Model
	selectedContext string
}

func UpdateContext(m model, msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.context.contexts.SetWidth(msg.Width)
		return m, nil

	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "esc":
			m.currentView = "pod"
			return m, nil
		case "enter":
			i, ok := m.context.contexts.SelectedItem().(item)
			if ok {
				m.context.selectedContext = i.name
				m.namespace.selectedNamespace = i.namespace
				setContext(i.name, i.namespace, i.user)
				m.namespace.namespaces = buildNamespacesList()
			}
			m.currentView = "pod"
			return refreshPods(m), nil
		}
	}

	var cmd tea.Cmd
	m.context.contexts, cmd = m.context.contexts.Update(msg)
	return m, cmd
}

func contextView(m model) string {
	return "\n" + m.context.contexts.View()
}

func buildContextList() list.Model {
	var items []list.Item
	ctxs := listContexts()
	for _, ctx := range ctxs {
		items = append(items, item{
			name:      ctx.Cluster,
			namespace: ctx.Namespace,
			user:      ctx.AuthInfo,
		})
	}
	l := list.New(items, itemDelegate{}, defaultWidth, listHeight)
	l.Title = "Select Context"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle
	return l
}
