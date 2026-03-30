package ui

type StepStatus int

const (
	StepPending StepStatus = iota
	StepRunning
	StepDone
	StepError
)

type BuildStep struct {
	Name   string
	Status StepStatus
	Detail string
}
