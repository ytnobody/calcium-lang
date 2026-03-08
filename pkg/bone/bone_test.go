package bone

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---- helpers ----

// tempHome creates a temporary HOME directory and sets HOME env var.
// Returns a cleanup function.
func tempHome(t *testing.T) (string, func()) {
	t.Helper()
	dir := t.TempDir()
	old := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	return dir, func() {
		os.Setenv("HOME", old)
	}
}

// tempProject creates a temp dir with a meta.toml and changes to it.
func tempProject(t *testing.T) (string, func()) {
	t.Helper()
	dir := t.TempDir()
	metaContent := `name = "test"
author = "testuser"
description = "Test module"
`
	if err := os.WriteFile(filepath.Join(dir, MetaFileName), []byte(metaContent), 0644); err != nil {
		t.Fatalf("failed to create meta.toml: %v", err)
	}
	old, _ := os.Getwd()
	os.Chdir(dir)
	return dir, func() {
		os.Chdir(old)
	}
}

// ---- config.go tests ----

func TestLoadBoneConfig_Default(t *testing.T) {
	_, cleanup := tempHome(t)
	defer cleanup()

	config, err := LoadBoneConfig()
	if err != nil {
		t.Fatalf("LoadBoneConfig() unexpected error: %v", err)
	}
	if config.RegistryURL != "" {
		t.Errorf("expected empty RegistryURL, got %q", config.RegistryURL)
	}
}

func TestSaveAndLoadBoneConfig(t *testing.T) {
	_, cleanup := tempHome(t)
	defer cleanup()

	cfg := &BoneConfig{RegistryURL: "https://example.com/registry"}
	if err := SaveBoneConfig(cfg); err != nil {
		t.Fatalf("SaveBoneConfig() unexpected error: %v", err)
	}

	loaded, err := LoadBoneConfig()
	if err != nil {
		t.Fatalf("LoadBoneConfig() unexpected error: %v", err)
	}
	if loaded.RegistryURL != cfg.RegistryURL {
		t.Errorf("RegistryURL = %q, want %q", loaded.RegistryURL, cfg.RegistryURL)
	}
}

func TestGetRegistryURL_Default(t *testing.T) {
	_, cleanup := tempHome(t)
	defer cleanup()

	url := GetRegistryURL()
	if url != DefaultRegistryURL {
		t.Errorf("GetRegistryURL() = %q, want %q", url, DefaultRegistryURL)
	}
}

func TestGetRegistryURL_Custom(t *testing.T) {
	_, cleanup := tempHome(t)
	defer cleanup()

	customURL := "https://custom.registry.example.com"
	cfg := &BoneConfig{RegistryURL: customURL}
	if err := SaveBoneConfig(cfg); err != nil {
		t.Fatalf("SaveBoneConfig() unexpected error: %v", err)
	}

	url := GetRegistryURL()
	if url != customURL {
		t.Errorf("GetRegistryURL() = %q, want %q", url, customURL)
	}
}

func TestSetConfig_RegistryURL(t *testing.T) {
	_, cleanup := tempHome(t)
	defer cleanup()

	customURL := "https://my.registry.example.com"
	if err := SetConfig("registry_url", customURL); err != nil {
		t.Fatalf("SetConfig() unexpected error: %v", err)
	}

	url := GetRegistryURL()
	if url != customURL {
		t.Errorf("GetRegistryURL() = %q, want %q", url, customURL)
	}
}

func TestSetConfig_Registry(t *testing.T) {
	_, cleanup := tempHome(t)
	defer cleanup()

	customURL := "https://another.registry.example.com"
	if err := SetConfig("registry", customURL); err != nil {
		t.Fatalf("SetConfig('registry') unexpected error: %v", err)
	}

	url := GetRegistryURL()
	if url != customURL {
		t.Errorf("GetRegistryURL() = %q, want %q", url, customURL)
	}
}

func TestSetConfig_UnknownKey(t *testing.T) {
	_, cleanup := tempHome(t)
	defer cleanup()

	err := SetConfig("unknown_key", "value")
	if err == nil {
		t.Error("expected error for unknown config key")
	}
}

