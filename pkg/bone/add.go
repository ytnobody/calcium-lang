package bone

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// getGlobalCacheDir returns the global module cache directory (~/.calcium/cache)
func getGlobalCacheDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".calcium", "cache")
	}
	return filepath.Join(home, ".calcium", "cache")
}

// Add adds a module from Boneyard
// If global is true, installs to ~/.calcium/cache/
// If global is false (default), installs to ./calcium_modules/
func Add(moduleSpec string, global bool) error {
	// Parse module specification
	author, name, version, err := ParseModuleSpec(moduleSpec)
	if err != nil {
		return err
	}

	fmt.Printf("Resolving %s/%s", author, name)
	if version != "" {
		fmt.Printf("@%s", version)
	}
	if global {
		fmt.Print(" (global)")
	}
	fmt.Println("...")

	// Find project root (only required for local install)
	projectRoot, err := FindProjectRoot()
	if err != nil {
		return err
	}
	if projectRoot == "" && !global {
		return fmt.Errorf("no meta.toml found. Run 'bone init' first, or use --global")
	}

	// Load existing lock file (for local installs)
	var lock *LockFile
	if !global && projectRoot != "" {
		lock, err = LoadLockFile(projectRoot)
		if err != nil {
			return fmt.Errorf("failed to load lock file: %w", err)
		}
	} else {
		lock = &LockFile{Modules: []LockedModule{}}
	}

	// Track installed modules to avoid duplicates
	installed := make(map[string]bool)
	for _, m := range lock.Modules {
		installed[m.Author+"/"+m.Name] = true
	}

	// Install module and its dependencies
	if err := installModuleRecursive(projectRoot, author, name, version, lock, installed, 0, global); err != nil {
		return err
	}

	// Save updated lock file (only for local installs)
	if !global && projectRoot != "" {
		if err := SaveLockFile(projectRoot, lock); err != nil {
			return fmt.Errorf("failed to save lock file: %w", err)
		}
	}

	return nil
}

// installModuleRecursive installs a module and its dependencies recursively
func installModuleRecursive(projectRoot, author, name, version string, lock *LockFile, installed map[string]bool, depth int, global bool) error {
	moduleKey := author + "/" + name
	indent := strings.Repeat("  ", depth)

	// Skip if already installed
	if installed[moduleKey] {
		fmt.Printf("%s- %s/%s (already installed)\n", indent, author, name)
		return nil
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
		resolvedVersion = versionInfo.Commit[:8]
	}

	fmt.Printf("%sInstalling %s/%s@%s...\n", indent, author, name, resolvedVersion)

	// Determine install directory
	var installDir string
	if global {
		// Global: ~/.calcium/cache/author/name
		installDir = filepath.Join(getGlobalCacheDir(), author, name)
	} else {
		// Local: projectRoot/calcium_modules/author/name
		installDir = filepath.Join(projectRoot, ModulesDir, author, name)
	}

	if err := os.MkdirAll(installDir, 0755); err != nil {
		return fmt.Errorf("failed to create install directory: %w", err)
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

	// Save entry file (always as mod.ca for consistency)
	entryPath := filepath.Join(installDir, "mod.ca")
	if err := os.WriteFile(entryPath, content, 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", entry, err)
	}

	// Mark as installed
	installed[moduleKey] = true

	// Update lock file entry (only for local installs)
	if !global {
		updateLockEntry(lock, author, name, resolvedVersion, versionInfo.Commit)
	}

	fmt.Printf("%s+ %s/%s@%s\n", indent, author, name, resolvedVersion)

	// Install dependencies
	if len(meta.Dependencies) > 0 {
		fmt.Printf("%s  Dependencies:\n", indent)
		for depSpec, dep := range meta.Dependencies {
			depAuthor, depName, _, err := ParseModuleSpec(depSpec)
			if err != nil {
				return fmt.Errorf("invalid dependency spec %q: %w", depSpec, err)
			}

			if err := installModuleRecursive(projectRoot, depAuthor, depName, dep.Version, lock, installed, depth+1, global); err != nil {
				return fmt.Errorf("failed to install dependency %s: %w", depSpec, err)
			}
		}
	}

	return nil
}

// updateLockEntry updates or adds a module entry in the lock file
func updateLockEntry(lock *LockFile, author, name, version, commit string) {
	// Remove existing entry if present
	newModules := make([]LockedModule, 0, len(lock.Modules)+1)
	for _, m := range lock.Modules {
		if !(m.Author == author && m.Name == name) {
			newModules = append(newModules, m)
		}
	}

	// Add new entry
	newModules = append(newModules, LockedModule{
		Name:    name,
		Author:  author,
		Version: version,
		Commit:  commit,
	})
	lock.Modules = newModules
}
