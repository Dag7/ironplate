package tiltmgr

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// TiltResource represents a running Tilt resource's status.
type TiltResource struct {
	Name         string
	Status       string // ok, pending, error, unknown
	Type         string
	UpdateStatus string
}

// IsRunning checks if Tilt is currently running by querying its session.
func IsRunning() bool {
	cmd := exec.Command("tilt", "get", "session")
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run() == nil
}

// Up starts Tilt with the given profile's resources.
func Up(profile *Profile, noBrowser bool) error {
	args := buildTiltArgs(profile, noBrowser)
	cmd := exec.Command("tilt", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Down stops the running Tilt instance.
func Down() error {
	cmd := exec.Command("tilt", "down")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// GetStatus returns the status of all running Tilt resources.
func GetStatus() ([]TiltResource, error) {
	cmd := exec.Command("tilt", "get", "uiresources", "-o", "json")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("tilt not running or not accessible: %w", err)
	}

	var data struct {
		Items []struct {
			Metadata struct {
				Name   string            `json:"name"`
				Labels map[string]string `json:"labels"`
			} `json:"metadata"`
			Status struct {
				RuntimeStatus string `json:"runtimeStatus"`
				UpdateStatus  string `json:"updateStatus"`
			} `json:"status"`
		} `json:"items"`
	}

	if err := json.Unmarshal(out, &data); err != nil {
		return nil, fmt.Errorf("parse tilt status: %w", err)
	}

	var resources []TiltResource
	for _, item := range data.Items {
		resources = append(resources, TiltResource{
			Name:         item.Metadata.Name,
			Status:       normalizeStatus(item.Status.RuntimeStatus),
			Type:         item.Metadata.Labels["tilt.dev/resource-type"],
			UpdateStatus: item.Status.UpdateStatus,
		})
	}
	return resources, nil
}

func buildTiltArgs(profile *Profile, noBrowser bool) []string {
	args := []string{"up"}

	allResources := make([]string, 0, len(profile.Services)+len(profile.Infra))
	allResources = append(allResources, profile.Infra...)
	allResources = append(allResources, profile.Services...)

	for _, r := range allResources {
		args = append(args, "--only", r)
	}

	if noBrowser {
		args = append(args, "--no-browser")
	}

	return args
}

// Enable enables disabled Tilt resources.
func Enable(resources []string) error {
	for _, r := range resources {
		cmd := exec.Command("tilt", "enable", r)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("enable %s: %w", r, err)
		}
	}
	return nil
}

// Disable disables Tilt resources.
func Disable(resources []string) error {
	for _, r := range resources {
		cmd := exec.Command("tilt", "disable", r)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("disable %s: %w", r, err)
		}
	}
	return nil
}

// Retry triggers a rebuild of Tilt resources.
func Retry(resources []string) error {
	for _, r := range resources {
		cmd := exec.Command("tilt", "trigger", r)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("retry %s: %w", r, err)
		}
	}
	return nil
}

// GetDisabledResources returns resources that are currently disabled.
func GetDisabledResources() ([]string, error) {
	resources, err := GetStatus()
	if err != nil {
		return nil, err
	}
	var disabled []string
	for _, r := range resources {
		if r.Status == "disabled" {
			disabled = append(disabled, r.Name)
		}
	}
	return disabled, nil
}

// GetErroredResources returns resources that are in error state.
func GetErroredResources() ([]string, error) {
	resources, err := GetStatus()
	if err != nil {
		return nil, err
	}
	var errored []string
	for _, r := range resources {
		if r.Status == "error" {
			errored = append(errored, r.Name)
		}
	}
	return errored, nil
}

// StatusSummary holds counts of resources by status.
type StatusSummary struct {
	Total    int
	Running  int
	Building int
	Pending  int
	Error    int
	Disabled int
}

// GetStatusSummary returns a summary of resource counts by status.
func GetStatusSummary() (*StatusSummary, error) {
	resources, err := GetStatus()
	if err != nil {
		return nil, err
	}
	s := &StatusSummary{Total: len(resources)}
	for _, r := range resources {
		switch r.Status {
		case "ok":
			s.Running++
		case "pending":
			s.Pending++
		case "error":
			s.Error++
		case "disabled":
			s.Disabled++
		default:
			s.Pending++
		}
	}
	return s, nil
}

func normalizeStatus(s string) string {
	s = strings.ToLower(s)
	switch s {
	case "ok", "running":
		return "ok"
	case "pending", "building", "in_progress":
		return "pending"
	case "error", "failed", "crashloopbackoff":
		return "error"
	case "disabled":
		return "disabled"
	default:
		if s == "" {
			return "unknown"
		}
		return s
	}
}
