package bone

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

const (
	MetaFileName   = "meta.toml"
	LockFileName   = "calcium.lock"
	ModulesDir     = "calcium_modules"
	BoneyardURL    = "https://boneyard.ca.land"
	BoneyardRawURL = "https://raw.githubusercontent.com/ytnobody/boneyard/main/index"
)

// Dependency represents a module dependency with version constraint
// Supports both simple string format and detailed object format:
//
//	[dependencies]
//	"author/module" = "^1.0.0"                    # Simple format
//	"author/module" = { version = "^1.0.0" }      # Detailed format
type Dependency struct {
	Version string // Version constraint (e.g., "^1.0.0", ">=2.0.0", "1.2.3")
}

// UnmarshalTOML implements custom TOML unmarshaling for Dependency
// to support both string and object formats
func (d *Dependency) UnmarshalTOML(data interface{}) error {
	switch v := data.(type) {
	case string:
		// Simple format: "^1.0.0"
		d.Version = v
	case map[string]interface{}:
		// Detailed format: { version = "^1.0.0" }
		if version, ok := v["version"].(string); ok {
			d.Version = version
		}
	}
	return nil
}

// Meta represents the meta.toml file for a Calcium module
type Meta struct {
	Name         string                `toml:"name"`
	Author       string                `toml:"author"`
	Description  string                `toml:"description"`
	License      string                `toml:"license,omitempty"`
	Keywords     []string              `toml:"keywords,omitempty"`
	Entry        string                `toml:"entry,omitempty"`
	SourceURL    string                `toml:"source_url,omitempty"`
	Dependencies map[string]Dependency `toml:"dependencies,omitempty"` // "author/module" -> Dependency
}

// LockFile represents the calcium.lock file
type LockFile struct {
	Modules []LockedModule `toml:"modules"`
}

// LockedModule represents a locked dependency
type LockedModule struct {
	Name    string `toml:"name"`
	Author  string `toml:"author"`
	Version string `toml:"version"`
	Commit  string `toml:"commit"`
}

// VersionInfo represents version information from Boneyard
type VersionInfo struct {
	Version   string `toml:"version"`
	Tag       string `toml:"tag"`
	Commit    string `toml:"commit"`
	Published string `toml:"published,omitempty"`
	Updated   string `toml:"updated,omitempty"`
}

// LoadMeta loads meta.toml from the given directory
func LoadMeta(dir string) (*Meta, error) {
	path := filepath.Join(dir, MetaFileName)
	var meta Meta
	if _, err := toml.DecodeFile(path, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

// SaveMeta saves meta.toml to the given directory
func SaveMeta(dir string, meta *Meta) error {
	path := filepath.Join(dir, MetaFileName)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	encoder := toml.NewEncoder(f)
	return encoder.Encode(meta)
}

// LoadLockFile loads calcium.lock from the given directory
func LoadLockFile(dir string) (*LockFile, error) {
	path := filepath.Join(dir, LockFileName)

	// Return empty lock file if doesn't exist
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return &LockFile{Modules: []LockedModule{}}, nil
	}

	var lock LockFile
	if _, err := toml.DecodeFile(path, &lock); err != nil {
		return nil, err
	}
	return &lock, nil
}

// SaveLockFile saves calcium.lock to the given directory
func SaveLockFile(dir string, lock *LockFile) error {
	path := filepath.Join(dir, LockFileName)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	encoder := toml.NewEncoder(f)
	return encoder.Encode(lock)
}

// FindProjectRoot finds the nearest directory containing meta.toml
func FindProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		metaPath := filepath.Join(dir, MetaFileName)
		if _, err := os.Stat(metaPath); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root
			return "", nil
		}
		dir = parent
	}
}
