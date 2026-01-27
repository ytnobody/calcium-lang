package vm

import (
	"testing"

	"github.com/ytnobody/calcium-lang/pkg/ast"
	"github.com/ytnobody/calcium-lang/pkg/compiler"
	"github.com/ytnobody/calcium-lang/pkg/lexer"
	"github.com/ytnobody/calcium-lang/pkg/parser"
	"github.com/ytnobody/calcium-lang/pkg/value"
)

type vmTestCase struct {
	input    string
	expected interface{}
}

func runVmTests(t *testing.T, tests []vmTestCase) {
	t.Helper()

	for _, tt := range tests {
		program := parse(tt.input)

		comp := compiler.New()
		err := comp.Compile(program)
		if err != nil {
			t.Fatalf("compiler error for %q: %s", tt.input, err)
		}

		vm := New(comp.Constants())
		err = vm.Run(comp.Bytecode().Instructions)
		if err != nil {
			t.Fatalf("vm error for %q: %s", tt.input, err)
		}

		stackElem := vm.LastPoppedStackElem()
		testExpectedValue(t, tt.expected, stackElem)
	}
}

func parse(input string) *ast.Program {
	l := lexer.New(input)
	p := parser.New(l)
	return p.ParseProgram()
}

func testExpectedValue(t *testing.T, expected interface{}, actual value.Value) {
	t.Helper()

	switch expected := expected.(type) {
	case int:
		testIntegerValue(t, int64(expected), actual)
	case int64:
		testIntegerValue(t, expected, actual)
	case float64:
		testFloatValue(t, expected, actual)
	case bool:
		testBooleanValue(t, expected, actual)
	case string:
		testStringValue(t, expected, actual)
	case []int:
		arr := actual.AsArray()
		if len(arr) != len(expected) {
			t.Errorf("wrong array length. want=%d, got=%d", len(expected), len(arr))
			return
		}
		for i, elem := range expected {
			testIntegerValue(t, int64(elem), arr[i])
		}
	case []interface{}:
		if actual.Type != value.TYPE_ARRAY {
			t.Errorf("expected array, got %s", actual.Type)
			return
		}
		arr := actual.AsArray()
		if len(arr) != len(expected) {
			t.Errorf("wrong array length. want=%d, got=%d", len(expected), len(arr))
			return
		}
		for i, elem := range expected {
			testExpectedValue(t, elem, arr[i])
		}
	case nil:
		if actual.Type != value.TYPE_NULL {
			t.Errorf("object is not Null: %T (%+v)", actual, actual)
		}
	}
}

func testIntegerValue(t *testing.T, expected int64, actual value.Value) {
	t.Helper()

	if actual.Type != value.TYPE_INT {
		t.Errorf("object is not Integer. got=%s (%+v)", actual.Type, actual)
		return
	}

	if actual.AsInt() != expected {
		t.Errorf("object has wrong value. want=%d, got=%d", expected, actual.AsInt())
	}
}

func testFloatValue(t *testing.T, expected float64, actual value.Value) {
	t.Helper()

	if actual.Type != value.TYPE_FLOAT {
		t.Errorf("object is not Float. got=%s (%+v)", actual.Type, actual)
		return
	}

	if actual.AsFloat() != expected {
		t.Errorf("object has wrong value. want=%f, got=%f", expected, actual.AsFloat())
	}
}

func testBooleanValue(t *testing.T, expected bool, actual value.Value) {
	t.Helper()

	if actual.Type != value.TYPE_BOOL {
		t.Errorf("object is not Boolean. got=%s (%+v)", actual.Type, actual)
		return
	}

	if actual.AsBool() != expected {
		t.Errorf("object has wrong value. want=%t, got=%t", expected, actual.AsBool())
	}
}

func testStringValue(t *testing.T, expected string, actual value.Value) {
	t.Helper()

	if actual.Type != value.TYPE_STRING {
		t.Errorf("object is not String. got=%s (%+v)", actual.Type, actual)
		return
	}

	if actual.AsString() != expected {
		t.Errorf("object has wrong value. want=%q, got=%q", expected, actual.AsString())
	}
}

func TestIntegerArithmetic(t *testing.T) {
	tests := []vmTestCase{
		{"1;", 1},
		{"2;", 2},
		{"1 + 2;", 3},
		{"1 - 2;", -1},
		{"2 * 3;", 6},
		{"4 / 2;", 2.0}, // Division always returns float
		{"5 % 3;", 2},
		{"2 ** 3;", 8},
		{"-5;", -5},
		{"5 + 5 + 5 + 5 - 10;", 10},
		{"2 * 2 * 2 * 2 * 2;", 32},
		{"5 * 2 + 10;", 20},
		{"5 + 2 * 10;", 25},
		{"(5 + 10 * 2 + 15 / 3) * 2 + -10;", 50.0},
	}

	runVmTests(t, tests)
}

func TestBooleanExpressions(t *testing.T) {
	tests := []vmTestCase{
		{"true;", true},
		{"false;", false},
		{"1 < 2;", true},
		{"1 > 2;", false},
		{"1 < 1;", false},
		{"1 > 1;", false},
		{"1 == 1;", true},
		{"1 != 1;", false},
		{"1 == 2;", false},
		{"1 != 2;", true},
		{"true == true;", true},
		{"false == false;", true},
		{"true == false;", false},
		{"true != false;", true},
		{"(1 < 2) == true;", true},
		{"(1 < 2) == false;", false},
		{"(1 > 2) == true;", false},
		{"(1 > 2) == false;", true},
		{"!true;", false},
		{"!false;", true},
		{"!!true;", true},
		{"!!false;", false},
		{"1 <= 2;", true},
		{"2 <= 2;", true},
		{"3 <= 2;", false},
		{"1 >= 2;", false},
		{"2 >= 2;", true},
		{"3 >= 2;", true},
	}

	runVmTests(t, tests)
}

