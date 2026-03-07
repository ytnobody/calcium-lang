package formatter_test

import (
	"strings"
	"testing"

	"github.com/ytnobody/calcium-lang/pkg/formatter"
)

// assertFormat checks that formatting src produces expected output.
// If expected is empty, it verifies that formatting is idempotent (fmt(src) == fmt(fmt(src))).
func assertFormat(t *testing.T, name, src, expected string) {
	t.Helper()
	result, err := formatter.Format(src)
	if err != nil {
		t.Fatalf("%s: Format error: %v", name, err)
	}
	if expected != "" && result != expected {
		t.Errorf("%s:\ngot:\n%s\nwant:\n%s", name, result, expected)
	}

	// Always verify idempotency
	result2, err := formatter.Format(result)
	if err != nil {
		t.Fatalf("%s: idempotency re-format error: %v", name, err)
	}
	if result != result2 {
		t.Errorf("%s: formatter is not idempotent\nfirst pass:\n%s\nsecond pass:\n%s", name, result, result2)
	}
}

func TestFormatUseStatement(t *testing.T) {
	assertFormat(t, "use", `use core.io!;`, "use core.io!;\n")
	assertFormat(t, "use no-effect", `use core.array;`, "use core.array;\n")
}

func TestFormatNamespace(t *testing.T) {
	assertFormat(t, "namespace", `namespace my.module;`, "namespace my.module;\n")
}

func TestFormatAssignment(t *testing.T) {
	assertFormat(t, "simple assign", `x = 42;`, "x = 42;\n")
	assertFormat(t, "string assign", `msg = "hello";`, `msg = "hello";`+"\n")
}

func TestFormatFunctionDeclaration(t *testing.T) {
	assertFormat(t, "simple func", `func add(a, b) = a + b;`, "func add(a, b) = a + b;\n")
	assertFormat(t, "effect func", `func! say(msg) = msg;`, "func! say(msg) = msg;\n")
}

func TestFormatFunctionWithMatch(t *testing.T) {
	src := `func fib(n) = match n
  0 => 0
  1 => 1
  true => fib(n - 1) + fib(n - 2);`

	result, err := formatter.Format(src)
	if err != nil {
		t.Fatalf("Format error: %v", err)
	}

	// Should contain "func fib(n) = match n"
	if !strings.Contains(result, "func fib(n) = match n") {
		t.Errorf("expected 'func fib(n) = match n' in output:\n%s", result)
	}
	// Should have match cases
	if !strings.Contains(result, "0 => 0") {
		t.Errorf("expected '0 => 0' in output:\n%s", result)
	}
	if !strings.Contains(result, "1 => 1") {
		t.Errorf("expected '1 => 1' in output:\n%s", result)
	}
}

func TestFormatConstraint(t *testing.T) {
	assertFormat(t, "constraint", `constraint Positive(n) = n > 0;`,
		"constraint Positive(n) = n > 0;\n")
}

func TestFormatTypeDeclaration(t *testing.T) {
	result, err := formatter.Format(`type Maybe = Some(value) | None;`)
	if err != nil {
		t.Fatalf("Format error: %v", err)
	}
	if !strings.Contains(result, "type Maybe =") {
		t.Errorf("expected 'type Maybe =' in output:\n%s", result)
	}
	if !strings.Contains(result, "Some(value)") {
		t.Errorf("expected 'Some(value)' in output:\n%s", result)
	}
	if !strings.Contains(result, "None") {
		t.Errorf("expected 'None' in output:\n%s", result)
	}
}

func TestFormatArrayLiteral(t *testing.T) {
	assertFormat(t, "array", `x = [1, 2, 3];`, "x = [1, 2, 3];\n")
	assertFormat(t, "empty array", `x = [];`, "x = [];\n")
}

func TestFormatHashLiteral(t *testing.T) {
	result, err := formatter.Format(`x = {name: "Alice", age: 30};`)
	if err != nil {
		t.Fatalf("Format error: %v", err)
	}
	// The parser converts identifier keys to string literals in the AST,
	// so formatted output has quoted keys.
	if !strings.Contains(result, `"Alice"`) {
		t.Errorf("expected hash value in output:\n%s", result)
	}
	if !strings.Contains(result, "30") {
		t.Errorf("expected hash numeric value in output:\n%s", result)
	}
}

func TestFormatPipeExpression(t *testing.T) {
	assertFormat(t, "pipe", `result = x |> double;`, "result = x |> double;\n")
	assertFormat(t, "effect pipe", `msg !> io.say;`, "msg !> io.say;\n")
	assertFormat(t, "prop pipe", `x = val |>? validate;`, "x = val |>? validate;\n")
}

