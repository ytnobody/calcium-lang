package vm

import (
	"bytes"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/ytnobody/calcium-lang/pkg/value"
)

// =============================================================================
// core.string builtin function tests
// =============================================================================

func TestBuiltinSplit(t *testing.T) {
	tests := []struct {
		name     string
		args     []value.Value
		wantErr  bool
		expected []string
	}{
		{
			name:     "basic split",
			args:     []value.Value{value.String("a,b,c"), value.String(",")},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "split by space",
			args:     []value.Value{value.String("hello world"), value.String(" ")},
			expected: []string{"hello", "world"},
		},
		{
			name:     "no match",
			args:     []value.Value{value.String("hello"), value.String(",")},
			expected: []string{"hello"},
		},
		{
			name:    "wrong arg count",
			args:    []value.Value{value.String("hello")},
			wantErr: true,
		},
		{
			name:    "non-string args",
			args:    []value.Value{value.Int(1), value.String(",")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := builtinSplit(tt.args...)
			if tt.wantErr {
				if result.Type != value.TYPE_FAILURE {
					t.Errorf("expected failure, got %v", result)
				}
				return
			}
			if result.Type != value.TYPE_ARRAY {
				t.Fatalf("expected array, got %s", result.Type)
			}
			arr := result.AsArray()
			if len(arr) != len(tt.expected) {
				t.Fatalf("expected %d elements, got %d", len(tt.expected), len(arr))
			}
			for i, v := range arr {
				if v.AsString() != tt.expected[i] {
					t.Errorf("element %d: expected %q, got %q", i, tt.expected[i], v.AsString())
				}
			}
		})
	}
}

func TestBuiltinJoin(t *testing.T) {
	tests := []struct {
		name     string
		args     []value.Value
		wantErr  bool
		expected string
	}{
		{
			name:     "join with comma",
			args:     []value.Value{value.Array([]value.Value{value.String("a"), value.String("b"), value.String("c")}), value.String(",")},
			expected: "a,b,c",
		},
		{
			name:     "join with space",
			args:     []value.Value{value.Array([]value.Value{value.String("hello"), value.String("world")}), value.String(" ")},
			expected: "hello world",
		},
		{
			name:    "wrong arg count",
			args:    []value.Value{value.Array([]value.Value{})},
			wantErr: true,
		},
		{
			name:    "non-array first arg",
			args:    []value.Value{value.String("hello"), value.String(",")},
			wantErr: true,
		},
		{
			name:    "non-string separator",
			args:    []value.Value{value.Array([]value.Value{}), value.Int(1)},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := builtinJoin(tt.args...)
			if tt.wantErr {
				if result.Type != value.TYPE_FAILURE {
					t.Errorf("expected failure, got %v", result)
				}
				return
			}
			if result.AsString() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result.AsString())
			}
		})
	}
}

func TestBuiltinTrim(t *testing.T) {
	tests := []struct {
		name     string
		args     []value.Value
		wantErr  bool
		expected string
	}{
		{
			name:     "trim spaces",
			args:     []value.Value{value.String("  hello  ")},
			expected: "hello",
		},
		{
			name:     "trim tabs and newlines",
			args:     []value.Value{value.String("\t hello \n")},
			expected: "hello",
		},
		{
			name:    "wrong arg count",
			args:    []value.Value{},
			wantErr: true,
		},
		{
			name:    "non-string arg",
			args:    []value.Value{value.Int(42)},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := builtinTrim(tt.args...)
			if tt.wantErr {
				if result.Type != value.TYPE_FAILURE {
					t.Errorf("expected failure, got %v", result)
				}
				return
			}
			if result.AsString() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result.AsString())
			}
		})
	}
}

func TestBuiltinUpperLower(t *testing.T) {
	// Upper
	result := builtinUpper(value.String("hello"))
	if result.AsString() != "HELLO" {
		t.Errorf("upper: expected HELLO, got %s", result.AsString())
	}
	result = builtinUpper()
	if result.Type != value.TYPE_FAILURE {
		t.Error("upper: expected failure for no args")
	}
	result = builtinUpper(value.Int(1))
	if result.Type != value.TYPE_FAILURE {
		t.Error("upper: expected failure for non-string")
	}

	// Lower
	result = builtinLower(value.String("HELLO"))
	if result.AsString() != "hello" {
		t.Errorf("lower: expected hello, got %s", result.AsString())
	}
	result = builtinLower()
	if result.Type != value.TYPE_FAILURE {
		t.Error("lower: expected failure for no args")
	}
	result = builtinLower(value.Int(1))
	if result.Type != value.TYPE_FAILURE {
		t.Error("lower: expected failure for non-string")
	}
}

func TestBuiltinStartsWith(t *testing.T) {
	tests := []struct {
		name     string
		args     []value.Value
		wantErr  bool
		expected bool
	}{
		{
			name:     "true case",
			args:     []value.Value{value.String("hello world"), value.String("hello")},
			expected: true,
		},
		{
			name:     "false case",
			args:     []value.Value{value.String("hello world"), value.String("world")},
			expected: false,
		},
		{
			name:    "wrong arg count",
			args:    []value.Value{value.String("hello")},
			wantErr: true,
		},
		{
			name:    "non-string args",
			args:    []value.Value{value.Int(1), value.String("h")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := builtinStartsWith(tt.args...)
			if tt.wantErr {
				if result.Type != value.TYPE_FAILURE {
					t.Errorf("expected failure, got %v", result)
				}
				return
			}
			if result.AsBool() != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result.AsBool())
			}
		})
	}
}

func TestBuiltinEndsWith(t *testing.T) {
	result := builtinEndsWith(value.String("hello world"), value.String("world"))
	if !result.AsBool() {
		t.Error("expected true")
	}
	result = builtinEndsWith(value.String("hello world"), value.String("hello"))
	if result.AsBool() {
		t.Error("expected false")
	}
	result = builtinEndsWith(value.String("hello"))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for wrong arg count")
	}
	result = builtinEndsWith(value.Int(1), value.String("x"))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-string")
	}
}

func TestBuiltinContains(t *testing.T) {
	result := builtinContains(value.String("hello world"), value.String("lo wo"))
	if !result.AsBool() {
		t.Error("expected true")
	}
	result = builtinContains(value.String("hello"), value.String("xyz"))
	if result.AsBool() {
		t.Error("expected false")
	}
	result = builtinContains(value.String("hello"))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for wrong arg count")
	}
	result = builtinContains(value.Int(1), value.String("x"))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-string")
	}
}

func TestBuiltinReplace(t *testing.T) {
	result := builtinReplace(value.String("hello world"), value.String("world"), value.String("go"))
	if result.AsString() != "hello go" {
		t.Errorf("expected 'hello go', got %q", result.AsString())
	}
	result = builtinReplace(value.String("aaa"), value.String("a"), value.String("b"))
	if result.AsString() != "bbb" {
		t.Errorf("expected 'bbb', got %q", result.AsString())
	}
	result = builtinReplace(value.String("hello"))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for wrong arg count")
	}
	result = builtinReplace(value.Int(1), value.String("a"), value.String("b"))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-string")
	}
}

func TestBuiltinSubstring(t *testing.T) {
	tests := []struct {
		name     string
		args     []value.Value
		wantErr  bool
		expected string
	}{
		{
			name:     "basic substring",
			args:     []value.Value{value.String("hello world"), value.Int(0), value.Int(5)},
			expected: "hello",
		},
		{
			name:     "substring to end",
			args:     []value.Value{value.String("hello world"), value.Int(6)},
			expected: "world",
		},
		{
			name:     "start beyond length",
			args:     []value.Value{value.String("hi"), value.Int(10)},
			expected: "",
		},
		{
			name:     "negative start clamped",
			args:     []value.Value{value.String("hello"), value.Int(-1), value.Int(3)},
			expected: "hel",
		},
		{
			name:     "start >= end",
			args:     []value.Value{value.String("hello"), value.Int(3), value.Int(2)},
			expected: "",
		},
		{
			name:     "end beyond length",
			args:     []value.Value{value.String("hi"), value.Int(0), value.Int(100)},
			expected: "hi",
		},
		{
			name:    "wrong arg count",
			args:    []value.Value{value.String("hello")},
			wantErr: true,
		},
		{
			name:    "non-string first arg",
			args:    []value.Value{value.Int(1), value.Int(0)},
			wantErr: true,
		},
		{
			name:    "non-int start",
			args:    []value.Value{value.String("hello"), value.String("0")},
			wantErr: true,
		},
		{
			name:    "non-int end",
			args:    []value.Value{value.String("hello"), value.Int(0), value.String("5")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := builtinSubstring(tt.args...)
			if tt.wantErr {
				if result.Type != value.TYPE_FAILURE {
					t.Errorf("expected failure, got %v", result)
				}
				return
			}
			if result.AsString() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result.AsString())
			}
		})
	}
}

