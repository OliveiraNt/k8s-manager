package logs

import (
	"context"
	"github.com/OliveiraNt/k8s-manager/kubernetes"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"strings"
)

type Model struct {
	logs      viewport.Model
	buffer    []string
	pod       string
	namespace string
	ready     bool
	ctx       context.Context
	cancel    context.CancelFunc
	logChan   chan string
}

type NewLogMsg string

func New(pod string, namespace string, width int, height int) Model {
	ctx, cancel := context.WithCancel(context.Background())
	logChan := make(chan string)
	vp := viewport.New(width, height)
	m := Model{
		pod:       pod,
		namespace: namespace,
		ready:     false,
		buffer:    []string{},
		ctx:       ctx,
		cancel:    cancel,
		logChan:   logChan,
		logs:      vp,
	}
	go kubernetes.GetPodLogs(ctx, namespace, pod, logChan)
	return m
}

func WatchLogs(m Model) tea.Cmd {
	return func() tea.Msg {
		select {
		case log := <-m.logChan:
			return NewLogMsg(log)
		}
	}
}

func (m Model) Stop() {
	m.cancel()
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.logs.Width = msg.Width
		m.logs.Height = msg.Height
		m.logs.SetContent(strings.Join(m.buffer, ""))
	case NewLogMsg:
		m.buffer = append(m.buffer, string(msg))
		m.logs.SetContent(strings.Join(m.buffer, ""))
		cmds = append(cmds, WatchLogs(m))
	}

	m.logs, cmd = m.logs.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	return m.logs.View()
}
