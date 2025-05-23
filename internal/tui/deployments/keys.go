package deployments

import "github.com/charmbracelet/bubbles/key"

type KeyMap struct {
	Context   key.Binding
	Namespace key.Binding
	Pods      key.Binding
}

func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Namespace, k.Context, k.Pods}
}

func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Namespace, k.Context, k.Pods},
	}
}

var keys = KeyMap{
	Context: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "context"),
	),
	Namespace: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "namespace"),
	),
	Pods: key.NewBinding(
		key.WithKeys("p"),
		key.WithHelp("p", "pods"),
	),
}
