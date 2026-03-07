package vm

import (
	"testing"

	"github.com/ytnobody/calcium-lang/pkg/value"
)

func assertJSONSuccess(t *testing.T, result value.Value) value.Value {
	t.Helper()
	if result.Type != value.TYPE_SUCCESS {
		t.Fatalf("expected success, got %s: %s", result.Type, result.String())
	}
	return result.AsSuccess()
}

func assertJSONFailure(t *testing.T, result value.Value) {
	t.Helper()
	if result.Type != value.TYPE_FAILURE {
		t.Fatalf("expected failure, got %s: %s", result.Type, result.String())
	}
}

func TestJSONParsePrimitives(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected value.Value
	}{
		{"integer", "42", value.Int(42)},
		{"negative integer", "-17", value.Int(-17)},
		{"zero", "0", value.Int(0)},
		{"float", "3.14", value.Float(3.14)},
		{"negative float", "-0.5", value.Float(-0.5)},
		{"exponent", "1e3", value.Float(1000)},
		{"true", "true", value.Bool(true)},
		{"false", "false", value.Bool(false)},
		{"null", "null", value.Null()},
		{"simple string", `"hello"`, value.String("hello")},
		{"empty string", `""`, value.String("")},
		{"string with escape", `"hello\nworld"`, value.String("hello\nworld")},
		{"string with tab", `"a\tb"`, value.String("a\tb")},
		{"string with unicode", `"caf\u00e9"`, value.String("caf\u00e9")},
		{"number with whitespace", "  42  ", value.Int(42)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := primitiveJSONParse(value.String(tt.input))
			v := assertJSONSuccess(t, result)
			if v.String() != tt.expected.String() {
				t.Errorf("expected %s, got %s", tt.expected.String(), v.String())
			}
		})
	}
}

func TestJSONParseArray(t *testing.T) {
	v := assertJSONSuccess(t, primitiveJSONParse(value.String("[1, 2, 3]")))
	arr := v.AsArray()
	if len(arr) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(arr))
	}
	if arr[0].AsInt() != 1 || arr[1].AsInt() != 2 || arr[2].AsInt() != 3 {
		t.Errorf("expected [1,2,3], got %v", arr)
	}
}

func TestJSONParseEmptyArray(t *testing.T) {
	v := assertJSONSuccess(t, primitiveJSONParse(value.String("[]")))
	arr := v.AsArray()
	if len(arr) != 0 {
		t.Fatalf("expected empty array, got %d elements", len(arr))
	}
}

func TestJSONParseObject(t *testing.T) {
	v := assertJSONSuccess(t, primitiveJSONParse(value.String(`{"name": "Alice", "age": 30}`)))
	h := v.AsHash()
	nameVal, ok := h.Get(value.String("name"))
	if !ok {
		t.Fatal("expected key 'name'")
	}
	if nameVal.AsString() != "Alice" {
		t.Errorf("expected 'Alice', got '%s'", nameVal.AsString())
	}
	ageVal, ok := h.Get(value.String("age"))
	if !ok {
		t.Fatal("expected key 'age'")
	}
	if ageVal.AsInt() != 30 {
		t.Errorf("expected 30, got %d", ageVal.AsInt())
	}
}

func TestJSONParseNested(t *testing.T) {
	input := `{"users": [{"name": "Alice", "scores": [95, 87]}, {"name": "Bob"}]}`
	v := assertJSONSuccess(t, primitiveJSONParse(value.String(input)))
	h := v.AsHash()
	users, ok := h.Get(value.String("users"))
	if !ok {
		t.Fatal("expected key 'users'")
	}
	arr := users.AsArray()
	if len(arr) != 2 {
		t.Fatalf("expected 2 users, got %d", len(arr))
	}
	user0 := arr[0].AsHash()
	name0, _ := user0.Get(value.String("name"))
	if name0.AsString() != "Alice" {
		t.Errorf("expected 'Alice', got '%s'", name0.AsString())
	}
	scores, _ := user0.Get(value.String("scores"))
	scoresArr := scores.AsArray()
	if len(scoresArr) != 2 || scoresArr[0].AsInt() != 95 {
		t.Errorf("unexpected scores: %v", scoresArr)
	}
}

func TestJSONParseErrors(t *testing.T) {
	errCases := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"unclosed brace", "{"},
		{"unclosed bracket", "["},
		{"trailing comma array", "[1, 2,]"},
		{"trailing comma object", `{"a": 1,}`},
		{"invalid token", "invalid"},
		{"missing colon", `{"a" 1}`},
	}

	for _, tt := range errCases {
		t.Run(tt.name, func(t *testing.T) {
			result := primitiveJSONParse(value.String(tt.input))
			assertJSONFailure(t, result)
		})
	}
}

func TestJSONStringify(t *testing.T) {
	tests := []struct {
		name     string
		input    value.Value
		expected string
	}{
		{"int", value.Int(42), "42"},
		{"negative int", value.Int(-17), "-17"},
		{"float", value.Float(3.14), "3.14"},
		{"true", value.Bool(true), "true"},
		{"false", value.Bool(false), "false"},
		{"null", value.Null(), "null"},
		{"string", value.String("hello"), `"hello"`},
		{"empty string", value.String(""), `""`},
		{"string with newline", value.String("a\nb"), `"a\nb"`},
		{"empty array", value.Array(nil), "[]"},
		{"int array", value.Array([]value.Value{value.Int(1), value.Int(2), value.Int(3)}), "[1,2,3]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := assertJSONSuccess(t, primitiveJSONStringify(tt.input))
			if v.AsString() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, v.AsString())
			}
		})
	}
}

func TestJSONStringifyObject(t *testing.T) {
	h := value.NewHash()
	h.Set(value.String("name"), value.String("Alice"))
	h.Set(value.String("age"), value.Int(30))

	v := assertJSONSuccess(t, primitiveJSONStringify(value.HashVal(h)))
	expected := `{"name":"Alice","age":30}`
	if v.AsString() != expected {
		t.Errorf("expected %s, got %s", expected, v.AsString())
	}
}

func TestJSONRoundTrip(t *testing.T) {
	inputs := []string{
		"42",
		"3.14",
		"true",
		"false",
		"null",
		`"hello"`,
		"[1,2,3]",
		`{"name":"Alice","age":30}`,
		`[{"id":1},{"id":2}]`,
	}

	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			parsed := assertJSONSuccess(t, primitiveJSONParse(value.String(input)))
			stringified := assertJSONSuccess(t, primitiveJSONStringify(parsed))
			if stringified.AsString() != input {
				t.Errorf("round-trip mismatch: %s -> %s", input, stringified.AsString())
			}
		})
	}
}

func TestJSONParseWrongArgType(t *testing.T) {
	assertJSONFailure(t, primitiveJSONParse(value.Int(42)))
}

func TestJSONParseWrongArgCount(t *testing.T) {
	assertJSONFailure(t, primitiveJSONParse())
}
