package cli

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/dag7/ironplate/internal/config"
	"github.com/dag7/ironplate/internal/secrets"
	"github.com/dag7/ironplate/internal/tui"
)

func newSecretsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "secrets",
		Short: "Manage Pulumi secrets for your project",
		Long: `Manage Pulumi IaC secrets interactively.

Instead of manually editing JSON files and running shell scripts,
use these commands to configure, inspect, and sync secrets.`,
	}

	cmd.AddCommand(newSecretsSetupCmd())
	cmd.AddCommand(newSecretsSyncCmd())
	cmd.AddCommand(newSecretsStatusCmd())

	return cmd
}

// --- iron secrets setup ---

func newSecretsSetupCmd() *cobra.Command {
	var env string
	var autoGenerate bool
	var syncAfter bool

	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Interactively configure secrets for an environment",
		Long: `Walk through each credential group, enter values, auto-generate
where possible, and optionally sync to Pulumi config.

This replaces the manual workflow of:
  1. Editing src/_secrets/{env}.json
  2. Running scripts/pulumi-secret-set.sh
  3. Running pulumi up`,
		RunE: func(cmd *cobra.Command, args []string) error {
			printer := tui.NewStatusPrinter()

			// Load project config
			cfgPath, err := config.FindConfigFile(".")
			if err != nil {
				return fmt.Errorf("not in an ironplate project: %w", err)
			}
			cfg, err := config.Load(cfgPath)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}
			projectRoot := filepath.Dir(cfgPath)

			// Prompt for environment if not provided
			if env == "" {
				if err := huh.NewSelect[string]().
					Title("Environment").
					Description("Which environment are you configuring?").
					Options(
						huh.NewOption("Staging", "staging"),
						huh.NewOption("Production", "production"),
					).
					Value(&env).
					Run(); err != nil {
					return err
				}
			}

			printer.Section(fmt.Sprintf("Secrets Setup — %s", env))

			// Get applicable groups
			groups := secrets.GroupsForConfig(cfg)
			mgr := secrets.NewManager(projectRoot, cfg.Metadata.Name)

			// Load existing data
			data, err := mgr.Load(env)
			if err != nil {
				return fmt.Errorf("load existing secrets: %w", err)
			}

			// If no data exists, initialize from groups
			if len(data) == 0 {
				data = secrets.InitFromGroups(groups)
				printer.Info("Initialized secrets template — fill in values below")
			}

			// Walk through each group
			totalGroups := len(groups)
			configured := 0

			for i, group := range groups {
				printer.Step(i+1, totalGroups, fmt.Sprintf("%s — %s", group.Name, group.Description))

				groupData := data[group.Name]
				if groupData == nil {
					groupData = make(map[string]string)
					data[group.Name] = groupData
				}

				groupModified := false

				for _, field := range group.Fields {
					if field.Type == secrets.FieldDerived {
						continue
					}

					currentVal := groupData[field.Key]
					maskedCurrent := maskValue(currentVal)

					// If auto-generate is on and field supports it, generate automatically
					if autoGenerate && field.Type == secrets.FieldGenerate && isEmptyOrPlaceholder(currentVal) {
						generated, err := secrets.Generate(field.Generator)
						if err != nil {
							printer.Warning(fmt.Sprintf("Could not generate %s: %v", field.Key, err))
							continue
						}
						groupData[field.Key] = generated
						groupModified = true
						fmt.Printf("    %s %s: auto-generated\n", tui.CheckMark, field.Key)
						continue
					}

					// Show current state and prompt
					prompt := field.Description
					if maskedCurrent != "" {
						prompt = fmt.Sprintf("%s [current: %s]", field.Description, maskedCurrent)
					}

					var action string
					options := []huh.Option[string]{
						huh.NewOption("Skip (keep current)", "skip"),
						huh.NewOption("Enter value", "enter"),
					}
					if field.Type == secrets.FieldGenerate {
						options = append([]huh.Option[string]{
							huh.NewOption("Auto-generate", "generate"),
						}, options...)
					}

					if err := huh.NewSelect[string]().
						Title(field.Key).
						Description(prompt).
						Options(options...).
						Value(&action).
						Run(); err != nil {
						return err
					}

					switch action {
					case "generate":
						generated, err := secrets.Generate(field.Generator)
						if err != nil {
							return fmt.Errorf("generate %s: %w", field.Key, err)
						}
						groupData[field.Key] = generated
						groupModified = true
						fmt.Printf("    %s Generated\n", tui.CheckMark)

					case "enter":
						var value string
						if err := huh.NewInput().
							Title(field.Key).
							Description(field.Description).
							Value(&value).
							Run(); err != nil {
							return err
						}
						if value != "" {
							groupData[field.Key] = value
							groupModified = true
						}

					case "skip":
						// Do nothing
					}
				}

				if groupModified {
					configured++
				}
			}

			// Save the updated JSON
			if err := mgr.Save(env, data); err != nil {
				return fmt.Errorf("save secrets: %w", err)
			}

			fmt.Println()
			printer.Success(fmt.Sprintf("Secrets saved to %s", mgr.JSONPath(env)))

			// Ask about syncing to Pulumi
			if !syncAfter {
				if err := huh.NewConfirm().
					Title("Sync to Pulumi?").
					Description("Set secrets as encrypted Pulumi config values").
					Value(&syncAfter).
					Run(); err != nil {
					return err
				}
			}

			if syncAfter {
				printer.Section("Syncing to Pulumi")
				synced, err := mgr.SyncToPulumi(env, data)
				if err != nil {
					return fmt.Errorf("sync to Pulumi: %w", err)
				}
				printer.Success(fmt.Sprintf("Synced %d credential group(s) to Pulumi stack %q", synced, env))
				fmt.Println()
				printer.Info("Next: run 'pulumi preview' then 'pulumi up' to deploy")
			}

			// Print summary
			fmt.Println()
			printer.Info(fmt.Sprintf("Configured %d/%d group(s) in this session", configured, totalGroups))

			return nil
		},
	}

	cmd.Flags().StringVarP(&env, "env", "e", "", "Environment (staging|production)")
	cmd.Flags().BoolVar(&autoGenerate, "auto-generate", false, "Auto-generate all generatable secrets")
	cmd.Flags().BoolVar(&syncAfter, "sync", false, "Sync to Pulumi after setup (skip confirmation)")

	return cmd
}

