package ui

import (
	"io"

	tea "github.com/charmbracelet/bubbletea"
)

type tuiNotifier struct {
	p      *tea.Program
	server int
}

func (n *tuiNotifier) Start(step int) {
	n.p.Send(stepStartMsg{server: n.server, step: step})
}

func (n *tuiNotifier) Done(step int, detail string) {
	n.p.Send(stepDoneMsg{server: n.server, step: step, detail: detail})
}

func (n *tuiNotifier) Error(step int, err error) {
	n.p.Send(stepErrorMsg{server: n.server, step: step, err: err})
}

func (n *tuiNotifier) ArtifactStart(name string) {
	n.p.Send(artifactStartMsg{server: n.server, name: name})
}

func (n *tuiNotifier) ArtifactProgress(name string, received, total int64) {
	n.p.Send(artifactProgressMsg{server: n.server, name: name, received: received, total: total})
}

func (n *tuiNotifier) ArtifactDone(name, detail string) {
	n.p.Send(artifactDoneMsg{server: n.server, name: name, detail: detail})
}

func (n *tuiNotifier) Elapsed(elapsed string) {
	n.p.Send(serverElapsedMsg{server: n.server, elapsed: elapsed})
}

func (n *tuiNotifier) LogWriter() io.Writer {
	return &tuiWriter{p: n.p, server: n.server}
}