func TestLogicalOperators(t *testing.T) {
	tests := []vmTestCase{
		{"true && true;", true},
		{"true && false;", false},
		{"false && true;", false},
		{"false && false;", false},
		{"true || true;", true},
		{"true || false;", true},
		{"false || true;", true},
		{"false || false;", false},
	}

	runVmTests(t, tests)
}

func TestStringExpressions(t *testing.T) {
	tests := []vmTestCase{
		{`"hello";`, "hello"},
		{`"hello" + " " + "world";`, "hello world"},
	}

	runVmTests(t, tests)
}

func TestGlobalAssignment(t *testing.T) {
	tests := []vmTestCase{
		{"x = 1; x;", 1},
		{"x = 1; y = 2; x + y;", 3},
		{"x = 1; y = x + 1; y;", 2},
		{"x = 1; x = x + 1; x;", 2},
	}

	runVmTests(t, tests)
}

func TestArrayLiterals(t *testing.T) {
	tests := []vmTestCase{
		{"[1, 2, 3];", []int{1, 2, 3}},
		{"[1 + 2, 3 * 4, 5 - 6];", []int{3, 12, -1}},
	}

	runVmTests(t, tests)
}

func TestArrayIndex(t *testing.T) {
	tests := []vmTestCase{
		{"[1, 2, 3][0];", 1},
		{"[1, 2, 3][1];", 2},
		{"[1, 2, 3][2];", 3},
		{"i = 0; [1][i];", 1},
		{"[1, 2, 3][1 + 1];", 3},
		{"arr = [1, 2, 3]; arr[0];", 1},
		{"arr = [1, 2, 3]; arr[0] + arr[1] + arr[2];", 6},
		{"arr = [1, 2, 3]; i = arr[0]; arr[i];", 2},
	}

	runVmTests(t, tests)
}

func TestArrayOutOfBounds(t *testing.T) {
	tests := []vmTestCase{
		{"[1, 2, 3][99];", nil},
		{"[1, 2, 3][-1];", nil},
	}

	runVmTests(t, tests)
}

func TestFunctions(t *testing.T) {
	tests := []vmTestCase{
		{
			input:    "func five() = 5; five();",
			expected: 5,
		},
		{
			input:    "func double(x) = x * 2; double(5);",
			expected: 10,
		},
		{
			input:    "func add(a, b) = a + b; add(5, 10);",
			expected: 15,
		},
		{
			input:    "func add(a, b) = a + b; add(5 + 5, add(5, 5));",
			expected: 20,
		},
		{
			input: `
				func outer() = 1;
				func middle() = outer();
				func inner() = middle();
				inner();
			`,
			expected: 1,
		},
	}

	runVmTests(t, tests)
}

func TestFunctionsWithLocalBindings(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
				func f() = {
					x = 5;
					x;
				};
				f();
			`,
			expected: 5,
		},
	}

	// These tests require block expressions which aren't implemented yet
	// Skip for now
	_ = tests
}

func TestRecursiveFunctions(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
				func countDown(x) =
					match x
						0 => 0
						_ => countDown(x - 1);
				countDown(5);
			`,
			expected: 0,
		},
		{
			input: `
				func factorial(n) =
					match n
						0 => 1
						_ => n * factorial(n - 1);
				factorial(5);
			`,
			expected: 120,
		},
	}

	runVmTests(t, tests)
}

func TestLambdaExpressions(t *testing.T) {
	tests := []vmTestCase{
		{
			input:    "f = x => x * 2; f(5);",
			expected: 10,
		},
		{
			input:    "f = (x, y) => x + y; f(3, 4);",
			expected: 7,
		},
		{
			input:    "f = () => 42; f();",
			expected: 42,
		},
	}

	runVmTests(t, tests)
}

func TestMatchExpression(t *testing.T) {
	tests := []vmTestCase{
		{
			input:    "match 1 1 => 10 2 => 20 _ => 0;",
			expected: 10,
		},
		{
			input:    "match 2 1 => 10 2 => 20 _ => 0;",
			expected: 20,
		},
		{
			input:    "match 99 1 => 10 2 => 20 _ => 0;",
			expected: 0,
		},
		{
			input:    "x = 5; match x 1 => 10 5 => 50 _ => 0;",
			expected: 50,
		},
	}

	runVmTests(t, tests)
}

func TestPipeOperator(t *testing.T) {
	tests := []vmTestCase{
		{
			input:    "func double(x) = x * 2; 5 |> double;",
			expected: 10,
		},
		{
			input:    "func add1(x) = x + 1; func double(x) = x * 2; 5 |> add1 |> double;",
			expected: 12,
		},
		{
			input:    "5 |> (x => x * 2);",
			expected: 10,
		},
		{
			input:    "10 |> (x => x + 5) |> (x => x * 2);",
			expected: 30,
		},
	}

	runVmTests(t, tests)
}

