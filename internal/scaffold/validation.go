package scaffold

import (
	"fmt"
	"regexp"

	"github.com/ironplate-dev/ironplate/internal/components"
	"github.com/ironplate-dev/ironplate/internal/config"
	"github.com/ironplate-dev/ironplate/pkg/fsutil"
)

var kebabCaseRegex = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)

// ValidationResult holds the results of config validation.
type ValidationResult struct {
	Errors   []string
	Warnings []string
}

// IsValid returns true if there are no errors.
func (r *ValidationResult) IsValid() bool {
	return len(r.Errors) == 0
}

// ValidateForScaffold validates a config before scaffolding a new project.
func ValidateForScaffold(cfg *config.ProjectConfig, outputDir string) *ValidationResult {
	result := &ValidationResult{}

	// Name must be kebab-case
	if !kebabCaseRegex.MatchString(cfg.Metadata.Name) {
		result.Errors = append(result.Errors,
			fmt.Sprintf("project name %q must be kebab-case (e.g., my-platform)", cfg.Metadata.Name))
	}

	// Organization must not be empty
	if cfg.Metadata.Organization == "" {
		result.Errors = append(result.Errors, "organization is required")
	}

	// Output directory should not already exist (or be empty)
	if fsutil.DirExists(outputDir) {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("output directory %q already exists", outputDir))
	}

	// Validate component dependencies
	_, err := components.ResolveDependencies(cfg.Spec.Infrastructure.Components)
	if err != nil {
		result.Errors = append(result.Errors,
			fmt.Sprintf("component dependency error: %s", err))
	}

	// Check for unknown components
	for _, name := range cfg.Spec.Infrastructure.Components {
		if components.Get(name) == nil {
			result.Errors = append(result.Errors,
				fmt.Sprintf("unknown infrastructure component %q", name))
		}
	}

	// Validate cloud provider if specified
	provider := cfg.Spec.Cloud.Provider
	if provider != "" && provider != "none" {
		if provider != "gcp" && provider != "aws" && provider != "azure" {
			result.Errors = append(result.Errors,
				fmt.Sprintf("unsupported cloud provider %q", provider))
		}
		if provider != "gcp" {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("cloud provider %q is not yet fully implemented, only GCP is available", provider))
		}
	}

	return result
}
