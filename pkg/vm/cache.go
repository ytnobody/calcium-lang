package vm

import (
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// getCacheDir returns the path to the module cache directory
func getCacheDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current directory
		return filepath.Join(".", ".calcium", "cache")
	}
	return filepath.Join(home, ".calcium", "cache")
}

// getModuleCachePath returns the path to a specific module's cache directory
func getModuleCachePath(author, name string) string {
	return filepath.Join(getCacheDir(), author, name)
}

// ensureCacheDir creates the cache directory for a module if it doesn't exist
func ensureCacheDir(author, name string) error {
	return os.MkdirAll(getModuleCachePath(author, name), 0755)
}

// readCachedModule reads a module from the cache directory
// Returns the content if found, nil otherwise
func readCachedModule(cacheDir string) []byte {
	modPath := filepath.Join(cacheDir, "mod.ca")
	content, err := os.ReadFile(modPath)
	if err != nil {
		return nil
	}
	return content
}

// saveToCacheDir saves a module file to the cache directory
func saveToCacheDir(cacheDir, filename string, content []byte) error {
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(cacheDir, filename), content, 0644)
}

// httpGet performs an HTTP GET request and returns the response body
// Returns nil on any error
func httpGet(url string) []byte {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	return body
}

// fetchFromGitHubDirect attempts to fetch a module directly from GitHub
// It tries the following URL patterns in order:
// 1. github.com/{author}/{name}-calcium/main/mod.ca (suffix convention)
// 2. github.com/{author}/{name}/main/mod.ca (direct name)
// 3. github.com/{author}/{name}-calcium/master/mod.ca (fallback branch)
// 4. github.com/{author}/{name}/master/mod.ca (fallback branch)
func fetchFromGitHubDirect(author, name string) []byte {
	// Try main branch first with -calcium suffix
	url := "https://raw.githubusercontent.com/" + author + "/" + name + "-calcium/main/mod.ca"
	if content := httpGet(url); content != nil {
		return content
	}

	// Try main branch without suffix
	url = "https://raw.githubusercontent.com/" + author + "/" + name + "/main/mod.ca"
	if content := httpGet(url); content != nil {
		return content
	}

	// Try master branch with -calcium suffix (fallback)
	url = "https://raw.githubusercontent.com/" + author + "/" + name + "-calcium/master/mod.ca"
	if content := httpGet(url); content != nil {
		return content
	}

	// Try master branch without suffix (fallback)
	url = "https://raw.githubusercontent.com/" + author + "/" + name + "/master/mod.ca"
	return httpGet(url)
}

// fetchRawGitHub fetches a file from GitHub's raw content URL
func fetchRawGitHub(author, repo, branch, file string) []byte {
	url := "https://raw.githubusercontent.com/" + author + "/" + repo + "/" + branch + "/" + file
	return httpGet(url)
}

// parseGitHubURL parses a GitHub URL and returns author and repo
// Input: "github.com/ytnobody/json-calcium"
// Output: "ytnobody", "json-calcium"
func parseGitHubURL(url string) (author, repo string) {
	// Remove protocol prefix if present
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")

	// Split by /
	parts := strings.Split(url, "/")
	if len(parts) >= 3 {
		// parts[0] = "github.com"
		// parts[1] = author
		// parts[2] = repo
		return parts[1], parts[2]
	}
	return "", ""
}

// ============================================================
// Exported functions for calcium cache command
// ============================================================

// GetCacheDir returns the path to the module cache directory
func GetCacheDir() string {
	return getCacheDir()
}

// CachedModule represents information about a cached module
type CachedModule struct {
	Author   string
	Name     string
	Path     string
	Size     int64
	ModTime  time.Time
}

// ListCachedModules returns a list of all cached modules
func ListCachedModules() ([]CachedModule, error) {
	cacheDir := getCacheDir()
	var modules []CachedModule

	// Check if cache directory exists
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		return modules, nil // Empty list if no cache
	}

	// Walk through cache directory
	err := filepath.WalkDir(cacheDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Look for mod.ca files
		if !d.IsDir() && d.Name() == "mod.ca" {
			info, err := d.Info()
			if err != nil {
				return nil // Skip this file
			}

			// Extract author/name from path
			rel, err := filepath.Rel(cacheDir, path)
			if err != nil {
				return nil
			}
			parts := strings.Split(filepath.Dir(rel), string(filepath.Separator))
			if len(parts) >= 2 {
				modules = append(modules, CachedModule{
					Author:  parts[0],
					Name:    parts[1],
					Path:    filepath.Dir(path),
					Size:    info.Size(),
					ModTime: info.ModTime(),
				})
			}
		}
		return nil
	})

	return modules, err
}

// GetCacheSize returns the total size of the cache directory in bytes
func GetCacheSize() (int64, error) {
	cacheDir := getCacheDir()
	var totalSize int64

	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		return 0, nil
	}

	err := filepath.WalkDir(cacheDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			info, err := d.Info()
			if err == nil {
				totalSize += info.Size()
			}
		}
		return nil
	})

	return totalSize, err
}

// ClearCache removes all cached modules
func ClearCache() error {
	cacheDir := getCacheDir()
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		return nil // Nothing to clear
	}
	return os.RemoveAll(cacheDir)
}

// FetchAndCacheModule fetches a remote module and caches it
// Returns the cached content and any error
func FetchAndCacheModule(author, name string) ([]byte, error) {
	content := fetchFromGitHubDirect(author, name)
	if content == nil {
		return nil, fmt.Errorf("module not found: %s/%s", author, name)
	}

	// Save to cache
	cacheDir := getModuleCachePath(author, name)
	if err := saveToCacheDir(cacheDir, "mod.ca", content); err != nil {
		return content, fmt.Errorf("failed to cache module: %w", err)
	}

	return content, nil
}

// FetchAndCacheGitHubURL fetches a module from a GitHub URL and caches it
func FetchAndCacheGitHubURL(url string) ([]byte, error) {
	author, repo := parseGitHubURL(url)
	if author == "" || repo == "" {
		return nil, fmt.Errorf("invalid GitHub URL: %s", url)
	}

	// Try main branch first, then master
	content := fetchRawGitHub(author, repo, "main", "mod.ca")
	if content == nil {
		content = fetchRawGitHub(author, repo, "master", "mod.ca")
	}
	if content == nil {
		return nil, fmt.Errorf("module not found: %s", url)
	}

	// Save to cache
	cacheDir := getModuleCachePath(author, repo)
	if err := saveToCacheDir(cacheDir, "mod.ca", content); err != nil {
		return content, fmt.Errorf("failed to cache module: %w", err)
	}

	return content, nil
}

// IsModuleCached checks if a module is already cached
func IsModuleCached(author, name string) bool {
	cacheDir := getModuleCachePath(author, name)
	return readCachedModule(cacheDir) != nil
}
