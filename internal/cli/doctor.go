package cli

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/dag7/ironplate/internal/config"
	"github.com/dag7/ironplate/internal/tui"
	"github.com/dag7/ironplate/internal/version"
	"github.com/spf13/cobra"
)

// toolCheck describes a prerequisite tool to verify.
type toolCheck struct {
	name       string   // human-readable name
	command    string   // command to run (e.g., "docker")
	args       []string // args (e.g., ["--version"])
	minVersion string   // minimum version (e.g., "20.0.0")
	required   bool     // is it required or optional?
	condition  string   // when is this needed? e.g., "devcontainer", "k3d", "go", "node", "yarn", "pnpm", "tilt", "argocd", "pulumi", "hasura", "gcloud"
}

// semver holds a parsed semantic version.
type semver struct {
	major int
	minor int
	patch int
}

// parseSemver extracts major.minor.patch from a version string.
// It accepts versions like "1.27.0", "v1.27.0", or partial like "1.27".
func parseSemver(v string) (semver, bool) {
	v = strings.TrimPrefix(v, "v")
	parts := strings.SplitN(v, ".", 4)
	if len(parts) < 2 {
		return semver{}, false
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return semver{}, false
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return semver{}, false
	}
	patch := 0
	if len(parts) >= 3 {
		// Trim anything after a hyphen (e.g., "1.27.0-rc1") or plus sign.
		patchStr := parts[2]
		if idx := strings.IndexAny(patchStr, "-+"); idx >= 0 {
			patchStr = patchStr[:idx]
		}
		patch, err = strconv.Atoi(patchStr)
		if err != nil {
			patch = 0
		}
	}

	return semver{major: major, minor: minor, patch: patch}, true
}

// compareSemver returns:
//
//	-1 if a < b
//	 0 if a == b
//	+1 if a > b
func compareSemver(a, b semver) int {
	if a.major != b.major {
		if a.major < b.major {
			return -1
		}
		return 1
	}
	if a.minor != b.minor {
		if a.minor < b.minor {
			return -1
		}
		return 1
	}
	if a.patch != b.patch {
		if a.patch < b.patch {
			return -1
		}
		return 1
	}
	return 0
}

// versionAtLeast returns true if version >= minVersion using semver comparison.
func versionAtLeast(ver, minVer string) bool {
	a, okA := parseSemver(ver)
	b, okB := parseSemver(minVer)
	if !okA || !okB {
		return false
	}
	return compareSemver(a, b) >= 0
}

// extractVersion attempts to extract a version string from command output using
// common patterns.
func extractVersion(output string) string {
	// Try common version patterns in order of specificity.
	patterns := []string{
		`v?(\d+\.\d+\.\d+[-\w.]*)`, // 1.27.0, v1.27.0, 1.27.0-rc1
	}
	for _, p := range patterns {
		re := regexp.MustCompile(p)
		matches := re.FindStringSubmatch(output)
		if len(matches) >= 2 {
			return matches[1]
		}
	}
	return ""
}

// coreChecks are tools always checked regardless of project config.
func coreChecks() []toolCheck {
	return []toolCheck{
		{
			name:       "Docker",
			command:    "docker",
			args:       []string{"--version"},
			minVersion: "20.0.0",
			required:   true,
		},
		{
			name:       "kubectl",
			command:    "kubectl",
			args:       []string{"version", "--client", "--short"},
			minVersion: "1.27.0",
			required:   true,
		},
		{
			name:       "k3d",
			command:    "k3d",
			args:       []string{"version"},
			minVersion: "5.0.0",
			required:   true,
			condition:  "k3d",
		},
		{
			name:       "Helm",
			command:    "helm",
			args:       []string{"version", "--short"},
			minVersion: "3.12.0",
			required:   true,
		},
		{
			name:       "Tilt",
			command:    "tilt",
			args:       []string{"version"},
			minVersion: "0.33.0",
			required:   true,
			condition:  "tilt",
		},
	}
}

// languageChecks are tools checked based on project languages.
func languageChecks() []toolCheck {
	return []toolCheck{
		{
			name:       "Go",
			command:    "go",
			args:       []string{"version"},
			minVersion: "1.22.0",
			required:   true,
			condition:  "go",
		},
		{
			name:       "Node.js",
			command:    "node",
			args:       []string{"--version"},
			minVersion: "20.0.0",
			required:   true,
			condition:  "node",
		},
		{
			name:       "Yarn",
			command:    "yarn",
			args:       []string{"--version"},
			minVersion: "4.0.0",
			required:   true,
			condition:  "yarn",
		},
		{
			name:       "pnpm",
			command:    "pnpm",
			args:       []string{"--version"},
			minVersion: "8.0.0",
			required:   true,
			condition:  "pnpm",
		},
	}
}

// optionalChecks are tools that are nice to have.
func optionalChecks() []toolCheck {
	return []toolCheck{
		{
			name:       "ArgoCD CLI",
			command:    "argocd",
			args:       []string{"version", "--client"},
			minVersion: "2.8.0",
			required:   false,
			condition:  "argocd",
		},
		{
			name:       "Pulumi",
			command:    "pulumi",
			args:       []string{"version"},
			minVersion: "3.0.0",
			required:   false,
			condition:  "pulumi",
		},
		{
			name:       "Hasura CLI",
			command:    "hasura",
			args:       []string{"version"},
			minVersion: "2.0.0",
			required:   false,
			condition:  "hasura",
		},
		{
			name:       "gcloud",
			command:    "gcloud",
			args:       []string{"version"},
			minVersion: "400.0.0",
			required:   false,
			condition:  "gcloud",
		},
	}
}

