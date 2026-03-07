package bytecode

import "testing"

func TestSourceMapAdd(t *testing.T) {
	sm := NewSourceMap()
	sm.File = "test.ca"

	sm.Add(0, 1, 1)
	sm.Add(3, 2, 5)
	sm.Add(10, 5, 1)

	if len(sm.Entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(sm.Entries))
	}

	if sm.Entries[0].Line != 1 || sm.Entries[0].Column != 1 {
		t.Errorf("entry 0: expected line 1, col 1, got line %d, col %d", sm.Entries[0].Line, sm.Entries[0].Column)
	}
	if sm.Entries[1].Line != 2 || sm.Entries[1].Column != 5 {
		t.Errorf("entry 1: expected line 2, col 5, got line %d, col %d", sm.Entries[1].Line, sm.Entries[1].Column)
	}
}

func TestSourceMapAddSkipsInvalidPosition(t *testing.T) {
	sm := NewSourceMap()
	sm.Add(0, 0, 0)  // line 0 is invalid
	sm.Add(0, -1, 1) // negative line is invalid

	if len(sm.Entries) != 0 {
		t.Errorf("expected 0 entries for invalid positions, got %d", len(sm.Entries))
	}
}

func TestSourceMapAddDeduplicates(t *testing.T) {
	sm := NewSourceMap()
	sm.Add(0, 1, 1)
	sm.Add(0, 1, 1) // duplicate

	if len(sm.Entries) != 1 {
		t.Errorf("expected 1 entry (deduplicated), got %d", len(sm.Entries))
	}
}

func TestSourceMapLookup(t *testing.T) {
	sm := NewSourceMap()
	sm.File = "test.ca"

	sm.Add(0, 1, 1)
	sm.Add(5, 3, 1)
	sm.Add(10, 5, 1)
	sm.Add(20, 10, 1)

	tests := []struct {
		offset       int
		expectedLine int
	}{
		{0, 1},  // exact match
		{3, 1},  // between entries, closest <= is 0->line 1
		{5, 3},  // exact match
		{7, 3},  // between entries, closest <= is 5->line 3
		{10, 5}, // exact match
		{15, 5}, // between entries
		{20, 10},
		{100, 10}, // beyond last entry
	}

	for _, tt := range tests {
		pos := sm.Lookup(tt.offset)
		if pos.Line != tt.expectedLine {
			t.Errorf("Lookup(%d): expected line %d, got line %d", tt.offset, tt.expectedLine, pos.Line)
		}
		if pos.File != "test.ca" {
			t.Errorf("Lookup(%d): expected file 'test.ca', got '%s'", tt.offset, pos.File)
		}
	}
}

func TestSourceMapLookupEmpty(t *testing.T) {
	sm := NewSourceMap()
	pos := sm.Lookup(0)
	if pos.IsValid() {
		t.Error("expected invalid position for empty source map")
	}
}

func TestSourceMapLookupNil(t *testing.T) {
	var sm *SourceMap
	pos := sm.Lookup(0)
	if pos.IsValid() {
		t.Error("expected invalid position for nil source map")
	}
}

func TestSourcePositionString(t *testing.T) {
	tests := []struct {
		pos      SourcePosition
		expected string
	}{
		{SourcePosition{Line: 1, Column: 5, File: "test.ca"}, "test.ca:1:5"},
		{SourcePosition{Line: 10, Column: 1}, "line 10, column 1"},
	}

	for _, tt := range tests {
		result := tt.pos.String()
		if result != tt.expected {
			t.Errorf("SourcePosition.String() = %q, expected %q", result, tt.expected)
		}
	}
}

func TestSourcePositionIsValid(t *testing.T) {
	valid := SourcePosition{Line: 1, Column: 1}
	if !valid.IsValid() {
		t.Error("expected valid position")
	}

	invalid := SourcePosition{Line: 0, Column: 0}
	if invalid.IsValid() {
		t.Error("expected invalid position")
	}
}

func TestSourceMapLookupEntry(t *testing.T) {
	sm := NewSourceMap()
	sm.AddWithFunc(0, 1, 1, "main")
	sm.AddWithFunc(5, 3, 1, "foo")

	entry := sm.LookupEntry(5)
	if entry == nil {
		t.Fatal("expected non-nil entry")
	}
	if entry.FuncName != "foo" {
		t.Errorf("expected func name 'foo', got '%s'", entry.FuncName)
	}

	entry = sm.LookupEntry(0)
	if entry == nil {
		t.Fatal("expected non-nil entry")
	}
	if entry.FuncName != "main" {
		t.Errorf("expected func name 'main', got '%s'", entry.FuncName)
	}
}