func TestBuiltinCharAt(t *testing.T) {
	result := builtinCharAt(value.String("hello"), value.Int(0))
	if result.AsString() != "h" {
		t.Errorf("expected 'h', got %q", result.AsString())
	}
	result = builtinCharAt(value.String("hello"), value.Int(4))
	if result.AsString() != "o" {
		t.Errorf("expected 'o', got %q", result.AsString())
	}
	// Out of bounds
	result = builtinCharAt(value.String("hi"), value.Int(10))
	if result.Type != value.TYPE_NULL {
		t.Errorf("expected null for out of bounds, got %s", result.Type)
	}
	// Negative index
	result = builtinCharAt(value.String("hi"), value.Int(-1))
	if result.Type != value.TYPE_NULL {
		t.Errorf("expected null for negative index, got %s", result.Type)
	}
	// Error cases
	result = builtinCharAt(value.String("hello"))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for wrong arg count")
	}
	result = builtinCharAt(value.Int(1), value.Int(0))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-string")
	}
	result = builtinCharAt(value.String("hi"), value.String("0"))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-int index")
	}
}

func TestBuiltinStringIndexOf(t *testing.T) {
	result := builtinStringIndexOf(value.String("hello world"), value.String("world"))
	if result.AsInt() != 6 {
		t.Errorf("expected 6, got %d", result.AsInt())
	}
	result = builtinStringIndexOf(value.String("hello"), value.String("xyz"))
	if result.AsInt() != -1 {
		t.Errorf("expected -1, got %d", result.AsInt())
	}
	// Error cases
	result = builtinStringIndexOf(value.String("hello"))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for wrong arg count")
	}
	result = builtinStringIndexOf(value.Int(1), value.String("x"))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-string")
	}
}

// =============================================================================
// core.array builtin function tests
// =============================================================================

func TestBuiltinReverse(t *testing.T) {
	arr := value.Array([]value.Value{value.Int(1), value.Int(2), value.Int(3)})
	result := builtinReverse(arr)
	if result.Type != value.TYPE_ARRAY {
		t.Fatalf("expected array, got %s", result.Type)
	}
	reversed := result.AsArray()
	if len(reversed) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(reversed))
	}
	if reversed[0].AsInt() != 3 || reversed[1].AsInt() != 2 || reversed[2].AsInt() != 1 {
		t.Errorf("expected [3,2,1], got %v", reversed)
	}

	// Error cases
	result = builtinReverse()
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for no args")
	}
	result = builtinReverse(value.String("hello"))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-array")
	}
}

func TestBuiltinSlice(t *testing.T) {
	arr := value.Array([]value.Value{value.Int(1), value.Int(2), value.Int(3), value.Int(4), value.Int(5)})

	tests := []struct {
		name     string
		args     []value.Value
		wantErr  bool
		expected []int64
	}{
		{
			name:     "basic slice",
			args:     []value.Value{arr, value.Int(1), value.Int(3)},
			expected: []int64{2, 3},
		},
		{
			name:     "slice to end",
			args:     []value.Value{arr, value.Int(3)},
			expected: []int64{4, 5},
		},
		{
			name:     "start beyond length",
			args:     []value.Value{arr, value.Int(10)},
			expected: []int64{},
		},
		{
			name:     "negative start clamped",
			args:     []value.Value{arr, value.Int(-1), value.Int(2)},
			expected: []int64{1, 2},
		},
		{
			name:     "start >= end",
			args:     []value.Value{arr, value.Int(3), value.Int(1)},
			expected: []int64{},
		},
		{
			name:     "end beyond length",
			args:     []value.Value{arr, value.Int(3), value.Int(100)},
			expected: []int64{4, 5},
		},
		{
			name:    "wrong arg count",
			args:    []value.Value{arr},
			wantErr: true,
		},
		{
			name:    "non-array first arg",
			args:    []value.Value{value.String("hello"), value.Int(0)},
			wantErr: true,
		},
		{
			name:    "non-int start",
			args:    []value.Value{arr, value.String("0")},
			wantErr: true,
		},
		{
			name:    "non-int end",
			args:    []value.Value{arr, value.Int(0), value.String("3")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := builtinSlice(tt.args...)
			if tt.wantErr {
				if result.Type != value.TYPE_FAILURE {
					t.Errorf("expected failure, got %v", result)
				}
				return
			}
			if result.Type != value.TYPE_ARRAY {
				t.Fatalf("expected array, got %s", result.Type)
			}
			got := result.AsArray()
			if len(got) != len(tt.expected) {
				t.Fatalf("expected %d elements, got %d", len(tt.expected), len(got))
			}
			for i, v := range got {
				if v.AsInt() != tt.expected[i] {
					t.Errorf("element %d: expected %d, got %d", i, tt.expected[i], v.AsInt())
				}
			}
		})
	}
}

func TestBuiltinArrayIndexOf(t *testing.T) {
	arr := value.Array([]value.Value{value.Int(10), value.Int(20), value.Int(30)})

	result := builtinArrayIndexOf(arr, value.Int(20))
	if result.AsInt() != 1 {
		t.Errorf("expected 1, got %d", result.AsInt())
	}

	result = builtinArrayIndexOf(arr, value.Int(99))
	if result.AsInt() != -1 {
		t.Errorf("expected -1, got %d", result.AsInt())
	}

	// Error cases
	result = builtinArrayIndexOf(arr)
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for wrong arg count")
	}
	result = builtinArrayIndexOf(value.String("hello"), value.Int(1))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-array")
	}
}

func TestBuiltinFlatten(t *testing.T) {
	nested := value.Array([]value.Value{
		value.Int(1),
		value.Array([]value.Value{value.Int(2), value.Int(3)}),
		value.Int(4),
		value.Array([]value.Value{value.Int(5)}),
	})

	result := builtinFlatten(nested)
	if result.Type != value.TYPE_ARRAY {
		t.Fatalf("expected array, got %s", result.Type)
	}
	flat := result.AsArray()
	if len(flat) != 5 {
		t.Fatalf("expected 5 elements, got %d", len(flat))
	}
	for i, expected := range []int64{1, 2, 3, 4, 5} {
		if flat[i].AsInt() != expected {
			t.Errorf("element %d: expected %d, got %d", i, expected, flat[i].AsInt())
		}
	}

	// Error cases
	result = builtinFlatten()
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for no args")
	}
	result = builtinFlatten(value.String("hello"))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-array")
	}
}

func TestBuiltinUnique(t *testing.T) {
	arr := value.Array([]value.Value{value.Int(1), value.Int(2), value.Int(2), value.Int(3), value.Int(1)})
	result := builtinUnique(arr)
	if result.Type != value.TYPE_ARRAY {
		t.Fatalf("expected array, got %s", result.Type)
	}
	unique := result.AsArray()
	if len(unique) != 3 {
		t.Fatalf("expected 3 unique elements, got %d", len(unique))
	}

	// Error cases
	result = builtinUnique()
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for no args")
	}
	result = builtinUnique(value.Int(1))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-array")
	}
}

