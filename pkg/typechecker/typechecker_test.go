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

// ---- TypeError.Error() ----

func TestTypeError_Error(t *testing.T) {
	err := typechecker.TypeError{Line: 5, Column: 10, Message: "type mismatch"}
	got := err.Error()
	expected := "[type error] line 5, col 10: type mismatch"
	if got != expected {
		t.Errorf("TypeError.Error() = %q, want %q", got, expected)
	}
}

// ---- Expression statement inference ----

func TestExpressionStatement(t *testing.T) {
	// An expression statement should be checked without errors
	errs := check(`42;`)
	if len(errs) != 0 {
		t.Errorf("expected no errors for expression statement, got: %v", errs)
	}
}

func TestExpressionStatement_StringLiteral(t *testing.T) {
	errs := check(`"hello";`)
	if len(errs) != 0 {
		t.Errorf("expected no errors, got: %v", errs)
	}
}

// ---- Literal type inference ----

func TestInferExpr_Literals(t *testing.T) {
	tests := []struct {
		name    string
		src     string
		wantErr bool
	}{
		{"float literal", `x: Float = 3.14;`, false},
		{"bool literal", `x: Bool = false;`, false},
		{"string literal", `x: String = "test";`, false},
		{"int literal", `x: Int = 100;`, false},
		{"array literal", `x = [1, 2, 3];`, false},
		{"hash literal", `x = {"a": 1};`, false},
		{"tuple literal assigned", `x = (1, 2, 3);`, false},
		{"success expression", `x = Ok(42);`, false},
		{"failure expression", `x = Err("bad");`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := check(tt.src)
			if tt.wantErr && len(errs) == 0 {
				t.Error("expected type error, got none")
			}
			if !tt.wantErr && len(errs) != 0 {
				t.Errorf("expected no errors, got: %v", errs)
			}
		})
	}
}

// ---- Regex literal ----

func TestInferExpr_RegexLiteral(t *testing.T) {
	errs := check(`x = /foo.*/;`)
	if len(errs) != 0 {
		t.Errorf("expected no errors for regex literal, got: %v", errs)
	}
}

// ---- Lambda expression ----

func TestInferExpr_LambdaExpression(t *testing.T) {
	errs := check(`f = x -> x + 1;`)
	if len(errs) != 0 {
		t.Errorf("expected no errors for lambda expression, got: %v", errs)
	}
}

// ---- Interpolated string ----

func TestInferExpr_InterpolatedString(t *testing.T) {
	errs := check("x: String = \"hello ${\"world\"}\";")
	if len(errs) != 0 {
		t.Errorf("expected no errors for interpolated string, got: %v", errs)
	}
}

// ---- Match expression ----

func TestInferExpr_MatchExpression(t *testing.T) {
	errs := check(`
x = 42;
y = match x
  | 1 => "one"
  | _ => "other"
end;
`)
	if len(errs) != 0 {
		t.Errorf("expected no errors for match expression, got: %v", errs)
	}
}

// ---- Prefix expression ----

func TestInferPrefix_Not(t *testing.T) {
	errs := check(`x: Bool = !true;`)
	if len(errs) != 0 {
		t.Errorf("expected no errors for !true -> Bool, got: %v", errs)
	}
}

func TestInferPrefix_NegateInt(t *testing.T) {
	errs := check(`x: Int = -42;`)
	if len(errs) != 0 {
		t.Errorf("expected no errors for -42 -> Int, got: %v", errs)
	}
}

func TestInferPrefix_NegateFloat(t *testing.T) {
	errs := check(`x: Float = -3.14;`)
	if len(errs) != 0 {
		t.Errorf("expected no errors for -3.14 -> Float, got: %v", errs)
	}
}

func TestInferPrefix_NegateUnknown(t *testing.T) {
	// Negating a non-numeric should produce Any
	errs := check(`
y = "hello";
x = -y;
`)
	if len(errs) != 0 {
		t.Errorf("expected no errors, got: %v", errs)
	}
}

func TestInferPrefix_NotMismatch(t *testing.T) {
	errs := check(`x: Int = !true;`)
	if len(errs) == 0 {
		t.Error("expected type error for !true assigned to Int, got none")
	}
}

// ---- Call expression ----

func TestInferCall_KnownFunction(t *testing.T) {
	errs := check(`
func add(a: Int, b: Int): Int = a + b;
x: Int = add(1, 2);
`)
	if len(errs) != 0 {
		t.Errorf("expected no errors for calling known function, got: %v", errs)
	}
}

func TestInferCall_KnownFunction_Mismatch(t *testing.T) {
	errs := check(`
func add(a: Int, b: Int): Int = a + b;
x: String = add(1, 2);
`)
	if len(errs) == 0 {
		t.Error("expected type error for Int return assigned to String, got none")
	}
}

