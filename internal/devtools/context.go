package devtools

import (
	"fmt"
	"strings"
)

// KubeContext represents a kubectl context entry.
type KubeContext struct {
	Name      string
	Cluster   string
	Namespace string
	Current   bool
}

// GetCurrentContext returns the current kubectl context name.
func GetCurrentContext() (string, error) {
	result, err := Kubectl("config", "current-context")
	if err != nil {
		return "", fmt.Errorf("no current context: %w", err)
	}
	return strings.TrimSpace(result.Stdout), nil
}

// ListContexts returns all available kubectl contexts.
func ListContexts() ([]KubeContext, error) {
	result, err := Kubectl("config", "get-contexts", "--no-headers")
	if err != nil {
		return nil, err
	}

	var contexts []KubeContext
	for _, line := range strings.Split(strings.TrimSpace(result.Stdout), "\n") {
		if line == "" {
			continue
		}
		ctx := parseContextLine(line)
		if ctx.Name != "" {
			contexts = append(contexts, ctx)
		}
	}
	return contexts, nil
}

// SwitchContext switches to the given kubectl context.
func SwitchContext(name string) error {
	_, err := Kubectl("config", "use-context", name)
	return err
}

// SetNamespace sets the default namespace for the current context.
func SetNamespace(namespace string) error {
	_, err := Kubectl("config", "set-context", "--current", "--namespace="+namespace)
	return err
}

// ContextExists checks if a context exists.
func ContextExists(name string) bool {
	contexts, err := ListContexts()
	if err != nil {
		return false
	}
	for _, ctx := range contexts {
		if ctx.Name == name {
			return true
		}
	}
	return false
}

// GetLocalContextName returns the k3d context name for a project.
func GetLocalContextName(projectName string) string {
	return "k3d-" + projectName + "-cluster"
}

func parseContextLine(line string) KubeContext {
	current := strings.HasPrefix(line, "*")
	if current {
		line = line[1:]
	}
	line = strings.TrimSpace(line)

	fields := strings.Fields(line)
	if len(fields) < 2 {
		return KubeContext{}
	}

	ctx := KubeContext{
		Name:    fields[0],
		Cluster: fields[1],
		Current: current,
	}
	if len(fields) >= 4 {
		ctx.Namespace = fields[3]
	}
	return ctx
}
