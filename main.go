package main

import (
	"fmt"
	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	"os"
)

type model struct {
	context         context
	namespace       namespace
	pod             pods
	currentView     string
	namespaceChange chan string
}

func newModel() model {
	ctx, ns := getCurrent()

	m := model{
		currentView: "pod",
		namespace: namespace{
			namespaces:        buildNamespacesList(),
			selectedNamespace: ns,
		},
		context: context{
			contexts:        buildContextList(),
			selectedContext: ctx,
		},
		pod: pods{
			sub:  make(chan struct{}),
			pods: buildPodsTable(),
			help: help.New(),
		},
		namespaceChange: make(chan string),
	}
	m = refreshPods(m)
	return m
}

func main() {
	m := newModel()

	go func() {
		for {
			select {
			case ns := <-m.namespaceChange:
				event := watchPods(ns)
				for range event {
					m.pod.sub <- struct{}{}
				}
			}
		}
	}()

	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}

func (m model) Init() tea.Cmd {
	return waitForActivity(m.pod.sub)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		k := msg.String()
		if k == "q" || k == "ctrl+c" {
			return m, tea.Quit
		}
	}
	switch m.currentView {
	case "context":
		return UpdateContext(m, msg)
	case "namespace":
		return UpdateNamespace(m, msg)
	case "pod":
		return UpdatePod(m, msg)
	}
	return m, nil
}

func (m model) View() string {
	switch m.currentView {
	case "namespace":
		return namespaceView(m)
	case "pod":
		return podsView(m)
	case "context":
		return contextView(m)
	default:
		panic("unknown view")
	}
}
