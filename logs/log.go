package logs

import (
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
	logsChan  chan<- string
}

type ChangeMsg string

func New(pod string, namespace string) Model {
	logsChan := make(chan string)
	go watchLogs(pod, namespace, logsChan)
	return Model{
		pod:       pod,
		namespace: namespace,
		ready:     false,
		buffer:    []string{},
		logsChan:  logsChan,
	}
}

func watchLogs(pod string, namespace string, logsChan chan<- string) {
	err := kubernetes.GetPodLogs(namespace, pod, logsChan)
	if err != nil {
		return
	}
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
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			close(m.logsChan)
			m.buffer = []string{}
		}
	case tea.WindowSizeMsg:
		if !m.ready {
			m.logs = viewport.New(msg.Width, msg.Height)
			m.ready = true
		} else {
			m.logs.Width = msg.Width
			m.logs.Height = msg.Height
			cmd = viewport.Sync(m.logs)
		}
	case ChangeMsg:
		scrollDown := false
		if m.logs.AtBottom() {
			scrollDown = true
		}

		m.buffer = append(m.buffer, "")

		m.logs.SetContent(strings.Join(m.buffer, "\n"))
		if scrollDown {
			m.logs.GotoBottom()
		}

	}

	m.logs, cmd = m.logs.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	return m.logs.View()
}

//type log struct {
//	pod     string
//	log     string
//	logChan chan string
//	sub     chan string
//	buffer  []string
//	index   int
//}
//
//type logEvents struct {
//	event string
//}
//
//func waitForLogActivity(sub chan string) tea.Cmd {
//	return func() tea.Msg {
//		return logEvents{<-sub}
//	}
//}
//
//func UpdateLog(m model, msg tea.Msg) (tea.Model, tea.Cmd) {
//	switch msg := msg.(type) {
//	case tea.KeyMsg:
//		switch keypress := msg.String(); keypress {
//		case "q", "ctrl+c":
//			return m, tea.Quit
//		case "esc":
//			close(m.log.logChan)
//			m.log = log{sub: make(chan string)}
//			m.currentView = "pod"
//			return m, nil
//		case "up": // scroll up
//			if m.log.index > 0 {
//				m.log.index--
//			}
//		case "down": // scroll down
//			if m.log.index < len(m.log.buffer)-1 {
//				m.log.index++
//			}
//		}
//	case logEvents:
//		m.log.log += msg.event
//		m.log.buffer = append(m.log.buffer, msg.event)
//		return m, waitForLogActivity(m.log.sub)
//	}
//	return m, nil
//}
//
//func logView(m model) string {
//	if len(m.log.buffer) == 0 {
//		return ""
//	}
//	start := m.log.index
//	end := start + 50
//	if end > len(m.log.buffer) {
//		end = len(m.log.buffer)
//	}
//	return strings.Join(m.log.buffer[start:end], "\n")
//}
