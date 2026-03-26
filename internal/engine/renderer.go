package engine

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/dag7/ironplate/pkg/fsutil"
)

// filePermission returns the appropriate file permission for the given path.
// Shell scripts get executable permission (0o755), all others get 0o644.
func filePermission(path string) os.FileMode {
	if strings.HasSuffix(path, ".sh") {
		return 0o755
	}
	return 0o644
}

// Renderer handles template rendering with custom functions.
type Renderer struct {
	funcMap    template.FuncMap
	leftDelim  string
	rightDelim string
}

// NewRenderer creates a new template renderer with the ironplate function map.
func NewRenderer() *Renderer {
	return &Renderer{
		funcMap:    IronFuncMap(),
		leftDelim:  "{{",
		rightDelim: "}}",
	}
}

// NewRendererWithDelimiters creates a renderer with custom delimiters.
// Use this for files that contain {{ }} syntax (Helm templates, GitHub Actions).
func NewRendererWithDelimiters(left, right string) *Renderer {
	return &Renderer{
		funcMap:    IronFuncMap(),
		leftDelim:  left,
		rightDelim: right,
	}
}

// RegisterFunc adds a custom template function.
func (r *Renderer) RegisterFunc(name string, fn interface{}) {
	r.funcMap[name] = fn
}

// RenderString renders a template string with the given context.
func (r *Renderer) RenderString(tmplStr string, ctx interface{}) (string, error) {
	tmpl, err := template.New("inline").
		Delims(r.leftDelim, r.rightDelim).
		Funcs(r.funcMap).
		Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, ctx); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}

	return buf.String(), nil
}

// RenderFile renders a single template file and writes the output.
func (r *Renderer) RenderFile(tmplContent []byte, outputPath string, ctx interface{}) error {
	tmpl, err := template.New(filepath.Base(outputPath)).
		Delims(r.leftDelim, r.rightDelim).
		Funcs(r.funcMap).
		Parse(string(tmplContent))
	if err != nil {
		return fmt.Errorf("parse template %s: %w", outputPath, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, ctx); err != nil {
		return fmt.Errorf("execute template %s: %w", outputPath, err)
	}

	// Skip empty output (e.g., templates that are entirely conditional)
	content := buf.String()
	if strings.TrimSpace(content) == "" {
		return nil
	}

	return fsutil.WriteFile(outputPath, []byte(content), filePermission(outputPath))
}

// RenderConcatFS renders all templates in a directory and concatenates them into a single output file.
// Templates are sorted by filename to ensure deterministic ordering.
// Templates that render to empty content (all-conditional) are skipped.
func (r *Renderer) RenderConcatFS(templateFS fs.FS, rootDir, outputPath string, ctx interface{}) error {
	entries, err := fs.ReadDir(templateFS, rootDir)
	if err != nil {
		return fmt.Errorf("read template dir %s: %w", rootDir, err)
	}

	var sections []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		path := filepath.Join(rootDir, entry.Name())
		content, err := fs.ReadFile(templateFS, path)
		if err != nil {
			return fmt.Errorf("read template %s: %w", path, err)
		}

		if !strings.HasSuffix(entry.Name(), ".tmpl") {
			// Non-template files are included verbatim
			if s := strings.TrimSpace(string(content)); s != "" {
				sections = append(sections, string(content))
			}
			continue
		}

		tmpl, err := template.New(entry.Name()).
			Delims(r.leftDelim, r.rightDelim).
			Funcs(r.funcMap).
			Parse(string(content))
		if err != nil {
			return fmt.Errorf("parse template %s: %w", path, err)
		}

		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, ctx); err != nil {
			return fmt.Errorf("execute template %s: %w", path, err)
		}

		// Skip sections that render to empty (conditional sections not applicable)
		if s := strings.TrimSpace(buf.String()); s != "" {
			sections = append(sections, buf.String())
		}
	}

	if len(sections) == 0 {
		return nil
	}

	combined := strings.Join(sections, "\n")
	return fsutil.WriteFile(outputPath, []byte(combined), 0o644)
}

// RenderFS renders all templates from an embedded filesystem into the output directory.
// Optional excludeDirs skips subdirectories matching those names (e.g., "helm").
func (r *Renderer) RenderFS(templateFS fs.FS, rootDir, outputDir string, ctx interface{}, excludeDirs ...string) error {
	excludeSet := make(map[string]bool, len(excludeDirs))
	for _, d := range excludeDirs {
		excludeSet[d] = true
	}

	return fs.WalkDir(templateFS, rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Compute relative path from the root template directory
		relPath, err := filepath.Rel(rootDir, path)
		if err != nil {
			return fmt.Errorf("compute relative path: %w", err)
		}

		// Skip the root itself
		if relPath == "." {
			return nil
		}

		// Skip excluded directories and their contents
		if d.IsDir() && excludeSet[d.Name()] {
			return fs.SkipDir
		}

		outputPath := filepath.Join(outputDir, relPath)

		if d.IsDir() {
			return fsutil.EnsureDir(outputPath)
		}

		content, err := fs.ReadFile(templateFS, path)
		if err != nil {
			return fmt.Errorf("read template %s: %w", path, err)
		}

		// Skip .gitkeep files
		if filepath.Base(path) == ".gitkeep" {
			return nil
		}

		// .tmpl files are rendered as templates; other files are copied verbatim
		if strings.HasSuffix(path, ".tmpl") {
			outputPath = strings.TrimSuffix(outputPath, ".tmpl")

			// Files with .tpl.tmpl extension contain native {{ }} Helm syntax,
			// so we use [[ ]] delimiters for ironplate template directives.
			if strings.HasSuffix(path, ".tpl.tmpl") {
				customRenderer := NewRendererWithDelimiters("[[", "]]")
				return customRenderer.RenderFile(content, outputPath, ctx)
			}

			return r.RenderFile(content, outputPath, ctx)
		}

		return fsutil.WriteFile(outputPath, content, filePermission(outputPath))
	})
}