func TestGetConfig_Default(t *testing.T) {
	_, cleanup := tempHome(t)
	defer cleanup()

	val, err := GetConfig("registry_url")
	if err != nil {
		t.Fatalf("GetConfig() unexpected error: %v", err)
	}
	if !strings.Contains(val, DefaultRegistryURL) {
		t.Errorf("GetConfig() = %q, expected to contain %q", val, DefaultRegistryURL)
	}
}

func TestGetConfig_Custom(t *testing.T) {
	_, cleanup := tempHome(t)
	defer cleanup()

	customURL := "https://custom.test.com"
	_ = SetConfig("registry_url", customURL)

	val, err := GetConfig("registry")
	if err != nil {
		t.Fatalf("GetConfig('registry') unexpected error: %v", err)
	}
	if val != customURL {
		t.Errorf("GetConfig() = %q, want %q", val, customURL)
	}
}

func TestGetConfig_UnknownKey(t *testing.T) {
	_, cleanup := tempHome(t)
	defer cleanup()

	_, err := GetConfig("unknown_key")
	if err == nil {
		t.Error("expected error for unknown config key")
	}
}

func TestShowConfig(t *testing.T) {
	_, cleanup := tempHome(t)
	defer cleanup()

	// Just verify it doesn't panic/error
	err := ShowConfig()
	if err != nil {
		t.Errorf("ShowConfig() unexpected error: %v", err)
	}
}

// ---- Dependency.UnmarshalTOML ----

func TestDependencyUnmarshalTOML_String(t *testing.T) {
	d := &Dependency{}
	err := d.UnmarshalTOML("^1.2.3")
	if err != nil {
		t.Fatalf("UnmarshalTOML(string) unexpected error: %v", err)
	}
	if d.Version != "^1.2.3" {
		t.Errorf("Version = %q, want %q", d.Version, "^1.2.3")
	}
}

func TestDependencyUnmarshalTOML_Map(t *testing.T) {
	d := &Dependency{}
	err := d.UnmarshalTOML(map[string]any{"version": ">=2.0.0"})
	if err != nil {
		t.Fatalf("UnmarshalTOML(map) unexpected error: %v", err)
	}
	if d.Version != ">=2.0.0" {
		t.Errorf("Version = %q, want %q", d.Version, ">=2.0.0")
	}
}

func TestDependencyUnmarshalTOML_MapNoVersion(t *testing.T) {
	d := &Dependency{}
	err := d.UnmarshalTOML(map[string]any{})
	if err != nil {
		t.Fatalf("UnmarshalTOML(empty map) unexpected error: %v", err)
	}
	if d.Version != "" {
		t.Errorf("Version = %q, want empty", d.Version)
	}
}

// ---- LoadMeta / SaveMeta ----

func TestSaveAndLoadMeta(t *testing.T) {
	dir := t.TempDir()

	meta := &Meta{
		Name:        "mymod",
		Author:      "alice",
		Description: "My test module",
		License:     "MIT",
		Entry:       "mod.ca",
	}

	if err := SaveMeta(dir, meta); err != nil {
		t.Fatalf("SaveMeta() unexpected error: %v", err)
	}

	loaded, err := LoadMeta(dir)
	if err != nil {
		t.Fatalf("LoadMeta() unexpected error: %v", err)
	}
	if loaded.Name != meta.Name {
		t.Errorf("Name = %q, want %q", loaded.Name, meta.Name)
	}
	if loaded.Author != meta.Author {
		t.Errorf("Author = %q, want %q", loaded.Author, meta.Author)
	}
}

func TestLoadMeta_NotFound(t *testing.T) {
	dir := t.TempDir()
	_, err := LoadMeta(dir)
	if err == nil {
		t.Error("expected error for missing meta.toml")
	}
}

// ---- LoadLockFile / SaveLockFile ----

