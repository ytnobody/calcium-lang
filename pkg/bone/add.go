package bone

import (
	"fmt"
	"os"
	"path/filepath"
)

// Add adds a module from Boneyard
func Add(moduleSpec string) error {
	// Parse module specification
	author, name, version, err := ParseModuleSpec(moduleSpec)
	if err != nil {
		return err
	}

	fmt.Printf("Resolving %s/%s", author, name)
	if version != "" {
		fmt.Printf("@%s", version)
	}
	fmt.Println("...")

	// Find project root
	projectRoot, err := FindProjectRoot()
	if err != nil {
		return err
	}
	if projectRoot == "" {
		return fmt.Errorf("no meta.toml found. Run 'bone init' first")
	}

	// Fetch module metadata from Boneyard
	meta, err := FetchMeta(author, name)
	if err != nil {
		return err
	}

	// Fetch version info
	versionInfo, err := FetchVersion(author, name, version)
	if err != nil {
		return err
	}

	resolvedVersion := versionInfo.Version
	if resolvedVersion == "" || resolvedVersion == "null" {
		resolvedVersion = versionInfo.Commit[:8] // Use short commit hash
	}

	fmt.Printf("Installing %s/%s@%s...\n", author, name, resolvedVersion)

	// Create modules directory
	modulesDir := filepath.Join(projectRoot, ModulesDir, author, name)
	if err := os.MkdirAll(modulesDir, 0755); err != nil {
		return fmt.Errorf("failed to create modules directory: %w", err)
	}

	// Fetch entry file
	entry := meta.Entry
	if entry == "" {
		entry = "mod.ca"
	}

	content, err := FetchFile(meta.SourceURL, versionInfo.Commit, entry)
	if err != nil {
		return err
	}

	// Save entry file
	entryPath := filepath.Join(modulesDir, entry)
	if err := os.WriteFile(entryPath, content, 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", entry, err)
	}

	// Update lock file
	lock, err := LoadLockFile(projectRoot)
	if err != nil {
		return fmt.Errorf("failed to load lock file: %w", err)
	}

	// Remove existing entry if present
	newModules := make([]LockedModule, 0, len(lock.Modules))
	for _, m := range lock.Modules {
		if !(m.Author == author && m.Name == name) {
			newModules = append(newModules, m)
		}
	}

	// Add new entry
	newModules = append(newModules, LockedModule{
		Name:    name,
		Author:  author,
		Version: resolvedVersion,
		Commit:  versionInfo.Commit,
	})
	lock.Modules = newModules

	if err := SaveLockFile(projectRoot, lock); err != nil {
		return fmt.Errorf("failed to save lock file: %w", err)
	}

	fmt.Printf("Added %s/%s@%s\n", author, name, resolvedVersion)
	fmt.Printf("  -> %s\n", filepath.Join(ModulesDir, author, name, entry))

	return nil
}