func TestFormatLambda(t *testing.T) {
	assertFormat(t, "single param lambda", `f = x => x * 2;`, "f = x => x * 2;\n")
	assertFormat(t, "multi param lambda", `f = (a, b) => a + b;`, "f = (a, b) => a + b;\n")
}

func TestFormatDestructuring(t *testing.T) {
	assertFormat(t, "array destructure", `[a, b, c] = arr;`, "[a, b, c] = arr;\n")
	assertFormat(t, "head-tail destructure", `[head | tail] = list;`, "[head | tail] = list;\n")
}

func TestFormatBooleans(t *testing.T) {
	assertFormat(t, "true", `x = true;`, "x = true;\n")
	assertFormat(t, "false", `x = false;`, "x = false;\n")
}

func TestFormatSuccessFailure(t *testing.T) {
	assertFormat(t, "success", `x = success(42);`, "x = success(42);\n")
	assertFormat(t, "failure", `x = failure("err");`, `x = failure("err");`+"\n")
}

func TestFormatMatchExpression(t *testing.T) {
	src := `x = match val 0 => "zero" 1 => "one" _ => "other";`
	result, err := formatter.Format(src)
	if err != nil {
		t.Fatalf("Format error: %v", err)
	}
	if !strings.Contains(result, "match val") {
		t.Errorf("expected 'match val' in output:\n%s", result)
	}
	if !strings.Contains(result, `0 => "zero"`) {
		t.Errorf("expected match arms in output:\n%s", result)
	}
}

func TestFormatConstraintCheck(t *testing.T) {
	assertFormat(t, "constraint check", `x = Positive?;`, "x = Positive?;\n")
}

func TestFormatParseError(t *testing.T) {
	_, err := formatter.Format(`func = ;`)
	if err == nil {
		t.Error("expected parse error, got nil")
	}
}

func TestFormatIdempotency(t *testing.T) {
	programs := []string{
		`use core.io!;
x = 42;
"hello" !> io.say;
`,
		`func fib(n) = match n
  0 => 0
  1 => 1
  true => fib(n - 1) + fib(n - 2);
`,
		`type Maybe = Some(value) | None;
constraint Positive(n) = n > 0;
func safe_sqrt(n: Positive?) = n ** 0.5;
`,
	}

	for i, src := range programs {
		result1, err := formatter.Format(src)
		if err != nil {
			t.Errorf("program %d: Format error: %v", i, err)
			continue
		}
		result2, err := formatter.Format(result1)
		if err != nil {
			t.Errorf("program %d: second Format error: %v", i, err)
			continue
		}
		if result1 != result2 {
			t.Errorf("program %d: not idempotent\nfirst:\n%s\nsecond:\n%s", i, result1, result2)
		}
	}
}

func TestFormatInfixOperators(t *testing.T) {
	tests := []struct {
		src      string
		expected string
	}{
		{`x = a + b;`, "x = a + b;\n"},
		{`x = a - b;`, "x = a - b;\n"},
		{`x = a * b;`, "x = a * b;\n"},
		{`x = a / b;`, "x = a / b;\n"},
		{`x = a % b;`, "x = a % b;\n"},
		{`x = a ** b;`, "x = a ** b;\n"},
		{`x = a == b;`, "x = a == b;\n"},
		{`x = a != b;`, "x = a != b;\n"},
		{`x = a < b;`, "x = a < b;\n"},
		{`x = a > b;`, "x = a > b;\n"},
		{`x = a && b;`, "x = a && b;\n"},
		{`x = a || b;`, "x = a || b;\n"},
	}
	for _, tt := range tests {
		assertFormat(t, tt.src, tt.src, tt.expected)
	}
}

func TestFormatCallExpression(t *testing.T) {
	assertFormat(t, "call", `x = foo(1, 2, 3);`, "x = foo(1, 2, 3);\n")
	assertFormat(t, "nested call", `x = foo(bar(1), 2);`, "x = foo(bar(1), 2);\n")
}

func TestFormatSpread(t *testing.T) {
	assertFormat(t, "spread", `x = foo(args...);`, "x = foo(args...);\n")
}

func TestFormatIndexExpression(t *testing.T) {
	assertFormat(t, "index", `x = arr[0];`, "x = arr[0];\n")
}

func TestFormatMemberExpression(t *testing.T) {
	assertFormat(t, "member", `x = obj.field;`, "x = obj.field;\n")
}

func TestFormatFunctionWithConstrainedParams(t *testing.T) {
	src := `func! safe_sqrt(n: Positive?) = n ** 0.5;`
	result, err := formatter.Format(src)
	if err != nil {
		t.Fatalf("Format error: %v", err)
	}
	if !strings.Contains(result, "n: Positive?") {
		t.Errorf("expected constrained parameter in output:\n%s", result)
	}
}