func TestBuiltinZip(t *testing.T) {
	arr1 := value.Array([]value.Value{value.Int(1), value.Int(2), value.Int(3)})
	arr2 := value.Array([]value.Value{value.String("a"), value.String("b"), value.String("c")})

	result := builtinZip(arr1, arr2)
	if result.Type != value.TYPE_ARRAY {
		t.Fatalf("expected array, got %s", result.Type)
	}
	zipped := result.AsArray()
	if len(zipped) != 3 {
		t.Fatalf("expected 3 pairs, got %d", len(zipped))
	}
	// Check first pair
	pair := zipped[0].AsArray()
	if pair[0].AsInt() != 1 || pair[1].AsString() != "a" {
		t.Errorf("first pair: expected [1, a], got %v", pair)
	}

	// Different lengths - use shorter
	short := value.Array([]value.Value{value.Int(1)})
	result = builtinZip(arr1, short)
	if len(result.AsArray()) != 1 {
		t.Errorf("expected 1 pair for different lengths, got %d", len(result.AsArray()))
	}

	// Error cases
	result = builtinZip(arr1)
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for wrong arg count")
	}
	result = builtinZip(value.String("a"), arr2)
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-array")
	}
}

func TestBuiltinTake(t *testing.T) {
	arr := value.Array([]value.Value{value.Int(1), value.Int(2), value.Int(3), value.Int(4), value.Int(5)})

	// Take first 3
	result := builtinTake(arr, value.Int(3))
	taken := result.AsArray()
	if len(taken) != 3 {
		t.Fatalf("expected 3, got %d", len(taken))
	}

	// Take 0
	result = builtinTake(arr, value.Int(0))
	if len(result.AsArray()) != 0 {
		t.Error("expected empty array for take 0")
	}

	// Take more than length
	result = builtinTake(arr, value.Int(100))
	if len(result.AsArray()) != 5 {
		t.Error("expected all elements for take > length")
	}

	// Negative take
	result = builtinTake(arr, value.Int(-1))
	if len(result.AsArray()) != 0 {
		t.Error("expected empty array for negative take")
	}

	// Error cases
	result = builtinTake(arr)
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for wrong arg count")
	}
	result = builtinTake(value.String("hello"), value.Int(1))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-array")
	}
	result = builtinTake(arr, value.String("1"))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-int count")
	}
}

func TestBuiltinDrop(t *testing.T) {
	arr := value.Array([]value.Value{value.Int(1), value.Int(2), value.Int(3), value.Int(4), value.Int(5)})

	// Drop first 2
	result := builtinDrop(arr, value.Int(2))
	dropped := result.AsArray()
	if len(dropped) != 3 {
		t.Fatalf("expected 3, got %d", len(dropped))
	}
	if dropped[0].AsInt() != 3 {
		t.Errorf("expected first element 3, got %d", dropped[0].AsInt())
	}

	// Drop 0
	result = builtinDrop(arr, value.Int(0))
	if len(result.AsArray()) != 5 {
		t.Error("expected all elements for drop 0")
	}

	// Drop more than length
	result = builtinDrop(arr, value.Int(100))
	if len(result.AsArray()) != 0 {
		t.Error("expected empty array for drop > length")
	}

	// Negative drop
	result = builtinDrop(arr, value.Int(-1))
	if len(result.AsArray()) != 5 {
		t.Error("expected all elements for negative drop")
	}

	// Error cases
	result = builtinDrop(arr)
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for wrong arg count")
	}
	result = builtinDrop(value.String("hello"), value.Int(1))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-array")
	}
	result = builtinDrop(arr, value.String("1"))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-int count")
	}
}

// =============================================================================
// core.io builtin function tests
// =============================================================================

func TestBuiltinPrintPrintln(t *testing.T) {
	vm := New(nil)
	var buf bytes.Buffer
	vm.SetOutput(&buf)

	vm.builtinPrint(value.String("hello"), value.String("world"))
	if buf.String() != "hello world" {
		t.Errorf("print: expected 'hello world', got %q", buf.String())
	}

	buf.Reset()
	vm.builtinPrintln(value.String("hello"), value.String("world"))
	if buf.String() != "hello world\n" {
		t.Errorf("println: expected 'hello world\\n', got %q", buf.String())
	}
}

func TestBuiltinFormat(t *testing.T) {
	result := builtinFormat(value.String("Hello, {}! You are {} years old."), value.String("Alice"), value.Int(30))
	if result.AsString() != "Hello, Alice! You are 30 years old." {
		t.Errorf("expected formatted string, got %q", result.AsString())
	}

	// No placeholders
	result = builtinFormat(value.String("no placeholders"), value.Int(1))
	if result.AsString() != "no placeholders" {
		t.Errorf("expected 'no placeholders', got %q", result.AsString())
	}

	// Error cases
	result = builtinFormat()
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for no args")
	}
	result = builtinFormat(value.Int(1))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-string first arg")
	}
}

func TestBuiltinReadWriteFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.txt")

	// Write
	result := builtinWriteFile(value.String(filePath), value.String("hello calcium"))
	if result.Type != value.TYPE_SUCCESS {
		t.Fatalf("write failed: %v", result)
	}

	// Read
	result = builtinReadFile(value.String(filePath))
	if result.Type != value.TYPE_SUCCESS {
		t.Fatalf("read failed: %v", result)
	}
	if result.AsSuccess().AsString() != "hello calcium" {
		t.Errorf("expected 'hello calcium', got %q", result.AsSuccess().AsString())
	}

	// Error: read non-existent
	result = builtinReadFile(value.String(filepath.Join(tmpDir, "nonexistent.txt")))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-existent file")
	}

	// Error: wrong args
	result = builtinReadFile()
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for no args")
	}
	result = builtinReadFile(value.Int(1))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-string path")
	}
	result = builtinWriteFile(value.String(filePath))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for missing content")
	}
	result = builtinWriteFile(value.Int(1), value.String("content"))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-string path")
	}
	result = builtinWriteFile(value.String(filePath), value.Int(1))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-string content")
	}
}

func TestBuiltinListDir(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "a.txt"), []byte("a"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "b.txt"), []byte("b"), 0644)

	result := builtinListDir(value.String(tmpDir))
	if result.Type != value.TYPE_SUCCESS {
		t.Fatalf("list_dir failed: %v", result)
	}
	arr := result.AsSuccess().AsArray()
	if len(arr) != 2 {
		t.Errorf("expected 2 entries, got %d", len(arr))
	}

	// Error cases
	result = builtinListDir()
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for no args")
	}
	result = builtinListDir(value.Int(1))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-string")
	}
	result = builtinListDir(value.String("/nonexistent_dir_xyz"))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-existent dir")
	}
}

func TestBuiltinMkdir(t *testing.T) {
	tmpDir := t.TempDir()
	newDir := filepath.Join(tmpDir, "a", "b", "c")

	result := builtinMkdir(value.String(newDir))
	if result.Type != value.TYPE_SUCCESS {
		t.Fatalf("mkdir failed: %v", result)
	}

	info, err := os.Stat(newDir)
	if err != nil || !info.IsDir() {
		t.Error("directory was not created")
	}

	// Error cases
	result = builtinMkdir()
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for no args")
	}
	result = builtinMkdir(value.Int(1))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-string")
	}
}

func TestBuiltinExists(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(filePath, []byte("test"), 0644)

	result := builtinExists(value.String(filePath))
	if !result.AsBool() {
		t.Error("expected true for existing file")
	}

	result = builtinExists(value.String(filepath.Join(tmpDir, "nonexistent")))
	if result.AsBool() {
		t.Error("expected false for non-existent file")
	}

	// Error cases
	result = builtinExists()
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for no args")
	}
	result = builtinExists(value.Int(1))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-string")
	}
}

func TestBuiltinDeleteFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "to_delete.txt")
	os.WriteFile(filePath, []byte("delete me"), 0644)

	result := builtinDeleteFile(value.String(filePath))
	if result.Type != value.TYPE_SUCCESS {
		t.Fatalf("delete failed: %v", result)
	}

	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Error("file should have been deleted")
	}

	// Delete non-existent
	result = builtinDeleteFile(value.String(filepath.Join(tmpDir, "nonexistent")))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-existent file")
	}

	// Error cases
	result = builtinDeleteFile()
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for no args")
	}
	result = builtinDeleteFile(value.Int(1))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-string")
	}
}

