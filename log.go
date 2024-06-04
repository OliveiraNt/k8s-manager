package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"strings"
)

type log struct {
	pod     string
	log     string
	logChan chan string
	sub     chan string
	buffer  []string
	index   int
}

type logEvents struct {
	event string
}

func waitForLogActivity(sub chan string) tea.Cmd {
	return func() tea.Msg {
		return logEvents{<-sub}
	}
}

func UpdateLog(m model, msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "esc":
			close(m.log.logChan)
			m.log = log{sub: make(chan string)}
			m.currentView = "pod"
			return m, nil
		case "up": // scroll up
			if m.log.index > 0 {
				m.log.index--
			}
		case "down": // scroll down
			if m.log.index < len(m.log.buffer)-1 {
				m.log.index++
			}
		}
	case logEvents:
		m.log.log += msg.event
		m.log.buffer = append(m.log.buffer, msg.event)
		return m, waitForLogActivity(m.log.sub)
	}
	return m, nil
}

func logView(m model) string {
	if len(m.log.buffer) == 0 {
		return ""
	}
	start := m.log.index
	end := start + 50
	if end > len(m.log.buffer) {
		end = len(m.log.buffer)
	}
	return strings.Join(m.log.buffer[start:end], "\n")
}
