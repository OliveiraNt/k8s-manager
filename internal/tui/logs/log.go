package logs

import (
	"context"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"strings"
)

type Model struct {
	logs    viewport.Model
	buffer  []string
	Ctx     context.Context
	cancel  context.CancelFunc
	LogChan chan string
}

type NewLogMsg string

func New(c context.Context, width int, height int) Model {
	ctx, cancel := context.WithCancel(c)
	logChan := make(chan string)
	vp := viewport.New(width, height)
	m := Model{
		buffer:  []string{},
		Ctx:     ctx,
		cancel:  cancel,
		LogChan: logChan,
		logs:    vp,
	}
	return m
}

func WatchLogs(m Model) tea.Cmd {
	return func() tea.Msg {
		select {
		case log := <-m.LogChan:
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
		gob := false
		if m.logs.AtBottom() {
			gob = true
		}
		m.buffer = append(m.buffer, string(msg))
		m.logs.SetContent(strings.Join(m.buffer, ""))
		cmds = append(cmds, WatchLogs(m))
		if gob {
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