func TestSaveAndLoadLockFile(t *testing.T) {
	dir := t.TempDir()

	lock := &LockFile{
		Modules: []LockedModule{
			{Name: "mymod", Author: "alice", Version: "1.0.0", Commit: "abc12345"},
		},
	}

	if err := SaveLockFile(dir, lock); err != nil {
		t.Fatalf("SaveLockFile() unexpected error: %v", err)
	}

	loaded, err := LoadLockFile(dir)
	if err != nil {
		t.Fatalf("LoadLockFile() unexpected error: %v", err)
	}
	if len(loaded.Modules) != 1 {
		t.Fatalf("len(Modules) = %d, want 1", len(loaded.Modules))
	}
	if loaded.Modules[0].Name != "mymod" {
		t.Errorf("Name = %q, want %q", loaded.Modules[0].Name, "mymod")
	}
	if loaded.Modules[0].Commit != "abc12345" {
		t.Errorf("Commit = %q, want %q", loaded.Modules[0].Commit, "abc12345")
	}
}

func TestLoadLockFile_NotFound(t *testing.T) {
	dir := t.TempDir()
	lock, err := LoadLockFile(dir)
	if err != nil {
		t.Fatalf("LoadLockFile() unexpected error for missing file: %v", err)
	}
	if len(lock.Modules) != 0 {
		t.Errorf("expected empty modules, got %d", len(lock.Modules))
	}
}

// ---- FindProjectRoot ----

func TestFindProjectRoot_Found(t *testing.T) {
	dir := t.TempDir()
	// Create meta.toml
	if err := os.WriteFile(filepath.Join(dir, MetaFileName), []byte(`name = "test"`), 0644); err != nil {
		t.Fatalf("failed to create meta.toml: %v", err)
	}

	old, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(old)

	root, err := FindProjectRoot()
	if err != nil {
		t.Fatalf("FindProjectRoot() unexpected error: %v", err)
	}
	if root == "" {
		t.Error("expected project root, got empty string")
	}
}

func TestFindProjectRoot_NotFound(t *testing.T) {
	// Use a temp dir with no meta.toml
	dir := t.TempDir()
	old, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(old)

	root, err := FindProjectRoot()
	if err != nil {
		t.Fatalf("FindProjectRoot() unexpected error: %v", err)
	}
	if root != "" {
		t.Errorf("expected empty root, got %q", root)
	}
}

// ---- ParseModuleSpec ----