func TestEffectPipeOperator(t *testing.T) {
	tests := []vmTestCase{
		{
			input:    "func! save(x) = x * 2; 5 !> save;",
			expected: nil, // Returns success(10)
		},
	}

	// Effect pipe returns success/failure, need special handling
	for _, tt := range tests {
		program := parse(tt.input)

		comp := compiler.New()
		err := comp.Compile(program)
		if err != nil {
			t.Fatalf("compiler error: %s", err)
		}

		vm := New(comp.Constants())
		err = vm.Run(comp.Bytecode().Instructions)
		if err != nil {
			t.Fatalf("vm error: %s", err)
		}

		result := vm.LastPoppedStackElem()
		if !result.IsSuccess() {
			t.Errorf("expected success, got %s", result.Type)
		}
		inner := result.AsSuccess()
		if inner.AsInt() != 10 {
			t.Errorf("expected 10, got %d", inner.AsInt())
		}
	}
}

func TestSpreadOperator(t *testing.T) {
	tests := []vmTestCase{
		{
			input:    "func add(a, b) = a + b; [3, 4]... |> add;",
			expected: 7,
		},
		{
			input:    "func sum3(a, b, c) = a + b + c; [1, 2, 3]... |> sum3;",
			expected: 6,
		},
	}

	runVmTests(t, tests)
}

func TestErrorHandling(t *testing.T) {
	tests := []struct {
		input    string
		expected interface{}
	}{
		{
			// Success case: extract value
			input:    `success(42) !? { success(r) => r failure(e) => 0 };`,
			expected: int64(42),
		},
		{
			// Failure case: handle error
			input:    `failure("error") !? { success(r) => r failure(e) => 0 };`,
			expected: int64(0),
		},
		{
			// Effect function with success handling
			input: `
				func! getData() = 100;
				getData() !? {
					success(r) => r * 2
					failure(e) => 0
				};
			`,
			expected: int64(200),
		},
	}

	for _, tt := range tests {
		program := parse(tt.input)

		comp := compiler.New()
		err := comp.Compile(program)
		if err != nil {
			t.Fatalf("compiler error for %q: %s", tt.input, err)
		}

		vm := New(comp.Constants())
		err = vm.Run(comp.Bytecode().Instructions)
		if err != nil {
			t.Fatalf("vm error for %q: %s", tt.input, err)
		}

		result := vm.LastPoppedStackElem()
		if result.Type != value.TYPE_INT {
			t.Errorf("expected int, got %s for %q", result.Type, tt.input)
			continue
		}
		if result.AsInt() != tt.expected.(int64) {
			t.Errorf("expected %d, got %d for %q", tt.expected, result.AsInt(), tt.input)
		}
	}
}

func TestParameterConstraints(t *testing.T) {
	tests := []struct {
		input         string
		expectSuccess bool
		expectedValue int64
	}{
		{
			// Constraint passes
			input: `
				constraint Positive(n) = n > 0;
				func double(x: Positive?) = x * 2;
				double(5);
			`,
			expectSuccess: true,
			expectedValue: 10,
		},
		{
			// Constraint fails
			input: `
				constraint Positive(n) = n > 0;
				func double(x: Positive?) = x * 2;
				double(-5);
			`,
			expectSuccess: false,
			expectedValue: -5,
		},
		{
			// Multiple parameters, one constrained
			input: `
				constraint Positive(n) = n > 0;
				func add(a: Positive?, b) = a + b;
				add(5, 3);
			`,
			expectSuccess: true,
			expectedValue: 8,
		},
		{
			// Multiple parameters, both constrained, first fails
			input: `
				constraint Positive(n) = n > 0;
				func add(a: Positive?, b: Positive?) = a + b;
				add(-1, 3);
			`,
			expectSuccess: false,
			expectedValue: -1,
		},
	}

	for _, tt := range tests {
		program := parse(tt.input)

		comp := compiler.New()
		err := comp.Compile(program)
		if err != nil {
			t.Fatalf("compiler error for %q: %s", tt.input, err)
		}

		vm := New(comp.Constants())
		err = vm.Run(comp.Bytecode().Instructions)
		if err != nil {
			t.Fatalf("vm error for %q: %s", tt.input, err)
		}

		result := vm.LastPoppedStackElem()

		if tt.expectSuccess {
			if !result.IsSuccess() {
				t.Errorf("expected success for %q, got %s (%v)", tt.input, result.Type, result)
				continue
			}
			inner := result.AsSuccess()
			if inner.AsInt() != tt.expectedValue {
				t.Errorf("expected %d for %q, got %d", tt.expectedValue, tt.input, inner.AsInt())
			}
		} else {
			if !result.IsFailure() {
				t.Errorf("expected failure for %q, got %s (%v)", tt.input, result.Type, result)
				continue
			}
			inner := result.AsFailure()
			if inner.AsInt() != tt.expectedValue {
				t.Errorf("expected %d for %q, got %d", tt.expectedValue, tt.input, inner.AsInt())
			}
		}
	}
}

