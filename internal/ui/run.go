package ui

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-isatty"
)

type tuiWriter struct {
	p      *tea.Program
	server int
	buf    []byte
}

func (w *tuiWriter) Write(p []byte) (int, error) {
	w.buf = append(w.buf, p...)
	for {
		idx := bytes.IndexByte(w.buf, '\n')
		if idx < 0 {
			break
		}
		line := strings.TrimRight(string(w.buf[:idx]), "\r")
		w.buf = w.buf[idx+1:]
		if line != "" {
			w.p.Send(logLineMsg{server: w.server, line: line})
		}
	}
	return len(p), nil
}

func isTTY() bool {
	if os.Getenv("NO_TUI") != "" {
		return false
	}
	return isatty.IsTerminal(os.Stderr.Fd()) || isatty.IsCygwinTerminal(os.Stderr.Fd())
}

func RunBuild(label string, stepNames []string, buildFn BuildFunc) error {
	return RunParallelBuild([]BuildEntry{{Label: label, StepNames: stepNames, BuildFn: buildFn}})
}

func RunParallelBuild(entries []BuildEntry) error {
	if !isTTY() {
		return runPlain(entries)
	}

	m := newModel(entries)
	p := tea.NewProgram(m)

	var wg sync.WaitGroup
	for i, entry := range entries {
		wg.Add(1)
		n := &tuiNotifier{p: p, server: i}
		go func(e BuildEntry, notifier Notifier) {
			defer wg.Done()
			e.BuildFn(notifier)
		}(entry, n)
	}

	go func() {
		wg.Wait()
		p.Send(allDoneMsg{})
	}()

	if _, err := p.Run(); err != nil {
		return err
	}
	return m.firstError()
}

func runPlain(entries []BuildEntry) error {
	for _, entry := range entries {
		Step("%s", entry.Label)
		n := &plainNotifier{names: entry.StepNames}
		if err := entry.BuildFn(n); err != nil {
			return err
		}
		Done("%s", entry.Label)
		fmt.Fprintln(os.Stderr)
	}
	return nil
}
