package devtools

import (
	"encoding/json"
	"fmt"
	"sort"
)

// ArgoApp represents an ArgoCD application.
type ArgoApp struct {
	Name         string
	Namespace    string
	Project      string
	SyncStatus   string // Synced, OutOfSync, Unknown
	HealthStatus string // Healthy, Progressing, Degraded, Suspended, Missing, Unknown
}

// ListArgoApps returns all ArgoCD applications.
func ListArgoApps() ([]ArgoApp, error) {
	result, err := Kubectl("get", "applications", "-n", "argocd", "-o", "json")
	if err != nil {
		return nil, fmt.Errorf("failed to list ArgoCD applications: %w", err)
	}

	var data struct {
		Items []struct {
			Metadata struct {
				Name      string `json:"name"`
				Namespace string `json:"namespace"`
			} `json:"metadata"`
			Spec struct {
				Project string `json:"project"`
			} `json:"spec"`
			Status struct {
				Sync struct {
					Status string `json:"status"`
				} `json:"sync"`
				Health struct {
					Status string `json:"status"`
				} `json:"health"`
			} `json:"status"`
		} `json:"items"`
	}

	if err := json.Unmarshal([]byte(result.Stdout), &data); err != nil {
		return nil, fmt.Errorf("parse ArgoCD response: %w", err)
	}

	var apps []ArgoApp
	for _, item := range data.Items {
		apps = append(apps, ArgoApp{
			Name:         item.Metadata.Name,
			Namespace:    item.Metadata.Namespace,
			Project:      item.Spec.Project,
			SyncStatus:   item.Status.Sync.Status,
			HealthStatus: item.Status.Health.Status,
		})
	}

	sort.Slice(apps, func(i, j int) bool {
		if apps[i].Project != apps[j].Project {
			return apps[i].Project < apps[j].Project
		}
		return apps[i].Name < apps[j].Name
	})

	return apps, nil
}

// SyncArgoApp triggers a sync on an ArgoCD application.
func SyncArgoApp(appName string) error {
	_, err := Kubectl("-n", "argocd", "patch", "application", appName,
		"--type=merge", "-p",
		`{"operation":{"sync":{"syncStrategy":{"hook":{}}}}}`)
	return err
}

// RefreshArgoApp refreshes an ArgoCD application.
func RefreshArgoApp(appName string, hard bool) error {
	refreshType := "normal"
	if hard {
		refreshType = "normal,hard"
	}
	_, err := Kubectl("-n", "argocd", "annotate", "application", appName,
		fmt.Sprintf("argocd.argoproj.io/refresh=%s", refreshType), "--overwrite")
	return err
}

// SyncIcon returns an icon for the sync status.
func SyncIcon(status string) string {
	switch status {
	case "Synced":
		return "✓"
	case "OutOfSync":
		return "○"
	default:
		return "?"
	}
}

// HealthIcon returns an icon for the health status.
func HealthIcon(status string) string {
	switch status {
	case "Healthy":
		return "●" // green when styled
	case "Progressing":
		return "◐"
	case "Degraded":
		return "✗"
	case "Suspended":
		return "⏸"
	case "Missing":
		return "?"
	default:
		return "○"
	}
}

// ArgoAppDetail holds detailed info about a single ArgoCD application.
type ArgoAppDetail struct {
	Name         string
	Project      string
	RepoURL      string
	Path         string
	Revision     string
	SyncStatus   string
	HealthStatus string
	LastSyncTime string
	Resources    []ArgoResource
}

// ArgoResource represents a resource managed by an ArgoCD application.
type ArgoResource struct {
	Kind      string
	Name      string
	Namespace string
	Status    string
	Health    string
	Message   string
}

// GetArgoAppStatus returns detailed status for a specific ArgoCD application.
func GetArgoAppStatus(appName string) (*ArgoAppDetail, error) {
	result, err := Kubectl("get", "application", appName, "-n", "argocd", "-o", "json")
	if err != nil {
		return nil, fmt.Errorf("failed to get application %s: %w", appName, err)
	}

	var data struct {
		Spec struct {
			Project string `json:"project"`
			Source  struct {
				RepoURL        string `json:"repoURL"`
				Path           string `json:"path"`
				TargetRevision string `json:"targetRevision"`
			} `json:"source"`
		} `json:"spec"`
		Status struct {
			Sync struct {
				Status   string `json:"status"`
				Revision string `json:"revision"`
			} `json:"sync"`
			Health struct {
				Status string `json:"status"`
			} `json:"health"`
			OperationState struct {
				FinishedAt string `json:"finishedAt"`
			} `json:"operationState"`
			Resources []struct {
				Kind      string `json:"kind"`
				Name      string `json:"name"`
				Namespace string `json:"namespace"`
				Status    string `json:"status"`
				Health    *struct {
					Status  string `json:"status"`
					Message string `json:"message"`
				} `json:"health"`
			} `json:"resources"`
		} `json:"status"`
	}

	if err := json.Unmarshal([]byte(result.Stdout), &data); err != nil {
		return nil, fmt.Errorf("parse application response: %w", err)
	}

	detail := &ArgoAppDetail{
		Name:         appName,
		Project:      data.Spec.Project,
		RepoURL:      data.Spec.Source.RepoURL,
		Path:         data.Spec.Source.Path,
		Revision:     data.Status.Sync.Revision,
		SyncStatus:   data.Status.Sync.Status,
		HealthStatus: data.Status.Health.Status,
		LastSyncTime: data.Status.OperationState.FinishedAt,
	}

	for _, r := range data.Status.Resources {
		res := ArgoResource{
			Kind:      r.Kind,
			Name:      r.Name,
			Namespace: r.Namespace,
			Status:    r.Status,
		}
		if r.Health != nil {
			res.Health = r.Health.Status
			res.Message = r.Health.Message
		}
		detail.Resources = append(detail.Resources, res)
	}

	return detail, nil
}

// SyncMultipleArgoApps triggers sync on multiple ArgoCD applications.
func SyncMultipleArgoApps(appNames []string) map[string]error {
	results := make(map[string]error)
	for _, name := range appNames {
		results[name] = SyncArgoApp(name)
	}
	return results
}

// GetOutOfSyncApps returns applications that are out of sync.
func GetOutOfSyncApps() ([]ArgoApp, error) {
	apps, err := ListArgoApps()
	if err != nil {
		return nil, err
	}
	var outOfSync []ArgoApp
	for _, app := range apps {
		if app.SyncStatus == "OutOfSync" {
			outOfSync = append(outOfSync, app)
		}
	}
	return outOfSync, nil
}

// GroupByProject groups ArgoCD apps by their project.
func GroupByProject(apps []ArgoApp) map[string][]ArgoApp {
	grouped := make(map[string][]ArgoApp)
	for _, app := range apps {
		project := app.Project
		if project == "" {
			project = "default"
		}
		grouped[project] = append(grouped[project], app)
	}
	return grouped
}