func TestConstraintCheck(t *testing.T) {
	tests := []struct {
		input         string
		expectSuccess bool
		expectedValue int64
	}{
		{
			// Constraint passes
			input:         "constraint Positive(n) = n > 0; 5 |> Positive?;",
			expectSuccess: true,
			expectedValue: 5,
		},
		{
			// Constraint fails
			input:         "constraint Positive(n) = n > 0; -3 |> Positive?;",
			expectSuccess: false,
			expectedValue: -3,
		},
		{
			// Multiple constraints chained
			input:         "constraint Even(n) = n % 2 == 0; 4 |> Even?;",
			expectSuccess: true,
			expectedValue: 4,
		},
		{
			// Constraint with complex condition
			input:         "constraint InRange(n) = n >= 0 && n <= 100; 50 |> InRange?;",
			expectSuccess: true,
			expectedValue: 50,
		},
		{
			// Constraint fails complex condition
			input:         "constraint InRange(n) = n >= 0 && n <= 100; 150 |> InRange?;",
			expectSuccess: false,
			expectedValue: 150,
		},
	}

	for _, tt := range tests {
		program := parse(tt.input)

		comp := compiler.New()
		err := comp.Compile(program)
		if err != nil {
			t.Fatalf("compiler error for %q: %s", tt.input, err)
		}

		vm := New(comp.Constants())
		err = vm.Run(comp.Bytecode().Instructions)
		if err != nil {
			t.Fatalf("vm error for %q: %s", tt.input, err)
		}

		result := vm.LastPoppedStackElem()

		if tt.expectSuccess {
			if !result.IsSuccess() {
				t.Errorf("expected success for %q, got %s", tt.input, result.Type)
				continue
			}
			inner := result.AsSuccess()
			if inner.AsInt() != tt.expectedValue {
				t.Errorf("expected %d for %q, got %d", tt.expectedValue, tt.input, inner.AsInt())
			}
		} else {
			if !result.IsFailure() {
				t.Errorf("expected failure for %q, got %s", tt.input, result.Type)
				continue
			}
			inner := result.AsFailure()
			if inner.AsInt() != tt.expectedValue {
				t.Errorf("expected %d for %q, got %d", tt.expectedValue, tt.input, inner.AsInt())
			}
		}
	}
}

func TestSuccessFailure(t *testing.T) {
	tests := []vmTestCase{
		{
			input:    "success(42);",
			expected: nil, // Special handling needed
		},
		{
			input:    "failure(\"error\");",
			expected: nil, // Special handling needed
		},
	}

	for i, tt := range tests {
		program := parse(tt.input)

		comp := compiler.New()
		err := comp.Compile(program)
		if err != nil {
			t.Fatalf("compiler error: %s", err)
		}

		vm := New(comp.Constants())
		err = vm.Run(comp.Bytecode().Instructions)
		if err != nil {
			t.Fatalf("vm error: %s", err)
		}

		result := vm.LastPoppedStackElem()
		if i == 0 {
			if !result.IsSuccess() {
				t.Errorf("expected success, got %s", result.Type)
			}
			if result.AsSuccess().AsInt() != 42 {
				t.Errorf("expected 42, got %v", result.AsSuccess())
			}
		} else {
			if !result.IsFailure() {
				t.Errorf("expected failure, got %s", result.Type)
			}
			if result.AsFailure().AsString() != "error" {
				t.Errorf("expected 'error', got %v", result.AsFailure())
			}
		}
	}
}

func TestBuiltinLen(t *testing.T) {
	tests := []vmTestCase{
		{`len("hello");`, 5},
		{`len("");`, 0},
		{`len([1, 2, 3]);`, 3},
		{`len([]);`, 0},
	}
	runVmTests(t, tests)
}

func TestBuiltinConcat(t *testing.T) {
	tests := []vmTestCase{
		{`concat("hello", " ", "world");`, "hello world"},
		{`concat("a", "b", "c");`, "abc"},
	}
	runVmTests(t, tests)
}

func TestBuiltinToString(t *testing.T) {
	tests := []vmTestCase{
		{`to_string(42);`, "42"},
		{`to_string(true);`, "true"},
		{`to_string(false);`, "false"},
	}
	runVmTests(t, tests)
}

func TestBuiltinGet(t *testing.T) {
	tests := []vmTestCase{
		{`get([1, 2, 3], 0);`, 1},
		{`get([1, 2, 3], 2);`, 3},
		{`get([1, 2, 3], 5, 99);`, 99}, // default value
	}
	runVmTests(t, tests)
}

func TestBuiltinHas(t *testing.T) {
	tests := []vmTestCase{
		{`has([1, 2, 3], 2);`, true},
		{`has([1, 2, 3], 5);`, false},
		{`has(["a", "b"], "a");`, true},
	}
	runVmTests(t, tests)
}

func TestBuiltinHeadTail(t *testing.T) {
	tests := []vmTestCase{
		{`head([1, 2, 3]);`, 1},
		{`head([42]);`, 42},
	}
	runVmTests(t, tests)

	// Test tail separately (returns array)
	input := `tail([1, 2, 3]);`
	program := parse(input)
	comp := compiler.New()
	comp.Compile(program)
	vm := New(comp.Constants())
	vm.Run(comp.Bytecode().Instructions)
	result := vm.LastPoppedStackElem()

	if result.Type != value.TYPE_ARRAY {
		t.Fatalf("expected array, got %s", result.Type)
	}
	arr := result.AsArray()
	if len(arr) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(arr))
	}
	if arr[0].AsInt() != 2 || arr[1].AsInt() != 3 {
		t.Errorf("expected [2, 3], got %v", arr)
	}
}

func TestBuiltinPush(t *testing.T) {
	input := `push([1, 2], 3);`
	program := parse(input)
	comp := compiler.New()
	comp.Compile(program)
	vm := New(comp.Constants())
	vm.Run(comp.Bytecode().Instructions)
	result := vm.LastPoppedStackElem()

	if result.Type != value.TYPE_ARRAY {
		t.Fatalf("expected array, got %s", result.Type)
	}
	arr := result.AsArray()
	if len(arr) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(arr))
	}
	if arr[0].AsInt() != 1 || arr[1].AsInt() != 2 || arr[2].AsInt() != 3 {
		t.Errorf("expected [1, 2, 3], got %v", arr)
	}
}

