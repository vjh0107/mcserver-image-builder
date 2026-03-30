package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type artifactStartMsg struct {
	server int
	name   string
}
type artifactProgressMsg struct {
	server   int
	name     string
	received int64
	total    int64
}
type artifactDoneMsg struct {
	server int
	name   string
	detail string
}
type stepStartMsg struct{ server, step int }
type stepDoneMsg struct {
	server, step int
	detail       string
}
type stepErrorMsg struct {
	server, step int
	err          error
}
type logLineMsg struct {
	server int
	line   string
}
type serverElapsedMsg struct {
	server  int
	elapsed string
}
type allDoneMsg struct{}

type BuildFunc func(n Notifier) error

type BuildEntry struct {
	Label     string
	StepNames []string
	BuildFn   BuildFunc
}

const maxLogLines = 5

type artifactState struct {
	name     string
	detail   string
	done     bool
	received int64
	total    int64
}

type serverState struct {
	label     string
	steps     []BuildStep
	logTail   []string
	artifacts []artifactState
	elapsed   string
	err       error
}

type Model struct {
	servers  []serverState
	spinner  spinner.Model
	progress progress.Model
}

func newModel(entries []BuildEntry) *Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = styleRunning

	p := progress.New(progress.WithScaledGradient("#5A56E0", "#EE6FF8"))
	p.Width = 20

	servers := make([]serverState, len(entries))
	for i, e := range entries {
		steps := make([]BuildStep, len(e.StepNames))
		for j, name := range e.StepNames {
			steps[j] = BuildStep{Name: name, Status: StepPending}
		}
		servers[i] = serverState{label: e.Label, steps: steps}
	}

	return &Model{servers: servers, spinner: s, progress: p}
}

func (m *Model) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

	case artifactStartMsg:
		srv := &m.servers[msg.server]
		srv.artifacts = append(srv.artifacts, artifactState{name: msg.name})

	case artifactProgressMsg:
		srv := &m.servers[msg.server]
		for i := range srv.artifacts {
			if srv.artifacts[i].name == msg.name && !srv.artifacts[i].done {
				srv.artifacts[i].received = msg.received
				srv.artifacts[i].total = msg.total
				break
			}
		}

	case artifactDoneMsg:
		srv := &m.servers[msg.server]
		for i := range srv.artifacts {
			if srv.artifacts[i].name == msg.name && !srv.artifacts[i].done {
				srv.artifacts[i].done = true
				srv.artifacts[i].detail = msg.detail
				break
			}
		}

	case stepStartMsg:
		srv := &m.servers[msg.server]
		if msg.step < len(srv.steps) {
			srv.steps[msg.step].Status = StepRunning
		}

	case stepDoneMsg:
		srv := &m.servers[msg.server]
		if msg.step < len(srv.steps) {
			srv.steps[msg.step].Status = StepDone
			srv.steps[msg.step].Detail = msg.detail
		}
		srv.logTail = nil
		srv.artifacts = nil

	case stepErrorMsg:
		srv := &m.servers[msg.server]
		if msg.step < len(srv.steps) {
			srv.steps[msg.step].Status = StepError
			srv.steps[msg.step].Detail = msg.err.Error()
		}
		srv.err = msg.err

	case logLineMsg:
		srv := &m.servers[msg.server]
		srv.logTail = append(srv.logTail, msg.line)
		if len(srv.logTail) > maxLogLines {
			srv.logTail = srv.logTail[len(srv.logTail)-maxLogLines:]
		}

	case serverElapsedMsg:
		srv := &m.servers[msg.server]
		srv.elapsed = msg.elapsed

	case allDoneMsg:
		for i := range m.servers {
			for j := range m.servers[i].steps {
				if m.servers[i].steps[j].Status == StepRunning {
					m.servers[i].steps[j].Status = StepDone
				}
			}
			m.servers[i].logTail = nil
			m.servers[i].artifacts = nil
		}
		return m, tea.Quit

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *Model) View() string {
	var b strings.Builder

	for i, srv := range m.servers {
		if i > 0 {
			b.WriteString("\n")
		}

		labelStyle := styleHeader
		if srv.err != nil {
			labelStyle = styleError
		}
		labelLine := "  " + labelStyle.Render(srv.label)
		if srv.elapsed != "" {
			labelLine += " " + styleDim.Render(srv.elapsed)
		}
		b.WriteString(labelLine + "\n")

		for _, step := range srv.steps {
			var icon string
			var style = styleDim

			switch step.Status {
			case StepDone:
				icon = styleDone.Render("✓")
				style = styleDone
			case StepRunning:
				icon = m.spinner.View()
				style = styleRunning
			case StepError:
				icon = styleError.Render("✗")
				style = styleError
			default:
				icon = styleDim.Render("○")
			}

			line := fmt.Sprintf("    %s %s", icon, style.Render(step.Name))

			if step.Status == StepRunning && len(srv.artifacts) > 0 {
				done := 0
				for _, a := range srv.artifacts {
					if a.done {
						done++
					}
				}
				line += " " + styleDim.Render(fmt.Sprintf("%d/%d", done, len(srv.artifacts)))
			} else if step.Detail != "" {
				line += " " + renderDetail(step.Detail, style)
			}
			b.WriteString(line + "\n")

			if step.Status == StepRunning {
				if len(srv.artifacts) > 0 {
					lines := 0
					visible := srv.artifacts
					if len(visible) > maxLogLines {
						visible = visible[len(visible)-maxLogLines:]
					}
					for _, a := range visible {
						var bar string
						var label string
						if a.done {
							bar = styleDone.Render("✓")
							label = a.name
							if a.detail != "" {
								label += " " + a.detail
							}
						} else if a.total > 0 {
							pct := float64(a.received) / float64(a.total)
							bar = m.progress.ViewAs(pct)
							label = a.name
						} else {
							bar = m.spinner.View() + strings.Repeat(" ", m.progress.Width-1)
							label = a.name
						}
						b.WriteString(fmt.Sprintf("      %s %s\n", bar, styleDim.Render(label)))
						lines++
					}
					for j := lines; j < maxLogLines; j++ {
						b.WriteString("\n")
					}
				} else if len(srv.logTail) > 0 {
					for _, l := range srv.logTail {
						b.WriteString("      " + styleDim.Render(l) + "\n")
					}
					for j := len(srv.logTail); j < maxLogLines; j++ {
						b.WriteString("\n")
					}
				}
			}
		}

		if srv.err != nil {
			b.WriteString("    " + styleError.Render(fmt.Sprintf("Error: %s", srv.err)) + "\n")
		}
	}

	b.WriteString("\n")
	return b.String()
}

func renderDetail(detail string, stepStyle lipgloss.Style) string {
	var result strings.Builder
	for detail != "" {
		open := strings.Index(detail, "(")
		end := strings.Index(detail, ")")
		if open >= 0 && end > open {
			if open > 0 {
				result.WriteString(styleDim.Render(detail[:open]))
			}
			result.WriteString(stepStyle.Render(detail[open : end+1]))
			detail = detail[end+1:]
		} else {
			result.WriteString(styleDim.Render(detail))
			break
		}
	}
	return result.String()
}

func (m *Model) firstError() error {
	for _, srv := range m.servers {
		if srv.err != nil {
			return srv.err
		}
	}
	return nil
}
