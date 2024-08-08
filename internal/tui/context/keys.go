package context

import "github.com/charmbracelet/bubbles/key"

type KeyMap struct {
	Select key.Binding
}

func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Select}

}

var keys = KeyMap{
	Select: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
}
