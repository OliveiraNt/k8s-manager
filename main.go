package main

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"k8s-manager/context"
	"k8s-manager/kubernetes"
	"k8s-manager/namespace"
	"k8s-manager/pods"
	"k8s.io/apimachinery/pkg/watch"
	"os"
	"strings"
)

type Views uint8

const (
	Context Views = iota
	Namespace
	Pod
	//Log
)

type model struct {
	context   context.Model
	namespace namespace.Model
	pod       pods.Model
	watch     watch.Interface
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
		watch:       kubernetes.WatchPods(ns),
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

func watchPodEvents(sub <-chan watch.Event) tea.Cmd {
	return func() tea.Msg {
		return pods.ChangeMsg(<-sub)
	}
}
func watchPods(ns string) watch.Interface {
	return kubernetes.WatchPods(ns)
}

func (m model) Init() tea.Cmd {
	return watchPodEvents(m.watch.ResultChan())
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// Handle quit keys regardless of the message type
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keypress := keyMsg.String(); keypress == "q" || keypress == "ctrl+c" {
			return m, tea.Quit
		}
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.currentView {
		case Pod:
			m.updatePodView(msg, &cmd)
		case Context:
			m.updateContextView(msg, &cmd)
		case Namespace:
			m.updateNamespaceView(msg, &cmd)
		default:
		}
	case context.ChangeMsg:
		m.watch.Stop()
		var ctxModel tea.Model
		ctxModel, cmd = m.context.Update(msg)
		if ctx, ok := ctxModel.(context.Model); ok {
			m.context = ctx
			m.context.ShowLoadingText = false
			ns := m.context.SelectedContext.Namespace
			if ns == "" {
				ns = "default"
			}
			m.namespace = namespace.New(ns)
			m.pod.Namespace = ns
			pods.RefreshPods(&m.pod, true)
			m.watch = watchPods(ns)
			cmd = tea.Batch(cmd, watchPodEvents(m.watch.ResultChan()))
			m.currentView = Pod
		}
	case pods.ChangeMsg:
		switch m.currentView {
		case Pod:
			var podModel tea.Model
			podModel, cmd = m.pod.Update(msg)
			if pod, ok := podModel.(pods.Model); ok {
				m.pod = pod
			}
		}
		cmd = tea.Batch(cmd, watchPodEvents(m.watch.ResultChan()))

	default:
		// Handle other message types
		switch m.currentView {
		case Pod:
			var podModel tea.Model
			podModel, cmd = m.pod.Update(msg)
			if pod, ok := podModel.(pods.Model); ok {
				m.pod = pod
			}
		case Context:
			var ctxModel tea.Model
			ctxModel, cmd = m.context.Update(msg)
			if ctx, ok := ctxModel.(context.Model); ok {
				m.context = ctx
			}
		case Namespace:
			var nsModel tea.Model
			nsModel, cmd = m.namespace.Update(msg)
			if ns, ok := nsModel.(namespace.Model); ok {
				m.namespace = ns
			}
		default:
		}
	}

	return m, cmd
}

func (m *model) updatePodView(msg tea.Msg, cmd *tea.Cmd) {
	keypress := msg.(tea.KeyMsg).String()
	switch keypress {
	case "c":
		m.currentView = Context
	case "n":
		m.currentView = Namespace
	default:
		var podModel tea.Model
		var c tea.Cmd
		podModel, c = m.pod.Update(msg)
		*cmd = c
		if pod, ok := podModel.(pods.Model); ok {
			m.pod = pod
		}
	}
}

func (m *model) updateContextView(msg tea.Msg, cmd *tea.Cmd) {
	keypress := msg.(tea.KeyMsg).String()
	var ctxModel tea.Model
	var c tea.Cmd
	switch keypress {
	case "esc":
		m.currentView = Pod
	case "enter":
		ctxModel, c = m.context.Update(msg)
		*cmd = c
		if ctx, ok := ctxModel.(context.Model); ok {
			m.context = ctx
		}
	default:
		ctxModel, c = m.context.Update(msg)
		*cmd = c
		if ctx, ok := ctxModel.(context.Model); ok {
			m.context = ctx
		}
	}
}

func (m *model) updateNamespaceView(msg tea.Msg, cmd *tea.Cmd) {
	keypress := msg.(tea.KeyMsg).String()
	switch keypress {
	case "esc":
		m.currentView = Pod
	case "enter":
		m.watch.Stop()
		var nsModel tea.Model
		var c tea.Cmd
		nsModel, c = m.namespace.Update(msg)
		*cmd = c
		if ns, ok := nsModel.(namespace.Model); ok {
			m.namespace = ns
			m.pod.Namespace = m.namespace.SelectedNamespace
			pods.RefreshPods(&m.pod, true)
			m.watch = watchPods(m.namespace.SelectedNamespace)
			*cmd = tea.Batch(*cmd, watchPodEvents(m.watch.ResultChan()))
			m.currentView = Pod
		}

	default:
		var nsModel tea.Model
		var c tea.Cmd
		nsModel, c = m.namespace.Update(msg)
		*cmd = c
		if ns, ok := nsModel.(namespace.Model); ok {
			m.namespace = ns
		}
	}
}

func (m model) View() string {
	var b strings.Builder
	_, _ = fmt.Fprintf(&b, "CONTEXT: %s\n", m.context.SelectedContext.Name)
	_, _ = fmt.Fprintf(&b, "NAMESPACE: %s\n", m.pod.Namespace)
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
