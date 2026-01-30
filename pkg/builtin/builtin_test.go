package builtin

import (
	"testing"

	"github.com/ytnobody/calcium-lang/pkg/value"
)

// ============================================================
// GetBuiltins / GetBuiltinNames Tests
// ============================================================

func TestGetBuiltins(t *testing.T) {
	builtins := GetBuiltins()

	if len(builtins) == 0 {
		t.Fatal("expected at least one builtin function")
	}

	expectedNames := []string{"len", "concat", "to_string", "get", "has", "head", "tail", "push", "range"}
	if len(builtins) != len(expectedNames) {
		t.Fatalf("expected %d builtins, got %d", len(expectedNames), len(builtins))
	}

	for i, expected := range expectedNames {
		if builtins[i].Name != expected {
			t.Errorf("expected builtin[%d].Name = %q, got %q", i, expected, builtins[i].Name)
		}
		if builtins[i].Fn == nil {
			t.Errorf("builtin[%d].Fn is nil", i)
		}
	}
}

func TestGetBuiltinNames(t *testing.T) {
	names := GetBuiltinNames()

	expectedNames := []string{"len", "concat", "to_string", "get", "has", "head", "tail", "push", "range"}
	if len(names) != len(expectedNames) {
		t.Fatalf("expected %d names, got %d", len(expectedNames), len(names))
	}

	for i, expected := range expectedNames {
		if names[i] != expected {
			t.Errorf("expected names[%d] = %q, got %q", i, expected, names[i])
		}
	}
}

// ============================================================
// builtinLen Tests
// ============================================================

