package tui

import (
	ctx "context"
	"fmt"
	"github.com/OliveiraNt/k8s-manager/internal/kubernetes"
	"github.com/OliveiraNt/k8s-manager/internal/tui/context"
	"github.com/OliveiraNt/k8s-manager/internal/tui/deployments"
	"github.com/OliveiraNt/k8s-manager/internal/tui/logs"
	"github.com/OliveiraNt/k8s-manager/internal/tui/namespace"
	"github.com/OliveiraNt/k8s-manager/internal/tui/pods"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"k8s.io/apimachinery/pkg/watch"
	"strings"
)

type Views uint8

const (
	Context Views = iota
	Namespace
	Pod
	Deployment
	Log
)

var titleStyle = lipgloss.NewStyle().MarginLeft(2).Bold(true)

type Model struct {
	context         context.Model
	namespace       namespace.Model
	pod             pods.Model
	deployment      deployments.Model
	watch           watch.Interface
	deploymentWatch watch.Interface
	log             logs.Model
	currentView     Views
	width           int
	height          int
}

func NewModel() Model {
	_, ns, _ := kubernetes.GetCurrent()
	if ns == "" {
		ns = "default"
	}
	c, cancelFunc := ctx.WithCancel(ctx.Background())
	defer cancelFunc()
	w, err := kubernetes.WatchPods(c, ns)
	if err != nil {
		panic(err)
	}
	dw, err := kubernetes.WatchDeployments(c, ns)
	if err != nil {
		panic(err)
	}
	m := Model{
		currentView:     Pod,
		context:         context.New(),
		pod:             pods.New(ns),
		deployment:      deployments.New(ns),
		namespace:       namespace.New(ns),
		watch:           w,
		deploymentWatch: dw,
	}
	return m
}

func watchPodEvents(sub <-chan watch.Event) tea.Cmd {
	return func() tea.Msg {
		return pods.ChangeMsg(<-sub)
	}
}

func watchDeploymentEvents(sub <-chan watch.Event) tea.Cmd {
	return func() tea.Msg {
		return deployments.ChangeMsg(<-sub)
	}
}

func watchPods(ns string) watch.Interface {
	c, cancelFunc := ctx.WithCancel(ctx.Background())
	defer cancelFunc()
	w, err := kubernetes.WatchPods(c, ns)
	if err != nil {
		panic(err)
	}
	return w
}

func watchDeployments(ns string) watch.Interface {
	c, cancelFunc := ctx.WithCancel(ctx.Background())
	defer cancelFunc()
	w, err := kubernetes.WatchDeployments(c, ns)
	if err != nil {
		panic(err)
	}
	return w
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		watchPodEvents(m.watch.ResultChan()),
		watchDeploymentEvents(m.deploymentWatch.ResultChan()),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

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
		case Deployment:
			m.updateDeploymentView(msg, &cmd)
		case Log:
			m.updateLogView(msg, &cmd)
		default:
		}
	case context.ChangeMsg:
		m.watch.Stop()
		m.deploymentWatch.Stop()
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
			m.deployment.Namespace = ns
			pods.RefreshPods(&m.pod, true)
			deployments.RefreshDeployments(&m.deployment, true)
			m.watch = watchPods(ns)
			m.deploymentWatch = watchDeployments(ns)
			cmd = tea.Batch(
				cmd,
				watchPodEvents(m.watch.ResultChan()),
				watchDeploymentEvents(m.deploymentWatch.ResultChan()),
			)
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
	case deployments.ChangeMsg:
		switch m.currentView {
		case Deployment:
			var depModel tea.Model
			depModel, cmd = m.deployment.Update(msg)
			if dep, ok := depModel.(deployments.Model); ok {
				m.deployment = dep
			}
		default:
		}
		cmd = tea.Batch(cmd, watchDeploymentEvents(m.deploymentWatch.ResultChan()))
	case logs.NewLogMsg:
		switch m.currentView {
		case Log:
			var logModel tea.Model
			logModel, cmd = m.log.Update(msg)
			if logM, ok := logModel.(logs.Model); ok {
				m.log = logM
			}
		default:
		}
	default:
		// Handle other message types
		cmd = handleOtherMsgTypes(m, cmd, msg)
	}

	return m, cmd
}

func handleOtherMsgTypes(m Model, cmd tea.Cmd, msg tea.Msg) tea.Cmd {
	switch m.currentView {
	case Pod:
		var podModel tea.Model
		podModel, cmd = m.pod.Update(msg)
		if pod, ok := podModel.(pods.Model); ok {
			m.pod = pod
		}
	case Deployment:
		var depModel tea.Model
		depModel, cmd = m.deployment.Update(msg)
		if dep, ok := depModel.(deployments.Model); ok {
			m.deployment = dep
		}
	case Log:
		var logModel tea.Model
		logModel, cmd = m.log.Update(msg)
		if logM, ok := logModel.(logs.Model); ok {
			m.log = logM
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

func (m *Model) updatePodView(msg tea.Msg, cmd *tea.Cmd) {
	keypress := msg.(tea.KeyMsg).String()
	switch keypress {
	case "c":
		m.currentView = Context
	case "n":
		m.currentView = Namespace
	case "d":
		m.currentView = Deployment
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

func (m *Model) updateLogView(msg tea.Msg, cmd *tea.Cmd) {
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
		if logM, ok := logModel.(logs.Model); ok {
			m.log = logM
		}
	}
}

func (m *Model) updateContextView(msg tea.Msg, cmd *tea.Cmd) {
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

func (m *Model) updateNamespaceView(msg tea.Msg, cmd *tea.Cmd) {
	keypress := msg.(tea.KeyMsg).String()
	switch keypress {
	case "esc":
		m.currentView = Pod
	case "enter":
		m.watch.Stop()
		m.deploymentWatch.Stop()
		var nsModel tea.Model
		var c tea.Cmd
		nsModel, c = m.namespace.Update(msg)
		*cmd = c
		if ns, ok := nsModel.(namespace.Model); ok {
			m.namespace = ns
			m.pod.Namespace = m.namespace.SelectedNamespace
			m.deployment.Namespace = m.namespace.SelectedNamespace
			pods.RefreshPods(&m.pod, true)
			deployments.RefreshDeployments(&m.deployment, true)
			m.watch = watchPods(m.namespace.SelectedNamespace)
			m.deploymentWatch = watchDeployments(m.namespace.SelectedNamespace)
			*cmd = tea.Batch(
				*cmd,
				watchPodEvents(m.watch.ResultChan()),
				watchDeploymentEvents(m.deploymentWatch.ResultChan()),
			)
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

func (m *Model) updateDeploymentView(msg tea.Msg, cmd *tea.Cmd) {
	keypress := msg.(tea.KeyMsg).String()
	switch keypress {
	case "esc":
		m.currentView = Pod
	case "p":
		m.currentView = Pod
	case "c":
		m.currentView = Context
	case "n":
		m.currentView = Namespace
	default:
		var depModel tea.Model
		var c tea.Cmd
		depModel, c = m.deployment.Update(msg)
		*cmd = c
		if dep, ok := depModel.(deployments.Model); ok {
			m.deployment = dep
		}
	}
}

func (m Model) View() string {
	var b strings.Builder
	_, _ = fmt.Fprintf(&b, "CONTEXT: %s\n", m.context.SelectedContext.Name)
	_, _ = fmt.Fprintf(&b, "NAMESPACE: %s\n", m.pod.Namespace)
	s := titleStyle.Render(b.String()) + "\n\n"
	switch m.currentView {
	case Pod:
		return s + m.pod.View()
	case Deployment:
		return s + m.deployment.View()
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