func TestBuiltinFileInfo(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "info_test.txt")
	os.WriteFile(filePath, []byte("hello"), 0644)

	result := builtinFileInfo(value.String(filePath))
	if result.Type != value.TYPE_SUCCESS {
		t.Fatalf("file_info failed: %v", result)
	}
	hash := result.AsSuccess().AsHash()

	name, ok := hash.Get(value.String("name"))
	if !ok || name.AsString() != "info_test.txt" {
		t.Errorf("expected name 'info_test.txt', got %v", name)
	}
	size, ok := hash.Get(value.String("size"))
	if !ok || size.AsInt() != 5 {
		t.Errorf("expected size 5, got %v", size)
	}
	isDir, ok := hash.Get(value.String("is_dir"))
	if !ok || isDir.AsBool() {
		t.Error("expected is_dir false")
	}

	// Directory
	result = builtinFileInfo(value.String(tmpDir))
	if result.Type != value.TYPE_SUCCESS {
		t.Fatalf("file_info on dir failed: %v", result)
	}
	hash = result.AsSuccess().AsHash()
	isDir, _ = hash.Get(value.String("is_dir"))
	if !isDir.AsBool() {
		t.Error("expected is_dir true for directory")
	}

	// Error cases
	result = builtinFileInfo()
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for no args")
	}
	result = builtinFileInfo(value.Int(1))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-string")
	}
	result = builtinFileInfo(value.String("/nonexistent_file_xyz"))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-existent file")
	}
}

// =============================================================================
// Regex builtin function tests
// =============================================================================

func makeRegex(pattern string) value.Value {
	re := regexp.MustCompile(pattern)
	return value.RegexVal(&value.Regex{Pattern: pattern, Re: re})
}

func TestBuiltinMatches(t *testing.T) {
	result := builtinMatches(value.String("hello123"), makeRegex(`\d+`))
	if !result.AsBool() {
		t.Error("expected true for matching string")
	}
	result = builtinMatches(value.String("hello"), makeRegex(`\d+`))
	if result.AsBool() {
		t.Error("expected false for non-matching string")
	}

	// Error cases
	result = builtinMatches(value.String("hello"))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for wrong arg count")
	}
	result = builtinMatches(value.Int(1), makeRegex(`\d+`))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-string first arg")
	}
	result = builtinMatches(value.String("hello"), value.String("pattern"))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-regex second arg")
	}
}

func TestBuiltinFind(t *testing.T) {
	result := builtinFind(value.String("hello123world"), makeRegex(`\d+`))
	if result.Type != value.TYPE_SUCCESS {
		t.Fatalf("expected success, got %v", result)
	}
	if result.AsSuccess().AsString() != "123" {
		t.Errorf("expected '123', got %q", result.AsSuccess().AsString())
	}

	// No match
	result = builtinFind(value.String("hello"), makeRegex(`\d+`))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for no match")
	}

	// Error cases
	result = builtinFind(value.String("hello"))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for wrong arg count")
	}
	result = builtinFind(value.Int(1), makeRegex(`\d+`))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-string")
	}
	result = builtinFind(value.String("hello"), value.String("x"))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-regex")
	}
}

func TestBuiltinFindAll(t *testing.T) {
	result := builtinFindAll(value.String("a1b2c3"), makeRegex(`\d`))
	if result.Type != value.TYPE_ARRAY {
		t.Fatalf("expected array, got %s", result.Type)
	}
	arr := result.AsArray()
	if len(arr) != 3 {
		t.Fatalf("expected 3 matches, got %d", len(arr))
	}
	if arr[0].AsString() != "1" || arr[1].AsString() != "2" || arr[2].AsString() != "3" {
		t.Errorf("unexpected matches: %v", arr)
	}

	// Error cases
	result = builtinFindAll(value.String("hello"))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for wrong arg count")
	}
	result = builtinFindAll(value.Int(1), makeRegex(`\d`))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-string")
	}
	result = builtinFindAll(value.String("hello"), value.String("x"))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-regex")
	}
}

func TestBuiltinRegexReplace(t *testing.T) {
	result := builtinRegexReplace(value.String("hello 123 world 456"), makeRegex(`\d+`), value.String("NUM"))
	if result.AsString() != "hello NUM world NUM" {
		t.Errorf("expected 'hello NUM world NUM', got %q", result.AsString())
	}

	// Error cases
	result = builtinRegexReplace(value.String("hello"))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for wrong arg count")
	}
	result = builtinRegexReplace(value.Int(1), makeRegex(`\d`), value.String("x"))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-string first arg")
	}
	result = builtinRegexReplace(value.String("hello"), value.String("x"), value.String("y"))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-regex second arg")
	}
	result = builtinRegexReplace(value.String("hello"), makeRegex(`x`), value.Int(1))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-string third arg")
	}
}

func TestBuiltinRegexSplit(t *testing.T) {
	result := builtinRegexSplit(value.String("a1b2c3d"), makeRegex(`\d`))
	if result.Type != value.TYPE_ARRAY {
		t.Fatalf("expected array, got %s", result.Type)
	}
	arr := result.AsArray()
	if len(arr) != 4 {
		t.Fatalf("expected 4 parts, got %d", len(arr))
	}
	if arr[0].AsString() != "a" || arr[1].AsString() != "b" || arr[2].AsString() != "c" || arr[3].AsString() != "d" {
		t.Errorf("unexpected parts: %v", arr)
	}

	// Error cases
	result = builtinRegexSplit(value.String("hello"))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for wrong arg count")
	}
	result = builtinRegexSplit(value.Int(1), makeRegex(`\d`))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-string")
	}
	result = builtinRegexSplit(value.String("hello"), value.String("x"))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-regex")
	}
}

func TestBuiltinCapture(t *testing.T) {
	result := builtinCapture(value.String("2024-01-15"), makeRegex(`(\d{4})-(\d{2})-(\d{2})`))
	if result.Type != value.TYPE_SUCCESS {
		t.Fatalf("expected success, got %v", result)
	}
	arr := result.AsSuccess().AsArray()
	if len(arr) != 4 { // full match + 3 groups
		t.Fatalf("expected 4 groups, got %d", len(arr))
	}
	if arr[0].AsString() != "2024-01-15" {
		t.Errorf("expected full match '2024-01-15', got %q", arr[0].AsString())
	}
	if arr[1].AsString() != "2024" || arr[2].AsString() != "01" || arr[3].AsString() != "15" {
		t.Error("unexpected capture groups")
	}

	// No match
	result = builtinCapture(value.String("hello"), makeRegex(`(\d+)`))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for no match")
	}

	// Error cases
	result = builtinCapture(value.String("hello"))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for wrong arg count")
	}
	result = builtinCapture(value.Int(1), makeRegex(`\d`))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-string")
	}
	result = builtinCapture(value.String("hello"), value.String("x"))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-regex")
	}
}

// =============================================================================
// Helper function tests
// =============================================================================

func TestMapKeys(t *testing.T) {
	m := map[string]value.Value{
		"a": value.Int(1),
		"b": value.Int(2),
		"c": value.Int(3),
	}
	keys := mapKeys(m)
	if len(keys) != 3 {
		t.Errorf("expected 3 keys, got %d", len(keys))
	}
}

// =============================================================================
// VM methods tests
// =============================================================================

func TestGetModule(t *testing.T) {
	vm := New(nil)
	mod, ok := vm.GetModule("core.io")
	if !ok {
		t.Fatal("expected core.io module to exist")
	}
	if mod.Name != "core.io" {
		t.Errorf("expected module name 'core.io', got %q", mod.Name)
	}

	_, ok = vm.GetModule("nonexistent")
	if ok {
		t.Error("expected nonexistent module to not exist")
	}
}

func TestSetSourcePath(t *testing.T) {
	vm := New(nil)
	vm.SetSourcePath("/some/path/file.ca")
	if vm.sourcePath != "/some/path/file.ca" {
		t.Errorf("expected source path to be set")
	}
}

