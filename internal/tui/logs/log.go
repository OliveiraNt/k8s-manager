// Package logs provides functionality for displaying and managing logs in the TUI.
// It includes memory optimization to prevent excessive memory usage during long-running sessions
// by limiting the number of log entries stored in the buffer.
// The package also includes scroll speed optimization to provide a better user experience
// when navigating through logs with the mouse wheel.
package logs

import (
	"context"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"strings"
)

type Model struct {
	logs          viewport.Model
	buffer        []string
	maxBufferSize int
	Ctx           context.Context
	cancel        context.CancelFunc
	LogChan       chan string
}

type NewLogMsg string

func New(c context.Context, width int, height int) Model {
	ctx, cancel := context.WithCancel(c)
	logChan := make(chan string)
	vp := viewport.New(width, height)
	vp.MouseWheelDelta = 50 // Increase scroll speed for better user experience
	m := Model{
		buffer:        []string{},
		maxBufferSize: 1000, // Limit buffer to 1000 lines to prevent memory issues
		Ctx:           ctx,
		cancel:        cancel,
		LogChan:       logChan,
		logs:          vp,
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

		// Add the new log entry to the buffer
		m.buffer = append(m.buffer, string(msg))

		// If buffer exceeds maximum size, remove oldest entries
		if len(m.buffer) > m.maxBufferSize {
			// Calculate how many entries to remove
			removeCount := len(m.buffer) - m.maxBufferSize
			// Remove oldest entries (from the beginning of the slice)
			m.buffer = m.buffer[removeCount:]
		}

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
