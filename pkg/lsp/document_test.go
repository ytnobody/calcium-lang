package lsp

import (
	"strings"
	"testing"
)

func TestDocumentStore_OpenAndGet(t *testing.T) {
	store := NewDocumentStore()
	uri := "file:///test.ca"
	text := "x = 42;"
	doc := store.Open(uri, 1, text)
	if doc == nil {
		t.Fatal("Open returned nil")
	}
	got, ok := store.Get(uri)
	if !ok {
		t.Fatal("Get returned false")
	}
	if got.URI != uri {
		t.Errorf("URI mismatch: got %q want %q", got.URI, uri)
	}
}

func TestDocumentStore_Update(t *testing.T) {
	store := NewDocumentStore()
	uri := "file:///test.ca"
	store.Open(uri, 1, "x = 42;")
	updated := store.Update(uri, 2, "y = 99;")
	if updated.Version != 2 {
		t.Errorf("expected version 2, got %d", updated.Version)
	}
}

func TestDocumentStore_Close(t *testing.T) {
	store := NewDocumentStore()
	uri := "file:///test.ca"
	store.Open(uri, 1, "x = 42;")
	store.Close(uri)
	_, ok := store.Get(uri)
	if ok {
		t.Error("expected document to be removed after Close")
	}
}

func TestDocument_ParseAndSymbols(t *testing.T) {
	store := NewDocumentStore()
	uri := "file:///test.ca"
	src := `
func add(a, b) = a + b;
x = 42;
constraint Positive(n) = n > 0;
type Color = Red | Green | Blue;
`
	doc := store.Open(uri, 1, src)

	// Should have no parse errors
	if len(doc.ParseErrors) != 0 {
		t.Errorf("unexpected parse errors: %v", doc.ParseErrors)
	}

	// Check symbols
	if _, ok := doc.Symbols["add"]; !ok {
		t.Error("expected 'add' function in symbols")
	}
	if _, ok := doc.Symbols["x"]; !ok {
		t.Error("expected 'x' variable in symbols")
	}
	if _, ok := doc.Symbols["Positive"]; !ok {
		t.Error("expected 'Positive' constraint in symbols")
	}
	if _, ok := doc.Symbols["Color"]; !ok {
		t.Error("expected 'Color' type in symbols")
	}
	// Variant constructors
	for _, name := range []string{"Red", "Green", "Blue"} {
		if _, ok := doc.Symbols[name]; !ok {
			t.Errorf("expected variant constructor %q in symbols", name)
		}
	}
}

func TestDocument_Diagnostics_NoErrors(t *testing.T) {
	store := NewDocumentStore()
	doc := store.Open("file:///ok.ca", 1, "x = 1;")
	diags := doc.Diagnostics()
	if len(diags) != 0 {
		t.Errorf("expected no diagnostics, got %d", len(diags))
	}
}

func TestDocument_Diagnostics_WithErrors(t *testing.T) {
	store := NewDocumentStore()
	// Intentionally broken: assignment to function without body
	doc := store.Open("file:///bad.ca", 1, "func (;")
	diags := doc.Diagnostics()
	if len(diags) == 0 {
		t.Error("expected diagnostics for invalid code, got none")
	}
}

func TestDocument_WordAtPosition(t *testing.T) {
	store := NewDocumentStore()
	src := "func add(a, b) = a + b;"
	doc := store.Open("file:///test.ca", 1, src)

	tests := []struct {
		pos  Position
		want string
	}{
		{Position{0, 0}, "func"},
		{Position{0, 5}, "add"},
		{Position{0, 9}, "a"},
	}
	for _, tt := range tests {
		got := doc.WordAtPosition(tt.pos)
		if got != tt.want {
			t.Errorf("WordAtPosition(%v) = %q, want %q", tt.pos, got, tt.want)
		}
	}
}

func TestDocument_CompletionAt_Keywords(t *testing.T) {
	store := NewDocumentStore()
	doc := store.Open("file:///test.ca", 1, "fu")
	items := doc.CompletionAt(Position{0, 2})

	found := false
	for _, item := range items {
		if item.Label == "func" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'func' keyword in completions")
	}
}

func TestDocument_CompletionAt_UserSymbols(t *testing.T) {
	store := NewDocumentStore()
	src := "func myFunc(x) = x + 1;\nmy"
	doc := store.Open("file:///test.ca", 1, src)
	items := doc.CompletionAt(Position{1, 2})

	found := false
	for _, item := range items {
		if item.Label == "myFunc" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'myFunc' in completions")
	}
}

func TestDocument_HoverAt_Keyword(t *testing.T) {
	store := NewDocumentStore()
	src := "func add(a, b) = a + b;"
	doc := store.Open("file:///test.ca", 1, src)
	hover := doc.HoverAt(Position{0, 0})
	if hover == nil {
		t.Fatal("expected hover info for 'func', got nil")
	}
	if !strings.Contains(hover.Contents.Value, "func") {
		t.Errorf("expected hover to mention 'func', got: %s", hover.Contents.Value)
	}
}

func TestDocument_HoverAt_UserSymbol(t *testing.T) {
	store := NewDocumentStore()
	src := "myVar = 42;\nmyVar"
	doc := store.Open("file:///test.ca", 1, src)
	hover := doc.HoverAt(Position{1, 0})
	if hover == nil {
		t.Fatal("expected hover info for 'myVar', got nil")
	}
}

func TestDocument_HoverAt_Unknown(t *testing.T) {
	store := NewDocumentStore()
	src := "unknownSymbol"
	doc := store.Open("file:///test.ca", 1, src)
	hover := doc.HoverAt(Position{0, 0})
	if hover != nil {
		t.Error("expected nil hover for unknown symbol")
	}
}

func TestDocument_DefinitionAt(t *testing.T) {
	store := NewDocumentStore()
	src := "func greet(name) = name;\ngreet"
	doc := store.Open("file:///test.ca", 1, src)
	loc := doc.DefinitionAt(Position{1, 0})
	if loc == nil {
		t.Fatal("expected location for 'greet', got nil")
	}
	if loc.URI != "file:///test.ca" {
		t.Errorf("unexpected URI: %s", loc.URI)
	}
}

func TestDocument_DefinitionAt_NotFound(t *testing.T) {
	store := NewDocumentStore()
	src := "unknown"
	doc := store.Open("file:///test.ca", 1, src)
	loc := doc.DefinitionAt(Position{0, 0})
	if loc != nil {
		t.Error("expected nil location for undefined symbol")
	}
}

func TestExtractPosition(t *testing.T) {
	tests := []struct {
		msg     string
		wantLn  int
		wantCol int
	}{
		{"[1:5] unexpected token", 1, 5},
		{"at line 3, column 10: error", 3, 10},
		{"line 7: something wrong", 7, 0},
		{"some other error", 0, 0},
	}
	for _, tt := range tests {
		ln, col := extractPosition(tt.msg)
		if ln != tt.wantLn || col != tt.wantCol {
			t.Errorf("extractPosition(%q) = (%d,%d), want (%d,%d)",
				tt.msg, ln, col, tt.wantLn, tt.wantCol)
		}
	}
}