func TestSetSourceMap(t *testing.T) {
	vm := New(nil)
	vm.SetSourceMap(nil)
	if vm.sourceMap != nil {
		t.Error("expected nil source map")
	}
}

func TestSetSourceInput(t *testing.T) {
	vm := New(nil)
	vm.SetSourceInput("test input")
	if vm.sourceInput != "test input" {
		t.Error("expected source input to be set")
	}
}

func TestSetOutput(t *testing.T) {
	vm := New(nil)
	var buf bytes.Buffer
	vm.SetOutput(&buf)
	if vm.getOutput() != &buf {
		t.Error("expected custom output writer")
	}
}

func TestGetOutputDefault(t *testing.T) {
	vm := New(nil)
	if vm.getOutput() != os.Stdout {
		t.Error("expected default output to be stdout")
	}
}

func TestSetRunOptions(t *testing.T) {
	vm := New(nil)
	opts := DefaultRunOptions()
	vm.SetRunOptions(opts)
	if vm.GetRunOptions() != opts {
		t.Error("expected run options to be set")
	}
}

func TestNewWithGlobals(t *testing.T) {
	globals := make([]value.Value, GlobalsSize)
	globals[0] = value.Int(42)
	vm := NewWithGlobals(nil, globals)
	if vm.globals[0].AsInt() != 42 {
		t.Error("expected globals to be preserved")
	}
}

func TestCloneForSpawn(t *testing.T) {
	vm := New(nil)
	vm.SetSourcePath("/test/path.ca")
	opts := DefaultRunOptions()
	vm.SetRunOptions(opts)

	clone := vm.CloneForSpawn()
	if clone.sourcePath != vm.sourcePath {
		t.Error("clone should share source path")
	}
	if clone.runOptions != vm.runOptions {
		t.Error("clone should share run options")
	}
	// Stack should be independent
	if &clone.stack[0] == &vm.stack[0] {
		t.Error("clone should have its own stack")
	}
}

func TestNewForREPL(t *testing.T) {
	globals := make([]value.Value, GlobalsSize)
	vm := NewForREPL(nil, globals)
	if vm == nil {
		t.Fatal("NewForREPL returned nil")
	}
	// Should have core.io module
	_, ok := vm.GetModule("core.io")
	if !ok {
		t.Error("expected core.io module in REPL VM")
	}
}

func TestGetSourceLine(t *testing.T) {
	vm := New(nil)
	vm.SetSourceInput("line1\nline2\nline3")

	tests := []struct {
		line     int
		expected string
	}{
		{1, "line1"},
		{2, "line2"},
		{3, "line3"},
		{0, ""},
		{-1, ""},
		{4, ""},
	}

	for _, tt := range tests {
		result := vm.getSourceLine(tt.line)
		if result != tt.expected {
			t.Errorf("getSourceLine(%d) = %q, want %q", tt.line, result, tt.expected)
		}
	}

	// Empty source
	vm2 := New(nil)
	if vm2.getSourceLine(1) != "" {
		t.Error("expected empty string for no source input")
	}
}

// =============================================================================
// Primitive math function tests (sin, cos, tan, log, exp)
// =============================================================================

func TestPrimitiveSin(t *testing.T) {
	result := primitiveSin(value.Float(0))
	if result.Type != value.TYPE_FLOAT {
		t.Fatalf("expected float, got %s", result.Type)
	}
	if result.AsFloat() != 0 {
		t.Errorf("sin(0) expected 0, got %f", result.AsFloat())
	}

	result = primitiveSin(value.Float(math.Pi / 2))
	if math.Abs(result.AsFloat()-1.0) > 1e-10 {
		t.Errorf("sin(pi/2) expected 1, got %f", result.AsFloat())
	}

	// Error cases
	result = primitiveSin()
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for no args")
	}
	result = primitiveSin(value.String("hello"))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-number")
	}
}

func TestPrimitiveCos(t *testing.T) {
	result := primitiveCos(value.Float(0))
	if math.Abs(result.AsFloat()-1.0) > 1e-10 {
		t.Errorf("cos(0) expected 1, got %f", result.AsFloat())
	}

	result = primitiveCos()
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for no args")
	}
	result = primitiveCos(value.String("x"))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-number")
	}
}

func TestPrimitiveTan(t *testing.T) {
	result := primitiveTan(value.Float(0))
	if math.Abs(result.AsFloat()) > 1e-10 {
		t.Errorf("tan(0) expected 0, got %f", result.AsFloat())
	}

	result = primitiveTan()
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for no args")
	}
	result = primitiveTan(value.String("x"))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-number")
	}
}

func TestPrimitiveLog(t *testing.T) {
	result := primitiveLog(value.Float(1))
	if math.Abs(result.AsFloat()) > 1e-10 {
		t.Errorf("log(1) expected 0, got %f", result.AsFloat())
	}

	result = primitiveLog(value.Float(math.E))
	if math.Abs(result.AsFloat()-1.0) > 1e-10 {
		t.Errorf("log(e) expected 1, got %f", result.AsFloat())
	}

	// Negative
	result = primitiveLog(value.Float(-1))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for negative arg")
	}

	// Zero
	result = primitiveLog(value.Float(0))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for zero arg")
	}

	result = primitiveLog()
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for no args")
	}
	result = primitiveLog(value.String("x"))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-number")
	}
}

func TestPrimitiveExp(t *testing.T) {
	result := primitiveExp(value.Float(0))
	if math.Abs(result.AsFloat()-1.0) > 1e-10 {
		t.Errorf("exp(0) expected 1, got %f", result.AsFloat())
	}

	result = primitiveExp(value.Float(1))
	if math.Abs(result.AsFloat()-math.E) > 1e-10 {
		t.Errorf("exp(1) expected e, got %f", result.AsFloat())
	}

	result = primitiveExp()
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for no args")
	}
	result = primitiveExp(value.String("x"))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-number")
	}
}

// =============================================================================
// Primitive I/O function tests
// =============================================================================

func TestPrimitivePrint(t *testing.T) {
	result := primitiveprint(value.String("hello"))
	if result.Type != value.TYPE_NULL {
		t.Errorf("expected null return, got %s", result.Type)
	}
	result = primitiveprint()
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for no args")
	}
}

func TestPrimitivePrintln(t *testing.T) {
	result := primitivePrintln(value.String("hello"))
	if result.Type != value.TYPE_NULL {
		t.Errorf("expected null return, got %s", result.Type)
	}
	result = primitivePrintln()
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for no args")
	}
}

func TestPrimitiveReadWriteFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.txt")

	// Write
	result := primitiveWriteFile(value.String(filePath), value.String("test content"))
	if result.Type != value.TYPE_SUCCESS {
		t.Fatalf("write failed: %v", result)
	}

	// Read
	result = primitiveReadFile(value.String(filePath))
	if result.Type != value.TYPE_SUCCESS {
		t.Fatalf("read failed: %v", result)
	}
	if result.AsSuccess().AsString() != "test content" {
		t.Errorf("expected 'test content', got %q", result.AsSuccess().AsString())
	}

	// Error cases
	result = primitiveReadFile()
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for no args")
	}
	result = primitiveReadFile(value.Int(1))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-string")
	}
	result = primitiveWriteFile()
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for no args")
	}
	result = primitiveWriteFile(value.Int(1), value.String("x"))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-string path")
	}
	result = primitiveWriteFile(value.String(filePath), value.Int(1))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-string content")
	}
}

func TestPrimitiveListDir(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "file.txt"), []byte("x"), 0644)

	result := primitiveListDir(value.String(tmpDir))
	if result.Type != value.TYPE_SUCCESS {
		t.Fatalf("list_dir failed: %v", result)
	}

	result = primitiveListDir()
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for no args")
	}
	result = primitiveListDir(value.Int(1))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-string")
	}
}

