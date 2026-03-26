// Package manifest tracks generated file checksums for iron update.
package manifest

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dag7/ironplate/internal/version"
	"github.com/dag7/ironplate/pkg/fsutil"
)

const (
	// Dir is the directory where ironplate stores project metadata.
	Dir = ".ironplate"
	// FileName is the manifest file name.
	FileName = "manifest.json"
)

// Manifest tracks which files were generated and their checksums,
// so iron update can detect user modifications.
type Manifest struct {
	Version     string               `json:"version"`
	GeneratedAt string               `json:"generated_at"`
	Files       map[string]FileEntry `json:"files"`
}

// FileEntry records the checksum of a generated file.
type FileEntry struct {
	Checksum string `json:"checksum"`
}

// New creates a new empty manifest stamped with the current version.
func New() *Manifest {
	return &Manifest{
		Version:     version.Short(),
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Files:       make(map[string]FileEntry),
	}
}

// Path returns the manifest file path for a project directory.
func Path(projectDir string) string {
	return filepath.Join(projectDir, Dir, FileName)
}

// Load reads a manifest from disk. Returns nil if it doesn't exist.
func Load(projectDir string) (*Manifest, error) {
	path := Path(projectDir)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read manifest: %w", err)
	}

	m := &Manifest{}
	if err := json.Unmarshal(data, m); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}
	return m, nil
}

// Save writes the manifest to disk.
func (m *Manifest) Save(projectDir string) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	data = append(data, '\n')
	return fsutil.WriteFile(Path(projectDir), data, 0o644)
}

// RecordFile adds or updates a file entry with its SHA-256 checksum.
func (m *Manifest) RecordFile(relPath string, content []byte) {
	m.Files[relPath] = FileEntry{
		Checksum: Checksum(content),
	}
}

// Checksum computes the SHA-256 hex digest of content.
func Checksum(content []byte) string {
	h := sha256.Sum256(content)
	return fmt.Sprintf("%x", h)
}

// ChecksumFile computes the SHA-256 of a file on disk.
func ChecksumFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return Checksum(data), nil
}