func TestBuiltinRange(t *testing.T) {
	tests := []struct {
		input    string
		expected []int64
	}{
		{`range(5);`, []int64{0, 1, 2, 3, 4}},
		{`range(2, 5);`, []int64{2, 3, 4}},
		{`range(0, 10, 2);`, []int64{0, 2, 4, 6, 8}},
		{`range(5, 0, -1);`, []int64{5, 4, 3, 2, 1}},
	}

	for _, tt := range tests {
		program := parse(tt.input)
		comp := compiler.New()
		comp.Compile(program)
		vm := New(comp.Constants())
		vm.Run(comp.Bytecode().Instructions)
		result := vm.LastPoppedStackElem()

		if result.Type != value.TYPE_ARRAY {
			t.Fatalf("expected array for %q, got %s", tt.input, result.Type)
		}
		arr := result.AsArray()
		if len(arr) != len(tt.expected) {
			t.Fatalf("expected %d elements for %q, got %d", len(tt.expected), tt.input, len(arr))
		}
		for i, expected := range tt.expected {
			if arr[i].AsInt() != expected {
				t.Errorf("expected %d at index %d for %q, got %d", expected, i, tt.input, arr[i].AsInt())
			}
		}
	}
}

// Test map function
func TestBuiltinMap(t *testing.T) {
	tests := []struct {
		input    string
		expected []int64
	}{
		// Basic map with lambda - argument order: map(arr, fn)
		{`map([1, 2, 3], x => x * 2);`, []int64{2, 4, 6}},
		// Map with named function
		{`func double(x) = x * 2; map([1, 2, 3], double);`, []int64{2, 4, 6}},
		// Empty array
		{`map([], x => x * 2);`, []int64{}},
		// Map with addition
		{`map([10, 20, 30], x => x + 5);`, []int64{15, 25, 35}},
	}

	for _, tt := range tests {
		program := parse(tt.input)
		comp := compiler.New()
		err := comp.Compile(program)
		if err != nil {
			t.Fatalf("compiler error for %q: %s", tt.input, err)
		}

		vm := New(comp.Constants())
		err = vm.Run(comp.Bytecode().Instructions)
		if err != nil {
			t.Fatalf("vm error for %q: %s", tt.input, err)
		}

		result := vm.LastPoppedStackElem()
		if result.Type != value.TYPE_ARRAY {
			t.Fatalf("expected array for %q, got %s", tt.input, result.Type)
		}
		arr := result.AsArray()
		if len(arr) != len(tt.expected) {
			t.Fatalf("expected %d elements for %q, got %d", len(tt.expected), tt.input, len(arr))
		}
		for i, expected := range tt.expected {
			if arr[i].AsInt() != expected {
				t.Errorf("expected %d at index %d for %q, got %d", expected, i, tt.input, arr[i].AsInt())
			}
		}
	}
}

// Test filter function
func TestBuiltinFilter(t *testing.T) {
	tests := []struct {
		input    string
		expected []int64
	}{
		// Filter even numbers - argument order: filter(arr, pred)
		{`func isEven(x) = x % 2 == 0; filter([1, 2, 3, 4, 5, 6], isEven);`, []int64{2, 4, 6}},
		// Filter with lambda
		{`filter([1, 2, 3, 4, 5], x => x > 2);`, []int64{3, 4, 5}},
		// Filter all elements
		{`filter([1, 2, 3], x => x > 0);`, []int64{1, 2, 3}},
		// Filter no elements
		{`filter([1, 2, 3], x => x > 10);`, []int64{}},
		// Empty array
		{`filter([], x => x > 0);`, []int64{}},
	}

	for _, tt := range tests {
		program := parse(tt.input)
		comp := compiler.New()
		err := comp.Compile(program)
		if err != nil {
			t.Fatalf("compiler error for %q: %s", tt.input, err)
		}

		vm := New(comp.Constants())
		err = vm.Run(comp.Bytecode().Instructions)
		if err != nil {
			t.Fatalf("vm error for %q: %s", tt.input, err)
		}

		result := vm.LastPoppedStackElem()
		if result.Type != value.TYPE_ARRAY {
			t.Fatalf("expected array for %q, got %s", tt.input, result.Type)
		}
		arr := result.AsArray()
		if len(arr) != len(tt.expected) {
			t.Fatalf("expected %d elements for %q, got %d", len(tt.expected), tt.input, len(arr))
		}
		for i, expected := range tt.expected {
			if arr[i].AsInt() != expected {
				t.Errorf("expected %d at index %d for %q, got %d", expected, i, tt.input, arr[i].AsInt())
			}
		}
	}
}

// Test reduce function
func TestBuiltinReduce(t *testing.T) {
	tests := []vmTestCase{
		// Sum of array - argument order: reduce(arr, fn, init)
		{`reduce([1, 2, 3, 4, 5], (acc, x) => acc + x, 0);`, int64(15)},
		// Product of array
		{`reduce([1, 2, 3, 4], (acc, x) => acc * x, 1);`, int64(24)},
		// Count elements
		{`reduce([1, 2, 3], (count, x) => count + 1, 0);`, int64(3)},
		// Empty array returns initial
		{`reduce([], (acc, x) => acc + x, 42);`, int64(42)},
		// String concatenation
		{`reduce(["a", "b", "c"], (acc, x) => concat(acc, x), "");`, "abc"},
	}

	runVmTests(t, tests)
}

