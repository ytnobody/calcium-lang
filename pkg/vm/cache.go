package vm

import (
	"io"
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
