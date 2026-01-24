package bone

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Init initializes a new Calcium module
func Init(name string) error {
	var dir string
	var moduleName string

	if name != "" {
		// Create new directory
		dir = name
		moduleName = name
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	} else {
		// Use current directory
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return err
		}
		moduleName = filepath.Base(dir)
	}

	// Check if meta.toml already exists
	metaPath := filepath.Join(dir, MetaFileName)
	if _, err := os.Stat(metaPath); err == nil {
		return fmt.Errorf("meta.toml already exists in %s", dir)
	}

	// Interactive prompts
	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("Module name (%s): ", moduleName)
	if input, _ := reader.ReadString('\n'); strings.TrimSpace(input) != "" {
		moduleName = strings.TrimSpace(input)
	}

	fmt.Print("Author (UPPERCASE): ")
	author, _ := reader.ReadString('\n')
	author = strings.TrimSpace(author)
	if author == "" {
		return fmt.Errorf("author is required")
	}
	author = strings.ToUpper(author)

	fmt.Print("Description: ")
	description, _ := reader.ReadString('\n')
	description = strings.TrimSpace(description)

	fmt.Print("License (MIT): ")
	license, _ := reader.ReadString('\n')
	license = strings.TrimSpace(license)
	if license == "" {
		license = "MIT"
	}

	fmt.Print("Keywords (comma-separated): ")
	keywordsInput, _ := reader.ReadString('\n')
	keywordsInput = strings.TrimSpace(keywordsInput)
	var keywords []string
	if keywordsInput != "" {
		for _, kw := range strings.Split(keywordsInput, ",") {
			kw = strings.TrimSpace(kw)
			if kw != "" {
				keywords = append(keywords, kw)
			}
		}
	}

	// Create meta.toml with comments
	metaContent := fmt.Sprintf(`name = %q
author = %q
description = %q
license = %q
`, moduleName, author, description, license)

	// Add keywords if present
	if len(keywords) > 0 {
		metaContent += "keywords = ["
		for i, kw := range keywords {
			if i > 0 {
				metaContent += ", "
			}
			metaContent += fmt.Sprintf("%q", kw)
		}
		metaContent += "]\n"
	}

	metaContent += `entry = "mod.ca"

# External dependencies (core modules like core.io don't need to be listed)
# [dependencies]
# "author/module" = "^1.0.0"
`

	if err := os.WriteFile(metaPath, []byte(metaContent), 0644); err != nil {
		return fmt.Errorf("failed to create meta.toml: %w", err)
	}

	// Create mod.ca if it doesn't exist
	modPath := filepath.Join(dir, "mod.ca")
	if _, err := os.Stat(modPath); os.IsNotExist(err) {
		// Convert module name to valid namespace (replace hyphens with underscores)
		namespace := strings.ReplaceAll(moduleName, "-", "_")
		content := fmt.Sprintf(`// %s - %s
// Author: %s
// License: %s

namespace %s;

use core.io!;

func hello() {
    io.println("Hello from %s!");
}
`, moduleName, description, author, license, namespace, moduleName)

		if err := os.WriteFile(modPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to create mod.ca: %w", err)
		}
	}

	// Create .gitignore if it doesn't exist
	gitignorePath := filepath.Join(dir, ".gitignore")
	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		content := `# Calcium
calcium_modules/
*.bone
`
		if err := os.WriteFile(gitignorePath, []byte(content), 0644); err != nil {
			// Not critical, just warn
			fmt.Fprintf(os.Stderr, "Warning: failed to create .gitignore: %v\n", err)
		}
	}

	fmt.Println()
	fmt.Printf("Initialized Calcium module '%s' in %s\n", moduleName, dir)
	fmt.Println()
	fmt.Println("Created files:")
	fmt.Println("  meta.toml   - Module metadata for Boneyard")
	fmt.Println("  mod.ca      - Entry point")
	fmt.Println("  .gitignore  - Git ignore file")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Edit mod.ca to add your code")
	fmt.Println("  2. Test with: calcium mod.ca")
	fmt.Println("  3. Create a git tag: git tag v0.1.0 && git push --tags")
	fmt.Println("  4. Register at Boneyard: https://github.com/calcium-lang/boneyard/issues/new")

	return nil
}