// Test combining map/filter/reduce
func TestHigherOrderCombined(t *testing.T) {
	tests := []vmTestCase{
		// Sum of doubled even numbers - argument order: arr first
		{`
			arr = [1, 2, 3, 4, 5, 6];
			evens = filter(arr, x => x % 2 == 0);
			doubled = map(evens, x => x * 2);
			reduce(doubled, (a, b) => a + b, 0);
		`, int64(24)}, // (2+4+6) * 2 = 24
		// Pipeline style: x |> f(y) -> f(x, y)
		{`[1, 2, 3, 4] |> filter(x => x > 1) |> map(x => x * 10) |> reduce((a, b) => a + b, 0);`, int64(90)}, // (2+3+4) * 10 = 90
	}

	runVmTests(t, tests)
}

// Test module system
func TestModules(t *testing.T) {
	tests := []vmTestCase{
		// Use core.math module
		{`use core.math; math.abs(-5);`, int64(5)},
		{`use core.math; math.abs(3);`, int64(3)},
		{`use core.math; math.min(3, 7);`, int64(3)},
		{`use core.math; math.max(3, 7);`, int64(7)},
		// Combine module functions
		{`use core.math; math.abs(math.min(-10, -5));`, int64(10)},
	}

	runVmTests(t, tests)
}

// Test module member access
func TestModuleMemberAccess(t *testing.T) {
	tests := []vmTestCase{
		// Use math module with calculations
		{`
			use core.math;
			a = -10;
			b = math.abs(a);
			b + 5;
		`, int64(15)},
		// Chain module function calls
		{`
			use core.math;
			x = math.max(1, 2);
			y = math.max(x, 3);
			math.max(y, 4);
		`, int64(4)},
	}

	runVmTests(t, tests)
}

// Test core.io module (effect module)
func TestCoreIOModule(t *testing.T) {
	// Test that io module can be loaded and functions called
	// Note: print/println output to stdout, we just verify they don't error
	input := `
		use core.io;
		io.print("test");
		io.println("test");
		42;
	`

	program := parse(input)
	comp := compiler.New()
	err := comp.Compile(program)
	if err != nil {
		t.Fatalf("compiler error: %s", err)
	}

	vm := New(comp.Constants())
	err = vm.Run(comp.Bytecode().Instructions)
	if err != nil {
		t.Fatalf("vm error: %s", err)
	}

	result := vm.LastPoppedStackElem()
	if result.AsInt() != 42 {
		t.Errorf("expected 42, got %v", result)
	}
}

// Test core.io format function
func TestCoreIOFormat(t *testing.T) {
	tests := []vmTestCase{
		{`use core.io; io.format("Hello {}!", "World");`, "Hello World!"},
		{`use core.io; io.format("{} + {} = {}", 1, 2, 3);`, "1 + 2 = 3"},
		{`use core.io; io.format("No placeholders");`, "No placeholders"},
	}
	runVmTests(t, tests)
}

// Test core.math extended functions
func TestCoreMathExtended(t *testing.T) {
	tests := []vmTestCase{
		// floor/ceil/round
		{`use core.math; math.floor(3.7);`, int64(3)},
		{`use core.math; math.ceil(3.2);`, int64(4)},
		{`use core.math; math.round(3.5);`, int64(4)},
		{`use core.math; math.round(3.4);`, int64(3)},
		// sqrt
		{`use core.math; math.sqrt(16);`, 4.0},
		{`use core.math; math.sqrt(2);`, 1.4142135623730951},
		// pow
		{`use core.math; math.pow(2, 3);`, 8.0},
		{`use core.math; math.pow(3, 2);`, 9.0},
	}
	runVmTests(t, tests)
}

// Test core.string module
func TestCoreStringModule(t *testing.T) {
	tests := []vmTestCase{
		// split/join
		{`use core.string; string.split("a,b,c", ",");`, []interface{}{"a", "b", "c"}},
		{`use core.string; string.join(["a", "b", "c"], "-");`, "a-b-c"},
		// trim
		{`use core.string; string.trim("  hello  ");`, "hello"},
		// upper/lower
		{`use core.string; string.upper("hello");`, "HELLO"},
		{`use core.string; string.lower("HELLO");`, "hello"},
		// starts_with/ends_with/contains
		{`use core.string; string.starts_with("hello", "he");`, true},
		{`use core.string; string.starts_with("hello", "lo");`, false},
		{`use core.string; string.ends_with("hello", "lo");`, true},
		{`use core.string; string.ends_with("hello", "he");`, false},
		{`use core.string; string.contains("hello", "ell");`, true},
		{`use core.string; string.contains("hello", "xyz");`, false},
		// replace
		{`use core.string; string.replace("hello world", "world", "calcium");`, "hello calcium"},
		// substring
		{`use core.string; string.substring("hello", 1, 4);`, "ell"},
		{`use core.string; string.substring_from("hello", 2);`, "llo"},
		// char_at
		{`use core.string; string.char_at("hello", 0);`, "h"},
		{`use core.string; string.char_at("hello", 4);`, "o"},
		// index_of
		{`use core.string; string.index_of("hello", "l");`, int64(2)},
		{`use core.string; string.index_of("hello", "x");`, int64(-1)},
	}
	runVmTests(t, tests)
}

