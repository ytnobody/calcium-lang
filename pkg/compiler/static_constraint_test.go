package compiler

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Static constraint checking tests
// ---------------------------------------------------------------------------

func TestStaticConstraintCheck_UserDefinedConstraint_Violation(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr string
	}{
		{
			name: "positive constraint violation with 0",
			input: `
constraint Positive(x) = x > 0;
func add_pos(a: Positive, b: Positive) = a + b;
add_pos(0, 5);
`,
			wantErr: "constraint `Positive` violated",
		},
		{
			name: "positive constraint violation with negative literal",
			input: `
constraint Positive(x) = x > 0;
func add_pos(a: Positive, b: Positive) = a + b;
add_pos(-1, 5);
`,
			wantErr: "constraint `Positive` violated",
		},
		{
			name: "even constraint violation with odd number",
			input: `
constraint Even(n) = n % 2 == 0;
func double_even(n: Even) = n * 2;
double_even(3);
`,
			wantErr: "constraint `Even` violated",
		},
		{
			name: "range constraint violation - below range",
			input: `
constraint InRange(x) = x >= 0 && x <= 100;
func check(x: InRange) = x;
check(-1);
`,
			wantErr: "constraint `InRange` violated",
		},
		{
			name: "range constraint violation - above range",
			input: `
constraint InRange(x) = x >= 0 && x <= 100;
func check(x: InRange) = x;
check(101);
`,
			wantErr: "constraint `InRange` violated",
		},
		{
			name: "second parameter violation",
			input: `
constraint Positive(x) = x > 0;
func add_pos(a: Positive, b: Positive) = a + b;
add_pos(5, -3);
`,
			wantErr: "argument 2 (`b`)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := compileSource(tt.input)
			if err == nil {
				t.Fatal("expected compile error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("expected error containing %q, got %q", tt.wantErr, err.Error())
			}
		})
	}
}

func TestStaticConstraintCheck_UserDefinedConstraint_Valid(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name: "positive constraint passes with positive literal",
			input: `
constraint Positive(x) = x > 0;
func add_pos(a: Positive, b: Positive) = a + b;
add_pos(3, 5);
`,
		},
		{
			name: "even constraint passes with even number",
			input: `
constraint Even(n) = n % 2 == 0;
func double_even(n: Even) = n * 2;
double_even(4);
`,
		},
		{
			name: "range constraint passes - boundary values",
			input: `
constraint InRange(x) = x >= 0 && x <= 100;
func check(x: InRange) = x;
check(0);
check(100);
check(50);
`,
		},
		{
			name: "no constraint - no static check",
			input: `
func add(a, b) = a + b;
add(-1, 5);
`,
		},
		{
			name: "dynamic value - no static check",
			input: `
constraint Positive(x) = x > 0;
func add_pos(a: Positive, b: Positive) = a + b;
func identity(x) = x;
add_pos(identity(-1), 5);
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := compileSource(tt.input)
			if err != nil {
				t.Fatalf("unexpected compile error: %s", err)
			}
		})
	}
}

func TestStaticConstraintCheck_BuiltinTypeConstraint_Violation(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr string
	}{
		{
			name: "Int constraint violation with string",
			input: `
func int_add(a: Int, b: Int) = a + b;
int_add("hello", 5);
`,
			wantErr: "constraint `Int` violated",
		},
		{
			name: "String constraint violation with int",
			input: `
func str_fn(s: String) = s;
str_fn(42);
`,
			wantErr: "constraint `String` violated",
		},
		{
			name: "Bool constraint violation with int",
			input: `
func bool_fn(b: Bool) = b;
bool_fn(42);
`,
			wantErr: "constraint `Bool` violated",
		},
		{
			name: "Array constraint violation with int",
			input: `
func arr_fn(a: Array) = a;
arr_fn(42);
`,
			wantErr: "constraint `Array` violated",
		},
		{
			name: "Number constraint violation with string",
			input: `
func num_fn(n: Number) = n;
num_fn("hello");
`,
			wantErr: "constraint `Number` violated",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := compileSource(tt.input)
			if err == nil {
				t.Fatal("expected compile error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("expected error containing %q, got %q", tt.wantErr, err.Error())
			}
		})
	}
}

func TestStaticConstraintCheck_BuiltinTypeConstraint_Valid(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name: "Int constraint passes with integer",
			input: `
func int_add(a: Int, b: Int) = a + b;
int_add(3, 5);
`,
		},
		{
			name: "Int constraint passes with negative integer",
			input: `
func int_fn(a: Int) = a;
int_fn(-5);
`,
		},
		{
			name: "String constraint passes with string",
			input: `
func str_fn(s: String) = s;
str_fn("hello");
`,
		},
		{
			name: "Bool constraint passes with boolean",
			input: `
func bool_fn(b: Bool) = b;
bool_fn(true);
bool_fn(false);
`,
		},
		{
			name: "Number constraint passes with int",
			input: `
func num_fn(n: Number) = n;
num_fn(42);
`,
		},
		{
			name: "Number constraint passes with float",
			input: `
func num_fn(n: Number) = n;
num_fn(3.14);
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := compileSource(tt.input)
			if err != nil {
				t.Fatalf("unexpected compile error: %s", err)
			}
		})
	}
}

func TestStaticConstraintCheck_ComposedConstraint(t *testing.T) {
	// Test that composed constraints can be evaluated statically
	input := `
constraint Positive(x) = x > 0;
constraint Even(n) = n % 2 == 0;
constraint PositiveEven(x) = Positive(x) && Even(x);
func check(x: PositiveEven) = x;
check(3);
`
	_, err := compileSource(input)
	if err == nil {
		t.Fatal("expected compile error for check(3) with PositiveEven constraint")
	}
	if !strings.Contains(err.Error(), "constraint `PositiveEven` violated") {
		t.Errorf("expected PositiveEven violation, got: %s", err)
	}

	// Valid case
	input2 := `
constraint Positive(x) = x > 0;
constraint Even(n) = n % 2 == 0;
constraint PositiveEven(x) = Positive(x) && Even(x);
func check(x: PositiveEven) = x;
check(4);
`
	_, err = compileSource(input2)
	if err != nil {
		t.Fatalf("unexpected compile error for check(4): %s", err)
	}
}

func TestStaticConstraintCheck_ErrorMessage(t *testing.T) {
	input := `
constraint Positive(x) = x > 0;
func add_pos(a: Positive, b: Positive) = a + b;
add_pos(0, 5);
`
	_, err := compileSource(input)
	if err == nil {
		t.Fatal("expected compile error")
	}

	errStr := err.Error()
	// Check error message contains useful information
	if !strings.Contains(errStr, "Positive") {
		t.Error("error should mention the constraint name")
	}
	if !strings.Contains(errStr, "argument 1") {
		t.Error("error should mention the argument position")
	}
	if !strings.Contains(errStr, "`a`") {
		t.Error("error should mention the parameter name")
	}
	if !strings.Contains(errStr, "0") {
		t.Error("error should mention the value")
	}
}
