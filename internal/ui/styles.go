package ui

import "github.com/charmbracelet/lipgloss"

var (
	styleDone    = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	styleRunning = lipgloss.NewStyle().Foreground(lipgloss.Color("5"))
	styleError   = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	styleHeader  = lipgloss.NewStyle().Bold(true)
	styleDim     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)
