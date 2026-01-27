package bone

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInstall_NoMetaToml(t *testing.T) {
	// Create temp dir without meta.toml
	tmpDir := t.TempDir()

	// Change to temp dir
	oldDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldDir)

	err := Install()
	if err == nil {
		t.Error("expected error for missing meta.toml")
	}
	if err.Error() != "no meta.toml found. Run 'bone init' first" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestInstall_NoLockFile(t *testing.T) {
	// Create temp project with meta.toml but no lock file
	tmpDir := t.TempDir()

	metaContent := `name = "test"
author = "testuser"
description = "test module"
`
	os.WriteFile(filepath.Join(tmpDir, "meta.toml"), []byte(metaContent), 0644)

	// Change to temp dir
	oldDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldDir)

	err := Install()
	if err == nil {
		t.Error("expected error for empty lock file")
	}
	// Lock file will be empty (LoadLockFile returns empty struct if file doesn't exist)
	if err.Error() != "no modules in calcium.lock. Use 'bone add <module>' to add dependencies" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestInstall_EmptyLockFile(t *testing.T) {
	// Create temp project with meta.toml and empty lock file
	tmpDir := t.TempDir()

	metaContent := `name = "test"
author = "testuser"
description = "test module"
`
	os.WriteFile(filepath.Join(tmpDir, "meta.toml"), []byte(metaContent), 0644)

	lockContent := `# Empty lock file
`
	os.WriteFile(filepath.Join(tmpDir, "calcium.lock"), []byte(lockContent), 0644)

	// Change to temp dir
	oldDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldDir)

	err := Install()
	if err == nil {
		t.Error("expected error for empty lock file")
	}
	if err.Error() != "no modules in calcium.lock. Use 'bone add <module>' to add dependencies" {
		t.Errorf("unexpected error: %v", err)
	}
}
