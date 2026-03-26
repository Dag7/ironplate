// Package scaffold orchestrates the full project generation pipeline.
package scaffold

import (
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/ironplate-dev/ironplate/internal/components"
	"github.com/ironplate-dev/ironplate/internal/config"
	"github.com/ironplate-dev/ironplate/internal/engine"
	"github.com/ironplate-dev/ironplate/internal/tui"
	"github.com/ironplate-dev/ironplate/pkg/fsutil"
)

// Scaffolder generates a complete ironplate project from configuration.
type Scaffolder struct {
	cfg       *config.ProjectConfig
	outputDir string
	renderer  *engine.Renderer
	printer   *tui.StatusPrinter
	templates fs.FS
}

// NewScaffolder creates a new project scaffolder.
func NewScaffolder(cfg *config.ProjectConfig, outputDir string, templates fs.FS) *Scaffolder {
	return &Scaffolder{
		cfg:       cfg,
		outputDir: outputDir,
		renderer:  engine.NewRenderer(),
		printer:   tui.NewStatusPrinter(),
		templates: templates,
	}
}

// Scaffold generates the complete project.
func (s *Scaffolder) Scaffold() error {
	ctx := engine.NewTemplateContext(s.cfg)

	s.printer.Section("Scaffolding " + s.cfg.Metadata.Name)

	steps := s.buildSteps(ctx)
	total := len(steps)

	for i, step := range steps {
		s.printer.Step(i+1, total, step.description)
		if err := step.execute(); err != nil {
			return fmt.Errorf("step %d (%s): %w", i+1, step.description, err)
		}
	}

	fmt.Println()
	s.printer.Success("Project scaffolded successfully!")
	return nil
}

type scaffoldStep struct {
	description string
	execute     func() error
}

func (s *Scaffolder) buildSteps(ctx *engine.TemplateContext) []scaffoldStep {
	steps := []scaffoldStep{
		{"Creating project directory", s.createProjectDir},
		{"Generating base files", s.stepRenderDir("base", ctx)},
	}

	// Monorepo config based on languages
	if ctx.Computed.HasNode {
		steps = append(steps, scaffoldStep{
			"Generating Node.js monorepo config",
			s.stepRenderDir("monorepo/node", ctx),
		})
	}
	if ctx.Computed.HasGo {
		steps = append(steps, scaffoldStep{
			"Generating Go workspace config",
			s.stepRenderDir("monorepo/go", ctx),
		})
	}

	// Development environment
	if s.cfg.Spec.DevEnvironment.Type == "devcontainer" {
		steps = append(steps, scaffoldStep{
			"Generating devcontainer",
			s.stepRenderDirTo("devcontainer", ".devcontainer", ctx),
		})
	}

	// k3d cluster config
	if s.cfg.Spec.DevEnvironment.K8sLocal == "k3d" {
		steps = append(steps, scaffoldStep{
			"Generating k3d cluster config",
			s.stepRenderDirTo("k3d", ".devcontainer/k3s", ctx),
		})
	}

	// Dockerfiles
	steps = append(steps, scaffoldStep{
		"Generating Dockerfiles",
		s.stepRenderDir("dockerfiles", ctx),
	})

	// Tilt
	if s.cfg.Spec.DevEnvironment.DevTool == "tilt" {
		steps = append(steps, scaffoldStep{
			"Generating Tilt configuration",
			s.stepRenderDir("tilt", ctx),
		})
	}

	// Helm library chart + ingress chart
	steps = append(steps, scaffoldStep{
		"Generating Helm library chart",
		s.stepRenderDirTo("k8s/helm/_lib", "k8s/helm/_lib", ctx),
	})
	steps = append(steps, scaffoldStep{
		"Generating ingress chart",
		s.stepRenderDirTo("k8s/helm/ingress", "k8s/helm/ingress", ctx),
	})

	// Local development manifests (postgres, redis, pgadmin, traefik)
	steps = append(steps, scaffoldStep{
		"Generating local dev manifests",
		s.stepRenderDirTo("k8s/deployment/local", "k8s/deployment/local", ctx),
	})

	// Infrastructure components
	resolved, err := components.ResolveDependencies(s.cfg.Spec.Infrastructure.Components)
	if err != nil {
		return steps // Return what we have; validation should catch this earlier
	}
	for _, compName := range resolved {
		comp := components.Get(compName)
		if comp == nil {
			continue
		}
		for _, tmplDir := range comp.Templates {
			name := compName
			dir := tmplDir
			steps = append(steps, scaffoldStep{
				fmt.Sprintf("Generating %s component", name),
				s.stepRenderDirTo(dir, fmt.Sprintf("k8s/helm/infra/%s", name), ctx),
			})
		}
		for _, mapping := range comp.ExtraTemplates {
			m := mapping
			steps = append(steps, scaffoldStep{
				fmt.Sprintf("Generating %s extras", compName),
				s.stepRenderDirTo(m.Source, m.Output, ctx),
			})
		}
	}

	// IaC (Pulumi)
	if s.cfg.Spec.Cloud.Provider != "" && s.cfg.Spec.Cloud.Provider != "none" {
		iacDir := fmt.Sprintf("iac/pulumi/%s", s.cfg.Spec.Cloud.Provider)
		steps = append(steps, scaffoldStep{
			fmt.Sprintf("Generating %s IaC", s.cfg.Spec.Cloud.Provider),
			s.stepRenderDirTo(iacDir, "iac/pulumi", ctx),
		})
	}

	// CI/CD
	if s.cfg.Spec.CICD.Platform == "github-actions" {
		steps = append(steps, scaffoldStep{
			"Generating CI/CD pipelines",
			s.stepRenderDirTo("cicd/github-actions", ".github", ctx),
		})
	}

	// Scripts
	if ctx.Computed.HasNode {
		steps = append(steps, scaffoldStep{
			"Generating utility scripts",
			s.stepRenderDir("scripts", ctx),
		})
	}

	// CLAUDE.md
	if s.cfg.Spec.AI.ClaudeMD {
		steps = append(steps, scaffoldStep{
			"Generating CLAUDE.md",
			s.stepRenderDir("claude-md", ctx),
		})
	}

	// Skills
	if s.cfg.Spec.AI.ClaudeCode {
		steps = append(steps, scaffoldStep{
			"Generating Claude skills",
			s.stepRenderDirTo("skills", ".claude/skills", ctx),
		})
	}

	return steps
}

func (s *Scaffolder) createProjectDir() error {
	return fsutil.EnsureDir(s.outputDir)
}

func (s *Scaffolder) stepRenderDir(templateDir string, ctx *engine.TemplateContext) func() error {
	return func() error {
		// Check if the template directory exists before attempting to render
		if _, err := fs.Stat(s.templates, templateDir); err != nil {
			return nil
		}
		return s.renderer.RenderFS(s.templates, templateDir, s.outputDir, ctx)
	}
}

// stepRenderDirTo renders templates from one directory into a different output subdirectory.
func (s *Scaffolder) stepRenderDirTo(templateDir, outputSubDir string, ctx *engine.TemplateContext) func() error {
	return func() error {
		if _, err := fs.Stat(s.templates, templateDir); err != nil {
			return nil
		}
		outputPath := filepath.Join(s.outputDir, outputSubDir)
		return s.renderer.RenderFS(s.templates, templateDir, outputPath, ctx)
	}
}