func TestParseModuleSpec(t *testing.T) {
	tests := []struct {
		spec          string
		wantAuthor    string
		wantName      string
		wantVersion   string
		wantErr       bool
	}{
		{"alice/mymod", "alice", "mymod", "", false},
		{"Alice/MyMod", "alice", "mymod", "", false},
		{"alice/mymod@1.0.0", "alice", "mymod", "1.0.0", false},
		{"alice/mymod@^2.0.0", "alice", "mymod", "^2.0.0", false},
		{"invalid", "", "", "", true},
		{"only/one@ver@extra", "only", "one", "ver@extra", false},
	}

	for _, tt := range tests {
		author, name, version, err := ParseModuleSpec(tt.spec)
		if tt.wantErr {
			if err == nil {
				t.Errorf("ParseModuleSpec(%q) expected error", tt.spec)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseModuleSpec(%q) unexpected error: %v", tt.spec, err)
			continue
		}
		if author != tt.wantAuthor {
			t.Errorf("ParseModuleSpec(%q).author = %q, want %q", tt.spec, author, tt.wantAuthor)
		}
		if name != tt.wantName {
			t.Errorf("ParseModuleSpec(%q).name = %q, want %q", tt.spec, name, tt.wantName)
		}
		if version != tt.wantVersion {
			t.Errorf("ParseModuleSpec(%q).version = %q, want %q", tt.spec, version, tt.wantVersion)
		}
	}
}

// ---- GetIndexPath ----

func TestGetIndexPath(t *testing.T) {
	tests := []struct {
		author   string
		name     string
		expected string
	}{
		{"alice", "mymod", "a/al/alice/mymod"},
		{"bob", "coolmod", "b/bo/bob/coolmod"},
		{"ab", "pkg", "a/ab/ab/pkg"},
		{"a", "mod", "a/a/a/mod"},
	}

	for _, tt := range tests {
		result := GetIndexPath(tt.author, tt.name)
		if result != tt.expected {
			t.Errorf("GetIndexPath(%q, %q) = %q, want %q", tt.author, tt.name, result, tt.expected)
		}
	}
}

// ---- updateLockEntry ----

func TestUpdateLockEntry_Add(t *testing.T) {
	lock := &LockFile{Modules: []LockedModule{}}
	updateLockEntry(lock, "alice", "mymod", "1.0.0", "abcd1234")

	if len(lock.Modules) != 1 {
		t.Fatalf("len(Modules) = %d, want 1", len(lock.Modules))
	}
	if lock.Modules[0].Author != "alice" {
		t.Errorf("Author = %q, want alice", lock.Modules[0].Author)
	}
	if lock.Modules[0].Version != "1.0.0" {
		t.Errorf("Version = %q, want 1.0.0", lock.Modules[0].Version)
	}
}

func TestUpdateLockEntry_Update(t *testing.T) {
	lock := &LockFile{
		Modules: []LockedModule{
			{Name: "mymod", Author: "alice", Version: "1.0.0", Commit: "old"},
			{Name: "other", Author: "bob", Version: "2.0.0", Commit: "xyz"},
		},
	}
	updateLockEntry(lock, "alice", "mymod", "1.1.0", "newcommit")

	if len(lock.Modules) != 2 {
		t.Fatalf("len(Modules) = %d, want 2", len(lock.Modules))
	}

	// Find alice/mymod
	var found *LockedModule
	for i := range lock.Modules {
		if lock.Modules[i].Author == "alice" && lock.Modules[i].Name == "mymod" {
			found = &lock.Modules[i]
			break
		}
	}
	if found == nil {
		t.Fatal("alice/mymod not found in lock file")
	}
	if found.Version != "1.1.0" {
		t.Errorf("Version = %q, want 1.1.0", found.Version)
	}
	if found.Commit != "newcommit" {
		t.Errorf("Commit = %q, want newcommit", found.Commit)
	}
}

// ---- List ----

func TestList_NoMetaToml(t *testing.T) {
	dir := t.TempDir()
	old, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(old)

	err := List()
	if err == nil {
		t.Error("expected error for missing meta.toml")
	}
}

func TestList_Empty(t *testing.T) {
	_, cleanup := tempProject(t)
	defer cleanup()

	// No lock file = no modules
	err := List()
	if err != nil {
		t.Errorf("List() unexpected error: %v", err)
	}
}

func TestList_WithModules(t *testing.T) {
	dir, cleanup := tempProject(t)
	defer cleanup()

	lock := &LockFile{
		Modules: []LockedModule{
			{Name: "mymod", Author: "alice", Version: "1.0.0", Commit: "abc12345678"},
			{Name: "other", Author: "bob", Version: "2.1.0", Commit: "def987"},
		},
	}
	if err := SaveLockFile(dir, lock); err != nil {
		t.Fatalf("SaveLockFile() unexpected error: %v", err)
	}

	err := List()
	if err != nil {
		t.Errorf("List() unexpected error: %v", err)
	}
}

// ---- Remove ----

func TestRemove_NoMetaToml(t *testing.T) {
	dir := t.TempDir()
	old, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(old)

	err := Remove("alice/mymod")
	if err == nil {
		t.Error("expected error for missing meta.toml")
	}
}

func TestRemove_ModuleNotInstalled(t *testing.T) {
	_, cleanup := tempProject(t)
	defer cleanup()

	err := Remove("alice/mymod")
	if err == nil {
		t.Error("expected error for module not installed")
	}
	if !strings.Contains(err.Error(), "not installed") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestRemove_InvalidSpec(t *testing.T) {
	_, cleanup := tempProject(t)
	defer cleanup()

	err := Remove("invalid-module-spec")
	if err == nil {
		t.Error("expected error for invalid module spec")
	}
}

func TestRemove_Success(t *testing.T) {
	dir, cleanup := tempProject(t)
	defer cleanup()

	// Create lock file with a module
	lock := &LockFile{
		Modules: []LockedModule{
			{Name: "mymod", Author: "alice", Version: "1.0.0", Commit: "abc12345"},
		},
	}
	if err := SaveLockFile(dir, lock); err != nil {
		t.Fatalf("SaveLockFile() unexpected error: %v", err)
	}

	// Create module directory
	moduleDir := filepath.Join(dir, ModulesDir, "alice", "mymod")
	if err := os.MkdirAll(moduleDir, 0755); err != nil {
		t.Fatalf("failed to create module dir: %v", err)
	}

	err := Remove("alice/mymod")
	if err != nil {
		t.Errorf("Remove() unexpected error: %v", err)
	}

	// Verify module is removed from lock file
	updated, _ := LoadLockFile(dir)
	if len(updated.Modules) != 0 {
		t.Errorf("expected 0 modules in lock file, got %d", len(updated.Modules))
	}

	// Verify directory is removed
	if _, err := os.Stat(moduleDir); !os.IsNotExist(err) {
		t.Error("expected module directory to be removed")
	}
}

func TestRemove_WithAuthorDirCleanup(t *testing.T) {
	dir, cleanup := tempProject(t)
	defer cleanup()

	// Create lock file with a single module from author
	lock := &LockFile{
		Modules: []LockedModule{
			{Name: "mymod", Author: "alice", Version: "1.0.0", Commit: "abc12345"},
		},
	}
	if err := SaveLockFile(dir, lock); err != nil {
		t.Fatalf("SaveLockFile() unexpected error: %v", err)
	}

	// Create module directory (but don't create files - so author dir will be empty after removal)
	moduleDir := filepath.Join(dir, ModulesDir, "alice", "mymod")
	if err := os.MkdirAll(moduleDir, 0755); err != nil {
		t.Fatalf("failed to create module dir: %v", err)
	}

	err := Remove("alice/mymod")
	if err != nil {
		t.Errorf("Remove() unexpected error: %v", err)
	}

	// Verify author directory is cleaned up (empty)
	authorDir := filepath.Join(dir, ModulesDir, "alice")
	if _, err := os.Stat(authorDir); !os.IsNotExist(err) {
		t.Error("expected author directory to be cleaned up")
	}
}

// ---- Add errors (without network) ----

func TestAdd_InvalidSpec(t *testing.T) {
	_, cleanup := tempProject(t)
	defer cleanup()

	err := Add("invalid-spec", false)
	if err == nil {
		t.Error("expected error for invalid module spec")
	}
}

func TestAdd_NoMetaToml(t *testing.T) {
	dir := t.TempDir()
	old, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(old)

	err := Add("alice/mymod", false)
	if err == nil {
		t.Error("expected error for missing meta.toml with local install")
	}
}

// ---- Update errors (without network) ----

func TestUpdate_NoMetaToml(t *testing.T) {
	dir := t.TempDir()
	old, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(old)

	err := Update("")
	if err == nil {
		t.Error("expected error for missing meta.toml")
	}
}

func TestUpdate_NoModules(t *testing.T) {
	_, cleanup := tempProject(t)
	defer cleanup()

	err := Update("")
	if err != nil {
		t.Errorf("Update() with no modules unexpected error: %v", err)
	}
}

func TestUpdate_ModuleNotInstalled(t *testing.T) {
	dir, cleanup := tempProject(t)
	defer cleanup()

	// Pre-populate lock file with a different module so we don't hit "no modules" early return
	lock := &LockFile{
		Modules: []LockedModule{
			{Name: "othermod", Author: "bob", Version: "1.0.0", Commit: "xyz"},
		},
	}
	if err := SaveLockFile(dir, lock); err != nil {
		t.Fatalf("SaveLockFile() unexpected error: %v", err)
	}

	err := Update("alice/mymod")
	if err == nil {
		t.Fatal("expected error for module not installed")
	}
	if !strings.Contains(err.Error(), "not installed") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestUpdate_InvalidSpec(t *testing.T) {
	dir, cleanup := tempProject(t)
	defer cleanup()

	// Pre-populate lock file so we don't hit "no modules" early return
	lock := &LockFile{
		Modules: []LockedModule{
			{Name: "somemod", Author: "alice", Version: "1.0.0", Commit: "abc"},
		},
	}
	if err := SaveLockFile(dir, lock); err != nil {
		t.Fatalf("SaveLockFile() unexpected error: %v", err)
	}

	err := Update("invalid-spec")
	if err == nil {
		t.Fatal("expected error for invalid module spec")
	}
}

// ---- newMockServer creates a mock registry + source HTTP server ----

// newMockServer creates a test server that serves mock registry and source responses.
// The returned server URL is used as both registry URL and source URL base.
// The server handles:
//   - Any path ending with /meta.toml → returns mock meta.toml
//   - Any path ending with .toml (version files) → returns mock version info
//   - Any path ending with /mod.ca → returns mock file content
func newMockServer(t *testing.T) *httptest.Server {
	t.Helper()
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case strings.HasSuffix(path, "/meta.toml"):
			// mock meta.toml: point source_url back to the mock server
			fmt.Fprintf(w, `name = "testmod"
author = "testauthor"
description = "Test module"
license = "MIT"
entry = "mod.ca"
source_url = "%s"
`, srv.URL)
		case strings.HasSuffix(path, ".toml"):
			// mock version info
			fmt.Fprint(w, `version = "1.0.0"
tag = "v1.0.0"
commit = "abc1234567890"
published = "2024-01-01"
`)
		case strings.HasSuffix(path, "/mod.ca"):
			// mock module file
			fmt.Fprint(w, `// Test module content
namespace testmod;
`)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// setMockRegistry sets the registry URL in a temp HOME dir.
// Returns cleanup function.
func setMockRegistry(t *testing.T, registryURL string) func() {
	t.Helper()
	_, cleanup := tempHome(t)
	if err := SetConfig("registry_url", registryURL); err != nil {
		cleanup()
		t.Fatalf("SetConfig() failed: %v", err)
	}
	return cleanup
}

// ---- FetchMeta / FetchVersion / FetchFile tests ----

func TestFetchMeta_Success(t *testing.T) {
	srv := newMockServer(t)
	cleanup := setMockRegistry(t, srv.URL)
	defer cleanup()

	meta, err := FetchMeta("testauthor", "testmod")
	if err != nil {
		t.Fatalf("FetchMeta() unexpected error: %v", err)
	}
	if meta.Name != "testmod" {
		t.Errorf("Name = %q, want testmod", meta.Name)
	}
	if meta.Author != "testauthor" {
		t.Errorf("Author = %q, want testauthor", meta.Author)
	}
}

func TestFetchMeta_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()

	cleanup := setMockRegistry(t, srv.URL)
	defer cleanup()

	_, err := FetchMeta("nonexistent", "module")
	if err == nil {
		t.Error("expected error for 404 response")
	}
}

func TestFetchMeta_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	cleanup := setMockRegistry(t, srv.URL)
	defer cleanup()

	_, err := FetchMeta("author", "module")
	if err == nil {
		t.Error("expected error for 500 response")
	}
}

func TestFetchVersion_Latest(t *testing.T) {
	srv := newMockServer(t)
	cleanup := setMockRegistry(t, srv.URL)
	defer cleanup()

	info, err := FetchVersion("testauthor", "testmod", "")
	if err != nil {
		t.Fatalf("FetchVersion() unexpected error: %v", err)
	}
	if info.Version != "1.0.0" {
		t.Errorf("Version = %q, want 1.0.0", info.Version)
	}
}

func TestFetchVersion_Specific(t *testing.T) {
	srv := newMockServer(t)
	cleanup := setMockRegistry(t, srv.URL)
	defer cleanup()

	info, err := FetchVersion("testauthor", "testmod", "1.0.0")
	if err != nil {
		t.Fatalf("FetchVersion(specific) unexpected error: %v", err)
	}
	if info.Version != "1.0.0" {
		t.Errorf("Version = %q, want 1.0.0", info.Version)
	}
}

func TestFetchVersion_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()

	cleanup := setMockRegistry(t, srv.URL)
	defer cleanup()

	_, err := FetchVersion("author", "module", "")
	if err == nil {
		t.Error("expected error for 404 (latest not found)")
	}

	_, err = FetchVersion("author", "module", "9.9.9")
	if err == nil {
		t.Error("expected error for 404 (specific version not found)")
	}
}

func TestFetchVersion_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	cleanup := setMockRegistry(t, srv.URL)
	defer cleanup()

	_, err := FetchVersion("author", "module", "")
	if err == nil {
		t.Error("expected error for 500 response")
	}
}

