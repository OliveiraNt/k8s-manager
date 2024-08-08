package namespace

import (
	"context"
	"github.com/OliveiraNt/k8s-manager/internal/kubernetes"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	Namespaces        list.Model
	SelectedNamespace string
}

func (m Model) Init() tea.Cmd {
	return nil

}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Namespaces.SetWidth(msg.Width)

	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "enter":
			i, ok := m.Namespaces.SelectedItem().(item)
			if ok {
				m.SelectedNamespace = i.name
			}
		default:
			m.Namespaces, cmd = m.Namespaces.Update(msg)
		}

	default:
		m.Namespaces, cmd = m.Namespaces.Update(msg)
	}
	return m, cmd
}

func (m Model) View() string {
	return "\n" + m.Namespaces.View()
}

func New(ns string) Model {
	return Model{
		Namespaces:        buildNamespacesList(),
		SelectedNamespace: ns,
	}
}

func buildNamespacesList() list.Model {
	var items []list.Item
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	namespaces, err := kubernetes.GetNamespaces(ctx)
	if err != nil {
		panic(err.Error())
	}
	ns := namespaces
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
	l.AdditionalShortHelpKeys = keys.ShortHelp
	return l
}
