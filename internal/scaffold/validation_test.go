package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dag7/ironplate/internal/config"
)

// validConfig returns a minimal valid *config.ProjectConfig suitable for
// validation tests. Individual tests mutate the field under test.
func validConfig() *config.ProjectConfig {
	return &config.ProjectConfig{
		APIVersion: "ironplate.dev/v1",
		Kind:       "Project",
		Metadata: config.Metadata{
			Name:         "my-project",
			Organization: "acme",
		},
		Spec: config.ProjectSpec{
			Languages: []string{"node"},
			Cloud: config.CloudSpec{
				Provider: "gcp",
			},
		},
	}
}

func TestValidateForScaffold_ValidConfig(t *testing.T) {
	cfg := validConfig()
	outputDir := filepath.Join(t.TempDir(), "does-not-exist")

	result := ValidateForScaffold(cfg, outputDir)

	assert.True(t, result.IsValid(), "expected no errors, got: %v", result.Errors)
	assert.Empty(t, result.Warnings)
}

func TestValidateForScaffold_InvalidName(t *testing.T) {
	tests := []struct {
		name      string
		projName  string
		wantError bool
	}{
		{"uppercase", "MyProject", true},
		{"spaces", "my project", true},
		{"underscores", "my_project", true},
		{"starts with number", "1project", true},
		{"empty", "", true},
		{"trailing hyphen", "my-project-", true},
		{"double hyphen", "my--project", true},
		{"valid kebab", "my-project", false},
		{"single word", "project", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			cfg.Metadata.Name = tt.projName
			outputDir := filepath.Join(t.TempDir(), "out")

			result := ValidateForScaffold(cfg, outputDir)

			if tt.wantError {
				assert.False(t, result.IsValid(), "expected error for name %q", tt.projName)
				assert.NotEmpty(t, result.Errors)
				assert.Contains(t, result.Errors[0], "kebab-case")
			} else {
				// Filter out errors that are NOT about the name
				var nameErrors []string
				for _, e := range result.Errors {
					if containsSubstring(e, "kebab-case") {
						nameErrors = append(nameErrors, e)
					}
				}
				assert.Empty(t, nameErrors, "expected no name errors for %q", tt.projName)
			}
		})
	}
}

func TestValidateForScaffold_EmptyOrganization(t *testing.T) {
	cfg := validConfig()
	cfg.Metadata.Organization = ""
	outputDir := filepath.Join(t.TempDir(), "out")

	result := ValidateForScaffold(cfg, outputDir)

	assert.False(t, result.IsValid())
	require.NotEmpty(t, result.Errors)
	assert.Contains(t, result.Errors[0], "organization is required")
}

func TestValidateForScaffold_ExistingDirectory(t *testing.T) {
	cfg := validConfig()

	// Create a directory that already exists
	outputDir := t.TempDir()

	result := ValidateForScaffold(cfg, outputDir)

	// Existing directory should produce a warning, not an error
	assert.True(t, result.IsValid(), "existing dir should be a warning, not an error; errors: %v", result.Errors)
	require.NotEmpty(t, result.Warnings)
	assert.Contains(t, result.Warnings[0], "already exists")
}

func TestValidateForScaffold_UnknownComponent(t *testing.T) {
	cfg := validConfig()
	cfg.Spec.Infrastructure.Components = []string{"totally-unknown-component"}
	outputDir := filepath.Join(t.TempDir(), "out")

	result := ValidateForScaffold(cfg, outputDir)

	assert.False(t, result.IsValid())
	found := false
	for _, e := range result.Errors {
		if containsSubstring(e, "unknown") && containsSubstring(e, "totally-unknown-component") {
			found = true
			break
		}
	}
	assert.True(t, found, "expected an error about unknown component, got: %v", result.Errors)
}

func TestValidateForScaffold_InvalidProvider(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		wantErr  bool
	}{
		{"empty provider is fine", "", false},
		{"none is fine", "none", false},
		{"gcp is valid", "gcp", false},
		{"aws is valid", "aws", false},
		{"azure is valid", "azure", false},
		{"digitalocean is invalid", "digitalocean", true},
		{"random string", "foobar", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			cfg.Spec.Cloud.Provider = tt.provider
			outputDir := filepath.Join(t.TempDir(), "out")

			result := ValidateForScaffold(cfg, outputDir)

			if tt.wantErr {
				assert.False(t, result.IsValid())
				found := false
				for _, e := range result.Errors {
					if containsSubstring(e, "unsupported cloud provider") {
						found = true
						break
					}
				}
				assert.True(t, found, "expected unsupported cloud provider error for %q, got: %v", tt.provider, result.Errors)
			} else {
				// No provider-related errors
				for _, e := range result.Errors {
					assert.NotContains(t, e, "cloud provider",
						"did not expect cloud provider error for %q", tt.provider)
				}
			}
		})
	}
}

func TestValidateForScaffold_NonGCPProviderWarning(t *testing.T) {
	tests := []struct {
		provider    string
		wantWarning bool
	}{
		{"aws", true},
		{"azure", true},
		{"gcp", false},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			cfg := validConfig()
			cfg.Spec.Cloud.Provider = tt.provider
			outputDir := filepath.Join(t.TempDir(), "out")

			result := ValidateForScaffold(cfg, outputDir)

			if tt.wantWarning {
				found := false
				for _, w := range result.Warnings {
					if containsSubstring(w, "not yet fully implemented") {
						found = true
						break
					}
				}
				assert.True(t, found, "expected 'not yet fully implemented' warning for %q, got warnings: %v",
					tt.provider, result.Warnings)
			} else {
				for _, w := range result.Warnings {
					assert.NotContains(t, w, "not yet fully implemented",
						"did not expect 'not fully implemented' warning for %q", tt.provider)
				}
			}
		})
	}
}

func TestValidateForScaffold_ValidComponents(t *testing.T) {
	cfg := validConfig()
	cfg.Spec.Infrastructure.Components = []string{"kafka", "redis"}
	outputDir := filepath.Join(t.TempDir(), "out")

	result := ValidateForScaffold(cfg, outputDir)

	assert.True(t, result.IsValid(), "expected no errors for valid components, got: %v", result.Errors)
}

func TestValidateForScaffold_ComponentDependencyError(t *testing.T) {
	// "argocd" depends on "external-secrets". Requesting only "argocd" should
	// still resolve cleanly because ResolveDependencies auto-pulls hard deps.
	// However, requesting an unknown component in the list triggers the
	// dependency resolver to return an error.
	cfg := validConfig()
	cfg.Spec.Infrastructure.Components = []string{"kafka", "nonexistent-dep"}
	outputDir := filepath.Join(t.TempDir(), "out")

	result := ValidateForScaffold(cfg, outputDir)

	assert.False(t, result.IsValid(), "expected errors for unknown dependency")

	found := false
	for _, e := range result.Errors {
		if containsSubstring(e, "component") {
			found = true
			break
		}
	}
	assert.True(t, found, "expected component-related error, got: %v", result.Errors)
}

func TestValidateForScaffold_ExistingDirectoryNonExistent(t *testing.T) {
	cfg := validConfig()
	outputDir := filepath.Join(os.TempDir(), "ironplate-test-nonexistent-"+t.Name())
	// Ensure it doesn't exist
	_ = os.RemoveAll(outputDir)

	result := ValidateForScaffold(cfg, outputDir)

	// No warning about existing directory
	for _, w := range result.Warnings {
		assert.NotContains(t, w, "already exists")
	}
}

// containsSubstring is a convenience alias for strings.Contains.
func containsSubstring(s, substr string) bool {
	return strings.Contains(s, substr)
}