// --- iron secrets sync ---

func newSecretsSyncCmd() *cobra.Command {
	var env string

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync JSON secrets to Pulumi config",
		Long: `Read the secrets JSON file and set each credential group
as an encrypted Pulumi config value.

Replaces: ./scripts/pulumi-secret-set.sh -e <env>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			printer := tui.NewStatusPrinter()

			cfgPath, err := config.FindConfigFile(".")
			if err != nil {
				return fmt.Errorf("not in an ironplate project: %w", err)
			}
			cfg, err := config.Load(cfgPath)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}
			projectRoot := filepath.Dir(cfgPath)

			if env == "" {
				if err := huh.NewSelect[string]().
					Title("Environment").
					Options(
						huh.NewOption("Staging", "staging"),
						huh.NewOption("Production", "production"),
					).
					Value(&env).
					Run(); err != nil {
					return err
				}
			}

			mgr := secrets.NewManager(projectRoot, cfg.Metadata.Name)
			data, err := mgr.Load(env)
			if err != nil {
				return fmt.Errorf("load secrets: %w", err)
			}

			if len(data) == 0 {
				return fmt.Errorf("no secrets found at %s — run 'iron secrets setup' first", mgr.JSONPath(env))
			}

			printer.Section(fmt.Sprintf("Syncing secrets — %s", env))

			synced, err := mgr.SyncToPulumi(env, data)
			if err != nil {
				return fmt.Errorf("sync: %w", err)
			}

			fmt.Println()
			printer.Success(fmt.Sprintf("Synced %d credential group(s) to Pulumi stack %q", synced, env))
			printer.Info("Next: run 'pulumi preview' then 'pulumi up' to deploy")

			return nil
		},
	}

	cmd.Flags().StringVarP(&env, "env", "e", "", "Environment (staging|production)")

	return cmd
}

// --- iron secrets status ---

func newSecretsStatusCmd() *cobra.Command {
	var env string

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show secrets configuration status",
		Long:  `Display which credential groups are configured vs missing for an environment.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			printer := tui.NewStatusPrinter()

			cfgPath, err := config.FindConfigFile(".")
			if err != nil {
				return fmt.Errorf("not in an ironplate project: %w", err)
			}
			cfg, err := config.Load(cfgPath)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}
			projectRoot := filepath.Dir(cfgPath)

			if env == "" {
				if err := huh.NewSelect[string]().
					Title("Environment").
					Options(
						huh.NewOption("Staging", "staging"),
						huh.NewOption("Production", "production"),
					).
					Value(&env).
					Run(); err != nil {
					return err
				}
			}

			groups := secrets.GroupsForConfig(cfg)
			mgr := secrets.NewManager(projectRoot, cfg.Metadata.Name)

			statuses, err := mgr.Status(env, groups)
			if err != nil {
				return fmt.Errorf("check status: %w", err)
			}

			printer.Section(fmt.Sprintf("Secrets Status — %s", env))

			totalConfigured := 0
			totalGroups := 0
			for _, s := range statuses {
				totalGroups++
				icon := tui.CrossMark
				detail := fmt.Sprintf("%d/%d fields", s.Configured, s.Total)

				if s.Configured == s.Total && s.Total > 0 {
					icon = tui.CheckMark
					totalConfigured++
				} else if s.Configured > 0 {
					icon = tui.WarnMark
				}

				fmt.Printf("  %s %s (%s) — %s\n", icon, s.Name, s.Description, detail)
				if len(s.Missing) > 0 {
					fmt.Printf("      missing: %s\n", strings.Join(s.Missing, ", "))
				}
			}

			fmt.Println()
			if totalConfigured == totalGroups {
				printer.Success(fmt.Sprintf("All %d credential groups fully configured", totalGroups))
			} else {
				printer.Info(fmt.Sprintf("%d/%d groups fully configured — run 'iron secrets setup -e %s' to fill in missing values",
					totalConfigured, totalGroups, env))
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&env, "env", "e", "", "Environment (staging|production)")

	return cmd
}

// maskValue returns a masked version of a secret value for display.
func maskValue(val string) string {
	if val == "" {
		return ""
	}
	if isEmptyOrPlaceholder(val) {
		return "(placeholder)"
	}
	if len(val) <= 8 {
		return "****"
	}
	return val[:4] + "****" + val[len(val)-4:]
}

func isEmptyOrPlaceholder(val string) bool {
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