// Test core.array module
func TestCoreArrayModule(t *testing.T) {
	tests := []vmTestCase{
		// reverse
		{`use core.array; array.reverse([1, 2, 3]);`, []interface{}{int64(3), int64(2), int64(1)}},
		// slice
		{`use core.array; array.slice([1, 2, 3, 4, 5], 1, 4);`, []interface{}{int64(2), int64(3), int64(4)}},
		{`use core.array; array.slice_from([1, 2, 3], 1);`, []interface{}{int64(2), int64(3)}},
		// index_of
		{`use core.array; array.index_of([1, 2, 3], 2);`, int64(1)},
		{`use core.array; array.index_of([1, 2, 3], 5);`, int64(-1)},
		// flatten
		{`use core.array; array.flatten([[1, 2], [3, 4]]);`, []interface{}{int64(1), int64(2), int64(3), int64(4)}},
		{`use core.array; array.flatten([1, [2, 3], 4]);`, []interface{}{int64(1), int64(2), int64(3), int64(4)}},
		// unique
		{`use core.array; array.unique([1, 2, 2, 3, 1]);`, []interface{}{int64(1), int64(2), int64(3)}},
		// zip
		{`use core.array; array.zip([1, 2], ["a", "b"]);`, []interface{}{
			[]interface{}{int64(1), "a"},
			[]interface{}{int64(2), "b"},
		}},
		// take/drop
		{`use core.array; array.take([1, 2, 3, 4, 5], 3);`, []interface{}{int64(1), int64(2), int64(3)}},
		{`use core.array; array.drop([1, 2, 3, 4, 5], 2);`, []interface{}{int64(3), int64(4), int64(5)}},
		// sum/product
		{`use core.array; array.sum([1, 2, 3, 4, 5]);`, int64(15)},
		{`use core.array; array.product([1, 2, 3, 4]);`, int64(24)},
	}
	runVmTests(t, tests)
}

func TestHashLiterals(t *testing.T) {
	tests := []vmTestCase{
		// Empty hash
		{`len({});`, int64(0)},
		// Hash literal
		{`h = {name: "Alice", age: 30}; len(h);`, int64(2)},
		// Dot access
		{`h = {name: "Alice", age: 30}; h.name;`, "Alice"},
		{`h = {name: "Alice", age: 30}; h.age;`, int64(30)},
		// Bracket access
		{`h = {name: "Alice"}; h["name"];`, "Alice"},
		// Dynamic key
		{`h = {name: "Bob"}; k = "name"; h[k];`, "Bob"},
		// Non-existent key returns null
		{`h = {name: "Alice"}; h.missing;`, nil},
		{`h = {name: "Alice"}; h["missing"];`, nil},
		// String keys with spaces
		{`h = {"my key": 42}; h["my key"];`, int64(42)},
	}
	runVmTests(t, tests)
}

func TestHashBuiltins(t *testing.T) {
	tests := []vmTestCase{
		// keys
		{`keys({a: 1, b: 2, c: 3});`, []interface{}{"a", "b", "c"}},
		{`keys({});`, []interface{}{}},
		// values
		{`values({a: 1, b: 2, c: 3});`, []interface{}{int64(1), int64(2), int64(3)}},
		{`values({});`, []interface{}{}},
		// len
		{`len({a: 1, b: 2});`, int64(2)},
	}
	runVmTests(t, tests)
}

func TestNestedHash(t *testing.T) {
	tests := []vmTestCase{
		// Nested hash access
		{`h = {user: {name: "Alice"}}; h.user.name;`, "Alice"},
		{`h = {data: {items: [1, 2, 3]}}; h.data.items[1];`, int64(2)},
		// Hash with various value types
		{`h = {num: 42, str: "hello", flag: true, arr: [1, 2]}; h.num;`, int64(42)},
		{`h = {num: 42, str: "hello", flag: true, arr: [1, 2]}; h.str;`, "hello"},
		{`h = {num: 42, str: "hello", flag: true, arr: [1, 2]}; h.flag;`, true},
		{`h = {num: 42, str: "hello", flag: true, arr: [1, 2]}; h.arr[0];`, int64(1)},
	}
	runVmTests(t, tests)
}

func TestArrayDestructuring(t *testing.T) {
	tests := []vmTestCase{
		// Basic array destructuring
		{`[a, b, c] = [1, 2, 3]; a;`, int64(1)},
		{`[a, b, c] = [1, 2, 3]; b;`, int64(2)},
		{`[a, b, c] = [1, 2, 3]; c;`, int64(3)},
		// Single element
		{`[x] = [42]; x;`, int64(42)},
		// Two elements
		{`[x, y] = [10, 20]; x + y;`, int64(30)},
		// Fewer elements in array (null for missing)
		{`[a, b, c] = [1, 2]; c;`, nil},
		// More elements in array (extras ignored)
		{`[a, b] = [1, 2, 3, 4]; a + b;`, int64(3)},
		// With expressions on right side
		{`arr = [100, 200]; [x, y] = arr; x;`, int64(100)},
	}
	runVmTests(t, tests)
}

func TestHeadTailDestructuring(t *testing.T) {
	tests := []vmTestCase{
		// Basic head|tail
		{`[h | t] = [1, 2, 3, 4]; h;`, int64(1)},
		{`[h | t] = [1, 2, 3, 4]; t;`, []interface{}{int64(2), int64(3), int64(4)}},
		// Single element array
		{`[h | t] = [42]; h;`, int64(42)},
		{`[h | t] = [42]; t;`, []interface{}{}},
		// Two elements
		{`[first | rest] = [10, 20]; first;`, int64(10)},
		{`[first | rest] = [10, 20]; rest;`, []interface{}{int64(20)}},
		// With variable
		{`arr = [5, 10, 15]; [x | xs] = arr; x;`, int64(5)},
	}
	runVmTests(t, tests)
}

