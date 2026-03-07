package bytecode

import "fmt"

// SourcePosition represents a position in source code
type SourcePosition struct {
	Line   int    // 1-based line number
	Column int    // 1-based column number
	File   string // source file name (optional)
}

// String returns a human-readable position string
func (p SourcePosition) String() string {
	if p.File != "" {
		return fmt.Sprintf("%s:%d:%d", p.File, p.Line, p.Column)
	}
	return fmt.Sprintf("line %d, column %d", p.Line, p.Column)
}

// IsValid returns true if this position has meaningful data
func (p SourcePosition) IsValid() bool {
	return p.Line > 0
}

// SourceMapEntry maps an instruction offset to a source position
type SourceMapEntry struct {
	Offset   int // byte offset in instruction stream
	Line     int
	Column   int
	FuncName string // function name for stack traces (empty for top-level)
}

// SourceMap maps bytecode instruction offsets to source positions
type SourceMap struct {
	File    string
	Entries []SourceMapEntry
}

// NewSourceMap creates a new empty source map
func NewSourceMap() *SourceMap {
	return &SourceMap{
		Entries: make([]SourceMapEntry, 0, 64),
	}
}

// Add records a source position for an instruction offset
func (sm *SourceMap) Add(offset, line, column int) {
	// Skip invalid positions
	if line <= 0 {
		return
	}
	// Deduplicate: if the last entry has the same line, skip
	if len(sm.Entries) > 0 {
		last := &sm.Entries[len(sm.Entries)-1]
		if last.Offset == offset && last.Line == line && last.Column == column {
			return
		}
	}
	sm.Entries = append(sm.Entries, SourceMapEntry{
		Offset: offset,
		Line:   line,
		Column: column,
	})
}

// AddWithFunc records a source position with function name context
func (sm *SourceMap) AddWithFunc(offset, line, column int, funcName string) {
	if line <= 0 {
		return
	}
	sm.Entries = append(sm.Entries, SourceMapEntry{
		Offset:   offset,
		Line:     line,
		Column:   column,
		FuncName: funcName,
	})
}

// Lookup finds the source position for a given instruction offset
// Uses binary search on sorted entries to find the closest entry <= offset
func (sm *SourceMap) Lookup(offset int) SourcePosition {
	if sm == nil || len(sm.Entries) == 0 {
		return SourcePosition{}
	}

	// Binary search for the largest entry offset <= target offset
	lo, hi := 0, len(sm.Entries)-1
	result := -1

	for lo <= hi {
		mid := (lo + hi) / 2
		if sm.Entries[mid].Offset <= offset {
			result = mid
			lo = mid + 1
		} else {
			hi = mid - 1
		}
	}

	if result < 0 {
		return SourcePosition{}
	}

	entry := sm.Entries[result]
	return SourcePosition{
		Line:   entry.Line,
		Column: entry.Column,
		File:   sm.File,
	}
}

// LookupEntry finds the full source map entry for a given instruction offset
func (sm *SourceMap) LookupEntry(offset int) *SourceMapEntry {
	if sm == nil || len(sm.Entries) == 0 {
		return nil
	}

	lo, hi := 0, len(sm.Entries)-1
	result := -1

	for lo <= hi {
		mid := (lo + hi) / 2
		if sm.Entries[mid].Offset <= offset {
			result = mid
			lo = mid + 1
		} else {
			hi = mid - 1
		}
	}

	if result < 0 {
		return nil
	}

	return &sm.Entries[result]
}
