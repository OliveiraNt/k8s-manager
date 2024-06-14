package main

import (
	ctx "context"
	"fmt"
	"github.com/OliveiraNt/k8s-manager/context"
	"github.com/OliveiraNt/k8s-manager/kubernetes"
	"github.com/OliveiraNt/k8s-manager/logs"
	"github.com/OliveiraNt/k8s-manager/namespace"
	"github.com/OliveiraNt/k8s-manager/pods"
	tea "github.com/charmbracelet/bubbletea"
	"k8s.io/apimachinery/pkg/watch"
	"log"
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
	context     context.Model
	namespace   namespace.Model
	pod         pods.Model
	watch       watch.Interface
	log         logs.Model
	currentView Views
	width       int
	height      int
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
		log.Fatal(err)
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
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		cmd = handleOtherMsgTypes(m, cmd, msg)
	case tea.KeyMsg:
		switch m.currentView {
		case Pod:
			m.updatePodView(msg, &cmd)
		case Context:
			m.updateContextView(msg, &cmd)
		case Namespace:
			m.updateNamespaceView(msg, &cmd)
		case Log:
			m.updateLogView(msg, &cmd)
		default:
		}
	case context.ChangeMsg:
		m.watch.Stop()
		var ctxModel tea.Model
		ctxModel, cmd = m.context.Update(msg)
		if ctxM, ok := ctxModel.(context.Model); ok {
			m.context = ctxM
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
		default:
		}
		cmd = tea.Batch(cmd, watchPodEvents(m.watch.ResultChan()))
	case logs.NewLogMsg:
		switch m.currentView {
		case Log:
			var logModel tea.Model
			logModel, cmd = m.log.Update(msg)
			if log, ok := logModel.(logs.Model); ok {
				m.log = log
			}
		default:
		}
	default:
		// Handle other message types
		cmd = handleOtherMsgTypes(m, cmd, msg)
	}

	return m, cmd
}

func handleOtherMsgTypes(m model, cmd tea.Cmd, msg tea.Msg) tea.Cmd {
	switch m.currentView {
	case Pod:
		var podModel tea.Model
		podModel, cmd = m.pod.Update(msg)
		if pod, ok := podModel.(pods.Model); ok {
			m.pod = pod
		}
	case Log:
		var logModel tea.Model
		logModel, cmd = m.log.Update(msg)
		if log, ok := logModel.(logs.Model); ok {
			m.log = log
		}
	case Context:
		var ctxModel tea.Model
		ctxModel, cmd = m.context.Update(msg)
		if ctxM, ok := ctxModel.(context.Model); ok {
			m.context = ctxM
		}
	case Namespace:
		var nsModel tea.Model
		nsModel, cmd = m.namespace.Update(msg)
		if ns, ok := nsModel.(namespace.Model); ok {
			m.namespace = ns
		}
	default:
	}
	return cmd
}

func (m *model) updatePodView(msg tea.Msg, cmd *tea.Cmd) {
	keypress := msg.(tea.KeyMsg).String()
	switch keypress {
	case "c":
		m.currentView = Context
	case "n":
		m.currentView = Namespace
	case "enter":
		m.log = logs.New(ctx.Background(), m.width, m.height)
		go func() {
			_ = kubernetes.GetPodLogs(
				m.log.Ctx,
				m.namespace.SelectedNamespace,
				m.pod.Pods.SelectedRow()[0],
				m.log.LogChan)
		}()
		*cmd = logs.WatchLogs(m.log)
		m.currentView = Log
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

func (m *model) updateLogView(msg tea.Msg, cmd *tea.Cmd) {
	keypress := msg.(tea.KeyMsg).String()
	switch keypress {
	case "esc":
		m.log.Stop()
		m.currentView = Pod
	default:
		var logModel tea.Model
		var c tea.Cmd
		logModel, c = m.log.Update(msg)
		*cmd = c
		if log, ok := logModel.(logs.Model); ok {
			m.log = log
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
		if ctxM, ok := ctxModel.(context.Model); ok {
			m.context = ctxM
		}
	default:
		ctxModel, c = m.context.Update(msg)
		*cmd = c
		if ctxM, ok := ctxModel.(context.Model); ok {
			m.context = ctxM
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
	case Log:
		return m.log.View()
	default:
		return s
	}
}