func TestPrimitiveMkdir(t *testing.T) {
	tmpDir := t.TempDir()
	newDir := filepath.Join(tmpDir, "new_dir")

	result := primitiveMkdir(value.String(newDir))
	if result.Type != value.TYPE_SUCCESS {
		t.Fatalf("mkdir failed: %v", result)
	}

	result = primitiveMkdir()
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for no args")
	}
	result = primitiveMkdir(value.Int(1))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-string")
	}
}

func TestPrimitiveExists(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "exists.txt")
	os.WriteFile(filePath, []byte("x"), 0644)

	result := primitiveExists(value.String(filePath))
	if !result.AsBool() {
		t.Error("expected true")
	}
	result = primitiveExists(value.String(filepath.Join(tmpDir, "nope")))
	if result.AsBool() {
		t.Error("expected false")
	}

	result = primitiveExists()
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for no args")
	}
	result = primitiveExists(value.Int(1))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-string")
	}
}

func TestPrimitiveDeleteFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "del.txt")
	os.WriteFile(filePath, []byte("x"), 0644)

	result := primitiveDeleteFile(value.String(filePath))
	if result.Type != value.TYPE_SUCCESS {
		t.Fatalf("delete failed: %v", result)
	}

	result = primitiveDeleteFile(value.String(filePath))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for already-deleted file")
	}

	result = primitiveDeleteFile()
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for no args")
	}
	result = primitiveDeleteFile(value.Int(1))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-string")
	}
}

func TestPrimitiveFileInfo(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "info.txt")
	os.WriteFile(filePath, []byte("hello"), 0644)

	result := primitiveFileInfo(value.String(filePath))
	if result.Type != value.TYPE_SUCCESS {
		t.Fatalf("file_info failed: %v", result)
	}
	hash := result.AsSuccess().AsHash()
	name, _ := hash.Get(value.String("name"))
	if name.AsString() != "info.txt" {
		t.Errorf("expected name 'info.txt', got %q", name.AsString())
	}

	result = primitiveFileInfo()
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for no args")
	}
	result = primitiveFileInfo(value.Int(1))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-string")
	}
	result = primitiveFileInfo(value.String("/nonexistent_xyz"))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-existent")
	}
}

// =============================================================================
// Primitive string function tests
// =============================================================================

func TestPrimitiveStrLen(t *testing.T) {
	result := primitiveStrLen(value.String("hello"))
	if result.AsInt() != 5 {
		t.Errorf("expected 5, got %d", result.AsInt())
	}

	result = primitiveStrLen()
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for no args")
	}
	result = primitiveStrLen(value.Int(1))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-string")
	}
}

func TestPrimitiveStrSplit(t *testing.T) {
	result := primitiveStrSplit(value.String("a,b,c"), value.String(","))
	arr := result.AsArray()
	if len(arr) != 3 || arr[0].AsString() != "a" {
		t.Errorf("unexpected split result: %v", arr)
	}

	result = primitiveStrSplit(value.String("hello"))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for wrong arg count")
	}
	result = primitiveStrSplit(value.Int(1), value.String(","))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-string")
	}
}

func TestPrimitiveStrJoin(t *testing.T) {
	arr := value.Array([]value.Value{value.String("a"), value.String("b")})
	result := primitiveStrJoin(arr, value.String("-"))
	if result.AsString() != "a-b" {
		t.Errorf("expected 'a-b', got %q", result.AsString())
	}

	result = primitiveStrJoin(arr)
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for wrong arg count")
	}
	result = primitiveStrJoin(value.String("x"), value.String("-"))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-array")
	}
	result = primitiveStrJoin(arr, value.Int(1))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-string sep")
	}
}

func TestPrimitiveStrTrim(t *testing.T) {
	result := primitiveStrTrim(value.String("  hello  "))
	if result.AsString() != "hello" {
		t.Errorf("expected 'hello', got %q", result.AsString())
	}

	result = primitiveStrTrim()
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for no args")
	}
	result = primitiveStrTrim(value.Int(1))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-string")
	}
}

func TestPrimitiveStrUpperLower(t *testing.T) {
	result := primitiveStrUpper(value.String("hello"))
	if result.AsString() != "HELLO" {
		t.Errorf("upper: expected 'HELLO', got %q", result.AsString())
	}
	result = primitiveStrLower(value.String("HELLO"))
	if result.AsString() != "hello" {
		t.Errorf("lower: expected 'hello', got %q", result.AsString())
	}

	// Error cases
	for _, fn := range []func(...value.Value) value.Value{primitiveStrUpper, primitiveStrLower} {
		if fn().Type != value.TYPE_FAILURE {
			t.Error("expected failure for no args")
		}
		if fn(value.Int(1)).Type != value.TYPE_FAILURE {
			t.Error("expected failure for non-string")
		}
	}
}

func TestPrimitiveStrReplace(t *testing.T) {
	result := primitiveStrReplace(value.String("hello world"), value.String("world"), value.String("go"))
	if result.AsString() != "hello go" {
		t.Errorf("expected 'hello go', got %q", result.AsString())
	}

	result = primitiveStrReplace(value.String("hello"))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for wrong arg count")
	}
	result = primitiveStrReplace(value.Int(1), value.String("a"), value.String("b"))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-string")
	}
}

func TestPrimitiveStrSubstring(t *testing.T) {
	result := primitiveStrSubstring(value.String("hello"), value.Int(1), value.Int(3))
	if result.AsString() != "el" {
		t.Errorf("expected 'el', got %q", result.AsString())
	}

	// Negative indices
	result = primitiveStrSubstring(value.String("hello"), value.Int(-2))
	if result.AsString() != "lo" {
		t.Errorf("expected 'lo', got %q", result.AsString())
	}

	// start > end
	result = primitiveStrSubstring(value.String("hello"), value.Int(3), value.Int(1))
	if result.AsString() != "" {
		t.Errorf("expected empty string, got %q", result.AsString())
	}

	// Error cases
	result = primitiveStrSubstring(value.String("hello"))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for wrong arg count")
	}
	result = primitiveStrSubstring(value.Int(1), value.Int(0))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-string")
	}
	result = primitiveStrSubstring(value.String("hello"), value.String("0"))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-int start")
	}
	result = primitiveStrSubstring(value.String("hello"), value.Int(0), value.String("3"))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-int end")
	}
}

func TestPrimitiveStrCharAt(t *testing.T) {
	result := primitiveStrCharAt(value.String("hello"), value.Int(0))
	if result.AsString() != "h" {
		t.Errorf("expected 'h', got %q", result.AsString())
	}

	// Negative index
	result = primitiveStrCharAt(value.String("hello"), value.Int(-1))
	if result.AsString() != "o" {
		t.Errorf("expected 'o', got %q", result.AsString())
	}

	// Out of bounds
	result = primitiveStrCharAt(value.String("hi"), value.Int(10))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for out of bounds")
	}

	// Error cases
	result = primitiveStrCharAt(value.String("hello"))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for wrong arg count")
	}
	result = primitiveStrCharAt(value.Int(1), value.Int(0))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-string")
	}
	result = primitiveStrCharAt(value.String("hello"), value.String("0"))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-int index")
	}
}

func TestPrimitiveStrIndexOf(t *testing.T) {
	result := primitiveStrIndexOf(value.String("hello"), value.String("ll"))
	if result.AsInt() != 2 {
		t.Errorf("expected 2, got %d", result.AsInt())
	}

	result = primitiveStrIndexOf(value.String("hello"), value.String("xyz"))
	if result.AsInt() != -1 {
		t.Errorf("expected -1, got %d", result.AsInt())
	}

	result = primitiveStrIndexOf(value.String("hello"))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for wrong arg count")
	}
	result = primitiveStrIndexOf(value.Int(1), value.String("x"))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-string")
	}
}

