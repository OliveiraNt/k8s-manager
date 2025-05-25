package tui

import (
	ctx "context"
	"fmt"
	"github.com/OliveiraNt/k8s-manager/internal/errors"
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
	error           *errors.AppError
	showError       bool
}

func NewModel() Model {
	_, ns, _ := kubernetes.GetCurrent()
	if ns == "" {
		ns = "default"
	}
	c, cancelFunc := ctx.WithCancel(ctx.Background())
	defer cancelFunc()

	m := Model{
		currentView: Pod,
		context:     context.New(),
		pod:         pods.New(ns),
		deployment:  deployments.New(ns),
		namespace:   namespace.New(ns),
		error:       nil,
		showError:   false,
	}

	w, err := kubernetes.WatchPods(c, ns)
	if err != nil {
		m.handleError(errors.New("Failed to watch pods", errors.Error, err))
	} else {
		m.watch = w
	}

	dw, err := kubernetes.WatchDeployments(c, ns)
	if err != nil {
		m.handleError(errors.New("Failed to watch deployments", errors.Error, err))
	} else {
		m.deploymentWatch = dw
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

func watchPods(ns string, m Model) watch.Interface {
	c, cancelFunc := ctx.WithCancel(ctx.Background())
	defer cancelFunc()
	w, err := kubernetes.WatchPods(c, ns)
	if err != nil {
		// Since we can't modify m, we'll just log the error
		errors.New("Failed to watch pods", errors.Error, err)
		return nil
	}
	return w
}

func watchDeployments(ns string, m Model) watch.Interface {
	c, cancelFunc := ctx.WithCancel(ctx.Background())
	defer cancelFunc()
	w, err := kubernetes.WatchDeployments(c, ns)
	if err != nil {
		// Since we can't modify m, we'll just log the error
		errors.New("Failed to watch deployments", errors.Error, err)
		return nil
	}
	return w
}

func (m Model) Init() tea.Cmd {
	switch m.currentView {
	case Pod:
		return watchPodEvents(m.watch.ResultChan())
	case Deployment:
		return watchDeploymentEvents(m.deploymentWatch.ResultChan())
	default:
		return nil
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	var cmd tea.Cmd

	// Handle quit keys regardless of the message type
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keypress := keyMsg.String(); keypress == "q" || keypress == "ctrl+c" {
			return m, tea.Quit
		}

		// If there's an error being displayed, clear it on any key press
		if m.showError && m.error != nil {
			m.clearError()
			return m, cmd
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
			m.watch = watchPods(ns, m)
			m.deploymentWatch = watchDeployments(ns, m)
			if m.watch != nil && m.deploymentWatch != nil {
				cmd = tea.Batch(
					cmd,
					watchPodEvents(m.watch.ResultChan()),
				)
			}
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
			cmd = tea.Batch(cmd, watchPodEvents(m.watch.ResultChan()))
		default:
		}
	case deployments.ChangeMsg:
		switch m.currentView {
		case Deployment:
			var depModel tea.Model
			depModel, cmd = m.deployment.Update(msg)
			if dep, ok := depModel.(deployments.Model); ok {
				m.deployment = dep
			}
			cmd = tea.Batch(cmd, watchDeploymentEvents(m.deploymentWatch.ResultChan()))
		default:
		}
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
		*cmd = tea.Batch(*cmd, watchDeploymentEvents(m.deploymentWatch.ResultChan()))
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
		*cmd = tea.Batch(*cmd, watchPodEvents(m.watch.ResultChan()))
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
		*cmd = tea.Batch(*cmd, watchPodEvents(m.watch.ResultChan()))
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
		*cmd = tea.Batch(*cmd, watchPodEvents(m.watch.ResultChan()))
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
			m.watch = watchPods(m.namespace.SelectedNamespace, *m)
			m.deploymentWatch = watchDeployments(m.namespace.SelectedNamespace, *m)
			if m.watch != nil && m.deploymentWatch != nil {
				*cmd = tea.Batch(
					*cmd,
					watchPodEvents(m.watch.ResultChan()),
				)
			}
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
		*cmd = tea.Batch(*cmd, watchPodEvents(m.watch.ResultChan()))
	case "p":
		m.currentView = Pod
		*cmd = tea.Batch(*cmd, watchPodEvents(m.watch.ResultChan()))
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

// handleError sets the error state and shows the error
func (m *Model) handleError(err *errors.AppError) {
	m.error = err
	m.showError = true

	// If the error is fatal, we should exit the application
	if err.Level == errors.Fatal {
		panic(err) // Still panic for fatal errors
	}
}

// clearError clears the error state
func (m *Model) clearError() {
	m.error = nil
	m.showError = false
}

// errorView renders the error message
func (m Model) errorView() string {
	if m.error == nil {
		return ""
	}

	var style lipgloss.Style
	switch m.error.Level {
	case errors.Info:
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("blue"))
	case errors.Warning:
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("yellow"))
	case errors.Error:
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("red"))
	case errors.Fatal:
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("red")).Bold(true)
	}

	return "\n" + style.Render(fmt.Sprintf("Error: %s", m.error.String())) + "\n\nPress any key to continue..."
}

func (m Model) View() string {
	// If there's an error and we're showing it, display the error view
	if m.showError && m.error != nil {
		return m.errorView()
	}

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
