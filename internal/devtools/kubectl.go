// Package devtools provides developer tooling for Kubernetes cluster management.
package devtools

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// KubectlResult holds the output of a kubectl command.
type KubectlResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// Kubectl runs a kubectl command and returns its output.
func Kubectl(args ...string) (*KubectlResult, error) {
	cmd := exec.Command("kubectl", args...)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	result := &KubectlResult{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}
	if cmd.ProcessState != nil {
		result.ExitCode = cmd.ProcessState.ExitCode()
	}
	if err != nil {
		return result, fmt.Errorf("kubectl %s: %s", strings.Join(args, " "), strings.TrimSpace(stderr.String()))
	}
	return result, nil
}

// KubectlJSON runs kubectl with -o json and unmarshals the result.
func KubectlJSON(out interface{}, args ...string) error {
	args = append(args, "-o", "json")
	result, err := Kubectl(args...)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(result.Stdout), out)
}

// KubectlInteractive runs kubectl with inherited stdio for interactive commands.
func KubectlInteractive(args ...string) error {
	cmd := exec.Command("kubectl", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
