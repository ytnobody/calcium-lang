package bone

import (
	"fmt"
)

// Update updates installed modules to their latest versions
func Update(moduleSpec string) error {
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
		fmt.Println("No modules installed.")
		return nil
	}

	var modulesToUpdate []LockedModule

	if moduleSpec != "" {
		// Update specific module
		author, name, _, err := ParseModuleSpec(moduleSpec)
		if err != nil {
			return err
		}

		found := false
		for _, m := range lock.Modules {
			if m.Author == author && m.Name == name {
				modulesToUpdate = append(modulesToUpdate, m)
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("module %s/%s is not installed", author, name)
		}
	} else {
		// Update all modules
		modulesToUpdate = lock.Modules
	}

	fmt.Printf("Checking %d module(s) for updates...\n", len(modulesToUpdate))
	fmt.Println()

	updated := 0
	for _, m := range modulesToUpdate {
		// Fetch latest version
		versionInfo, err := FetchVersion(m.Author, m.Name, "latest")
		if err != nil {
			fmt.Printf("  %s/%s: failed to check (%v)\n", m.Author, m.Name, err)
			continue
		}

		// Compare commits
		if versionInfo.Commit == m.Commit {
			fmt.Printf("  %s/%s@%s: up to date\n", m.Author, m.Name, m.Version)
			continue
		}

		// Update needed
		newVersion := versionInfo.Version
		if newVersion == "" || newVersion == "null" {
			newVersion = versionInfo.Commit[:8]
		}

		fmt.Printf("  %s/%s: %s -> %s\n", m.Author, m.Name, m.Version, newVersion)

		// Re-add the module locally (this will update it)
		spec := fmt.Sprintf("%s/%s", m.Author, m.Name)
		if err := Add(spec, false); err != nil {
			fmt.Printf("    failed to update: %v\n", err)
			continue
		}

		updated++
	}

	fmt.Println()
	if updated > 0 {
		fmt.Printf("Updated %d module(s)\n", updated)
	} else {
		fmt.Println("All modules are up to date")
	}

	return nil
}
