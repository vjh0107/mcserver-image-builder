package ui

import "io"

type Notifier interface {
	Start(step int)
	Done(step int, detail string)
	Error(step int, err error)
	ArtifactStart(name string)
	ArtifactProgress(name string, received, total int64)
	ArtifactDone(name, detail string)
	Elapsed(elapsed string)
	LogWriter() io.Writer
}
