package secrets

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dag7/ironplate/pkg/executil"
)

// Manager handles reading/writing secret JSON files and syncing to Pulumi.
type Manager struct {
	ProjectRoot string
	ProjectName string
}

// NewManager creates a secrets manager rooted at the given project directory.
func NewManager(projectRoot, projectName string) *Manager {
	return &Manager{
		ProjectRoot: projectRoot,
		ProjectName: projectName,
	}
}

// SecretsDir returns the path to the _secrets directory.
func (m *Manager) SecretsDir() string {
	return filepath.Join(m.ProjectRoot, "iac", "pulumi", "src", "_secrets")
}

// JSONPath returns the path to the environment-specific JSON file.
func (m *Manager) JSONPath(env string) string {
	return filepath.Join(m.SecretsDir(), env+".json")
}

// PulumiDir returns the path to the iac/pulumi directory.
func (m *Manager) PulumiDir() string {
	return filepath.Join(m.ProjectRoot, "iac", "pulumi")
}

// SecretsData is the top-level JSON structure: group name -> field key -> value.
type SecretsData map[string]map[string]string

// Load reads and parses the environment JSON file.
// Returns empty data (not an error) if the file doesn't exist.
func (m *Manager) Load(env string) (SecretsData, error) {
	path := m.JSONPath(env)

	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(SecretsData), nil
		}
		return nil, fmt.Errorf("read secrets file: %w", err)
	}

	var data SecretsData
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, fmt.Errorf("parse secrets JSON: %w", err)
	}
	return data, nil
}

// Save writes the secrets data to the environment JSON file.
func (m *Manager) Save(env string, data SecretsData) error {
	if err := os.MkdirAll(m.SecretsDir(), 0o700); err != nil {
		return fmt.Errorf("create secrets directory: %w", err)
	}

	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal secrets JSON: %w", err)
	}

	path := m.JSONPath(env)
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		return fmt.Errorf("write secrets file: %w", err)
	}
	return nil
}

// InitFromGroups creates a new SecretsData with empty/placeholder values for the given groups.
func InitFromGroups(groups []Group) SecretsData {
	data := make(SecretsData)
	for _, g := range groups {
		fields := make(map[string]string)
		for _, f := range g.Fields {
			if f.Type == FieldDerived {
				continue
			}
			if f.Placeholder != "" {
				fields[f.Key] = f.Placeholder
			} else {
				fields[f.Key] = ""
			}
		}
		data[g.Name] = fields
	}
	return data
}

// isPlaceholder returns true if the value looks like an unfilled placeholder.
func isPlaceholder(val string) bool {
	if val == "" {
		return true
	}
	v := strings.ToUpper(val)
	return strings.HasPrefix(v, "REPLACE_WITH") ||
		strings.HasPrefix(v, "TODO") ||
		strings.HasPrefix(v, "CHANGE_ME") ||
		strings.HasPrefix(v, "__UNCONFIGURED__") ||
		v == "MISSING-SECRET"
}

// Status describes the configuration state of a credential group.
type GroupStatus struct {
	Name        string
	Description string
	Configured  int // Number of fields with non-empty, non-placeholder values
	Total       int // Total number of fields
	Missing     []string // Keys that are empty or placeholder
}

// Status returns the configuration status for each group.
func (m *Manager) Status(env string, groups []Group) ([]GroupStatus, error) {
	data, err := m.Load(env)
	if err != nil {
		return nil, err
	}

	var statuses []GroupStatus
	for _, g := range groups {
		gs := GroupStatus{
			Name:        g.Name,
			Description: g.Description,
			Total:       len(g.Fields),
		}
		groupData := data[g.Name]
		for _, f := range g.Fields {
			if f.Type == FieldDerived {
				gs.Total--
				continue
			}
			val := ""
			if groupData != nil {
				val = groupData[f.Key]
			}
			if isPlaceholder(val) {
				gs.Missing = append(gs.Missing, f.Key)
			} else {
				gs.Configured++
			}
		}
		statuses = append(statuses, gs)
	}
	return statuses, nil
}

// SyncToPulumi syncs the JSON secrets data to Pulumi config for the given stack.
// Returns the number of groups successfully synced.
func (m *Manager) SyncToPulumi(env string, data SecretsData) (int, error) {
	if !executil.CommandExists("pulumi") {
		return 0, fmt.Errorf("pulumi CLI not found — install from https://www.pulumi.com/docs/install/")
	}

	pulumiDir := m.PulumiDir()
	stack := env
	synced := 0

	for groupName, fields := range data {
		// Skip groups where all values are empty/placeholder
		hasValue := false
		filtered := make(map[string]string)
		for k, v := range fields {
			if !isPlaceholder(v) {
				filtered[k] = v
				hasValue = true
			}
		}
		if !hasValue {
			continue
		}

		// Marshal the filtered group as JSON
		jsonBytes, err := json.Marshal(filtered)
		if err != nil {
			return synced, fmt.Errorf("marshal group %s: %w", groupName, err)
		}

		// Set as Pulumi secret: pulumi config set --secret --stack <stack> <project>:<group> <json>
		configKey := fmt.Sprintf("%s:%s", m.ProjectName, groupName)
		_, err = executil.RunInDir(pulumiDir, "pulumi", "config", "set", "--secret",
			"--stack", stack, configKey, string(jsonBytes))
		if err != nil {
			return synced, fmt.Errorf("set pulumi secret for %s: %w", groupName, err)
		}
		synced++
	}

	return synced, nil
}
