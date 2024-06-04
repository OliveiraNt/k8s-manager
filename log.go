package main

import (
	tea "github.com/charmbracelet/bubbletea"
)

type log struct {
	pod     string
	log     string
	logChan chan string
	sub     chan string
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
		}
	case logEvents:
		m.log.log += msg.event
		return m, waitForLogActivity(m.log.sub)
	}
	return m, nil
}

func logView(m model) string {
	return m.log.log
}
