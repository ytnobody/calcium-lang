package bone

import (
	"fmt"
	"os"
	"path/filepath"
)

// Remove removes an installed module
func Remove(moduleSpec string) error {
	// Parse module specification (version is ignored for remove)
	author, name, _, err := ParseModuleSpec(moduleSpec)
	if err != nil {
		return err
	}

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

	// Find and remove the module
	found := false
	newModules := make([]LockedModule, 0, len(lock.Modules))
	for _, m := range lock.Modules {
		if m.Author == author && m.Name == name {
			found = true
		} else {
			newModules = append(newModules, m)
		}
	}

	if !found {
		return fmt.Errorf("module %s/%s is not installed", author, name)
	}

	// Remove module directory
	modulesDir := filepath.Join(projectRoot, ModulesDir, author, name)
	if err := os.RemoveAll(modulesDir); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to remove directory %s: %v\n", modulesDir, err)
	}

	// Clean up empty author directory
	authorDir := filepath.Join(projectRoot, ModulesDir, author)
	entries, _ := os.ReadDir(authorDir)
	if len(entries) == 0 {
		os.Remove(authorDir)
	}

	// Update lock file
	lock.Modules = newModules
	if err := SaveLockFile(projectRoot, lock); err != nil {
		return fmt.Errorf("failed to save lock file: %w", err)
	}

	fmt.Printf("Removed %s/%s\n", author, name)

	return nil
}
