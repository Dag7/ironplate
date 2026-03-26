package scaffold

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/dag7/ironplate/internal/config"
	"github.com/dag7/ironplate/internal/engine"
	"github.com/dag7/ironplate/internal/manifest"
)

// FileChange represents a single file difference found during update.
type FileChange struct {
	RelPath    string
	ChangeType ChangeType
	OldContent []byte // current file content (nil for new files)
	NewContent []byte // re-rendered content (nil for removed files)
}

// ChangeType classifies how a file differs between current and re-rendered.
type ChangeType int

const (
	// ChangeUpdate — file exists and was not modified by user; safe to overwrite.
	ChangeUpdate ChangeType = iota
	// ChangeConflict — file exists but user modified it; needs manual review.
	ChangeConflict
	// ChangeNew — file didn't exist before; will be created.
	ChangeNew
	// ChangeRemoved — file was in manifest but template no longer generates it.
	ChangeRemoved
)

func (c ChangeType) String() string {
	switch c {
	case ChangeUpdate:
		return "update"
	case ChangeConflict:
		return "conflict"
	case ChangeNew:
		return "new"
	case ChangeRemoved:
		return "removed"
	default:
		return "unknown"
	}
}

// protectedPrefixes are paths that iron update should never touch.
var protectedPrefixes = []string{
	"apps/",
	"packages/",
}

// protectedFiles are specific files that should never be overwritten.
var protectedFiles = []string{
	"ironplate.yaml",
}

// Updater compares a project's current state against freshly rendered templates
// and produces a list of changes that can be applied.
type Updater struct {
	cfg        *config.ProjectConfig
	projectDir string
	templates  fs.FS
	manifest   *manifest.Manifest
}

// NewUpdater creates an updater for the given project.
func NewUpdater(cfg *config.ProjectConfig, projectDir string, templates fs.FS, m *manifest.Manifest) *Updater {
	return &Updater{
		cfg:        cfg,
		projectDir: projectDir,
		templates:  templates,
		manifest:   m,
	}
}

// ComputeChanges re-renders all templates to a temp directory and compares
// against the current project state + manifest to classify each file change.
func (u *Updater) ComputeChanges() ([]FileChange, error) {
	// Re-render the full project to a temp directory
	tempDir, err := os.MkdirTemp("", "iron-update-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Run the scaffold pipeline into the temp dir (without manifest — we don't need one)
	scaffolder := NewScaffolder(u.cfg, tempDir, u.templates)
	if err := scaffolder.scaffoldQuiet(); err != nil {
		return nil, fmt.Errorf("re-render templates: %w", err)
	}

	// Collect all re-rendered files
	newFiles := make(map[string][]byte)
	err = filepath.WalkDir(tempDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		relPath, err := filepath.Rel(tempDir, path)
		if err != nil {
			return err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		newFiles[relPath] = content
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk temp dir: %w", err)
	}

	var changes []FileChange

	for relPath, newContent := range newFiles {
		if isProtected(relPath) {
			continue
		}

		currentPath := filepath.Join(u.projectDir, relPath)
		currentContent, err := os.ReadFile(currentPath)

		if os.IsNotExist(err) {
			// File doesn't exist in project — it's new
			changes = append(changes, FileChange{
				RelPath:    relPath,
				ChangeType: ChangeNew,
				NewContent: newContent,
			})
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("read current %s: %w", relPath, err)
		}

		// File exists — check if the new content differs
		newChecksum := manifest.Checksum(newContent)
		currentChecksum := manifest.Checksum(currentContent)

		if newChecksum == currentChecksum {
			// Content identical — nothing to do
			continue
		}

		// Content differs. Was the current file modified by the user?
		if u.manifest != nil {
			if entry, ok := u.manifest.Files[relPath]; ok {
				if currentChecksum == entry.Checksum {
					// User hasn't touched it — safe to update
					changes = append(changes, FileChange{
						RelPath:    relPath,
						ChangeType: ChangeUpdate,
						OldContent: currentContent,
						NewContent: newContent,
					})
					continue
				}
				// User modified the file — conflict
				changes = append(changes, FileChange{
					RelPath:    relPath,
					ChangeType: ChangeConflict,
					OldContent: currentContent,
					NewContent: newContent,
				})
				continue
			}
		}

		// No manifest or file not in manifest — conservative: treat as conflict
		changes = append(changes, FileChange{
			RelPath:    relPath,
			ChangeType: ChangeConflict,
			OldContent: currentContent,
			NewContent: newContent,
		})
	}

	// Check for removed files (in manifest but not re-rendered)
	if u.manifest != nil {
		for relPath := range u.manifest.Files {
			if isProtected(relPath) {
				continue
			}
			if _, exists := newFiles[relPath]; !exists {
				// Template no longer generates this file
				currentPath := filepath.Join(u.projectDir, relPath)
				currentContent, err := os.ReadFile(currentPath)
				if err == nil {
					changes = append(changes, FileChange{
						RelPath:    relPath,
						ChangeType: ChangeRemoved,
						OldContent: currentContent,
					})
				}
			}
		}
	}

	return changes, nil
}

// ApplyChange writes a single change to the project directory.
func ApplyChange(projectDir string, change FileChange) error {
	path := filepath.Join(projectDir, change.RelPath)

	switch change.ChangeType {
	case ChangeUpdate, ChangeConflict, ChangeNew:
		perm := os.FileMode(0o644)
		if strings.HasSuffix(change.RelPath, ".sh") {
			perm = 0o755
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		return os.WriteFile(path, change.NewContent, perm)

	case ChangeRemoved:
		return os.Remove(path)

	default:
		return fmt.Errorf("unknown change type for %s", change.RelPath)
	}
}

// scaffoldQuiet runs the scaffold pipeline without printing status messages.
func (s *Scaffolder) scaffoldQuiet() error {
	ctx := engine.NewTemplateContext(s.cfg)
	steps := s.buildSteps(ctx)
	for _, step := range steps {
		if err := step.execute(); err != nil {
			return err
		}
	}
	return nil
}

func isProtected(relPath string) bool {
	for _, f := range protectedFiles {
		if relPath == f {
			return true
		}
	}
	for _, prefix := range protectedPrefixes {
		if strings.HasPrefix(relPath, prefix) {
			return true
		}
	}
	return false
}