// shouldCheck determines whether a tool check should be executed based on the
// loaded project configuration. When no config is available, core checks always
// run and conditional checks are skipped unless they have no condition.
func shouldCheck(tc toolCheck, cfg *config.ProjectConfig) bool {
	// No condition means always check.
	if tc.condition == "" {
		return true
	}

	// No project config: skip conditional checks.
	if cfg == nil {
		return false
	}

	switch tc.condition {
	case "k3d":
		return cfg.Spec.DevEnvironment.K8sLocal == "k3d"
	case "tilt":
		return cfg.Spec.DevEnvironment.DevTool == "tilt"
	case "go":
		return cfg.Spec.HasLanguage("go")
	case "node":
		return cfg.Spec.HasLanguage("node")
	case "yarn":
		return cfg.Spec.HasLanguage("node") && cfg.Spec.Monorepo.PackageManager == "yarn"
	case "pnpm":
		return cfg.Spec.HasLanguage("node") && cfg.Spec.Monorepo.PackageManager == "pnpm"
	case "argocd":
		return cfg.Spec.GitOps.Enabled && cfg.Spec.GitOps.Tool == "argocd"
	case "pulumi":
		// Check if pulumi is listed as an additional tool.
		for _, t := range cfg.Spec.DevEnvironment.Tools {
			if t == "pulumi" {
				return true
			}
		}
		return false
	case "hasura":
		return cfg.Spec.Infrastructure.HasComponent("hasura")
	case "gcloud":
		return cfg.Spec.Cloud.Provider == "gcp"
	}

	return false
}

func newDoctorCmd() *cobra.Command {
	var fix bool

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Validate environment prerequisites",
		Long:  `Check that all required tools (docker, kubectl, k3d, helm, tilt, etc.) are installed and meet minimum version requirements.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = fix
			return runDoctor()
		},
	}

	cmd.Flags().BoolVar(&fix, "fix", false, "Attempt to fix missing tools")

	return cmd
}

func runDoctor() error {
	p := tui.NewStatusPrinter()

	fmt.Println()
	fmt.Printf("  %s %s\n", tui.BoldStyle.Render("iron doctor"), tui.MutedStyle.Render(version.Short()))

	// Try to load project config (optional — doctor works outside projects too).
	var cfg *config.ProjectConfig
	if cfgPath, err := config.FindConfigFile("."); err == nil {
		if loaded, err := config.Load(cfgPath); err == nil {
			cfg = loaded
			p.Section("Project")
			p.Info(fmt.Sprintf("Found config: %s", tui.MutedStyle.Render(cfgPath)))
			p.Info(fmt.Sprintf("Project: %s (%s)", cfg.Metadata.Name, cfg.Metadata.Organization))
		}
	}

	// Build the full list of checks.
	var checks []toolCheck
	checks = append(checks, coreChecks()...)
	checks = append(checks, languageChecks()...)
	checks = append(checks, optionalChecks()...)

	// Filter to only applicable checks.
	var applicable []toolCheck
	for _, tc := range checks {
		if shouldCheck(tc, cfg) {
			applicable = append(applicable, tc)
		}
	}

	// Run checks.
	var okCount, warnCount, errCount int

	p.Section("Tool checks")

	for i, tc := range applicable {
		p.Step(i+1, len(applicable), fmt.Sprintf("Checking %s...", tc.name))

		// Run the version command.
		out, err := exec.Command(tc.command, tc.args...).CombinedOutput()
		if err != nil {
			if tc.required {
				p.Error(fmt.Sprintf("%s — not found (required, min %s)", tc.name, tc.minVersion))
				errCount++
			} else {
				p.Warning(fmt.Sprintf("%s — not found (optional)", tc.name))
				warnCount++
			}
			continue
		}

		// Extract and compare version.
		output := strings.TrimSpace(string(out))
		detectedVersion := extractVersion(output)

		if detectedVersion == "" {
			// Could not parse version, but the tool exists.
			p.Warning(fmt.Sprintf("%s — installed but could not detect version (need >= %s)", tc.name, tc.minVersion))
			warnCount++
			continue
		}

		if versionAtLeast(detectedVersion, tc.minVersion) {
			p.Success(fmt.Sprintf("%s %s %s",
				tc.name,
				tui.MutedStyle.Render(detectedVersion),
				tui.MutedStyle.Render(fmt.Sprintf("(>= %s)", tc.minVersion)),
			))
			okCount++
		} else {
			if tc.required {
				p.Error(fmt.Sprintf("%s %s — need >= %s", tc.name, detectedVersion, tc.minVersion))
				errCount++
			} else {
				p.Warning(fmt.Sprintf("%s %s — recommended >= %s", tc.name, detectedVersion, tc.minVersion))
				warnCount++
			}
		}
	}

	// Summary.
	p.Section("Summary")

	parts := []string{
		tui.SuccessStyle.Render(fmt.Sprintf("%d passed", okCount)),
	}
	if warnCount > 0 {
		parts = append(parts, tui.WarningStyle.Render(fmt.Sprintf("%d warnings", warnCount)))
	}
	if errCount > 0 {
		parts = append(parts, tui.ErrorStyle.Render(fmt.Sprintf("%d errors", errCount)))
	}
	fmt.Printf("  %s\n", strings.Join(parts, tui.MutedStyle.Render(" | ")))
	fmt.Println()

	if errCount > 0 {
		return fmt.Errorf("%d required tool(s) missing or below minimum version", errCount)
	}

	return nil
}