func TestInferCall_UnknownFunction(t *testing.T) {
	// Calling an unknown function returns Any, which is compatible with anything
	errs := check(`x: Int = unknown_func(1);`)
	if len(errs) != 0 {
		t.Errorf("expected no errors for unknown function call (returns Any), got: %v", errs)
	}
}

// ---- Do expression ----

func TestInferDo_Basic(t *testing.T) {
	errs := check(`
x: Int = do
  a = 1;
  b = 2;
  a + b
end;
`)
	if len(errs) != 0 {
		t.Errorf("expected no errors for do expression, got: %v", errs)
	}
}

func TestInferDo_TypeMismatch(t *testing.T) {
	errs := check(`
x: String = do
  a = 1;
  b = 2;
  a + b
end;
`)
	if len(errs) == 0 {
		t.Error("expected type error for do returning Int assigned to String, got none")
	}
}

func TestInferDo_WithAssignments(t *testing.T) {
	errs := check(`
result: Int = do
  x: Int = 10;
  y: Int = 20;
  x + y
end;
`)
	if len(errs) != 0 {
		t.Errorf("expected no errors, got: %v", errs)
	}
}

// ---- Infix expression: string concatenation ----

func TestInferInfix_StringConcat(t *testing.T) {
	errs := check(`
a: String = "hello";
b: String = " world";
c: String = a + b;
`)
	if len(errs) != 0 {
		t.Errorf("expected no errors for string concatenation, got: %v", errs)
	}
}

func TestInferInfix_StringConcat_Mismatch(t *testing.T) {
	errs := check(`
a: String = "hello";
b: String = " world";
c: Int = a + b;
`)
	if len(errs) == 0 {
		t.Error("expected type error for String + String assigned to Int, got none")
	}
}

// ---- Infix: float promotion ----

func TestInferInfix_FloatPromotion(t *testing.T) {
	errs := check(`
a: Int = 3;
b: Float = 1.5;
c: Float = a + b;
`)
	if len(errs) != 0 {
		t.Errorf("expected no errors for Int + Float -> Float, got: %v", errs)
	}
}

// ---- Infix: comparison operators ----

func TestInferInfix_Comparison(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{"equal", `x: Bool = 1 == 2;`},
		{"not-equal", `x: Bool = 1 != 2;`},
		{"less-than", `x: Bool = 1 < 2;`},
		{"greater-than", `x: Bool = 1 > 2;`},
		{"less-equal", `x: Bool = 1 <= 2;`},
		{"greater-equal", `x: Bool = 1 >= 2;`},
		{"and", `x: Bool = true && false;`},
		{"or", `x: Bool = true || false;`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := check(tt.src)
			if len(errs) != 0 {
				t.Errorf("expected no errors, got: %v", errs)
			}
		})
	}
}

// ---- Infix: unknown operator ----

func TestInferInfix_UnknownOperator(t *testing.T) {
	// Other numeric operators
	tests := []struct {
		name string
		src  string
	}{
		{"modulo int", `a: Int = 3; b: Int = 4; c: Int = a % b;`},
		{"power int", `a: Int = 2; b: Int = 3; c: Int = a ** b;`},
		{"subtract int", `a: Int = 5; b: Int = 2; c: Int = a - b;`},
		{"multiply int", `a: Int = 3; b: Int = 4; c: Int = a * b;`},
		{"divide int", `a: Int = 10; b: Int = 2; c: Int = a / b;`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := check(tt.src)
			if len(errs) != 0 {
				t.Errorf("expected no errors, got: %v", errs)
			}
		})
	}
}

// ---- Identifier: known vs unknown ----

func TestInferExpr_IdentifierKnown(t *testing.T) {
	errs := check(`
x: Int = 42;
y: Int = x;
`)
	if len(errs) != 0 {
		t.Errorf("expected no errors, got: %v", errs)
	}
}

func TestInferExpr_IdentifierUnknown(t *testing.T) {
	// Unknown identifier should return Any, which is compatible with anything
	errs := check(`x: Int = unknown_var;`)
	if len(errs) != 0 {
		t.Errorf("expected no errors for unknown identifier (returns Any), got: %v", errs)
	}
}

// ---- New() ----

func TestNew(t *testing.T) {
	c := typechecker.New()
	if c == nil {
		t.Error("New() returned nil")
	}
}

// ---- Function with more params than type annotations ----

func TestFunctionDeclaration_PartialParamTypes(t *testing.T) {
	// Only first param has type annotation
	errs := check(`func process(a: Int, b) = a + b;`)
	if len(errs) != 0 {
		t.Errorf("expected no errors for partial param types, got: %v", errs)
	}
}

// ---- Infix: Any operand ----

func TestInferInfix_AnyOperand(t *testing.T) {
	// When one operand is Any (unknown variable), result is Any
	errs := check(`c = unknown_var + 1;`)
	if len(errs) != 0 {
		t.Errorf("expected no errors for Any + Int, got: %v", errs)
	}
}
