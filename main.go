package main

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	context "k8s-manager/context"
	"k8s-manager/kubernetes"
	"k8s-manager/namespace"
	"k8s-manager/pods"
	"os"
	"strings"
)

type Views uint8

const (
	Context Views = iota
	Namespace
	Pod
	Log
)

type model struct {
	context   context.Model
	namespace namespace.Model
	pod       pods.Model
	//log             log
	currentView Views
}

func newModel() model {
	_, ns, _ := kubernetes.GetCurrent()
	if ns == "" {
		ns = "default"
	}
	m := model{
		currentView: Pod,
		context:     context.New(),
		pod:         pods.New(ns),
		namespace:   namespace.New(ns),
	}
	return m
}

func main() {
	m := newModel()

	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
		switch m.currentView {
		case Pod:
			switch keypress := msg.String(); keypress {
			case "c":
				m.currentView = Context
			case "n":
				m.currentView = Namespace
			default:
				var podModel tea.Model
				podModel, cmd = m.pod.Update(msg)
				if pod, ok := podModel.(pods.Model); ok {
					m.pod = pod
				}
			}
		case Context:
			var contextModel tea.Model
			switch keypress := msg.String(); keypress {
			case "esc":
				m.currentView = Pod
			case "enter":
				contextModel, cmd = m.context.Update(msg)
				if ctx, ok := contextModel.(context.Model); ok {
					m.context = ctx
					ns := m.context.SelectedContext.Namespace
					if ns == "" {
						ns = "default"
					}
					m.namespace = namespace.New(ns)
					m.pod.Namespace = ns
					pods.RefreshPods(&m.pod)
					m.currentView = Pod
				}
			default:
				contextModel, cmd = m.context.Update(msg)
				if ctx, ok := contextModel.(context.Model); ok {
					m.context = ctx
				}
			}
		case Namespace:
			var namespaceModel tea.Model
			switch keypress := msg.String(); keypress {
			case "esc":
				m.currentView = Pod
			case "enter":
				namespaceModel, cmd = m.namespace.Update(msg)
				if ns, ok := namespaceModel.(namespace.Model); ok {
					m.namespace = ns
					m.pod.Namespace = m.namespace.SelectedNamespace
					pods.RefreshPods(&m.pod)
					m.currentView = Pod
				}
			default:
				namespaceModel, cmd = m.namespace.Update(msg)
				if ns, ok := namespaceModel.(namespace.Model); ok {
					m.namespace = ns
				}
			}
		}
	default:
		switch m.currentView {
		case Pod:
			var podModel tea.Model
			podModel, cmd = m.pod.Update(msg)
			if pod, ok := podModel.(pods.Model); ok {
				m.pod = pod
			}
		case Context:
			var contextModel tea.Model
			contextModel, cmd = m.context.Update(msg)
			if ctx, ok := contextModel.(context.Model); ok {
				m.context = ctx
			}
		case Namespace:
			var namespaceModel tea.Model
			namespaceModel, cmd = m.namespace.Update(msg)
			if ns, ok := namespaceModel.(namespace.Model); ok {
				m.namespace = ns
			}
		}
	}
	return m, cmd
}

func (m model) View() string {
	var b strings.Builder
	fmt.Fprintf(&b, "CONTEXT: %s\n", m.context.SelectedContext.Name)
	fmt.Fprintf(&b, "NAMESPACE: %s\n", m.pod.Namespace)
	s := titleStyle.Render(b.String()) + "\n\n"
	switch m.currentView {
	case Pod:
		return s + m.pod.View()
	case Context:
		return m.context.View()
	case Namespace:
		return m.namespace.View()
	default:
		return s
	}
}