func TestFetchFile_Success(t *testing.T) {
	srv := newMockServer(t)

	content, err := FetchFile(srv.URL, "abc1234", "mod.ca")
	if err != nil {
		t.Fatalf("FetchFile() unexpected error: %v", err)
	}
	if len(content) == 0 {
		t.Error("expected non-empty file content")
	}
}

func TestFetchFile_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()

	_, err := FetchFile(srv.URL, "abc", "mod.ca")
	if err == nil {
		t.Error("expected error for 404 response")
	}
}

// ---- Install with mock server ----

func TestInstall_WithModules(t *testing.T) {
	srv := newMockServer(t)
	cleanup := setMockRegistry(t, srv.URL)
	defer cleanup()

	dir, projCleanup := tempProject(t)
	defer projCleanup()

	// Create lock file with a module
	lock := &LockFile{
		Modules: []LockedModule{
			{Name: "testmod", Author: "testauthor", Version: "1.0.0", Commit: "abc1234567890"},
		},
	}
	if err := SaveLockFile(dir, lock); err != nil {
		t.Fatalf("SaveLockFile() unexpected error: %v", err)
	}

	err := Install()
	if err != nil {
		t.Errorf("Install() unexpected error: %v", err)
	}

	// Verify module was installed
	modFile := filepath.Join(dir, ModulesDir, "testauthor", "testmod", "mod.ca")
	if _, err := os.Stat(modFile); os.IsNotExist(err) {
		t.Error("expected mod.ca to be installed")
	}
}

