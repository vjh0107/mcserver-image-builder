package docker

import (
	"fmt"
	"os"
	"path/filepath"
)

type ContextBuilder struct {
	name       string
	contextDir string
}

func NewContextBuilder(name string) *ContextBuilder {
	return &ContextBuilder{name: name}
}

func (cb *ContextBuilder) Prepare() (string, error) {
	dir, err := os.MkdirTemp("", fmt.Sprintf("mcserver-%s-*", cb.name))
	if err != nil {
		return "", fmt.Errorf("creating build context directory: %w", err)
	}
	cb.contextDir = dir

	subdirs := []string{"plugins", "config"}
	for _, sub := range subdirs {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
			return "", fmt.Errorf("creating %s directory: %w", sub, err)
		}
	}

	return dir, nil
}

func (cb *ContextBuilder) Cleanup() {
	if cb.contextDir != "" {
		os.RemoveAll(cb.contextDir)
	}
}