func TestPrimitiveStrStartsEndsWith(t *testing.T) {
	result := primitiveStrStartsWith(value.String("hello"), value.String("hel"))
	if !result.AsBool() {
		t.Error("expected true")
	}
	result = primitiveStrStartsWith(value.String("hello"), value.String("xyz"))
	if result.AsBool() {
		t.Error("expected false")
	}

	result = primitiveStrEndsWith(value.String("hello"), value.String("llo"))
	if !result.AsBool() {
		t.Error("expected true")
	}
	result = primitiveStrEndsWith(value.String("hello"), value.String("xyz"))
	if result.AsBool() {
		t.Error("expected false")
	}

	// Error cases
	for _, fn := range []func(...value.Value) value.Value{primitiveStrStartsWith, primitiveStrEndsWith} {
		if fn(value.String("hello")).Type != value.TYPE_FAILURE {
			t.Error("expected failure for wrong arg count")
		}
		if fn(value.Int(1), value.String("x")).Type != value.TYPE_FAILURE {
			t.Error("expected failure for non-string")
		}
	}
}

func TestPrimitiveStrContains(t *testing.T) {
	result := primitiveStrContains(value.String("hello world"), value.String("lo wo"))
	if !result.AsBool() {
		t.Error("expected true")
	}
	result = primitiveStrContains(value.String("hello"), value.String("xyz"))
	if result.AsBool() {
		t.Error("expected false")
	}

	result = primitiveStrContains(value.String("hello"))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for wrong arg count")
	}
	result = primitiveStrContains(value.Int(1), value.String("x"))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-string")
	}
}

// =============================================================================
// Primitive regex function tests
// =============================================================================

func TestPrimitiveRegexTest(t *testing.T) {
	re := makeRegex(`\d+`)
	result := primitiveRegexTest(value.String("abc123"), re)
	if !result.AsBool() {
		t.Error("expected true")
	}
	result = primitiveRegexTest(value.String("abc"), re)
	if result.AsBool() {
		t.Error("expected false")
	}

	result = primitiveRegexTest(value.String("x"))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for wrong arg count")
	}
	result = primitiveRegexTest(value.Int(1), re)
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-string")
	}
	result = primitiveRegexTest(value.String("x"), value.String("pattern"))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-regex")
	}
}

func TestPrimitiveRegexFind(t *testing.T) {
	re := makeRegex(`\d+`)
	result := primitiveRegexFind(value.String("abc123def"), re)
	if result.Type != value.TYPE_SUCCESS || result.AsSuccess().AsString() != "123" {
		t.Errorf("expected success(123), got %v", result)
	}

	result = primitiveRegexFind(value.String("abc"), re)
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for no match")
	}

	result = primitiveRegexFind(value.String("x"))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for wrong arg count")
	}
	result = primitiveRegexFind(value.Int(1), re)
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-string")
	}
	result = primitiveRegexFind(value.String("x"), value.String("y"))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-regex")
	}
}

func TestPrimitiveRegexFindAll(t *testing.T) {
	re := makeRegex(`\d+`)
	result := primitiveRegexFindAll(value.String("a1b22c333"), re)
	if result.Type != value.TYPE_ARRAY {
		t.Fatalf("expected array, got %s", result.Type)
	}
	arr := result.AsArray()
	if len(arr) != 3 || arr[0].AsString() != "1" || arr[1].AsString() != "22" || arr[2].AsString() != "333" {
		t.Errorf("unexpected result: %v", arr)
	}

	result = primitiveRegexFindAll(value.String("x"))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for wrong arg count")
	}
	result = primitiveRegexFindAll(value.Int(1), re)
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-string")
	}
	result = primitiveRegexFindAll(value.String("x"), value.String("y"))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-regex")
	}
}

func TestPrimitiveRegexReplace(t *testing.T) {
	re := makeRegex(`\d+`)
	result := primitiveRegexReplace(value.String("a1b2c3"), re, value.String("X"))
	if result.AsString() != "aXbXcX" {
		t.Errorf("expected 'aXbXcX', got %q", result.AsString())
	}

	result = primitiveRegexReplace(value.String("x"))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for wrong arg count")
	}
	result = primitiveRegexReplace(value.Int(1), re, value.String("x"))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-string first arg")
	}
	result = primitiveRegexReplace(value.String("x"), value.String("y"), value.String("z"))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-regex second arg")
	}
	result = primitiveRegexReplace(value.String("x"), re, value.Int(1))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-string third arg")
	}
}

func TestPrimitiveRegexReplaceFirst(t *testing.T) {
	re := makeRegex(`\d+`)
	result := primitiveRegexReplaceFirst(value.String("a1b2c3"), re, value.String("X"))
	if result.AsString() != "aXb2c3" {
		t.Errorf("expected 'aXb2c3', got %q", result.AsString())
	}

	// No match
	result = primitiveRegexReplaceFirst(value.String("abc"), re, value.String("X"))
	if result.AsString() != "abc" {
		t.Errorf("expected 'abc', got %q", result.AsString())
	}

	result = primitiveRegexReplaceFirst(value.String("x"))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for wrong arg count")
	}
	result = primitiveRegexReplaceFirst(value.Int(1), re, value.String("x"))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-string first arg")
	}
	result = primitiveRegexReplaceFirst(value.String("x"), value.String("y"), value.String("z"))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-regex second arg")
	}
	result = primitiveRegexReplaceFirst(value.String("x"), re, value.Int(1))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-string third arg")
	}
}

func TestPrimitiveRegexSplit(t *testing.T) {
	re := makeRegex(`[,;]`)
	result := primitiveRegexSplit(value.String("a,b;c"), re)
	arr := result.AsArray()
	if len(arr) != 3 || arr[0].AsString() != "a" || arr[1].AsString() != "b" || arr[2].AsString() != "c" {
		t.Errorf("unexpected result: %v", arr)
	}

	result = primitiveRegexSplit(value.String("x"))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for wrong arg count")
	}
	result = primitiveRegexSplit(value.Int(1), re)
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-string")
	}
	result = primitiveRegexSplit(value.String("x"), value.String("y"))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-regex")
	}
}

func TestPrimitiveRegexCapture(t *testing.T) {
	re := makeRegex(`(\d{4})-(\d{2})`)
	result := primitiveRegexCapture(value.String("date: 2024-01"), re)
	if result.Type != value.TYPE_SUCCESS {
		t.Fatalf("expected success, got %v", result)
	}
	arr := result.AsSuccess().AsArray()
	if len(arr) != 3 || arr[0].AsString() != "2024-01" || arr[1].AsString() != "2024" || arr[2].AsString() != "01" {
		t.Errorf("unexpected captures: %v", arr)
	}

	// No match
	result = primitiveRegexCapture(value.String("no digits"), re)
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for no match")
	}

	result = primitiveRegexCapture(value.String("x"))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for wrong arg count")
	}
	result = primitiveRegexCapture(value.Int(1), re)
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-string")
	}
	result = primitiveRegexCapture(value.String("x"), value.String("y"))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-regex")
	}
}

func TestPrimitiveRegexCompile(t *testing.T) {
	result := primitiveRegexCompile(value.String(`\d+`))
	if result.Type != value.TYPE_SUCCESS {
		t.Fatalf("expected success, got %v", result)
	}
	if result.AsSuccess().Type != value.TYPE_REGEX {
		t.Errorf("expected regex, got %s", result.AsSuccess().Type)
	}

	// With flags
	result = primitiveRegexCompile(value.String(`hello`), value.String("i"))
	if result.Type != value.TYPE_SUCCESS {
		t.Fatalf("expected success with flags, got %v", result)
	}

	// Invalid regex
	result = primitiveRegexCompile(value.String(`[invalid`))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for invalid regex")
	}

	// Error cases
	result = primitiveRegexCompile()
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for no args")
	}
	result = primitiveRegexCompile(value.Int(1))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-string pattern")
	}
	result = primitiveRegexCompile(value.String(`\d+`), value.Int(1))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-string flags")
	}
}

// =============================================================================
// Type conversion primitive tests
// =============================================================================

