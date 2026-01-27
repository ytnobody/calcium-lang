package vm

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// RunOptions holds runtime options for the VM
type RunOptions struct {
	// ImportMap maps module names to URLs
	// e.g., "json" -> "github.com/ytnobody/json-calcium"
	ImportMap map[string]string

	// LockFile contains version/hash info for dependencies
	LockFile *LockFile

	// CachedOnly disables network access, only uses cached modules
	CachedOnly bool
}

// ImportMapFile represents the structure of calcium.imports.toml
type ImportMapFile struct {
	Imports map[string]string `toml:"imports"`
}

// LockFile represents the structure of calcium.lock
type LockFile struct {
	Modules map[string]ModuleLock `toml:"modules"`
}

// ModuleLock represents a locked module entry
type ModuleLock struct {
	Version string            `toml:"version"`
	URL     string            `toml:"url"`
	Files   map[string]string `toml:"files"` // filename -> sha256 hash
}

// DefaultRunOptions returns default run options
func DefaultRunOptions() *RunOptions {
	return &RunOptions{
		ImportMap:  make(map[string]string),
		CachedOnly: false,
	}
}

// LoadImportMap loads import mappings from a TOML file
func LoadImportMap(path string) (map[string]string, error) {
	var imf ImportMapFile
	_, err := toml.DecodeFile(path, &imf)
	if err != nil {
		return nil, fmt.Errorf("failed to load import map: %w", err)
	}
	if imf.Imports == nil {
		return make(map[string]string), nil
	}
	return imf.Imports, nil
}

// LoadLockFile loads lock file from a TOML file
func LoadLockFile(path string) (*LockFile, error) {
	var lf LockFile
	_, err := toml.DecodeFile(path, &lf)
	if err != nil {
		return nil, fmt.Errorf("failed to load lock file: %w", err)
	}
	if lf.Modules == nil {
		lf.Modules = make(map[string]ModuleLock)
	}
	return &lf, nil
}

// ResolveModuleName resolves a module name using the import map
// Returns the original name if no mapping exists
func (opts *RunOptions) ResolveModuleName(name string) string {
	if opts == nil || opts.ImportMap == nil {
		return name
	}
	if mapped, ok := opts.ImportMap[name]; ok {
		return mapped
	}
	return name
}

// GetModuleLock returns the lock entry for a module if it exists
func (opts *RunOptions) GetModuleLock(name string) *ModuleLock {
	if opts == nil || opts.LockFile == nil {
		return nil
	}
	if lock, ok := opts.LockFile.Modules[name]; ok {
		return &lock
	}
	return nil
}

// VerifyFileHash verifies that a file's content matches its expected hash
func VerifyFileHash(content []byte, expectedHash string) bool {
	hash := sha256.Sum256(content)
	actualHash := hex.EncodeToString(hash[:])
	return actualHash == expectedHash
}

// VerifyCachedModule verifies a cached module against lock file
func (opts *RunOptions) VerifyCachedModule(moduleName string, cacheDir string) error {
	lock := opts.GetModuleLock(moduleName)
	if lock == nil {
		// No lock entry, skip verification
		return nil
	}

	for filename, expectedHash := range lock.Files {
		filePath := filepath.Join(cacheDir, filename)
		content, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("lock file verification failed: cannot read %s: %w", filename, err)
		}
		if !VerifyFileHash(content, expectedHash) {
			return fmt.Errorf("lock file verification failed: hash mismatch for %s", filename)
		}
	}

	return nil
}
