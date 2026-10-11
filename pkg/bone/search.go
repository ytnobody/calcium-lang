package bone

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// SearchResult is a single module entry from the registry search index.
type SearchResult struct {
	Author        string   `json:"author"`
	Name          string   `json:"name"`
	Path          string   `json:"path"`
	Description   string   `json:"description"`
	Keywords      []string `json:"keywords"`
	LatestVersion string   `json:"latest_version"`
	LatestTag     string   `json:"latest_tag"`
}

// Search fetches the registry search index and returns modules matching query.
// An empty query returns every module in the index.
func Search(query string) ([]SearchResult, error) {
	registryURL := GetRegistryURL()
	url := registryURL + "/search.json"

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch search index: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("search index not found at %s (is the registry populated?)", url)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to fetch search index: HTTP %d", resp.StatusCode)
	}

	var modules []SearchResult
	if err := json.NewDecoder(resp.Body).Decode(&modules); err != nil {
		return nil, fmt.Errorf("failed to parse search index: %w", err)
	}

	return FilterSearch(modules, query), nil
}

// FilterSearch filters modules by a case-insensitive substring query, matching
// against author, name, description, and keywords. Supports "AUTHOR/name" specs.
func FilterSearch(modules []SearchResult, query string) []SearchResult {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return modules
	}

	var qAuthor, qName string
	if idx := strings.Index(q, "/"); idx >= 0 {
		qAuthor = q[:idx]
		qName = q[idx+1:]
	}

	out := make([]SearchResult, 0, len(modules))
	for _, m := range modules {
		if matches(m, q, qAuthor, qName) {
			out = append(out, m)
		}
	}
	return out
}

func matches(m SearchResult, q, qAuthor, qName string) bool {
	author := strings.ToLower(m.Author)
	name := strings.ToLower(m.Name)

	if qName != "" {
		// AUTHOR/name spec: author must match exactly, name a substring.
		return author == qAuthor && strings.Contains(name, qName)
	}

	if strings.Contains(author, q) || strings.Contains(name, q) ||
		strings.Contains(strings.ToLower(m.Description), q) {
		return true
	}
	for _, k := range m.Keywords {
		if strings.Contains(strings.ToLower(k), q) {
			return true
		}
	}
	return false
}