// ---- Add with mock server ----

func TestAdd_Global(t *testing.T) {
	srv := newMockServer(t)
	_, homeCleanup := tempHome(t)
	defer homeCleanup()
	if err := SetConfig("registry_url", srv.URL); err != nil {
		t.Fatalf("SetConfig() failed: %v", err)
	}

	// Create temp dir to work from (no meta.toml needed for global)
	dir := t.TempDir()
	old, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(old)

	err := Add("testauthor/testmod", true)
	if err != nil {
		t.Errorf("Add() global unexpected error: %v", err)
	}
}

func TestAdd_Local(t *testing.T) {
	srv := newMockServer(t)
	cleanup := setMockRegistry(t, srv.URL)
	defer cleanup()

	dir, projCleanup := tempProject(t)
	defer projCleanup()

	err := Add("testauthor/testmod", false)
	if err != nil {
		t.Errorf("Add() local unexpected error: %v", err)
	}

	// Verify module was installed
	modFile := filepath.Join(dir, ModulesDir, "testauthor", "testmod", "mod.ca")
	if _, err := os.Stat(modFile); os.IsNotExist(err) {
		t.Error("expected mod.ca to be installed locally")
	}
}

// ---- VersionConstraint.String ----

func TestVersionConstraintString(t *testing.T) {
	tests := []struct {
		constraint string
		expected   string
	}{
		{"*", "*"},
		{"^1.0.0", "^1.0.0"},
		{"~1.2.3", "~1.2.3"},
		{">=2.0.0", ">=2.0.0"},
		{"1.0.0", "1.0.0"},
	}

	for _, tt := range tests {
		c, err := ParseConstraint(tt.constraint)
		if err != nil {
			t.Errorf("ParseConstraint(%q) error: %v", tt.constraint, err)
			continue
		}
		result := c.String()
		if result != tt.expected {
			t.Errorf("VersionConstraint(%q).String() = %q, want %q", tt.constraint, result, tt.expected)
		}
	}
}

