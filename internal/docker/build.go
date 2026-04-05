package docker

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

const DefaultBaseImage = "eclipse-temurin:21-jre-alpine"

type BuildOptions struct {
	ContextDir string
	Tag        string
	NoCache    bool
	Output     io.Writer
}

func Build(opts BuildOptions) error {
	args := []string{"build", "-t", opts.Tag}
	if opts.NoCache {
		args = append(args, "--no-cache")
	}
	args = append(args, opts.ContextDir)

	if opts.Output != nil {
		return runDockerWithOutput(args, opts.Output)
	}
	return runDockerQuiet(args)
}

func Push(tag string, output io.Writer) error {
	args := []string{"push", tag}
	if output != nil {
		return runDockerWithOutput(args, output)
	}
	return runDockerQuiet(args)
}

func EnsureAvailable() error {
	if _, err := exec.LookPath("docker"); err != nil {
		return fmt.Errorf("docker is not installed or not in PATH")
	}
	return nil
}

func runDockerWithOutput(args []string, output io.Writer) error {
	var errBuf bytes.Buffer
	cmd := exec.Command("docker", args...)
	cmd.Stdout = output
	cmd.Stderr = io.MultiWriter(output, &errBuf)

	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(errBuf.String())
		if msg != "" {
			return fmt.Errorf("docker %s: %s", args[0], msg)
		}
		return fmt.Errorf("docker %s: %w", args[0], err)
	}
	return nil
}

func runDockerQuiet(args []string) error {
	var buf bytes.Buffer
	cmd := exec.Command("docker", args...)
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	if err := cmd.Run(); err != nil {
		os.Stderr.Write(buf.Bytes())
		return fmt.Errorf("docker %s: %w", args[0], err)
	}
	return nil
}
