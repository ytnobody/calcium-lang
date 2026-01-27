package vm

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadImportMap(t *testing.T) {
	// Create temp file
	tmpDir := t.TempDir()
	importMapPath := filepath.Join(tmpDir, "imports.toml")

	content := `[imports]
json = "github.com/ytnobody/json-calcium"
http = "ytnobody/http"
`
	if err := os.WriteFile(importMapPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write import map: %v", err)
	}

	imports, err := LoadImportMap(importMapPath)
	if err != nil {
		t.Fatalf("LoadImportMap failed: %v", err)
	}

	if imports["json"] != "github.com/ytnobody/json-calcium" {
		t.Errorf("expected json to map to github.com/ytnobody/json-calcium, got %s", imports["json"])
	}

	if imports["http"] != "ytnobody/http" {
		t.Errorf("expected http to map to ytnobody/http, got %s", imports["http"])
	}
}

func TestLoadLockFile(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "calcium.lock")

	content := `[modules."ytnobody/json"]
version = "1.0.0"
url = "https://raw.githubusercontent.com/ytnobody/json-calcium/v1.0.0/"
[modules."ytnobody/json".files]
"mod.ca" = "abc123def456"
"parser.ca" = "789xyz000"
`
	if err := os.WriteFile(lockPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write lock file: %v", err)
	}

	lf, err := LoadLockFile(lockPath)
	if err != nil {
		t.Fatalf("LoadLockFile failed: %v", err)
	}

	lock, ok := lf.Modules["ytnobody/json"]
	if !ok {
		t.Fatalf("expected ytnobody/json in lock file")
	}

	if lock.Version != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", lock.Version)
	}

	if lock.Files["mod.ca"] != "abc123def456" {
		t.Errorf("expected mod.ca hash abc123def456, got %s", lock.Files["mod.ca"])
	}
}

func TestRunOptionsResolveModuleName(t *testing.T) {
	opts := &RunOptions{
		ImportMap: map[string]string{
			"json":  "github.com/ytnobody/json-calcium",
			"utils": "myorg/utils",
		},
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"json", "github.com/ytnobody/json-calcium"},
		{"utils", "myorg/utils"},
		{"unknown", "unknown"}, // no mapping, returns original
		{"core.math", "core.math"},
	}

	for _, tt := range tests {
		result := opts.ResolveModuleName(tt.input)
		if result != tt.expected {
			t.Errorf("ResolveModuleName(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestVerifyFileHash(t *testing.T) {
	// SHA256 of "hello world\n" is known
	content := []byte("hello world\n")
	// sha256sum: a948904f2f0f479b8f8197694b30184b0d2ed1c1cd2a1ec0fb85d299a192a447
	expectedHash := "a948904f2f0f479b8f8197694b30184b0d2ed1c1cd2a1ec0fb85d299a192a447"

	if !VerifyFileHash(content, expectedHash) {
		t.Error("VerifyFileHash should return true for matching hash")
	}

	if VerifyFileHash(content, "wronghash") {
		t.Error("VerifyFileHash should return false for non-matching hash")
	}
}

func TestDefaultRunOptions(t *testing.T) {
	opts := DefaultRunOptions()

	if opts == nil {
		t.Fatal("DefaultRunOptions returned nil")
	}

	if opts.ImportMap == nil {
		t.Error("ImportMap should not be nil")
	}

	if opts.CachedOnly {
		t.Error("CachedOnly should be false by default")
	}

	if opts.LockFile != nil {
		t.Error("LockFile should be nil by default")
	}
}

func TestRunOptionsNil(t *testing.T) {
	var opts *RunOptions = nil

	// Should not panic
	result := opts.ResolveModuleName("test")
	if result != "test" {
		t.Errorf("nil RunOptions.ResolveModuleName should return original, got %s", result)
	}

	lock := opts.GetModuleLock("test")
	if lock != nil {
		t.Error("nil RunOptions.GetModuleLock should return nil")
	}
}
