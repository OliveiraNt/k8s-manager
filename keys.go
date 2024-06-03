package main

import "github.com/charmbracelet/bubbles/key"

type KeyMap struct {
	Context   key.Binding
	Namespace key.Binding
	Logs      key.Binding
}

func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Namespace, k.Context}

}

func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Namespace, k.Context, k.Logs},
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
}
