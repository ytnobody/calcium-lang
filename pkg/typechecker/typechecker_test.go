package typechecker_test

import (
	"testing"

	"github.com/ytnobody/calcium-lang/pkg/lexer"
	"github.com/ytnobody/calcium-lang/pkg/parser"
	"github.com/ytnobody/calcium-lang/pkg/typechecker"
)

// helper: parse source and run type checker, return errors
func check(src string) []typechecker.TypeError {
	l := lexer.New(src)
	p := parser.New(l)
	prog := p.ParseProgram()
	return typechecker.Check(prog)
}

func TestNoAnnotations(t *testing.T) {
	// Existing code without any type annotations should produce no errors.
	errs := check(`
x = 42;
y = "hello";
z = x + 1;
`)
	if len(errs) != 0 {
		t.Errorf("expected no type errors, got: %v", errs)
	}
}

func TestTypedAssignment_Valid(t *testing.T) {
	errs := check(`x: Int = 42;`)
	if len(errs) != 0 {
		t.Errorf("expected no type errors for x: Int = 42, got: %v", errs)
	}
}

func TestTypedAssignment_Valid_String(t *testing.T) {
	errs := check(`name: String = "hello";`)
	if len(errs) != 0 {
		t.Errorf("expected no type errors for name: String = \"hello\", got: %v", errs)
	}
}

func TestTypedAssignment_Valid_Bool(t *testing.T) {
	errs := check(`flag: Bool = true;`)
	if len(errs) != 0 {
		t.Errorf("expected no type errors, got: %v", errs)
	}
}

func TestTypedAssignment_Valid_Float(t *testing.T) {
	errs := check(`ratio: Float = 3.14;`)
	if len(errs) != 0 {
		t.Errorf("expected no type errors, got: %v", errs)
	}
}

func TestTypedAssignment_Valid_Any(t *testing.T) {
	// Any is compatible with everything
	errs := check(`val: Any = 42;`)
	if len(errs) != 0 {
		t.Errorf("expected no type errors, got: %v", errs)
	}
}

func TestTypedAssignment_Mismatch(t *testing.T) {
	// Assigning a string literal to an Int-annotated variable should fail.
	errs := check(`x: Int = "hello";`)
	if len(errs) == 0 {
		t.Error("expected a type error for x: Int = \"hello\", got none")
	}
}

func TestTypedAssignment_Mismatch_BoolToInt(t *testing.T) {
	errs := check(`x: Int = true;`)
	if len(errs) == 0 {
		t.Error("expected a type error for x: Int = true, got none")
	}
}

func TestMixedAnnotated_Unannotated(t *testing.T) {
	// Mixing annotated and unannotated variables: no error expected since
	// unannotated variables are Any.
	errs := check(`
a = 42;
b: Int = a;
`)
	if len(errs) != 0 {
		// 'a' has inferred type Int from literal 42, so b: Int = a should be fine.
		t.Errorf("unexpected errors: %v", errs)
	}
}

func TestFunctionDeclaration_TypedParams_Valid(t *testing.T) {
	errs := check(`func add(a: Int, b: Int): Int = a + b;`)
	if len(errs) != 0 {
		t.Errorf("expected no type errors, got: %v", errs)
	}
}

func TestFunctionDeclaration_ReturnTypeMismatch(t *testing.T) {
	// Function declared to return Int but body is a string literal.
	errs := check(`func greet(name: String): Int = "hello";`)
	if len(errs) == 0 {
		t.Error("expected a type error for return type mismatch, got none")
	}
}

func TestFunctionDeclaration_NoAnnotations(t *testing.T) {
	// No annotations — should produce no errors
	errs := check(`func identity(x) = x;`)
	if len(errs) != 0 {
		t.Errorf("expected no errors, got: %v", errs)
	}
}

func TestFunctionDeclaration_ReturnAny_Valid(t *testing.T) {
	// Returning Any — compatible with any body type
	errs := check(`func anything(x): Any = x;`)
	if len(errs) != 0 {
		t.Errorf("expected no errors, got: %v", errs)
	}
}

func TestFunctionDeclaration_ParamsNoReturn(t *testing.T) {
	// Typed params without return type annotation — still valid
	errs := check(`func double(n: Int) = n + n;`)
	if len(errs) != 0 {
		t.Errorf("expected no errors, got: %v", errs)
	}
}

func TestConstraint_Unchanged(t *testing.T) {
	// Constraint annotation (with ?) should still parse and not cause type errors
	errs := check(`
constraint Positive(x) = x > 0;
func square(n: Positive?) = n * n;
`)
	if len(errs) != 0 {
		t.Errorf("expected no type errors with constraint annotation, got: %v", errs)
	}
}

func TestInferArithmetic(t *testing.T) {
	// Int + Int → Int, assigned to Int: should be fine
	errs := check(`
a: Int = 3;
b: Int = 4;
c: Int = a + b;
`)
	if len(errs) != 0 {
		t.Errorf("expected no errors, got: %v", errs)
	}
}

func TestInferArithmetic_FloatMismatch(t *testing.T) {
	// Assigning Int + Int to Float variable — type mismatch
	errs := check(`
a: Int = 3;
b: Int = 4;
c: Float = a + b;
`)
	if len(errs) == 0 {
		t.Error("expected type error for Int + Int assigned to Float, got none")
	}
}