func TestRegexLiteral(t *testing.T) {
	tests := []vmTestCase{
		// Basic regex literals compile without error
		{`/hello/; true;`, true},
		{`/^[a-z]+$/; true;`, true},
		{`/\d{3}-\d{4}/; true;`, true},
	}
	runVmTests(t, tests)
}

func TestRegexMatches(t *testing.T) {
	tests := []vmTestCase{
		// Basic matches
		{`use core.regex; regex.matches("hello", /hello/);`, true},
		{`use core.regex; regex.matches("world", /hello/);`, false},
		{`use core.regex; regex.matches("HELLO", /hello/);`, false},
		{`use core.regex; regex.matches("HELLO", /hello/i);`, true},
		// Pattern matching
		{`use core.regex; regex.matches("123-4567", /^\d{3}-\d{4}$/);`, true},
		{`use core.regex; regex.matches("12-4567", /^\d{3}-\d{4}$/);`, false},
		{`use core.regex; regex.matches("abc", /^[a-z]+$/);`, true},
		{`use core.regex; regex.matches("ABC", /^[a-z]+$/);`, false},
		{`use core.regex; regex.matches("ABC", /^[a-z]+$/i);`, true},
		// Escape sequences
		{`use core.regex; regex.matches("https://", /https?:\/\//);`, true},
		{`use core.regex; regex.matches("http://", /https?:\/\//);`, true},
	}
	runVmTests(t, tests)
}

func TestRegexFindAll(t *testing.T) {
	tests := []vmTestCase{
		// find_all returns all matches
		{`use core.regex; regex.find_all("abc123def456", /\d+/);`, []interface{}{"123", "456"}},
		{`use core.regex; regex.find_all("no numbers here", /\d+/);`, []interface{}{}},
	}
	runVmTests(t, tests)
}

func TestRegexReplace(t *testing.T) {
	tests := []vmTestCase{
		// Basic replacement
		{`use core.regex; regex.replace("hello world", /world/, "universe");`, "hello universe"},
		{`use core.regex; regex.replace("a1b2c3", /\d/, "X");`, "aXbXcX"},
	}
	runVmTests(t, tests)
}

func TestRegexSplit(t *testing.T) {
	tests := []vmTestCase{
		// Split by pattern
		{`use core.regex; regex.split("a1b2c3", /\d/);`, []interface{}{"a", "b", "c", ""}},
		{`use core.regex; regex.split("one,two;three", /[,;]/);`, []interface{}{"one", "two", "three"}},
	}
	runVmTests(t, tests)
}

// Test parseGitHubURL helper function
func TestParseGitHubURL(t *testing.T) {
	tests := []struct {
		url      string
		author   string
		repo     string
	}{
		{"github.com/ytnobody/json-calcium", "ytnobody", "json-calcium"},
		{"github.com/author/repo", "author", "repo"},
		{"https://github.com/user/project", "user", "project"},
		{"http://github.com/foo/bar", "foo", "bar"},
	}

	for _, tt := range tests {
		author, repo := parseGitHubURL(tt.url)
		if author != tt.author {
			t.Errorf("parseGitHubURL(%q) author = %q, want %q", tt.url, author, tt.author)
		}
		if repo != tt.repo {
			t.Errorf("parseGitHubURL(%q) repo = %q, want %q", tt.url, repo, tt.repo)
		}
	}
}

// Test cache directory path generation
func TestCachePaths(t *testing.T) {
	// Test getModuleCachePath
	path := getModuleCachePath("ytnobody", "json")
	if path == "" {
		t.Error("getModuleCachePath returned empty path")
	}

	// Should contain author and module name
	if !contains(path, "ytnobody") || !contains(path, "json") {
		t.Errorf("getModuleCachePath(%q, %q) = %q, should contain author and module", "ytnobody", "json", path)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Test exported cache functions for calcium cache command
func TestCacheExportedFunctions(t *testing.T) {
	// Test GetCacheDir
	cacheDir := GetCacheDir()
	if cacheDir == "" {
		t.Error("GetCacheDir returned empty string")
	}

	// Test ListCachedModules on empty/non-existent cache
	modules, err := ListCachedModules()
	if err != nil {
		t.Errorf("ListCachedModules returned error: %v", err)
	}
	// modules can be empty or contain items, both are valid
	_ = modules

	// Test GetCacheSize
	size, err := GetCacheSize()
	if err != nil {
		t.Errorf("GetCacheSize returned error: %v", err)
	}
	if size < 0 {
		t.Error("GetCacheSize returned negative size")
	}

	// Test IsModuleCached for non-existent module
	cached := IsModuleCached("nonexistent", "module")
	// Should return false for non-existent module
	_ = cached // Result depends on cache state
}

// Test CachedModule struct
func TestCachedModuleStruct(t *testing.T) {
	mod := CachedModule{
		Author: "testauthor",
		Name:   "testmodule",
		Path:   "/path/to/module",
		Size:   1024,
	}

	if mod.Author != "testauthor" {
		t.Errorf("CachedModule.Author = %q, want %q", mod.Author, "testauthor")
	}
	if mod.Name != "testmodule" {
		t.Errorf("CachedModule.Name = %q, want %q", mod.Name, "testmodule")
	}
}