// ---- Init tests ----

func TestInit_AlreadyExists(t *testing.T) {
	dir := t.TempDir()
	// Pre-create meta.toml
	if err := os.WriteFile(filepath.Join(dir, MetaFileName), []byte(`name = "existing"`), 0644); err != nil {
		t.Fatalf("failed to create meta.toml: %v", err)
	}

	old, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(old)

	err := Init("")
	if err == nil {
		t.Error("expected error when meta.toml already exists")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestInit_WithName_AlreadyExists(t *testing.T) {
	dir := t.TempDir()

	old, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(old)

	// Create subdir with meta.toml
	subdir := filepath.Join(dir, "mymodule")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subdir, MetaFileName), []byte(`name = "existing"`), 0644); err != nil {
		t.Fatalf("failed to create meta.toml: %v", err)
	}

	err := Init("mymodule")
	if err == nil {
		t.Error("expected error when meta.toml already exists in named dir")
	}
}

func TestInit_WithStdin(t *testing.T) {
	dir := t.TempDir()
	old, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(old)

	// Replace os.Stdin with a pipe
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() failed: %v", err)
	}
	oldStdin := os.Stdin
	os.Stdin = r
	defer func() {
		os.Stdin = oldStdin
		r.Close()
	}()

	// Write stdin input: module name, author, description, license, keywords
	go func() {
		defer w.Close()
		fmt.Fprintf(w, "\n")         // accept default module name
		fmt.Fprintf(w, "testauthor\n") // author
		fmt.Fprintf(w, "A test module\n") // description
		fmt.Fprintf(w, "Apache-2.0\n")   // license
		fmt.Fprintf(w, "test,module\n")  // keywords
	}()

	err = Init("")
	if err != nil {
		t.Errorf("Init() unexpected error: %v", err)
	}

	// Verify meta.toml was created
	if _, err := os.Stat(filepath.Join(dir, MetaFileName)); os.IsNotExist(err) {
		t.Error("expected meta.toml to be created")
	}
}

