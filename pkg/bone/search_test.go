package bone

import (
	"testing"
)

func testModules() []SearchResult {
	return []SearchResult{
		{
			Author:        "ytnobody",
			Name:          "json",
			Description:   "json parser / builder",
			Keywords:      []string{"json", "data"},
			LatestVersion: "1.0.0",
		},
		{
			Author:        "JOHNDOE",
			Name:          "http",
			Description:   "http client",
			Keywords:      []string{"http", "net"},
			LatestVersion: "0.2.1",
		},
		{
			Author:      "mydev",
			Name:        "util",
			Description: "utility helpers",
			Keywords:    []string{},
		},
	}
}

func TestFilterSearchEmptyQuery(t *testing.T) {
	mods := testModules()
	got := FilterSearch(mods, "")
	if len(got) != len(mods) {
		t.Fatalf("empty query should return all modules, got %d", len(got))
	}
}

func TestFilterSearchByName(t *testing.T) {
	got := FilterSearch(testModules(), "json")
	if len(got) != 1 || got[0].Name != "json" {
		t.Fatalf("expected 1 match for 'json', got %d", len(got))
	}
}

func TestFilterSearchCaseInsensitive(t *testing.T) {
	got := FilterSearch(testModules(), "JSON")
	if len(got) != 1 || got[0].Name != "json" {
		t.Fatalf("case-insensitive search failed, got %d", len(got))
	}
}

func TestFilterSearchByAuthor(t *testing.T) {
	got := FilterSearch(testModules(), "johndoe")
	if len(got) != 1 || got[0].Name != "http" {
		t.Fatalf("expected 1 match by author, got %d", len(got))
	}
}

func TestFilterSearchByKeyword(t *testing.T) {
	got := FilterSearch(testModules(), "data")
	if len(got) != 1 || got[0].Name != "json" {
		t.Fatalf("expected keyword match, got %d", len(got))
	}
}

func TestFilterSearchByDescription(t *testing.T) {
	got := FilterSearch(testModules(), "client")
	if len(got) != 1 || got[0].Name != "http" {
		t.Fatalf("expected description match, got %d", len(got))
	}
}

func TestFilterSearchAuthorNameSpec(t *testing.T) {
	got := FilterSearch(testModules(), "ytnobody/json")
	if len(got) != 1 || got[0].Name != "json" {
		t.Fatalf("expected AUTHOR/name match, got %d", len(got))
	}
}

func TestFilterSearchNoMatch(t *testing.T) {
	got := FilterSearch(testModules(), "nonexistent99")
	if len(got) != 0 {
		t.Fatalf("expected no matches, got %d", len(got))
	}
}

func TestParseModuleSpecSearchCompat(t *testing.T) {
	// Ensure FilterSearch + ParseModuleSpec agree on the ref format.
	mods := testModules()
	matched := FilterSearch(mods, "ytnobody/json")
	if len(matched) != 1 {
		t.Fatalf("search did not match, got %d", len(matched))
	}
	author, name, _, err := ParseModuleSpec("ytnobody/json")
	if err != nil || author != "ytnobody" || name != "json" {
		t.Fatalf("ParseModuleSpec unexpected: %q/%q err=%v", author, name, err)
	}
}