func TestPrimitiveToInt(t *testing.T) {
	tests := []struct {
		name     string
		arg      value.Value
		wantErr  bool
		expected int64
	}{
		{"int passthrough", value.Int(42), false, 42},
		{"float to int", value.Float(3.7), false, 3},
		{"string to int", value.String("123"), false, 123},
		{"bool true", value.Bool(true), false, 1},
		{"bool false", value.Bool(false), false, 0},
		{"invalid string", value.String("abc"), true, 0},
		{"null", value.Null(), true, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := primitiveToInt(tt.arg)
			if tt.wantErr {
				if result.Type != value.TYPE_FAILURE {
					t.Errorf("expected failure, got %v", result)
				}
				return
			}
			if result.Type != value.TYPE_SUCCESS {
				t.Fatalf("expected success, got %v", result)
			}
			if result.AsSuccess().AsInt() != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result.AsSuccess().AsInt())
			}
		})
	}

	// Wrong arg count
	result := primitiveToInt()
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for no args")
	}
}

func TestPrimitiveToFloat(t *testing.T) {
	tests := []struct {
		name     string
		arg      value.Value
		wantErr  bool
		expected float64
	}{
		{"float passthrough", value.Float(3.14), false, 3.14},
		{"int to float", value.Int(42), false, 42.0},
		{"string to float", value.String("3.14"), false, 3.14},
		{"invalid string", value.String("abc"), true, 0},
		{"null", value.Null(), true, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := primitiveToFloat(tt.arg)
			if tt.wantErr {
				if result.Type != value.TYPE_FAILURE {
					t.Errorf("expected failure, got %v", result)
				}
				return
			}
			if result.Type != value.TYPE_SUCCESS {
				t.Fatalf("expected success, got %v", result)
			}
			if result.AsSuccess().AsFloat() != tt.expected {
				t.Errorf("expected %f, got %f", tt.expected, result.AsSuccess().AsFloat())
			}
		})
	}

	result := primitiveToFloat()
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for no args")
	}
}

func TestPrimitiveTypeOf(t *testing.T) {
	tests := []struct {
		arg      value.Value
		expected string
	}{
		{value.Int(1), "int"},
		{value.Float(1.0), "float"},
		{value.String("hello"), "string"},
		{value.Bool(true), "bool"},
		{value.Null(), "null"},
	}

	for _, tt := range tests {
		result := primitiveTypeOf(tt.arg)
		if result.AsString() != tt.expected {
			t.Errorf("type_of(%v) = %q, want %q", tt.arg, result.AsString(), tt.expected)
		}
	}

	result := primitiveTypeOf()
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for no args")
	}
}

func TestPrimitiveNull(t *testing.T) {
	result := primitiveNull()
	if result.Type != value.TYPE_NULL {
		t.Errorf("expected null, got %s", result.Type)
	}
}

// =============================================================================
// TOML primitive tests
// =============================================================================

func TestPrimitiveTomlParse(t *testing.T) {
	tomlStr := `
title = "test"
[database]
port = 5432
`
	result := primitiveTomlParse(value.String(tomlStr))
	if result.Type != value.TYPE_SUCCESS {
		t.Fatalf("expected success, got %v", result)
	}
	hash := result.AsSuccess().AsHash()
	title, ok := hash.Get(value.String("title"))
	if !ok || title.AsString() != "test" {
		t.Errorf("expected title 'test', got %v", title)
	}

	// Invalid TOML
	result = primitiveTomlParse(value.String("invalid = = toml"))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for invalid TOML")
	}

	// Error cases
	result = primitiveTomlParse()
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for no args")
	}
	result = primitiveTomlParse(value.Int(1))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-string")
	}
}

func TestPrimitiveTomlStringify(t *testing.T) {
	hash := value.NewHash()
	hash.Set(value.String("name"), value.String("test"))
	hash.Set(value.String("value"), value.Int(42))

	result := primitiveTomlStringify(value.HashVal(hash))
	if result.Type != value.TYPE_SUCCESS {
		t.Fatalf("expected success, got %v", result)
	}

	// Error cases
	result = primitiveTomlStringify()
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for no args")
	}
	result = primitiveTomlStringify(value.String("hello"))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-hash")
	}
}

// =============================================================================
// goValueToCalcium / calciumValueToGo conversion tests
// =============================================================================

func TestGoValueToCalcium(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected value.Type
	}{
		{"nil", nil, value.TYPE_NULL},
		{"bool", true, value.TYPE_BOOL},
		{"int", 42, value.TYPE_INT},
		{"int64", int64(42), value.TYPE_INT},
		{"float64", 3.14, value.TYPE_FLOAT},
		{"string", "hello", value.TYPE_STRING},
		{"slice", []interface{}{1, "two"}, value.TYPE_ARRAY},
		{"map", map[string]interface{}{"a": 1}, value.TYPE_HASH},
		{"other", struct{}{}, value.TYPE_STRING}, // fallback to string
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := goValueToCalcium(tt.input)
			if result.Type != tt.expected {
				t.Errorf("expected type %s, got %s", tt.expected, result.Type)
			}
		})
	}
}

func TestCalciumValueToGo(t *testing.T) {
	// Null
	result := calciumValueToGo(value.Null())
	if result != nil {
		t.Error("expected nil for null")
	}

	// Bool
	result = calciumValueToGo(value.Bool(true))
	if result != true {
		t.Error("expected true")
	}

	// Int
	result = calciumValueToGo(value.Int(42))
	if result != int64(42) {
		t.Errorf("expected 42, got %v", result)
	}

	// Float
	result = calciumValueToGo(value.Float(3.14))
	if result != 3.14 {
		t.Errorf("expected 3.14, got %v", result)
	}

	// String
	result = calciumValueToGo(value.String("hello"))
	if result != "hello" {
		t.Errorf("expected hello, got %v", result)
	}

	// Array
	arr := value.Array([]value.Value{value.Int(1), value.Int(2)})
	result = calciumValueToGo(arr)
	goArr, ok := result.([]interface{})
	if !ok || len(goArr) != 2 {
		t.Error("expected []interface{} with 2 elements")
	}

	// Hash
	hash := value.NewHash()
	hash.Set(value.String("key"), value.String("value"))
	result = calciumValueToGo(value.HashVal(hash))
	goMap, ok := result.(map[string]interface{})
	if !ok || goMap["key"] != "value" {
		t.Error("expected map with key=value")
	}

	// Unknown type returns nil
	fn := &value.Function{Name: "test"}
	result = calciumValueToGo(value.Func(fn))
	if result != nil {
		t.Error("expected nil for unknown type")
	}
}

func TestHashToStringMap(t *testing.T) {
	hash := value.NewHash()
	hash.Set(value.String("key1"), value.String("val1"))
	hash.Set(value.String("key2"), value.String("val2"))
	hash.Set(value.String("key3"), value.Int(42)) // non-string value should be skipped

	result := hashToStringMap(hash)
	if len(result) != 2 {
		t.Errorf("expected 2 entries, got %d", len(result))
	}
	if result["key1"] != "val1" || result["key2"] != "val2" {
		t.Errorf("unexpected map: %v", result)
	}
}

// =============================================================================
// Stdlib tests
// =============================================================================

func TestIsStdlibModule(t *testing.T) {
	if !IsStdlibModule("core.math") {
		t.Error("core.math should be a stdlib module")
	}
	if IsStdlibModule("nonexistent.module") {
		t.Error("nonexistent.module should not be a stdlib module")
	}
}

func TestGetPrimitives(t *testing.T) {
	prims := GetPrimitives()
	if len(prims) == 0 {
		t.Error("expected primitives to be non-empty")
	}

	// Check a few known primitives exist
	expectedNames := []string{"__floor", "__ceil", "__round", "__sqrt", "__sin", "__cos", "__tan", "__log", "__exp"}
	for _, name := range expectedNames {
		if _, ok := prims[name]; !ok {
			t.Errorf("expected primitive %q to exist", name)
		}
	}
}

// =============================================================================
// OS primitive tests
// =============================================================================

func TestPrimitiveOsArgs(t *testing.T) {
	result := primitiveOsArgs()
	if result.Type != value.TYPE_ARRAY {
		t.Fatalf("expected array, got %s", result.Type)
	}

	// Wrong arg count
	result = primitiveOsArgs(value.String("extra"))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for extra args")
	}
}
