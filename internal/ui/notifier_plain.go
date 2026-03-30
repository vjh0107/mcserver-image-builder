package ui

import (
	"io"
	"os"
)

type plainNotifier struct {
	names []string
}

func (n *plainNotifier) Start(step int) {
	if step < len(n.names) {
		Info("%s...", n.names[step])
	}
}

func (n *plainNotifier) Done(step int, detail string) {
	if step < len(n.names) {
		if detail != "" {
			Done("%s %s", n.names[step], detail)
		} else {
			Done("%s", n.names[step])
		}
	}
}

func (n *plainNotifier) Error(step int, err error) {
	if step < len(n.names) {
		Warn("%s: %s", n.names[step], err)
	}
}

func (n *plainNotifier) ArtifactStart(name string) {
	Info("  %s", name)
}

func (n *plainNotifier) ArtifactProgress(string, int64, int64) {}

func (n *plainNotifier) ArtifactDone(name, detail string) {
	if detail != "" {
		Done("  %s %s", name, detail)
	} else {
		Done("  %s", name)
	}
}

func (n *plainNotifier) Elapsed(string) {}

func (n *plainNotifier) LogWriter() io.Writer {
	return os.Stderr
}