func TestBuiltinLen(t *testing.T) {
	tests := []struct {
		name     string
		args     []value.Value
		expected value.Value
	}{
		{
			name:     "string length",
			args:     []value.Value{value.String("hello")},
			expected: value.Int(5),
		},
		{
			name:     "empty string",
			args:     []value.Value{value.String("")},
			expected: value.Int(0),
		},
		{
			name:     "array length",
			args:     []value.Value{value.Array([]value.Value{value.Int(1), value.Int(2), value.Int(3)})},
			expected: value.Int(3),
		},
		{
			name:     "empty array",
			args:     []value.Value{value.Array([]value.Value{})},
			expected: value.Int(0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := builtinLen(tt.args...)
			if !result.Equals(tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestBuiltinLen_WrongArgCount(t *testing.T) {
	result := builtinLen()
	if result.Type != value.TYPE_FAILURE {
		t.Errorf("expected failure for no args, got %v", result.Type)
	}

	result = builtinLen(value.String("a"), value.String("b"))
	if result.Type != value.TYPE_FAILURE {
		t.Errorf("expected failure for too many args, got %v", result.Type)
	}
}

func TestBuiltinLen_UnsupportedType(t *testing.T) {
	result := builtinLen(value.Int(42))
	if result.Type != value.TYPE_FAILURE {
		t.Errorf("expected failure for int, got %v", result.Type)
	}
}

// ============================================================
// builtinConcat Tests
// ============================================================

func TestBuiltinConcat(t *testing.T) {
	tests := []struct {
		name     string
		args     []value.Value
		expected value.Value
	}{
		{
			name:     "no args",
			args:     []value.Value{},
			expected: value.String(""),
		},
		{
			name:     "string concat",
			args:     []value.Value{value.String("hello"), value.String(" "), value.String("world")},
			expected: value.String("hello world"),
		},
		{
			name:     "string with non-string",
			args:     []value.Value{value.String("count: "), value.Int(42)},
			expected: value.String("count: 42"),
		},
		{
			name:     "array concat",
			args:     []value.Value{value.Array([]value.Value{value.Int(1)}), value.Array([]value.Value{value.Int(2), value.Int(3)})},
			expected: value.Array([]value.Value{value.Int(1), value.Int(2), value.Int(3)}),
		},
		{
			name:     "array with non-array element",
			args:     []value.Value{value.Array([]value.Value{value.Int(1)}), value.Int(2)},
			expected: value.Array([]value.Value{value.Int(1), value.Int(2)}),
		},
		{
			name:     "non-string first arg converts all to string",
			args:     []value.Value{value.Int(1), value.Int(2), value.Int(3)},
			expected: value.String("123"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := builtinConcat(tt.args...)
			if !result.Equals(tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// ============================================================
// builtinToString Tests
// ============================================================

func TestBuiltinToString(t *testing.T) {
	tests := []struct {
		name     string
		args     []value.Value
		expected string
	}{
		{"int to string", []value.Value{value.Int(42)}, "42"},
		{"float to string", []value.Value{value.Float(3.14)}, "3.14"},
		{"bool to string", []value.Value{value.Bool(true)}, "true"},
		{"string to string", []value.Value{value.String("hello")}, "hello"},
		{"null to string", []value.Value{value.Null()}, "null"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := builtinToString(tt.args...)
			if result.Type != value.TYPE_STRING {
				t.Errorf("expected string type, got %v", result.Type)
			}
			if result.AsString() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result.AsString())
			}
		})
	}
}

func TestBuiltinToString_WrongArgCount(t *testing.T) {
	result := builtinToString()
	if result.Type != value.TYPE_FAILURE {
		t.Errorf("expected failure for no args, got %v", result.Type)
	}

	result = builtinToString(value.Int(1), value.Int(2))
	if result.Type != value.TYPE_FAILURE {
		t.Errorf("expected failure for too many args, got %v", result.Type)
	}
}

// ============================================================
// builtinGet Tests
// ============================================================

func TestBuiltinGet(t *testing.T) {
	arr := value.Array([]value.Value{value.String("a"), value.String("b"), value.String("c")})

	tests := []struct {
		name     string
		args     []value.Value
		expected value.Value
	}{
		{"get first element", []value.Value{arr, value.Int(0)}, value.String("a")},
		{"get middle element", []value.Value{arr, value.Int(1)}, value.String("b")},
		{"get last element", []value.Value{arr, value.Int(2)}, value.String("c")},
		{"out of bounds returns null", []value.Value{arr, value.Int(10)}, value.Null()},
		{"negative index returns null", []value.Value{arr, value.Int(-1)}, value.Null()},
		{"out of bounds with default", []value.Value{arr, value.Int(10), value.String("default")}, value.String("default")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := builtinGet(tt.args...)
			if !result.Equals(tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestBuiltinGet_Errors(t *testing.T) {
	tests := []struct {
		name string
		args []value.Value
	}{
		{"no args", []value.Value{}},
		{"one arg", []value.Value{value.Array([]value.Value{})}},
		{"too many args", []value.Value{value.Array([]value.Value{}), value.Int(0), value.String("default"), value.Int(1)}},
		{"first arg not array", []value.Value{value.String("not array"), value.Int(0)}},
		{"index not int", []value.Value{value.Array([]value.Value{}), value.String("0")}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := builtinGet(tt.args...)
			if result.Type != value.TYPE_FAILURE {
				t.Errorf("expected failure, got %v", result.Type)
			}
		})
	}
}

// ============================================================
// builtinHas Tests
// ============================================================

func TestBuiltinHas(t *testing.T) {
	arr := value.Array([]value.Value{value.Int(1), value.Int(2), value.String("hello")})

	tests := []struct {
		name     string
		args     []value.Value
		expected bool
	}{
		{"has int element", []value.Value{arr, value.Int(1)}, true},
		{"has string element", []value.Value{arr, value.String("hello")}, true},
		{"does not have element", []value.Value{arr, value.Int(99)}, false},
		{"non-array returns false", []value.Value{value.String("not array"), value.Int(1)}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := builtinHas(tt.args...)
			if result.Type != value.TYPE_BOOL {
				t.Errorf("expected bool type, got %v", result.Type)
			}
			if result.AsBool() != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result.AsBool())
			}
		})
	}
}

func TestBuiltinHas_WrongArgCount(t *testing.T) {
	result := builtinHas()
	if result.Type != value.TYPE_FAILURE {
		t.Errorf("expected failure for no args, got %v", result.Type)
	}

	result = builtinHas(value.Array([]value.Value{}))
	if result.Type != value.TYPE_FAILURE {
		t.Errorf("expected failure for one arg, got %v", result.Type)
	}
}

// ============================================================
// builtinHead Tests
// ============================================================

func TestBuiltinHead(t *testing.T) {
	tests := []struct {
		name     string
		args     []value.Value
		expected value.Value
	}{
		{"head of non-empty array", []value.Value{value.Array([]value.Value{value.Int(1), value.Int(2), value.Int(3)})}, value.Int(1)},
		{"head of single element", []value.Value{value.Array([]value.Value{value.String("only")})}, value.String("only")},
		{"head of empty array", []value.Value{value.Array([]value.Value{})}, value.Null()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := builtinHead(tt.args...)
			if !result.Equals(tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestBuiltinHead_Errors(t *testing.T) {
	result := builtinHead()
	if result.Type != value.TYPE_FAILURE {
		t.Errorf("expected failure for no args, got %v", result.Type)
	}

	result = builtinHead(value.String("not array"))
	if result.Type != value.TYPE_FAILURE {
		t.Errorf("expected failure for non-array, got %v", result.Type)
	}
}

// ============================================================
// builtinTail Tests
// ============================================================

func TestBuiltinTail(t *testing.T) {
	tests := []struct {
		name     string
		args     []value.Value
		expected value.Value
	}{
		{
			name:     "tail of multiple elements",
			args:     []value.Value{value.Array([]value.Value{value.Int(1), value.Int(2), value.Int(3)})},
			expected: value.Array([]value.Value{value.Int(2), value.Int(3)}),
		},
		{
			name:     "tail of single element",
			args:     []value.Value{value.Array([]value.Value{value.Int(1)})},
			expected: value.Array([]value.Value{}),
		},
		{
			name:     "tail of empty array",
			args:     []value.Value{value.Array([]value.Value{})},
			expected: value.Array([]value.Value{}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := builtinTail(tt.args...)
			if !result.Equals(tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestBuiltinTail_Errors(t *testing.T) {
	result := builtinTail()
	if result.Type != value.TYPE_FAILURE {
		t.Errorf("expected failure for no args, got %v", result.Type)
	}

	result = builtinTail(value.String("not array"))
	if result.Type != value.TYPE_FAILURE {
		t.Errorf("expected failure for non-array, got %v", result.Type)
	}
}

// ============================================================
// builtinPush Tests
// ============================================================

func TestBuiltinPush(t *testing.T) {
	tests := []struct {
		name     string
		args     []value.Value
		expected value.Value
	}{
		{
			name:     "push to non-empty array",
			args:     []value.Value{value.Array([]value.Value{value.Int(1), value.Int(2)}), value.Int(3)},
			expected: value.Array([]value.Value{value.Int(1), value.Int(2), value.Int(3)}),
		},
		{
			name:     "push to empty array",
			args:     []value.Value{value.Array([]value.Value{}), value.String("first")},
			expected: value.Array([]value.Value{value.String("first")}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := builtinPush(tt.args...)
			if !result.Equals(tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestBuiltinPush_Errors(t *testing.T) {
	result := builtinPush()
	if result.Type != value.TYPE_FAILURE {
		t.Errorf("expected failure for no args, got %v", result.Type)
	}

	result = builtinPush(value.Array([]value.Value{}))
	if result.Type != value.TYPE_FAILURE {
		t.Errorf("expected failure for one arg, got %v", result.Type)
	}

	result = builtinPush(value.String("not array"), value.Int(1))
	if result.Type != value.TYPE_FAILURE {
		t.Errorf("expected failure for non-array first arg, got %v", result.Type)
	}
}

// ============================================================
// builtinRange Tests
// ============================================================

func TestBuiltinRange(t *testing.T) {
	tests := []struct {
		name     string
		args     []value.Value
		expected []int64
	}{
		{"range(5)", []value.Value{value.Int(5)}, []int64{0, 1, 2, 3, 4}},
		{"range(0)", []value.Value{value.Int(0)}, []int64{}},
		{"range(2, 5)", []value.Value{value.Int(2), value.Int(5)}, []int64{2, 3, 4}},
		{"range(0, 3)", []value.Value{value.Int(0), value.Int(3)}, []int64{0, 1, 2}},
		{"range(0, 10, 2)", []value.Value{value.Int(0), value.Int(10), value.Int(2)}, []int64{0, 2, 4, 6, 8}},
		{"range(10, 0, -2)", []value.Value{value.Int(10), value.Int(0), value.Int(-2)}, []int64{10, 8, 6, 4, 2}},
		{"range(5, 2)", []value.Value{value.Int(5), value.Int(2)}, []int64{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := builtinRange(tt.args...)
			if result.Type != value.TYPE_ARRAY {
				t.Fatalf("expected array type, got %v", result.Type)
			}

			arr := result.AsArray()
			if len(arr) != len(tt.expected) {
				t.Fatalf("expected %d elements, got %d", len(tt.expected), len(arr))
			}

			for i, exp := range tt.expected {
				if arr[i].AsInt() != exp {
					t.Errorf("element[%d]: expected %d, got %d", i, exp, arr[i].AsInt())
				}
			}
		})
	}
}

func TestBuiltinRange_Errors(t *testing.T) {
	tests := []struct {
		name string
		args []value.Value
	}{
		{"no args", []value.Value{}},
		{"too many args", []value.Value{value.Int(1), value.Int(2), value.Int(3), value.Int(4)}},
		{"non-int single arg", []value.Value{value.String("5")}},
		{"non-int first arg", []value.Value{value.String("1"), value.Int(5)}},
		{"non-int second arg", []value.Value{value.Int(1), value.String("5")}},
		{"non-int step", []value.Value{value.Int(0), value.Int(10), value.String("2")}},
		{"zero step", []value.Value{value.Int(0), value.Int(10), value.Int(0)}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := builtinRange(tt.args...)
			if result.Type != value.TYPE_FAILURE {
				t.Errorf("expected failure, got %v", result.Type)
			}
		})
	}
}

// ============================================================
// BuiltinParseInt Tests
// ============================================================

func TestBuiltinParseInt(t *testing.T) {
	tests := []struct {
		name     string
		args     []value.Value
		expected int64
	}{
		{"positive int", []value.Value{value.String("42")}, 42},
		{"negative int", []value.Value{value.String("-123")}, -123},
		{"zero", []value.Value{value.String("0")}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuiltinParseInt(tt.args...)
			if result.Type != value.TYPE_INT {
				t.Fatalf("expected int type, got %v", result.Type)
			}
			if result.AsInt() != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result.AsInt())
			}
		})
	}
}

func TestBuiltinParseInt_Errors(t *testing.T) {
	tests := []struct {
		name string
		args []value.Value
	}{
		{"no args", []value.Value{}},
		{"too many args", []value.Value{value.String("1"), value.String("2")}},
		{"non-string arg", []value.Value{value.Int(42)}},
		{"invalid number", []value.Value{value.String("not a number")}},
		{"float string", []value.Value{value.String("3.14")}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuiltinParseInt(tt.args...)
			if result.Type != value.TYPE_FAILURE {
				t.Errorf("expected failure, got %v", result.Type)
			}
		})
	}
}

// ============================================================
// BuiltinParseFloat Tests
// ============================================================

func TestBuiltinParseFloat(t *testing.T) {
	tests := []struct {
		name     string
		args     []value.Value
		expected float64
	}{
		{"positive float", []value.Value{value.String("3.14")}, 3.14},
		{"negative float", []value.Value{value.String("-2.5")}, -2.5},
		{"integer as float", []value.Value{value.String("42")}, 42.0},
		{"zero", []value.Value{value.String("0.0")}, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuiltinParseFloat(tt.args...)
			if result.Type != value.TYPE_FLOAT {
				t.Fatalf("expected float type, got %v", result.Type)
			}
			if result.AsFloat() != tt.expected {
				t.Errorf("expected %f, got %f", tt.expected, result.AsFloat())
			}
		})
	}
}

func TestBuiltinParseFloat_Errors(t *testing.T) {
	tests := []struct {
		name string
		args []value.Value
	}{
		{"no args", []value.Value{}},
		{"too many args", []value.Value{value.String("1.0"), value.String("2.0")}},
		{"non-string arg", []value.Value{value.Float(3.14)}},
		{"invalid number", []value.Value{value.String("not a number")}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuiltinParseFloat(tt.args...)
			if result.Type != value.TYPE_FAILURE {
				t.Errorf("expected failure, got %v", result.Type)
			}
		})
	}
}
