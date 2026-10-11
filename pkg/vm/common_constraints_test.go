package vm

import (
	"testing"

	"github.com/ytnobody/calcium-lang/pkg/compiler"
	"github.com/ytnobody/calcium-lang/pkg/value"
)

// Direct calls to the built-in common constraints return bool.
func TestBuiltinCommonConstraintsDirect(t *testing.T) {
	cases := []struct {
		name string
		call func() bool
		want bool
	}{
		{"Positive(3)", func() bool { return builtinConstraintPositive(value.Int(3)).AsBool() }, true},
		{"Positive(0)", func() bool { return builtinConstraintPositive(value.Int(0)).AsBool() }, false},
		{"Positive(-1)", func() bool { return builtinConstraintPositive(value.Int(-1)).AsBool() }, false},
		{"NonZero(0)", func() bool { return builtinConstraintNonZero(value.Int(0)).AsBool() }, false},
		{"NonZero(5)", func() bool { return builtinConstraintNonZero(value.Int(5)).AsBool() }, true},
		{"NonNegative(-1)", func() bool { return builtinConstraintNonNegative(value.Int(-1)).AsBool() }, false},
		{"NonNegative(0)", func() bool { return builtinConstraintNonNegative(value.Int(0)).AsBool() }, true},
		{"Natural(-1)", func() bool { return builtinConstraintNatural(value.Int(-1)).AsBool() }, false},
		{"Natural(7.5)", func() bool { return builtinConstraintNatural(value.Float(7.5)).AsBool() }, false},
		{"Finite(nan)", func() bool { return builtinConstraintFinite(value.Float(1)).AsBool() }, true},
		{"NonEmpty('')", func() bool { return builtinConstraintNonEmpty(value.String("")).AsBool() }, false},
		{"NonEmpty('a')", func() bool { return builtinConstraintNonEmpty(value.String("a")).AsBool() }, true},
		{"Empty([])", func() bool { return builtinConstraintEmpty(value.Array([]value.Value{})).AsBool() }, true},
		{"Digits('123')", func() bool { return builtinConstraintDigits(value.String("123")).AsBool() }, true},
		{"Digits('12a')", func() bool { return builtinConstraintDigits(value.String("12a")).AsBool() }, false},
		{"Lowercase('abc')", func() bool { return builtinConstraintLowercase(value.String("abc")).AsBool() }, true},
		{"Lowercase('Abc')", func() bool { return builtinConstraintLowercase(value.String("Abc")).AsBool() }, false},
		{"Uppercase('ABC')", func() bool { return builtinConstraintUppercase(value.String("ABC")).AsBool() }, true},
		{"Uppercase('abc')", func() bool { return builtinConstraintUppercase(value.String("abc")).AsBool() }, false},
	}
	for _, c := range cases {
		if got := c.call(); got != c.want {
			t.Errorf("%s = %v, want %v", c.name, got, c.want)
		}
	}
}

// run returns the last popped stack value for a program.
func runCommonProg(t *testing.T, input string) value.Value {
	t.Helper()
	program := parse(input)
	comp := compiler.New()
	if err := comp.Compile(program); err != nil {
		t.Fatalf("compile error: %v", err)
	}
	vm := New(comp.Constants())
	if err := vm.Run(comp.Bytecode().Instructions); err != nil {
		t.Fatalf("vm error: %v", err)
	}
	return vm.LastPoppedStackElem()
}

// Param constraint with a dynamic (non-literal) value is enforced at runtime.
func TestBuiltinCommonConstraintRuntime(t *testing.T) {
	// Dynamic negative value must fail the Positive constraint (returns failure).
	got := runCommonProg(t, `
func f(x: Positive) = x;
neg = 0 - 1;
r = f(neg);
r;
`)
	if got.Type != value.TYPE_FAILURE {
		t.Fatalf("expected failure for dynamic negative under :Positive, got %s", got.Type)
	}

	// Dynamic positive value passes; unwrap the success to check the value.
	got2 := runCommonProg(t, `
func f(x: Positive) = x;
p = 0 - -5;
r = f(p) !? { success(v) => v failure(_) => -999 };
r;
`)
	if got2.Type == value.TYPE_FAILURE || got2.AsInt() != 5 {
		t.Fatalf("expected success value 5, got %s", got2.Type)
	}
}

// Composite annotations (A & B, A | B) are enforced at runtime.
func TestBuiltinCompositeConstraintRuntime(t *testing.T) {
	// AND: negative dynamic value must violate `Positive & NonZero` -> failure.
	got := runCommonProg(t, `
func f(a: Positive & NonZero) = a;
neg = 0 - 1;
r = f(neg) !? { success(v) => 1 failure(_) => 0 };
r;
`)
	if got.Type != value.TYPE_INT || got.AsInt() != 0 {
		t.Fatalf("expected AND violation (failure), got %v", got)
	}

	// AND: valid value passes and is wrapped in success.
	got2 := runCommonProg(t, `
func f(a: Positive & NonZero) = a;
p = 0 + 5;
r = f(p) !? { success(v) => v failure(_) => -999 };
r;
`)
	if got2.Type == value.TYPE_FAILURE || got2.AsInt() != 5 {
		t.Fatalf("expected AND pass value 5, got %v", got2)
	}

	// UNION: a string value satisfies `Int | String`, an int violates Strict
	got3 := runCommonProg(t, `
func f(a: Int | String) = a;
s = f("hi") !? { success(v) => "OK" failure(_) => "NOPE" };
s;
`)
	if got3.AsString() != "OK" {
		t.Fatalf("expected Int|String to accept a string, got %q", got3.AsString())
	}

	// UNION: a runtime (non-constant) float violates `Int | String` -> failure.
	// (A literal `x = 1.5` would now be caught statically; route through a
	// helper so the value is dynamic.)
	got4 := runCommonProg(t, `
func pass(v) = v;
func f(a: Int | String) = a;
x = pass(1.5);
r = f(x) !? { success(v) => "OK" failure(_) => "NOPE" };
r;
`)
	if got4.AsString() != "NOPE" {
		t.Fatalf("expected Int|String to reject a float, got %q", got4.AsString())
	}
}
