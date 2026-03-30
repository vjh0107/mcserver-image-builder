package ui

import (
	"fmt"
	"os"
	"strings"
)

func Step(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "==> %s\n", fmt.Sprintf(format, args...))
}

func Info(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "    %s\n", fmt.Sprintf(format, args...))
}

func Done(format string, args ...any) {
	fmt.Fprintf(os.Stderr, " ✓  %s\n", fmt.Sprintf(format, args...))
}

func Warn(format string, args ...any) {
	fmt.Fprintf(os.Stderr, " ⚠  %s\n", fmt.Sprintf(format, args...))
}

func List(label string, items []string) {
	fmt.Fprintf(os.Stderr, "==> %s: %s\n", label, strings.Join(items, ", "))
}