func TestInit_WithNameAndStdin(t *testing.T) {
	dir := t.TempDir()
	old, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(old)

	// Replace os.Stdin with a pipe
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() failed: %v", err)
	}
	oldStdin := os.Stdin
	os.Stdin = r
	defer func() {
		os.Stdin = oldStdin
		r.Close()
	}()

	go func() {
		defer w.Close()
		fmt.Fprintf(w, "customname\n") // custom module name
		fmt.Fprintf(w, "myauthor\n")   // author
		fmt.Fprintf(w, "My module\n")  // description
		fmt.Fprintf(w, "\n")           // default license (MIT)
		fmt.Fprintf(w, "\n")           // no keywords
	}()

	err = Init("newmodule")
	if err != nil {
		t.Errorf("Init(name) unexpected error: %v", err)
	}

	// Verify meta.toml was created in the named directory
	metaPath := filepath.Join(dir, "newmodule", MetaFileName)
	if _, err := os.Stat(metaPath); os.IsNotExist(err) {
		t.Error("expected meta.toml to be created in named directory")
	}
}

func TestInit_NoAuthor(t *testing.T) {
	dir := t.TempDir()
	old, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(old)

	// Replace os.Stdin with a pipe
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() failed: %v", err)
	}
	oldStdin := os.Stdin
	os.Stdin = r
	defer func() {
		os.Stdin = oldStdin
		r.Close()
	}()

	// Write empty author (required field)
	go func() {
		defer w.Close()
		fmt.Fprintf(w, "\n") // module name
		fmt.Fprintf(w, "\n") // empty author → should error
	}()

	err = Init("")
	if err == nil {
		t.Error("expected error for empty author")
	}
	if !strings.Contains(err.Error(), "author is required") {
		t.Errorf("unexpected error message: %v", err)
	}
}
