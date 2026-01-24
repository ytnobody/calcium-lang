package bone

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/BurntSushi/toml"
)

// ParseModuleSpec parses a module specification like "AUTHOR/name@version"
func ParseModuleSpec(spec string) (author, name, version string, err error) {
	// Split by @
	parts := strings.SplitN(spec, "@", 2)
	moduleRef := parts[0]
	if len(parts) == 2 {
		version = parts[1]
	}

	// Split author/name
	slashParts := strings.SplitN(moduleRef, "/", 2)
	if len(slashParts) != 2 {
		return "", "", "", fmt.Errorf("invalid module format: expected AUTHOR/name, got %s", moduleRef)
	}

	author = strings.ToLower(slashParts[0])
	name = strings.ToLower(slashParts[1])

	return author, name, version, nil
}

// GetIndexPath returns the Boneyard index path for a module
func GetIndexPath(author, name string) string {
	// Path: A/AU/AUTHOR/name
	prefix1 := string(author[0])
	prefix2 := author[:min(2, len(author))]
	return fmt.Sprintf("%s/%s/%s/%s", prefix1, prefix2, author, name)
}

// FetchMeta fetches meta.toml from Boneyard
func FetchMeta(author, name string) (*Meta, error) {
	indexPath := GetIndexPath(author, name)
	url := fmt.Sprintf("%s/%s/meta.toml", BoneyardRawURL, indexPath)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch meta.toml: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("module %s/%s not found in Boneyard", author, name)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to fetch meta.toml: HTTP %d", resp.StatusCode)
	}

	var meta Meta
	if _, err := toml.NewDecoder(resp.Body).Decode(&meta); err != nil {
		return nil, fmt.Errorf("failed to parse meta.toml: %w", err)
	}

	return &meta, nil
}

// FetchVersion fetches version info from Boneyard
func FetchVersion(author, name, version string) (*VersionInfo, error) {
	indexPath := GetIndexPath(author, name)

	var filename string
	if version == "" || version == "latest" {
		filename = "latest.toml"
	} else {
		filename = version + ".toml"
	}

	url := fmt.Sprintf("%s/%s/%s", BoneyardRawURL, indexPath, filename)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch version info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		if version == "" || version == "latest" {
			return nil, fmt.Errorf("module %s/%s has no releases yet", author, name)
		}
		return nil, fmt.Errorf("version %s not found for %s/%s", version, author, name)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to fetch version info: HTTP %d", resp.StatusCode)
	}

	var info VersionInfo
	if _, err := toml.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to parse version info: %w", err)
	}

	return &info, nil
}

// FetchFile fetches a file from the module's source repository
func FetchFile(sourceURL, commit, filePath string) ([]byte, error) {
	// Convert GitHub URL to raw URL
	// https://github.com/author/repo -> https://raw.githubusercontent.com/author/repo/COMMIT/file
	rawURL := strings.Replace(sourceURL, "github.com", "raw.githubusercontent.com", 1)
	url := fmt.Sprintf("%s/%s/%s", rawURL, commit, filePath)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch %s: %w", filePath, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to fetch %s: HTTP %d", filePath, resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
