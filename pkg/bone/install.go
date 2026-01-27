package bone

import (
	"fmt"
	"os"
	"path/filepath"
)

// Install installs all dependencies from calcium.lock
func Install() error {
	// Find project root
	projectRoot, err := FindProjectRoot()
	if err != nil {
		return err
	}
	if projectRoot == "" {
		return fmt.Errorf("no meta.toml found. Run 'bone init' first")
	}

	// Load lock file
	lock, err := LoadLockFile(projectRoot)
	if err != nil {
		return fmt.Errorf("failed to load lock file: %w", err)
	}

	if len(lock.Modules) == 0 {
		return fmt.Errorf("no modules in calcium.lock. Use 'bone add <module>' to add dependencies")
	}

	fmt.Printf("Installing %d module(s) from calcium.lock...\n\n", len(lock.Modules))

	// Install each locked module
	for _, mod := range lock.Modules {
		if err := installFromLock(projectRoot, mod); err != nil {
			return fmt.Errorf("failed to install %s/%s: %w", mod.Author, mod.Name, err)
		}
	}

	fmt.Printf("\nDone! Installed %d module(s).\n", len(lock.Modules))
	return nil
}

// installFromLock installs a module from lock file entry (fixed commit).
// Note: This does not recursively resolve dependencies - calcium.lock is assumed
// to contain the full flattened dependency tree (written by 'bone add').
func installFromLock(projectRoot string, mod LockedModule) error {
	moduleKey := mod.Author + "/" + mod.Name

	// Fetch module metadata from Boneyard
	meta, err := FetchMeta(mod.Author, mod.Name)
	if err != nil {
		return fmt.Errorf("failed to fetch metadata for %s: %w", moduleKey, err)
	}

	version := mod.Version
	if version == "" {
		version = mod.Commit[:min(8, len(mod.Commit))]
	}

	fmt.Printf("Installing %s/%s@%s...\n", mod.Author, mod.Name, version)

	// Determine install directory
	installDir := filepath.Join(projectRoot, ModulesDir, mod.Author, mod.Name)

	if err := os.MkdirAll(installDir, 0755); err != nil {
		return fmt.Errorf("failed to create install directory: %w", err)
	}

	// Fetch entry file using the locked commit
	entry := meta.Entry
	if entry == "" {
		entry = "mod.ca"
	}

	content, err := FetchFile(meta.SourceURL, mod.Commit, entry)
	if err != nil {
		return fmt.Errorf("failed to fetch %s: %w", entry, err)
	}

	// Save entry file (always as mod.ca for consistency)
	entryPath := filepath.Join(installDir, "mod.ca")
	if err := os.WriteFile(entryPath, content, 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", entry, err)
	}

	fmt.Printf("+ %s/%s@%s\n", mod.Author, mod.Name, version)

	return nil
}
