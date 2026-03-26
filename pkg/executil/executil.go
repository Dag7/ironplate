// Package executil provides shell command execution helpers.
package executil

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// CommandResult holds the output of a command execution.
type CommandResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// Run executes a command and returns the result.
func Run(name string, args ...string) (*CommandResult, error) {
	cmd := exec.Command(name, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	result := &CommandResult{
		Stdout:   strings.TrimSpace(stdout.String()),
		Stderr:   strings.TrimSpace(stderr.String()),
		ExitCode: cmd.ProcessState.ExitCode(),
	}

	if err != nil {
		return result, fmt.Errorf("command %s %s failed: %w", name, strings.Join(args, " "), err)
	}

	return result, nil
}

// CommandExists checks if a command is available in PATH.
func CommandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// GetVersion runs a command with --version and returns the output.
func GetVersion(name string) (string, error) {
	result, err := Run(name, "--version")
	if err != nil {
		return "", err
	}
	return result.Stdout, nil
}

// RunInDir executes a command in a specific directory.
func RunInDir(dir, name string, args ...string) (*CommandResult, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	result := &CommandResult{
		Stdout:   strings.TrimSpace(stdout.String()),
		Stderr:   strings.TrimSpace(stderr.String()),
		ExitCode: cmd.ProcessState.ExitCode(),
	}

	if err != nil {
		return result, fmt.Errorf("command %s %s in %s failed: %w", name, strings.Join(args, " "), dir, err)
	}

	return result, nil
}
