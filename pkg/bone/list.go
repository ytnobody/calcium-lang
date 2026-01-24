package bone

import (
	"fmt"
)

// List lists installed modules
func List() error {
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
		fmt.Println("Use 'bone add <AUTHOR/module>' to install modules.")
		return nil
	}

	fmt.Println("Installed modules:")
	fmt.Println()

	for _, m := range lock.Modules {
		fmt.Printf("  %s/%s@%s\n", m.Author, m.Name, m.Version)
		if m.Commit != "" {
			fmt.Printf("    commit: %s\n", m.Commit[:min(8, len(m.Commit))])
		}
	}

	fmt.Println()
	fmt.Printf("Total: %d module(s)\n", len(lock.Modules))

	return nil
}
