package context

import (
	"github.com/OliveiraNt/k8s-manager/kubernetes"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	Contexts        list.Model
	SelectedContext Item
	ShowLoadingText bool
}

type ChangeMsg struct{}

func New() Model {
	name, namespace, user := kubernetes.GetCurrent()
	return Model{
		Contexts:        buildContextList(),
		SelectedContext: Item{name, namespace, user},
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Contexts.SetWidth(msg.Width)

	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "enter":
			m.ShowLoadingText = true
			cmd = func() tea.Msg { return ChangeMsg{} }
		default:
			m.Contexts, cmd = m.Contexts.Update(msg)
		}
	case ChangeMsg:
		i, ok := m.Contexts.SelectedItem().(Item)
		if ok {
			m.SelectedContext = i
			kubernetes.SetContext(i.Name, i.Namespace, i.User)
		}
	default:
		m.Contexts, cmd = m.Contexts.Update(msg)
	}

	return m, cmd
}

func (m Model) View() string {
	if m.ShowLoadingText {
		return "\nLoading..."

	}
	return "\n" + m.Contexts.View()
}

func buildContextList() list.Model {
	var items []list.Item
	ctxs := kubernetes.ListContexts()
	for _, ctx := range ctxs {
		items = append(items, Item{
			Name:      ctx.Cluster,
			Namespace: ctx.Namespace,
			User:      ctx.AuthInfo,
		})
	}
	l := list.New(items, itemDelegate{}, defaultWidth, listHeight)
	l.Title = "Select Context"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle
	l.AdditionalShortHelpKeys = keys.ShortHelp
	return l
}
