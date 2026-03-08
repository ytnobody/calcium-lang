package optimizer

import (
	"testing"

	"github.com/ytnobody/calcium-lang/pkg/ast"
	"github.com/ytnobody/calcium-lang/pkg/bytecode"
	"github.com/ytnobody/calcium-lang/pkg/lexer"
	"github.com/ytnobody/calcium-lang/pkg/parser"
	"github.com/ytnobody/calcium-lang/pkg/token"
)

func parse(input string) *ast.Program {
	l := lexer.New(input)
	p := parser.New(l)
	return p.ParseProgram()
}

// Helper to extract the expression from a single-statement program
func getExpr(program *ast.Program) ast.Expression {
	if len(program.Statements) == 0 {
		return nil
	}
	es, ok := program.Statements[0].(*ast.ExpressionStatement)
	if !ok {
		return nil
	}
	return es.Expression
}

// TestConstantFoldingArithmetic tests folding of arithmetic operations
func TestConstantFoldingArithmetic(t *testing.T) {
	tests := []struct {
		input    string
		expected any
	}{
		// Integer arithmetic
		{"2 + 3;", int64(5)},
		{"10 - 4;", int64(6)},
		{"3 * 4;", int64(12)},
		{"20 / 4;", int64(5)},
		{"17 % 5;", int64(2)},
		{"2 ** 3;", int64(8)},

		// Nested expressions
		{"(2 + 3) * 4;", int64(20)},
		{"2 + 3 * 4;", int64(14)},
		{"(10 - 2) / (1 + 1);", int64(4)},

		// Unary minus
		{"-5;", int64(-5)},
		{"-(-3);", int64(3)},
		{"-(2 + 3);", int64(-5)},

		// Float arithmetic
		{"2.5 + 1.5;", float64(4.0)},
		{"3.0 * 2.0;", float64(6.0)},

		// Mixed int/float
		{"2 + 3.5;", float64(5.5)},
		{"10.0 / 4;", float64(2.5)},
	}

	for _, tt := range tests {
		program := parse(tt.input)
		optimized := FoldConstants(program).(*ast.Program)
		expr := getExpr(optimized)

		switch expected := tt.expected.(type) {
		case int64:
			intLit, ok := expr.(*ast.IntegerLiteral)
			if !ok {
				t.Errorf("input %q: expected IntegerLiteral, got %T", tt.input, expr)
				continue
			}
			if intLit.Value != expected {
				t.Errorf("input %q: expected %d, got %d", tt.input, expected, intLit.Value)
			}
		case float64:
			floatLit, ok := expr.(*ast.FloatLiteral)
			if !ok {
				t.Errorf("input %q: expected FloatLiteral, got %T", tt.input, expr)
				continue
			}
			if floatLit.Value != expected {
				t.Errorf("input %q: expected %f, got %f", tt.input, expected, floatLit.Value)
			}
		}
	}
}

// TestConstantFoldingComparison tests folding of comparison operations
func TestConstantFoldingComparison(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"5 > 3;", true},
		{"2 < 1;", false},
		{"3 >= 3;", true},
		{"4 <= 3;", false},
		{"5 == 5;", true},
		{"5 != 5;", false},
		{"3 == 4;", false},
		{"3 != 4;", true},
	}

	for _, tt := range tests {
		program := parse(tt.input)
		optimized := FoldConstants(program).(*ast.Program)
		expr := getExpr(optimized)

		boolLit, ok := expr.(*ast.BooleanLiteral)
		if !ok {
			t.Errorf("input %q: expected BooleanLiteral, got %T", tt.input, expr)
			continue
		}
		if boolLit.Value != tt.expected {
			t.Errorf("input %q: expected %v, got %v", tt.input, tt.expected, boolLit.Value)
		}
	}
}

// TestConstantFoldingLogical tests folding of logical operations
func TestConstantFoldingLogical(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"true && true;", true},
		{"true && false;", false},
		{"false && true;", false},
		{"false && false;", false},
		{"true || true;", true},
		{"true || false;", true},
		{"false || true;", true},
		{"false || false;", false},
		{"!true;", false},
		{"!false;", true},
		{"!!true;", true},
	}

	for _, tt := range tests {
		program := parse(tt.input)
		optimized := FoldConstants(program).(*ast.Program)
		expr := getExpr(optimized)

		boolLit, ok := expr.(*ast.BooleanLiteral)
		if !ok {
			t.Errorf("input %q: expected BooleanLiteral, got %T", tt.input, expr)
			continue
		}
		if boolLit.Value != tt.expected {
			t.Errorf("input %q: expected %v, got %v", tt.input, tt.expected, boolLit.Value)
		}
	}
}

// TestConstantFoldingString tests folding of string concatenation
func TestConstantFoldingString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`"hello" + " " + "world";`, "hello world"},
		{`"a" + "b" + "c";`, "abc"},
	}

	for _, tt := range tests {
		program := parse(tt.input)
		optimized := FoldConstants(program).(*ast.Program)
		expr := getExpr(optimized)

		strLit, ok := expr.(*ast.StringLiteral)
		if !ok {
			t.Errorf("input %q: expected StringLiteral, got %T", tt.input, expr)
			continue
		}
		if strLit.Value != tt.expected {
			t.Errorf("input %q: expected %q, got %q", tt.input, tt.expected, strLit.Value)
		}
	}
}

// TestConstantFoldingArrayIndex tests folding of constant array indexing
func TestConstantFoldingArrayIndex(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"[1, 2, 3][0];", 1},
		{"[10, 20, 30][1];", 20},
		{"[100, 200, 300][2];", 300},
	}

	for _, tt := range tests {
		program := parse(tt.input)
		optimized := FoldConstants(program).(*ast.Program)
		expr := getExpr(optimized)

		intLit, ok := expr.(*ast.IntegerLiteral)
		if !ok {
			t.Errorf("input %q: expected IntegerLiteral, got %T", tt.input, expr)
			continue
		}
		if intLit.Value != tt.expected {
			t.Errorf("input %q: expected %d, got %d", tt.input, tt.expected, intLit.Value)
		}
	}
}

// TestConstantFoldingStringIndex tests folding of constant string indexing
func TestConstantFoldingStringIndex(t *testing.T) {
	program := parse(`"hello"[1];`)
	optimized := FoldConstants(program).(*ast.Program)
	expr := getExpr(optimized)

	strLit, ok := expr.(*ast.StringLiteral)
	if !ok {
		t.Fatalf("expected StringLiteral, got %T", expr)
	}
	if strLit.Value != "e" {
		t.Errorf("expected 'e', got %q", strLit.Value)
	}
}

// TestConstantFoldingNoFold tests that non-constant expressions are not folded
func TestConstantFoldingNoFold(t *testing.T) {
	tests := []string{
		"x + 3;",        // Variable involved
		"f(2);",         // Function call
		"10 / 0;",       // Division by zero
		"[1, 2, 3][x];", // Non-constant index
	}

	for _, input := range tests {
		program := parse(input)
		optimized := FoldConstants(program).(*ast.Program)
		expr := getExpr(optimized)

		// These should NOT be folded to literals
		switch expr.(type) {
		case *ast.IntegerLiteral, *ast.FloatLiteral, *ast.BooleanLiteral, *ast.StringLiteral:
			t.Errorf("input %q: should not have been folded to a literal", input)
		}
	}
}

// TestConstantFoldingModuloByZero tests that modulo by zero is not folded
func TestConstantFoldingModuloByZero(t *testing.T) {
	program := parse("10 % 0;")
	optimized := FoldConstants(program).(*ast.Program)
	expr := getExpr(optimized)

	// Should remain an infix expression (not folded)
	if _, ok := expr.(*ast.IntegerLiteral); ok {
		t.Error("modulo by zero should not be folded")
	}
}

// TestConstantFoldingFloatComparison tests folding of float comparison operations
func TestConstantFoldingFloatComparison(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"3.5 > 2.5;", true},
		{"1.0 < 2.0;", true},
		{"3.0 >= 3.0;", true},
		{"2.5 <= 3.5;", true},
		{"2.0 == 2.0;", true},
		{"2.0 != 3.0;", true},
	}

	for _, tt := range tests {
		program := parse(tt.input)
		optimized := FoldConstants(program).(*ast.Program)
		expr := getExpr(optimized)

		boolLit, ok := expr.(*ast.BooleanLiteral)
		if !ok {
			t.Errorf("input %q: expected BooleanLiteral, got %T", tt.input, expr)
			continue
		}
		if boolLit.Value != tt.expected {
			t.Errorf("input %q: expected %v, got %v", tt.input, tt.expected, boolLit.Value)
		}
	}
}

// TestConstantFoldingFloatDivByZero tests that float division by zero is not folded
func TestConstantFoldingFloatDivByZero(t *testing.T) {
	program := parse("10.0 / 0.0;")
	optimized := FoldConstants(program).(*ast.Program)
	expr := getExpr(optimized)

	if _, ok := expr.(*ast.FloatLiteral); ok {
		t.Error("float division by zero should not be folded")
	}
}

// TestConstantFoldingFloatPower tests folding of float power
func TestConstantFoldingFloatPower(t *testing.T) {
	program := parse("2.0 ** 3.0;")
	optimized := FoldConstants(program).(*ast.Program)
	expr := getExpr(optimized)

	floatLit, ok := expr.(*ast.FloatLiteral)
	if !ok {
		t.Fatalf("expected FloatLiteral, got %T", expr)
	}
	if floatLit.Value != 8.0 {
		t.Errorf("expected 8.0, got %f", floatLit.Value)
	}
}

// TestConstantFoldingBooleanEquality tests boolean == and != folding
func TestConstantFoldingBooleanEquality(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"true == true;", true},
		{"true == false;", false},
		{"true != false;", true},
		{"false != false;", false},
	}

	for _, tt := range tests {
		program := parse(tt.input)
		optimized := FoldConstants(program).(*ast.Program)
		expr := getExpr(optimized)

		boolLit, ok := expr.(*ast.BooleanLiteral)
		if !ok {
			t.Errorf("input %q: expected BooleanLiteral, got %T", tt.input, expr)
			continue
		}
		if boolLit.Value != tt.expected {
			t.Errorf("input %q: expected %v, got %v", tt.input, tt.expected, boolLit.Value)
		}
	}
}

// TestConstantFoldingUnknownOperator tests that unknown operators are not folded
func TestConstantFoldingUnknownOperator(t *testing.T) {
	// Create a custom infix expression with an unknown operator
	tok := token.Token{Type: token.PLUS, Literal: "?!"}
	expr := &ast.InfixExpression{
		Token:    tok,
		Left:     &ast.IntegerLiteral{Token: tok, Value: 1},
		Operator: "?!",
		Right:    &ast.IntegerLiteral{Token: tok, Value: 2},
	}
	program := &ast.Program{
		Statements: []ast.Statement{
			&ast.ExpressionStatement{Token: tok, Expression: expr},
		},
	}

	optimized := FoldConstants(program).(*ast.Program)
	result := getExpr(optimized)

	// Should not be folded to a literal
	if _, ok := result.(*ast.IntegerLiteral); ok {
		t.Error("unknown operator should not be folded")
	}
}

// TestConstantFoldingPrefixUnknown tests unknown prefix operators
func TestConstantFoldingPrefixUnknown(t *testing.T) {
	tok := token.Token{Type: token.BANG, Literal: "~"}
	expr := &ast.PrefixExpression{
		Token:    tok,
		Operator: "~",
		Right:    &ast.IntegerLiteral{Token: tok, Value: 5},
	}
	program := &ast.Program{
		Statements: []ast.Statement{
			&ast.ExpressionStatement{Token: tok, Expression: expr},
		},
	}

	optimized := FoldConstants(program).(*ast.Program)
	result := getExpr(optimized)

	// Should remain a prefix expression
	if _, ok := result.(*ast.PrefixExpression); !ok {
		t.Errorf("expected PrefixExpression, got %T", result)
	}
}

// TestConstantFoldingFunctionDeclaration tests folding within function bodies
func TestConstantFoldingFunctionDeclaration(t *testing.T) {
	program := parse("func add(a, b) = 2 + 3;")
	optimized := FoldConstants(program).(*ast.Program)

	if len(optimized.Statements) < 1 {
		t.Fatal("expected at least 1 statement")
	}
	fd, ok := optimized.Statements[0].(*ast.FunctionDeclaration)
	if !ok {
		t.Fatalf("expected FunctionDeclaration, got %T", optimized.Statements[0])
	}
	intLit, ok := fd.Body.(*ast.IntegerLiteral)
	if !ok {
		t.Fatalf("expected IntegerLiteral body, got %T", fd.Body)
	}
	if intLit.Value != 5 {
		t.Errorf("expected 5, got %d", intLit.Value)
	}
}

// TestConstantFoldingConstraintDeclaration tests folding within constraint bodies
func TestConstantFoldingConstraintDeclaration(t *testing.T) {
	program := parse("constraint Positive(n) = 1 + 1;")
	optimized := FoldConstants(program).(*ast.Program)

	if len(optimized.Statements) < 1 {
		t.Fatal("expected at least 1 statement")
	}
	cd, ok := optimized.Statements[0].(*ast.ConstraintDeclaration)
	if !ok {
		t.Fatalf("expected ConstraintDeclaration, got %T", optimized.Statements[0])
	}
	intLit, ok := cd.Body.(*ast.IntegerLiteral)
	if !ok {
		t.Fatalf("expected IntegerLiteral body, got %T", cd.Body)
	}
	if intLit.Value != 2 {
		t.Errorf("expected 2, got %d", intLit.Value)
	}
}

// TestConstantFoldingArrayDestructuring tests folding in destructuring value
func TestConstantFoldingArrayDestructuring(t *testing.T) {
	program := parse("[a, b] = [1 + 2, 3 + 4];")
	optimized := FoldConstants(program).(*ast.Program)

	if len(optimized.Statements) < 1 {
		t.Fatal("expected at least 1 statement")
	}
	ad, ok := optimized.Statements[0].(*ast.ArrayDestructuringStatement)
	if !ok {
		t.Fatalf("expected ArrayDestructuringStatement, got %T", optimized.Statements[0])
	}
	// The value should be an array literal with folded elements
	arr, ok := ad.Value.(*ast.ArrayLiteral)
	if !ok {
		t.Fatalf("expected ArrayLiteral value, got %T", ad.Value)
	}
	if len(arr.Elements) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(arr.Elements))
	}
	intLit, ok := arr.Elements[0].(*ast.IntegerLiteral)
	if !ok {
		t.Fatalf("expected IntegerLiteral, got %T", arr.Elements[0])
	}
	if intLit.Value != 3 {
		t.Errorf("expected 3, got %d", intLit.Value)
	}
}

// TestConstantFoldingHeadTailDestructuring tests folding in head-tail destructuring
func TestConstantFoldingHeadTailDestructuring(t *testing.T) {
	program := parse("[head | tail] = [1 + 1, 2, 3];")
	optimized := FoldConstants(program).(*ast.Program)

	if len(optimized.Statements) < 1 {
		t.Fatal("expected at least 1 statement")
	}
	ht, ok := optimized.Statements[0].(*ast.HeadTailDestructuringStatement)
	if !ok {
		t.Fatalf("expected HeadTailDestructuringStatement, got %T", optimized.Statements[0])
	}
	arr, ok := ht.Value.(*ast.ArrayLiteral)
	if !ok {
		t.Fatalf("expected ArrayLiteral value, got %T", ht.Value)
	}
	intLit, ok := arr.Elements[0].(*ast.IntegerLiteral)
	if !ok {
		t.Fatalf("expected IntegerLiteral, got %T", arr.Elements[0])
	}
	if intLit.Value != 2 {
		t.Errorf("expected 2, got %d", intLit.Value)
	}
}

// TestConstantFoldingCallExpression tests folding of call arguments
func TestConstantFoldingCallExpression(t *testing.T) {
	program := parse("f(2 + 3, 4 * 5);")
	optimized := FoldConstants(program).(*ast.Program)
	expr := getExpr(optimized)

	call, ok := expr.(*ast.CallExpression)
	if !ok {
		t.Fatalf("expected CallExpression, got %T", expr)
	}
	if len(call.Arguments) != 2 {
		t.Fatalf("expected 2 args, got %d", len(call.Arguments))
	}
	arg1, ok := call.Arguments[0].(*ast.IntegerLiteral)
	if !ok {
		t.Fatalf("expected IntegerLiteral arg, got %T", call.Arguments[0])
	}
	if arg1.Value != 5 {
		t.Errorf("expected 5, got %d", arg1.Value)
	}
	arg2, ok := call.Arguments[1].(*ast.IntegerLiteral)
	if !ok {
		t.Fatalf("expected IntegerLiteral arg, got %T", call.Arguments[1])
	}
	if arg2.Value != 20 {
		t.Errorf("expected 20, got %d", arg2.Value)
	}
}

// TestConstantFoldingLambdaExpression tests folding within lambda body
func TestConstantFoldingLambdaExpression(t *testing.T) {
	program := parse("x => 2 + 3;")
	optimized := FoldConstants(program).(*ast.Program)
	expr := getExpr(optimized)

	lambda, ok := expr.(*ast.LambdaExpression)
	if !ok {
		t.Fatalf("expected LambdaExpression, got %T", expr)
	}
	intLit, ok := lambda.Body.(*ast.IntegerLiteral)
	if !ok {
		t.Fatalf("expected IntegerLiteral body, got %T", lambda.Body)
	}
	if intLit.Value != 5 {
		t.Errorf("expected 5, got %d", intLit.Value)
	}
}

// TestConstantFoldingPipeExpression tests folding within pipe expressions
func TestConstantFoldingPipeExpression(t *testing.T) {
	program := parse("(2 + 3) |> f;")
	optimized := FoldConstants(program).(*ast.Program)
	expr := getExpr(optimized)

	pipe, ok := expr.(*ast.PipeExpression)
	if !ok {
		t.Fatalf("expected PipeExpression, got %T", expr)
	}
	intLit, ok := pipe.Left.(*ast.IntegerLiteral)
	if !ok {
		t.Fatalf("expected IntegerLiteral left, got %T", pipe.Left)
	}
	if intLit.Value != 5 {
		t.Errorf("expected 5, got %d", intLit.Value)
	}
}

// TestConstantFoldingHashLiteral tests folding within hash literals
func TestConstantFoldingHashLiteral(t *testing.T) {
	program := parse(`{"a": 1 + 2, "b": 3 + 4};`)
	optimized := FoldConstants(program).(*ast.Program)
	expr := getExpr(optimized)

	hash, ok := expr.(*ast.HashLiteral)
	if !ok {
		t.Fatalf("expected HashLiteral, got %T", expr)
	}
	if len(hash.Pairs) != 2 {
		t.Fatalf("expected 2 pairs, got %d", len(hash.Pairs))
	}
	val, ok := hash.Pairs[0].Value.(*ast.IntegerLiteral)
	if !ok {
		t.Fatalf("expected IntegerLiteral, got %T", hash.Pairs[0].Value)
	}
	if val.Value != 3 {
		t.Errorf("expected 3, got %d", val.Value)
	}
}

// TestConstantFoldingSuccessExpression tests folding within success expression
func TestConstantFoldingSuccessExpression(t *testing.T) {
	program := parse("success(2 + 3);")
	optimized := FoldConstants(program).(*ast.Program)
	expr := getExpr(optimized)

	succ, ok := expr.(*ast.SuccessExpression)
	if !ok {
		t.Fatalf("expected SuccessExpression, got %T", expr)
	}
	intLit, ok := succ.Value.(*ast.IntegerLiteral)
	if !ok {
		t.Fatalf("expected IntegerLiteral value, got %T", succ.Value)
	}
	if intLit.Value != 5 {
		t.Errorf("expected 5, got %d", intLit.Value)
	}
}

// TestConstantFoldingFailureExpression tests folding within failure expression
func TestConstantFoldingFailureExpression(t *testing.T) {
	program := parse(`failure("err");`)
	optimized := FoldConstants(program).(*ast.Program)
	expr := getExpr(optimized)

	fail, ok := expr.(*ast.FailureExpression)
	if !ok {
		t.Fatalf("expected FailureExpression, got %T", expr)
	}
	if fail.Value == nil {
		t.Fatal("expected non-nil value")
	}
}

// TestConstantFoldingReturnExpression tests folding within return expression
func TestConstantFoldingReturnExpression(t *testing.T) {
	program := parse("return 2 + 3;")
	optimized := FoldConstants(program).(*ast.Program)
	expr := getExpr(optimized)

	ret, ok := expr.(*ast.ReturnExpression)
	if !ok {
		t.Fatalf("expected ReturnExpression, got %T", expr)
	}
	intLit, ok := ret.Value.(*ast.IntegerLiteral)
	if !ok {
		t.Fatalf("expected IntegerLiteral value, got %T", ret.Value)
	}
	if intLit.Value != 5 {
		t.Errorf("expected 5, got %d", intLit.Value)
	}
}

// TestConstantFoldingMemberExpression tests folding within member expressions
func TestConstantFoldingMemberExpression(t *testing.T) {
	// Member expression object is typically an identifier, so folding won't change it
	program := parse("x.y;")
	optimized := FoldConstants(program).(*ast.Program)
	expr := getExpr(optimized)

	_, ok := expr.(*ast.MemberExpression)
	if !ok {
		t.Fatalf("expected MemberExpression, got %T", expr)
	}
}

// TestConstantFoldingSpreadExpression tests folding within spread expressions
func TestConstantFoldingSpreadExpression(t *testing.T) {
	program := parse("[1, 2, 3]... |> f;")
	optimized := FoldConstants(program).(*ast.Program)
	expr := getExpr(optimized)

	pipe, ok := expr.(*ast.PipeExpression)
	if !ok {
		t.Fatalf("expected PipeExpression, got %T", expr)
	}
	_, ok = pipe.Left.(*ast.SpreadExpression)
	if !ok {
		t.Fatalf("expected SpreadExpression, got %T", pipe.Left)
	}
}

// TestConstantFoldingMatchExpression tests folding within match expression cases
func TestConstantFoldingMatchExpression(t *testing.T) {
	program := parse("match x 1 => 2 + 3 _ => 4 + 5;")
	optimized := FoldConstants(program).(*ast.Program)
	expr := getExpr(optimized)

	me, ok := expr.(*ast.MatchExpression)
	if !ok {
		t.Fatalf("expected MatchExpression, got %T", expr)
	}
	if len(me.Cases) < 2 {
		t.Fatalf("expected at least 2 cases, got %d", len(me.Cases))
	}
	intLit, ok := me.Cases[0].Body.(*ast.IntegerLiteral)
	if !ok {
		t.Fatalf("expected IntegerLiteral body in first case, got %T", me.Cases[0].Body)
	}
	if intLit.Value != 5 {
		t.Errorf("expected 5, got %d", intLit.Value)
	}
}

// TestConstantFoldingEffectHandleExpression tests folding within effect handle
func TestConstantFoldingEffectHandleExpression(t *testing.T) {
	program := parse(`x !? { success(v) => 1 + 2 failure(e) => 3 + 4 };`)
	optimized := FoldConstants(program).(*ast.Program)
	expr := getExpr(optimized)

	handle, ok := expr.(*ast.EffectHandleExpression)
	if !ok {
		t.Fatalf("expected EffectHandleExpression, got %T", expr)
	}
	succBody, ok := handle.SuccessBody.(*ast.IntegerLiteral)
	if !ok {
		t.Fatalf("expected IntegerLiteral success body, got %T", handle.SuccessBody)
	}
	if succBody.Value != 3 {
		t.Errorf("expected 3, got %d", succBody.Value)
	}
	failBody, ok := handle.FailureBody.(*ast.IntegerLiteral)
	if !ok {
		t.Fatalf("expected IntegerLiteral failure body, got %T", handle.FailureBody)
	}
	if failBody.Value != 7 {
		t.Errorf("expected 7, got %d", failBody.Value)
	}
}

// TestConstantFoldingInterpolatedStringAllConstant tests all-constant interpolation
func TestConstantFoldingInterpolatedStringAllConstant(t *testing.T) {
	// Construct an interpolated string with all string literal parts directly
	tok := token.Token{Type: token.STRING, Literal: ""}
	is := &ast.InterpolatedString{
		Token: tok,
		Parts: []ast.Expression{
			&ast.StringLiteral{Token: tok, Value: "hello "},
			&ast.StringLiteral{Token: tok, Value: "world"},
		},
	}
	program := &ast.Program{
		Statements: []ast.Statement{
			&ast.ExpressionStatement{Token: tok, Expression: is},
		},
	}

	optimized := FoldConstants(program).(*ast.Program)
	expr := getExpr(optimized)

	strLit, ok := expr.(*ast.StringLiteral)
	if !ok {
		t.Fatalf("expected StringLiteral, got %T", expr)
	}
	if strLit.Value != "hello world" {
		t.Errorf("expected 'hello world', got %q", strLit.Value)
	}
}

// TestConstantFoldingInterpolatedStringWithVariable tests mixed interpolation
func TestConstantFoldingInterpolatedStringWithVariable(t *testing.T) {
	program := parse(`"hello ${name} world";`)
	optimized := FoldConstants(program).(*ast.Program)
	expr := getExpr(optimized)

	// Should remain an interpolated string since name is not constant
	_, ok := expr.(*ast.InterpolatedString)
	if !ok {
		t.Fatalf("expected InterpolatedString, got %T", expr)
	}
}

// TestConstantFoldingNilNode tests folding on nil input
func TestConstantFoldingNilNode(t *testing.T) {
	result := FoldConstants(nil)
	if result != nil {
		t.Error("expected nil result for nil input")
	}
}

// TestConstantFoldingEffectPipeExpression tests folding within effect pipe
func TestConstantFoldingEffectPipeExpression(t *testing.T) {
	tok := token.Token{Type: token.BANG, Literal: "!>"}
	ep := &ast.EffectPipeExpression{
		Token: tok,
		Left:  &ast.InfixExpression{Token: tok, Left: &ast.IntegerLiteral{Token: tok, Value: 1}, Operator: "+", Right: &ast.IntegerLiteral{Token: tok, Value: 2}},
		Right: &ast.Identifier{Token: tok, Value: "f"},
	}
	program := &ast.Program{
		Statements: []ast.Statement{
			&ast.ExpressionStatement{Token: tok, Expression: ep},
		},
	}

	optimized := FoldConstants(program).(*ast.Program)
	expr := getExpr(optimized)

	effPipe, ok := expr.(*ast.EffectPipeExpression)
	if !ok {
		t.Fatalf("expected EffectPipeExpression, got %T", expr)
	}
	intLit, ok := effPipe.Left.(*ast.IntegerLiteral)
	if !ok {
		t.Fatalf("expected IntegerLiteral left, got %T", effPipe.Left)
	}
	if intLit.Value != 3 {
		t.Errorf("expected 3, got %d", intLit.Value)
	}
}

// TestConstantFoldingStayExpression tests folding within stay expressions
func TestConstantFoldingStayExpression(t *testing.T) {
	tok := token.Token{Type: token.IDENT, Literal: "stay"}
	stay := &ast.StayExpression{
		Token: tok,
		StateInit: []ast.StayStatePair{
			{
				Name:  &ast.Identifier{Token: tok, Value: "count"},
				Value: &ast.InfixExpression{Token: tok, Left: &ast.IntegerLiteral{Token: tok, Value: 1}, Operator: "+", Right: &ast.IntegerLiteral{Token: tok, Value: 2}},
			},
		},
		Body: []ast.Statement{
			&ast.ExpressionStatement{Token: tok, Expression: &ast.InfixExpression{Token: tok, Left: &ast.IntegerLiteral{Token: tok, Value: 3}, Operator: "+", Right: &ast.IntegerLiteral{Token: tok, Value: 4}}},
		},
	}
	program := &ast.Program{
		Statements: []ast.Statement{
			&ast.ExpressionStatement{Token: tok, Expression: stay},
		},
	}

	optimized := FoldConstants(program).(*ast.Program)
	expr := getExpr(optimized)

	stayExpr, ok := expr.(*ast.StayExpression)
	if !ok {
		t.Fatalf("expected StayExpression, got %T", expr)
	}
	intLit, ok := stayExpr.StateInit[0].Value.(*ast.IntegerLiteral)
	if !ok {
		t.Fatalf("expected IntegerLiteral in state init, got %T", stayExpr.StateInit[0].Value)
	}
	if intLit.Value != 3 {
		t.Errorf("expected 3, got %d", intLit.Value)
	}
}

// TestConstantFoldingArrayOutOfBounds tests that out-of-bounds indexing is not folded
func TestConstantFoldingArrayOutOfBounds(t *testing.T) {
	program := parse("[1, 2, 3][5];")
	optimized := FoldConstants(program).(*ast.Program)
	expr := getExpr(optimized)

	// Should remain an index expression (not folded)
	if _, ok := expr.(*ast.IndexExpression); !ok {
		t.Errorf("expected IndexExpression for out-of-bounds, got %T", expr)
	}
}

// TestConstantFoldingNegativeIndex tests that negative indexing is not folded
func TestConstantFoldingNegativeIndex(t *testing.T) {
	tok := token.Token{Type: token.LBRACKET, Literal: "["}
	ie := &ast.IndexExpression{
		Token: tok,
		Left: &ast.ArrayLiteral{
			Token:    tok,
			Elements: []ast.Expression{&ast.IntegerLiteral{Token: tok, Value: 10}},
		},
		Index: &ast.IntegerLiteral{Token: tok, Value: -1},
	}
	program := &ast.Program{
		Statements: []ast.Statement{
			&ast.ExpressionStatement{Token: tok, Expression: ie},
		},
	}

	optimized := FoldConstants(program).(*ast.Program)
	expr := getExpr(optimized)

	if _, ok := expr.(*ast.IndexExpression); !ok {
		t.Errorf("expected IndexExpression for negative index, got %T", expr)
	}
}

// === Dead Code Elimination Tests ===

// TestDeadCodeEliminationMatch tests match expression simplification
func TestDeadCodeEliminationMatch(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		// Boolean match with constant subject
		{"match true true => 1 false => 2;", 1},
		{"match false true => 1 false => 2;", 2},

		// Integer match with constant subject
		{"match 1 1 => 10 2 => 20 _ => 30;", 10},
		{"match 2 1 => 10 2 => 20 _ => 30;", 20},
		{"match 3 1 => 10 2 => 20 _ => 30;", 30},
	}

	for _, tt := range tests {
		program := parse(tt.input)
		optimized := EliminateDeadCode(program).(*ast.Program)
		expr := getExpr(optimized)

		intLit, ok := expr.(*ast.IntegerLiteral)
		if !ok {
			t.Errorf("input %q: expected IntegerLiteral, got %T", tt.input, expr)
			continue
		}
		if intLit.Value != tt.expected {
			t.Errorf("input %q: expected %d, got %d", tt.input, tt.expected, intLit.Value)
		}
	}
}

// TestDeadCodeEliminationMatchString tests string match simplification
func TestDeadCodeEliminationMatchString(t *testing.T) {
	tok := token.Token{Type: token.MATCH, Literal: "match"}
	me := &ast.MatchExpression{
		Token:   tok,
		Subject: &ast.StringLiteral{Token: tok, Value: "hello"},
		Cases: []*ast.MatchCase{
			{Token: tok, Pattern: &ast.StringLiteral{Token: tok, Value: "hello"}, Body: &ast.IntegerLiteral{Token: tok, Value: 1}},
			{Token: tok, Pattern: &ast.StringLiteral{Token: tok, Value: "world"}, Body: &ast.IntegerLiteral{Token: tok, Value: 2}},
		},
	}
	program := &ast.Program{
		Statements: []ast.Statement{
			&ast.ExpressionStatement{Token: tok, Expression: me},
		},
	}

	optimized := EliminateDeadCode(program).(*ast.Program)
	expr := getExpr(optimized)

	intLit, ok := expr.(*ast.IntegerLiteral)
	if !ok {
		t.Fatalf("expected IntegerLiteral, got %T", expr)
	}
	if intLit.Value != 1 {
		t.Errorf("expected 1, got %d", intLit.Value)
	}
}

// TestDeadCodeEliminationMatchStringDefault tests string match with default
func TestDeadCodeEliminationMatchStringDefault(t *testing.T) {
	tok := token.Token{Type: token.MATCH, Literal: "match"}
	me := &ast.MatchExpression{
		Token:   tok,
		Subject: &ast.StringLiteral{Token: tok, Value: "other"},
		Cases: []*ast.MatchCase{
			{Token: tok, Pattern: &ast.StringLiteral{Token: tok, Value: "hello"}, Body: &ast.IntegerLiteral{Token: tok, Value: 1}},
			{Token: tok, Pattern: &ast.Identifier{Token: tok, Value: "_"}, Body: &ast.IntegerLiteral{Token: tok, Value: 99}, IsDefault: true},
		},
	}
	program := &ast.Program{
		Statements: []ast.Statement{
			&ast.ExpressionStatement{Token: tok, Expression: me},
		},
	}

	optimized := EliminateDeadCode(program).(*ast.Program)
	expr := getExpr(optimized)

	intLit, ok := expr.(*ast.IntegerLiteral)
	if !ok {
		t.Fatalf("expected IntegerLiteral, got %T", expr)
	}
	if intLit.Value != 99 {
		t.Errorf("expected 99, got %d", intLit.Value)
	}
}

// TestDeadCodeEliminationMatchNonConstant tests that non-constant match preserves all branches
func TestDeadCodeEliminationMatchNonConstant(t *testing.T) {
	program := parse("match x 1 => 10 _ => 20;")
	optimized := EliminateDeadCode(program).(*ast.Program)
	expr := getExpr(optimized)

	_, ok := expr.(*ast.MatchExpression)
	if !ok {
		t.Fatalf("expected MatchExpression for non-constant subject, got %T", expr)
	}
}

// TestDeadCodeEliminationShortCircuit tests short-circuit evaluation
func TestDeadCodeEliminationShortCircuit(t *testing.T) {
	// Test that false && x simplifies to false
	input := "false && expensive_call();"
	program := parse(input)
	optimized := EliminateDeadCode(program).(*ast.Program)
	expr := getExpr(optimized)

	boolLit, ok := expr.(*ast.BooleanLiteral)
	if !ok {
		t.Errorf("expected BooleanLiteral, got %T", expr)
	} else if boolLit.Value != false {
		t.Errorf("expected false, got %v", boolLit.Value)
	}

	// Test that true || x simplifies to true
	input = "true || expensive_call();"
	program = parse(input)
	optimized = EliminateDeadCode(program).(*ast.Program)
	expr = getExpr(optimized)

	boolLit, ok = expr.(*ast.BooleanLiteral)
	if !ok {
		t.Errorf("expected BooleanLiteral, got %T", expr)
	} else if boolLit.Value != true {
		t.Errorf("expected true, got %v", boolLit.Value)
	}

	// Test that true && x simplifies to x
	input = "true && some_expr;"
	program = parse(input)
	optimized = EliminateDeadCode(program).(*ast.Program)
	expr = getExpr(optimized)

	ident, ok := expr.(*ast.Identifier)
	if !ok {
		t.Errorf("expected Identifier, got %T", expr)
	} else if ident.Value != "some_expr" {
		t.Errorf("expected some_expr, got %s", ident.Value)
	}

	// Test that false || x simplifies to x
	input = "false || some_expr;"
	program = parse(input)
	optimized = EliminateDeadCode(program).(*ast.Program)
	expr = getExpr(optimized)

	ident, ok = expr.(*ast.Identifier)
	if !ok {
		t.Errorf("expected Identifier, got %T", expr)
	} else if ident.Value != "some_expr" {
		t.Errorf("expected some_expr, got %s", ident.Value)
	}
}

// TestDeadCodeEliminationFunctionDeclaration tests DCE within function bodies
func TestDeadCodeEliminationFunctionDeclaration(t *testing.T) {
	program := parse("func f(x) = false && expensive();")
	optimized := EliminateDeadCode(program).(*ast.Program)

	fd, ok := optimized.Statements[0].(*ast.FunctionDeclaration)
	if !ok {
		t.Fatalf("expected FunctionDeclaration, got %T", optimized.Statements[0])
	}
	boolLit, ok := fd.Body.(*ast.BooleanLiteral)
	if !ok {
		t.Fatalf("expected BooleanLiteral body, got %T", fd.Body)
	}
	if boolLit.Value != false {
		t.Errorf("expected false, got %v", boolLit.Value)
	}
}

// TestDeadCodeEliminationConstraintDeclaration tests DCE within constraint bodies
func TestDeadCodeEliminationConstraintDeclaration(t *testing.T) {
	program := parse("constraint C(n) = true || expensive();")
	optimized := EliminateDeadCode(program).(*ast.Program)

	cd, ok := optimized.Statements[0].(*ast.ConstraintDeclaration)
	if !ok {
		t.Fatalf("expected ConstraintDeclaration, got %T", optimized.Statements[0])
	}
	boolLit, ok := cd.Body.(*ast.BooleanLiteral)
	if !ok {
		t.Fatalf("expected BooleanLiteral body, got %T", cd.Body)
	}
	if boolLit.Value != true {
		t.Errorf("expected true, got %v", boolLit.Value)
	}
}

// TestDeadCodeEliminationArrayDestructuring tests DCE in destructuring
func TestDeadCodeEliminationArrayDestructuring(t *testing.T) {
	program := parse("[a, b] = [1, 2];")
	optimized := EliminateDeadCode(program).(*ast.Program)

	_, ok := optimized.Statements[0].(*ast.ArrayDestructuringStatement)
	if !ok {
		t.Fatalf("expected ArrayDestructuringStatement, got %T", optimized.Statements[0])
	}
}

// TestDeadCodeEliminationHeadTailDestructuring tests DCE in head-tail destructuring
func TestDeadCodeEliminationHeadTailDestructuring(t *testing.T) {
	program := parse("[head | tail] = [1, 2, 3];")
	optimized := EliminateDeadCode(program).(*ast.Program)

	_, ok := optimized.Statements[0].(*ast.HeadTailDestructuringStatement)
	if !ok {
		t.Fatalf("expected HeadTailDestructuringStatement, got %T", optimized.Statements[0])
	}
}

// TestDeadCodeEliminationPrefixExpression tests DCE within prefix expressions
func TestDeadCodeEliminationPrefixExpression(t *testing.T) {
	program := parse("!x;")
	optimized := EliminateDeadCode(program).(*ast.Program)
	expr := getExpr(optimized)

	_, ok := expr.(*ast.PrefixExpression)
	if !ok {
		t.Fatalf("expected PrefixExpression, got %T", expr)
	}
}

// TestDeadCodeEliminationArrayLiteral tests DCE within array literal
func TestDeadCodeEliminationArrayLiteral(t *testing.T) {
	program := parse("[1, false && x, 3];")
	optimized := EliminateDeadCode(program).(*ast.Program)
	expr := getExpr(optimized)

	arr, ok := expr.(*ast.ArrayLiteral)
	if !ok {
		t.Fatalf("expected ArrayLiteral, got %T", expr)
	}
	// The false && x should be simplified to false
	boolLit, ok := arr.Elements[1].(*ast.BooleanLiteral)
	if !ok {
		t.Fatalf("expected BooleanLiteral in array, got %T", arr.Elements[1])
	}
	if boolLit.Value != false {
		t.Errorf("expected false, got %v", boolLit.Value)
	}
}

// TestDeadCodeEliminationHashLiteral tests DCE within hash literal
func TestDeadCodeEliminationHashLiteral(t *testing.T) {
	program := parse(`{"a": true || x};`)
	optimized := EliminateDeadCode(program).(*ast.Program)
	expr := getExpr(optimized)

	hash, ok := expr.(*ast.HashLiteral)
	if !ok {
		t.Fatalf("expected HashLiteral, got %T", expr)
	}
	boolLit, ok := hash.Pairs[0].Value.(*ast.BooleanLiteral)
	if !ok {
		t.Fatalf("expected BooleanLiteral value, got %T", hash.Pairs[0].Value)
	}
	if boolLit.Value != true {
		t.Errorf("expected true, got %v", boolLit.Value)
	}
}

// TestDeadCodeEliminationIndexExpression tests DCE within index expression
func TestDeadCodeEliminationIndexExpression(t *testing.T) {
	program := parse("x[0];")
	optimized := EliminateDeadCode(program).(*ast.Program)
	expr := getExpr(optimized)

	_, ok := expr.(*ast.IndexExpression)
	if !ok {
		t.Fatalf("expected IndexExpression, got %T", expr)
	}
}

// TestDeadCodeEliminationCallExpression tests DCE within call arguments
func TestDeadCodeEliminationCallExpression(t *testing.T) {
	program := parse("f(false && x);")
	optimized := EliminateDeadCode(program).(*ast.Program)
	expr := getExpr(optimized)

	call, ok := expr.(*ast.CallExpression)
	if !ok {
		t.Fatalf("expected CallExpression, got %T", expr)
	}
	boolLit, ok := call.Arguments[0].(*ast.BooleanLiteral)
	if !ok {
		t.Fatalf("expected BooleanLiteral arg, got %T", call.Arguments[0])
	}
	if boolLit.Value != false {
		t.Errorf("expected false, got %v", boolLit.Value)
	}
}

// TestDeadCodeEliminationLambdaExpression tests DCE within lambda body
func TestDeadCodeEliminationLambdaExpression(t *testing.T) {
	program := parse("x => true || y;")
	optimized := EliminateDeadCode(program).(*ast.Program)
	expr := getExpr(optimized)

	lambda, ok := expr.(*ast.LambdaExpression)
	if !ok {
		t.Fatalf("expected LambdaExpression, got %T", expr)
	}
	boolLit, ok := lambda.Body.(*ast.BooleanLiteral)
	if !ok {
		t.Fatalf("expected BooleanLiteral body, got %T", lambda.Body)
	}
	if boolLit.Value != true {
		t.Errorf("expected true, got %v", boolLit.Value)
	}
}

// TestDeadCodeEliminationPipeExpression tests DCE within pipe expressions
func TestDeadCodeEliminationPipeExpression(t *testing.T) {
	program := parse("x |> f;")
	optimized := EliminateDeadCode(program).(*ast.Program)
	expr := getExpr(optimized)

	_, ok := expr.(*ast.PipeExpression)
	if !ok {
		t.Fatalf("expected PipeExpression, got %T", expr)
	}
}

// TestDeadCodeEliminationEffectPipeExpression tests DCE within effect pipe expressions
func TestDeadCodeEliminationEffectPipeExpression(t *testing.T) {
	tok := token.Token{Type: token.BANG, Literal: "!>"}
	ep := &ast.EffectPipeExpression{
		Token: tok,
		Left:  &ast.Identifier{Token: tok, Value: "x"},
		Right: &ast.Identifier{Token: tok, Value: "f"},
	}
	program := &ast.Program{
		Statements: []ast.Statement{
			&ast.ExpressionStatement{Token: tok, Expression: ep},
		},
	}

	optimized := EliminateDeadCode(program).(*ast.Program)
	expr := getExpr(optimized)

	_, ok := expr.(*ast.EffectPipeExpression)
	if !ok {
		t.Fatalf("expected EffectPipeExpression, got %T", expr)
	}
}

// TestDeadCodeEliminationEffectHandleExpression tests DCE within effect handle
func TestDeadCodeEliminationEffectHandleExpression(t *testing.T) {
	program := parse(`x !? { success(v) => v failure(e) => 0 };`)
	optimized := EliminateDeadCode(program).(*ast.Program)
	expr := getExpr(optimized)

	_, ok := expr.(*ast.EffectHandleExpression)
	if !ok {
		t.Fatalf("expected EffectHandleExpression, got %T", expr)
	}
}

// TestDeadCodeEliminationSuccessExpression tests DCE within success expressions
func TestDeadCodeEliminationSuccessExpression(t *testing.T) {
	program := parse("success(42);")
	optimized := EliminateDeadCode(program).(*ast.Program)
	expr := getExpr(optimized)

	_, ok := expr.(*ast.SuccessExpression)
	if !ok {
		t.Fatalf("expected SuccessExpression, got %T", expr)
	}
}

// TestDeadCodeEliminationFailureExpression tests DCE within failure expressions
func TestDeadCodeEliminationFailureExpression(t *testing.T) {
	program := parse(`failure("err");`)
	optimized := EliminateDeadCode(program).(*ast.Program)
	expr := getExpr(optimized)

	_, ok := expr.(*ast.FailureExpression)
	if !ok {
		t.Fatalf("expected FailureExpression, got %T", expr)
	}
}

// TestDeadCodeEliminationReturnExpression tests DCE within return expressions
func TestDeadCodeEliminationReturnExpression(t *testing.T) {
	program := parse("return 42;")
	optimized := EliminateDeadCode(program).(*ast.Program)
	expr := getExpr(optimized)

	_, ok := expr.(*ast.ReturnExpression)
	if !ok {
		t.Fatalf("expected ReturnExpression, got %T", expr)
	}
}

// TestDeadCodeEliminationMemberExpression tests DCE within member expressions
func TestDeadCodeEliminationMemberExpression(t *testing.T) {
	program := parse("x.y;")
	optimized := EliminateDeadCode(program).(*ast.Program)
	expr := getExpr(optimized)

	_, ok := expr.(*ast.MemberExpression)
	if !ok {
		t.Fatalf("expected MemberExpression, got %T", expr)
	}
}

// TestDeadCodeEliminationSpreadExpression tests DCE within spread expressions
func TestDeadCodeEliminationSpreadExpression(t *testing.T) {
	program := parse("[1, 2, 3]... |> f;")
	optimized := EliminateDeadCode(program).(*ast.Program)
	expr := getExpr(optimized)

	_, ok := expr.(*ast.PipeExpression)
	if !ok {
		t.Fatalf("expected PipeExpression, got %T", expr)
	}
}

// TestDeadCodeEliminationInterpolatedString tests DCE within interpolated strings
func TestDeadCodeEliminationInterpolatedString(t *testing.T) {
	program := parse(`"hello ${name} world";`)
	optimized := EliminateDeadCode(program).(*ast.Program)
	expr := getExpr(optimized)

	_, ok := expr.(*ast.InterpolatedString)
	if !ok {
		t.Fatalf("expected InterpolatedString, got %T", expr)
	}
}

// TestDeadCodeEliminationStayExpression tests DCE within stay expressions
func TestDeadCodeEliminationStayExpression(t *testing.T) {
	tok := token.Token{Type: token.IDENT, Literal: "stay"}
	stay := &ast.StayExpression{
		Token: tok,
		StateInit: []ast.StayStatePair{
			{
				Name:  &ast.Identifier{Token: tok, Value: "count"},
				Value: &ast.IntegerLiteral{Token: tok, Value: 0},
			},
		},
		Body: []ast.Statement{
			&ast.ExpressionStatement{Token: tok, Expression: &ast.Identifier{Token: tok, Value: "count"}},
		},
	}
	program := &ast.Program{
		Statements: []ast.Statement{
			&ast.ExpressionStatement{Token: tok, Expression: stay},
		},
	}

	optimized := EliminateDeadCode(program).(*ast.Program)
	expr := getExpr(optimized)

	_, ok := expr.(*ast.StayExpression)
	if !ok {
		t.Fatalf("expected StayExpression, got %T", expr)
	}
}

// TestDeadCodeEliminationNil tests DCE on nil input
func TestDeadCodeEliminationNil(t *testing.T) {
	result := EliminateDeadCode(nil)
	if result != nil {
		t.Error("expected nil result for nil input")
	}
}

// TestDeadCodeEliminationNonShortCircuit tests that non-constant && and || are preserved
func TestDeadCodeEliminationNonShortCircuit(t *testing.T) {
	program := parse("x && y;")
	optimized := EliminateDeadCode(program).(*ast.Program)
	expr := getExpr(optimized)

	_, ok := expr.(*ast.InfixExpression)
	if !ok {
		t.Fatalf("expected InfixExpression, got %T", expr)
	}
}

// TestDeadCodeEliminationNonBooleanInfix tests that non-boolean ops pass through
func TestDeadCodeEliminationNonBooleanInfix(t *testing.T) {
	program := parse("x + y;")
	optimized := EliminateDeadCode(program).(*ast.Program)
	expr := getExpr(optimized)

	_, ok := expr.(*ast.InfixExpression)
	if !ok {
		t.Fatalf("expected InfixExpression, got %T", expr)
	}
}

// === CSE Tests ===

// TestCSEBasic tests basic CSE functionality
func TestCSEBasic(t *testing.T) {
	program := parse("f(x, y);")
	optimized := EliminateCommonSubexpressions(program).(*ast.Program)

	if len(optimized.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(optimized.Statements))
	}
}

// TestCSENewCSEOptimizer tests creating a new CSE optimizer
func TestCSENewCSEOptimizer(t *testing.T) {
	opt := NewCSEOptimizer()
	if opt == nil {
		t.Fatal("expected non-nil CSE optimizer")
	}
	if opt.effectFuncs == nil {
		t.Fatal("expected non-nil effectFuncs map")
	}
	if opt.counter != 0 {
		t.Errorf("expected counter 0, got %d", opt.counter)
	}
}

// TestCSEGenerateVar tests CSE variable generation
func TestCSEGenerateVar(t *testing.T) {
	opt := NewCSEOptimizer()
	v1 := opt.generateCSEVar()
	v2 := opt.generateCSEVar()

	if v1 != "_cse_0" {
		t.Errorf("expected _cse_0, got %s", v1)
	}
	if v2 != "_cse_1" {
		t.Errorf("expected _cse_1, got %s", v2)
	}
}

// TestCSEIsPureExpr tests isPureExpr for various expression types
func TestCSEIsPureExpr(t *testing.T) {
	opt := NewCSEOptimizer()
	tok := token.Token{Type: token.INT, Literal: "1"}

	// Pure expressions
	if !opt.isPureExpr(&ast.IntegerLiteral{Token: tok, Value: 1}) {
		t.Error("IntegerLiteral should be pure")
	}
	if !opt.isPureExpr(&ast.FloatLiteral{Token: tok, Value: 1.0}) {
		t.Error("FloatLiteral should be pure")
	}
	if !opt.isPureExpr(&ast.StringLiteral{Token: tok, Value: "hello"}) {
		t.Error("StringLiteral should be pure")
	}
	if !opt.isPureExpr(&ast.BooleanLiteral{Token: tok, Value: true}) {
		t.Error("BooleanLiteral should be pure")
	}
	if !opt.isPureExpr(&ast.Identifier{Token: tok, Value: "x"}) {
		t.Error("Identifier should be pure")
	}

	// Lambda is pure
	if !opt.isPureExpr(&ast.LambdaExpression{Token: tok, Body: &ast.IntegerLiteral{Token: tok, Value: 1}}) {
		t.Error("LambdaExpression should be pure")
	}

	// Pure array
	if !opt.isPureExpr(&ast.ArrayLiteral{Token: tok, Elements: []ast.Expression{&ast.IntegerLiteral{Token: tok, Value: 1}}}) {
		t.Error("ArrayLiteral with pure elements should be pure")
	}

	// Pure hash
	if !opt.isPureExpr(&ast.HashLiteral{Token: tok, Pairs: []ast.HashPair{{Key: &ast.StringLiteral{Token: tok, Value: "a"}, Value: &ast.IntegerLiteral{Token: tok, Value: 1}}}}) {
		t.Error("HashLiteral with pure pairs should be pure")
	}

	// Pure infix
	if !opt.isPureExpr(&ast.InfixExpression{Token: tok, Left: &ast.IntegerLiteral{Token: tok, Value: 1}, Operator: "+", Right: &ast.IntegerLiteral{Token: tok, Value: 2}}) {
		t.Error("InfixExpression with pure operands should be pure")
	}

	// Pure prefix
	if !opt.isPureExpr(&ast.PrefixExpression{Token: tok, Operator: "-", Right: &ast.IntegerLiteral{Token: tok, Value: 1}}) {
		t.Error("PrefixExpression with pure operand should be pure")
	}

	// Pure index
	if !opt.isPureExpr(&ast.IndexExpression{Token: tok, Left: &ast.Identifier{Token: tok, Value: "arr"}, Index: &ast.IntegerLiteral{Token: tok, Value: 0}}) {
		t.Error("IndexExpression with pure operands should be pure")
	}

	// Pure member
	if !opt.isPureExpr(&ast.MemberExpression{Token: tok, Object: &ast.Identifier{Token: tok, Value: "obj"}, Property: &ast.Identifier{Token: tok, Value: "prop"}}) {
		t.Error("MemberExpression with pure object should be pure")
	}

	// Pure call (non-effect)
	if !opt.isPureExpr(&ast.CallExpression{Token: tok, Function: &ast.Identifier{Token: tok, Value: "f"}, Arguments: []ast.Expression{&ast.IntegerLiteral{Token: tok, Value: 1}}}) {
		t.Error("CallExpression with non-effect function should be pure")
	}

	// Effect function call is not pure
	opt.effectFuncs["sideEffect"] = true
	if opt.isPureExpr(&ast.CallExpression{Token: tok, Function: &ast.Identifier{Token: tok, Value: "sideEffect"}, Arguments: nil}) {
		t.Error("CallExpression with effect function should not be pure")
	}

	// Unknown expression type is not pure
	if opt.isPureExpr(&ast.SuccessExpression{Token: tok, Value: &ast.IntegerLiteral{Token: tok, Value: 1}}) {
		t.Error("SuccessExpression should not be pure (not in isPureExpr switch)")
	}
}

// TestCSEIsPureExprImpureArray tests that array with impure element is not pure
func TestCSEIsPureExprImpureArray(t *testing.T) {
	opt := NewCSEOptimizer()
	opt.effectFuncs["sideEffect"] = true
	tok := token.Token{Type: token.INT, Literal: "1"}

	arr := &ast.ArrayLiteral{
		Token: tok,
		Elements: []ast.Expression{
			&ast.CallExpression{Token: tok, Function: &ast.Identifier{Token: tok, Value: "sideEffect"}, Arguments: nil},
		},
	}
	if opt.isPureExpr(arr) {
		t.Error("Array with impure element should not be pure")
	}
}

// TestCSEIsPureExprImpureHash tests that hash with impure values is not pure
func TestCSEIsPureExprImpureHash(t *testing.T) {
	opt := NewCSEOptimizer()
	opt.effectFuncs["sideEffect"] = true
	tok := token.Token{Type: token.INT, Literal: "1"}

	hash := &ast.HashLiteral{
		Token: tok,
		Pairs: []ast.HashPair{
			{
				Key:   &ast.StringLiteral{Token: tok, Value: "a"},
				Value: &ast.CallExpression{Token: tok, Function: &ast.Identifier{Token: tok, Value: "sideEffect"}, Arguments: nil},
			},
		},
	}
	if opt.isPureExpr(hash) {
		t.Error("Hash with impure value should not be pure")
	}
}

// TestCSEIsPureExprCallWithImpureArg tests call with impure argument
func TestCSEIsPureExprCallWithImpureArg(t *testing.T) {
	opt := NewCSEOptimizer()
	opt.effectFuncs["sideEffect"] = true
	tok := token.Token{Type: token.INT, Literal: "1"}

	call := &ast.CallExpression{
		Token:    tok,
		Function: &ast.Identifier{Token: tok, Value: "f"},
		Arguments: []ast.Expression{
			&ast.CallExpression{Token: tok, Function: &ast.Identifier{Token: tok, Value: "sideEffect"}, Arguments: nil},
		},
	}
	if opt.isPureExpr(call) {
		t.Error("Call with impure argument should not be pure")
	}
}

// TestCSEIsPureExprMemberCall tests member expression call (conservative)
func TestCSEIsPureExprMemberCall(t *testing.T) {
	opt := NewCSEOptimizer()
	tok := token.Token{Type: token.INT, Literal: "1"}

	call := &ast.CallExpression{
		Token: tok,
		Function: &ast.MemberExpression{
			Token:    tok,
			Object:   &ast.Identifier{Token: tok, Value: "io"},
			Property: &ast.Identifier{Token: tok, Value: "println"},
		},
		Arguments: nil,
	}
	// Member expression calls are treated as potentially pure (conservative approach)
	if !opt.isPureExpr(call) {
		t.Error("Member call should be considered pure (conservative)")
	}
}

// TestCSEIsExpensiveExpr tests isExpensiveExpr for various expression types
func TestCSEIsExpensiveExpr(t *testing.T) {
	opt := NewCSEOptimizer()
	tok := token.Token{Type: token.INT, Literal: "1"}

	// Not expensive: literals and identifiers
	if opt.isExpensiveExpr(&ast.IntegerLiteral{Token: tok, Value: 1}) {
		t.Error("IntegerLiteral should not be expensive")
	}
	if opt.isExpensiveExpr(&ast.FloatLiteral{Token: tok, Value: 1.0}) {
		t.Error("FloatLiteral should not be expensive")
	}
	if opt.isExpensiveExpr(&ast.StringLiteral{Token: tok, Value: "hello"}) {
		t.Error("StringLiteral should not be expensive")
	}
	if opt.isExpensiveExpr(&ast.BooleanLiteral{Token: tok, Value: true}) {
		t.Error("BooleanLiteral should not be expensive")
	}
	if opt.isExpensiveExpr(&ast.Identifier{Token: tok, Value: "x"}) {
		t.Error("Identifier should not be expensive")
	}

	// Expensive: function calls
	if !opt.isExpensiveExpr(&ast.CallExpression{Token: tok, Function: &ast.Identifier{Token: tok, Value: "f"}, Arguments: nil}) {
		t.Error("CallExpression should be expensive")
	}

	// Expensive: infix with expensive operand
	if !opt.isExpensiveExpr(&ast.InfixExpression{
		Token:    tok,
		Left:     &ast.CallExpression{Token: tok, Function: &ast.Identifier{Token: tok, Value: "f"}, Arguments: nil},
		Operator: "+",
		Right:    &ast.IntegerLiteral{Token: tok, Value: 1},
	}) {
		t.Error("InfixExpression with expensive left should be expensive")
	}
	if !opt.isExpensiveExpr(&ast.InfixExpression{
		Token:    tok,
		Left:     &ast.IntegerLiteral{Token: tok, Value: 1},
		Operator: "+",
		Right:    &ast.CallExpression{Token: tok, Function: &ast.Identifier{Token: tok, Value: "f"}, Arguments: nil},
	}) {
		t.Error("InfixExpression with expensive right should be expensive")
	}

	// Expensive: index with expensive left
	if !opt.isExpensiveExpr(&ast.IndexExpression{
		Token: tok,
		Left:  &ast.CallExpression{Token: tok, Function: &ast.Identifier{Token: tok, Value: "f"}, Arguments: nil},
		Index: &ast.IntegerLiteral{Token: tok, Value: 0},
	}) {
		t.Error("IndexExpression with expensive left should be expensive")
	}

	// Not expensive: unknown types
	if opt.isExpensiveExpr(&ast.ArrayLiteral{Token: tok, Elements: nil}) {
		t.Error("ArrayLiteral should not be expensive by default")
	}
}

// TestCSEExprEqual tests expression equality checking
func TestCSEExprEqual(t *testing.T) {
	opt := NewCSEOptimizer()
	tok := token.Token{Type: token.INT, Literal: "1"}

	// Nil checks
	if !opt.exprEqual(nil, nil) {
		t.Error("nil == nil should be true")
	}
	if opt.exprEqual(nil, &ast.IntegerLiteral{Token: tok, Value: 1}) {
		t.Error("nil != non-nil should be false")
	}
	if opt.exprEqual(&ast.IntegerLiteral{Token: tok, Value: 1}, nil) {
		t.Error("non-nil != nil should be false")
	}

	// Integer equality
	if !opt.exprEqual(&ast.IntegerLiteral{Token: tok, Value: 5}, &ast.IntegerLiteral{Token: tok, Value: 5}) {
		t.Error("same integers should be equal")
	}
	if opt.exprEqual(&ast.IntegerLiteral{Token: tok, Value: 5}, &ast.IntegerLiteral{Token: tok, Value: 6}) {
		t.Error("different integers should not be equal")
	}

	// Float equality
	if !opt.exprEqual(&ast.FloatLiteral{Token: tok, Value: 1.5}, &ast.FloatLiteral{Token: tok, Value: 1.5}) {
		t.Error("same floats should be equal")
	}
	if opt.exprEqual(&ast.FloatLiteral{Token: tok, Value: 1.5}, &ast.FloatLiteral{Token: tok, Value: 2.5}) {
		t.Error("different floats should not be equal")
	}

	// String equality
	if !opt.exprEqual(&ast.StringLiteral{Token: tok, Value: "hello"}, &ast.StringLiteral{Token: tok, Value: "hello"}) {
		t.Error("same strings should be equal")
	}
	if opt.exprEqual(&ast.StringLiteral{Token: tok, Value: "hello"}, &ast.StringLiteral{Token: tok, Value: "world"}) {
		t.Error("different strings should not be equal")
	}

	// Boolean equality
	if !opt.exprEqual(&ast.BooleanLiteral{Token: tok, Value: true}, &ast.BooleanLiteral{Token: tok, Value: true}) {
		t.Error("same booleans should be equal")
	}
	if opt.exprEqual(&ast.BooleanLiteral{Token: tok, Value: true}, &ast.BooleanLiteral{Token: tok, Value: false}) {
		t.Error("different booleans should not be equal")
	}

	// Identifier equality
	if !opt.exprEqual(&ast.Identifier{Token: tok, Value: "x"}, &ast.Identifier{Token: tok, Value: "x"}) {
		t.Error("same identifiers should be equal")
	}
	if opt.exprEqual(&ast.Identifier{Token: tok, Value: "x"}, &ast.Identifier{Token: tok, Value: "y"}) {
		t.Error("different identifiers should not be equal")
	}

	// Infix equality
	infix1 := &ast.InfixExpression{Token: tok, Left: &ast.IntegerLiteral{Token: tok, Value: 1}, Operator: "+", Right: &ast.IntegerLiteral{Token: tok, Value: 2}}
	infix2 := &ast.InfixExpression{Token: tok, Left: &ast.IntegerLiteral{Token: tok, Value: 1}, Operator: "+", Right: &ast.IntegerLiteral{Token: tok, Value: 2}}
	if !opt.exprEqual(infix1, infix2) {
		t.Error("same infix expressions should be equal")
	}
	infix3 := &ast.InfixExpression{Token: tok, Left: &ast.IntegerLiteral{Token: tok, Value: 1}, Operator: "-", Right: &ast.IntegerLiteral{Token: tok, Value: 2}}
	if opt.exprEqual(infix1, infix3) {
		t.Error("different infix operators should not be equal")
	}

	// Prefix equality
	prefix1 := &ast.PrefixExpression{Token: tok, Operator: "-", Right: &ast.IntegerLiteral{Token: tok, Value: 1}}
	prefix2 := &ast.PrefixExpression{Token: tok, Operator: "-", Right: &ast.IntegerLiteral{Token: tok, Value: 1}}
	if !opt.exprEqual(prefix1, prefix2) {
		t.Error("same prefix expressions should be equal")
	}
	prefix3 := &ast.PrefixExpression{Token: tok, Operator: "!", Right: &ast.IntegerLiteral{Token: tok, Value: 1}}
	if opt.exprEqual(prefix1, prefix3) {
		t.Error("different prefix operators should not be equal")
	}

	// Call equality
	call1 := &ast.CallExpression{Token: tok, Function: &ast.Identifier{Token: tok, Value: "f"}, Arguments: []ast.Expression{&ast.IntegerLiteral{Token: tok, Value: 1}}}
	call2 := &ast.CallExpression{Token: tok, Function: &ast.Identifier{Token: tok, Value: "f"}, Arguments: []ast.Expression{&ast.IntegerLiteral{Token: tok, Value: 1}}}
	if !opt.exprEqual(call1, call2) {
		t.Error("same call expressions should be equal")
	}
	call3 := &ast.CallExpression{Token: tok, Function: &ast.Identifier{Token: tok, Value: "g"}, Arguments: []ast.Expression{&ast.IntegerLiteral{Token: tok, Value: 1}}}
	if opt.exprEqual(call1, call3) {
		t.Error("different call functions should not be equal")
	}
	call4 := &ast.CallExpression{Token: tok, Function: &ast.Identifier{Token: tok, Value: "f"}, Arguments: []ast.Expression{&ast.IntegerLiteral{Token: tok, Value: 1}, &ast.IntegerLiteral{Token: tok, Value: 2}}}
	if opt.exprEqual(call1, call4) {
		t.Error("different arg counts should not be equal")
	}
	call5 := &ast.CallExpression{Token: tok, Function: &ast.Identifier{Token: tok, Value: "f"}, Arguments: []ast.Expression{&ast.IntegerLiteral{Token: tok, Value: 99}}}
	if opt.exprEqual(call1, call5) {
		t.Error("different args should not be equal")
	}

	// Index equality
	idx1 := &ast.IndexExpression{Token: tok, Left: &ast.Identifier{Token: tok, Value: "arr"}, Index: &ast.IntegerLiteral{Token: tok, Value: 0}}
	idx2 := &ast.IndexExpression{Token: tok, Left: &ast.Identifier{Token: tok, Value: "arr"}, Index: &ast.IntegerLiteral{Token: tok, Value: 0}}
	if !opt.exprEqual(idx1, idx2) {
		t.Error("same index expressions should be equal")
	}

	// Member equality
	mem1 := &ast.MemberExpression{Token: tok, Object: &ast.Identifier{Token: tok, Value: "obj"}, Property: &ast.Identifier{Token: tok, Value: "prop"}}
	mem2 := &ast.MemberExpression{Token: tok, Object: &ast.Identifier{Token: tok, Value: "obj"}, Property: &ast.Identifier{Token: tok, Value: "prop"}}
	if !opt.exprEqual(mem1, mem2) {
		t.Error("same member expressions should be equal")
	}
	mem3 := &ast.MemberExpression{Token: tok, Object: &ast.Identifier{Token: tok, Value: "obj"}, Property: &ast.Identifier{Token: tok, Value: "other"}}
	if opt.exprEqual(mem1, mem3) {
		t.Error("different member properties should not be equal")
	}

	// Array equality
	arr1 := &ast.ArrayLiteral{Token: tok, Elements: []ast.Expression{&ast.IntegerLiteral{Token: tok, Value: 1}, &ast.IntegerLiteral{Token: tok, Value: 2}}}
	arr2 := &ast.ArrayLiteral{Token: tok, Elements: []ast.Expression{&ast.IntegerLiteral{Token: tok, Value: 1}, &ast.IntegerLiteral{Token: tok, Value: 2}}}
	if !opt.exprEqual(arr1, arr2) {
		t.Error("same arrays should be equal")
	}
	arr3 := &ast.ArrayLiteral{Token: tok, Elements: []ast.Expression{&ast.IntegerLiteral{Token: tok, Value: 1}}}
	if opt.exprEqual(arr1, arr3) {
		t.Error("different array lengths should not be equal")
	}
	arr4 := &ast.ArrayLiteral{Token: tok, Elements: []ast.Expression{&ast.IntegerLiteral{Token: tok, Value: 1}, &ast.IntegerLiteral{Token: tok, Value: 3}}}
	if opt.exprEqual(arr1, arr4) {
		t.Error("different array elements should not be equal")
	}

	// Type mismatch
	if opt.exprEqual(&ast.IntegerLiteral{Token: tok, Value: 1}, &ast.StringLiteral{Token: tok, Value: "1"}) {
		t.Error("different types should not be equal")
	}

	// Unsupported types
	if opt.exprEqual(&ast.SuccessExpression{Token: tok, Value: nil}, &ast.SuccessExpression{Token: tok, Value: nil}) {
		t.Error("unsupported types should not be equal")
	}
}

// TestCSECollectEffectFuncs tests collection of effect function declarations
func TestCSECollectEffectFuncs(t *testing.T) {
	// Parse a program with effect function
	tok := token.Token{Type: token.FUNC, Literal: "func"}
	program := &ast.Program{
		Statements: []ast.Statement{
			&ast.FunctionDeclaration{
				Token:    tok,
				Name:     &ast.Identifier{Token: tok, Value: "normalFunc"},
				Body:     &ast.IntegerLiteral{Token: tok, Value: 1},
				IsEffect: false,
			},
			&ast.FunctionDeclaration{
				Token:    tok,
				Name:     &ast.Identifier{Token: tok, Value: "effectFunc"},
				Body:     &ast.IntegerLiteral{Token: tok, Value: 1},
				IsEffect: true,
			},
		},
	}

	opt := NewCSEOptimizer()
	opt.collectEffectFuncs(program)

	if opt.effectFuncs["normalFunc"] {
		t.Error("normalFunc should not be marked as effect")
	}
	if !opt.effectFuncs["effectFunc"] {
		t.Error("effectFunc should be marked as effect")
	}
}

// TestCSEOptimizeAssignment tests CSE on assignment statements
func TestCSEOptimizeAssignment(t *testing.T) {
	program := parse("x = 1 + 2;")
	optimized := EliminateCommonSubexpressions(program).(*ast.Program)

	if len(optimized.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(optimized.Statements))
	}
	_, ok := optimized.Statements[0].(*ast.AssignmentStatement)
	if !ok {
		t.Fatalf("expected AssignmentStatement, got %T", optimized.Statements[0])
	}
}

// TestCSEOptimizeFunctionDeclaration tests CSE on function declaration
func TestCSEOptimizeFunctionDeclaration(t *testing.T) {
	program := parse("func f(x) = x + x;")
	optimized := EliminateCommonSubexpressions(program).(*ast.Program)

	if len(optimized.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(optimized.Statements))
	}
	_, ok := optimized.Statements[0].(*ast.FunctionDeclaration)
	if !ok {
		t.Fatalf("expected FunctionDeclaration, got %T", optimized.Statements[0])
	}
}

// TestCSEOptimizeConstraintDeclaration tests CSE on constraint declaration
func TestCSEOptimizeConstraintDeclaration(t *testing.T) {
	program := parse("constraint C(n) = n > 0;")
	optimized := EliminateCommonSubexpressions(program).(*ast.Program)

	if len(optimized.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(optimized.Statements))
	}
	_, ok := optimized.Statements[0].(*ast.ConstraintDeclaration)
	if !ok {
		t.Fatalf("expected ConstraintDeclaration, got %T", optimized.Statements[0])
	}
}

// TestCSEOptimizeNilNode tests CSE on nil input
func TestCSEOptimizeNilNode(t *testing.T) {
	opt := NewCSEOptimizer()
	result := opt.optimize(nil)
	if result != nil {
		t.Error("expected nil result for nil input")
	}
}

// TestCSEOptimizeDefaultNode tests CSE on unhandled node types
func TestCSEOptimizeDefaultNode(t *testing.T) {
	opt := NewCSEOptimizer()
	tok := token.Token{Type: token.IDENT, Literal: "x"}
	node := &ast.Identifier{Token: tok, Value: "x"}

	result := opt.optimize(node)
	if result != node {
		t.Error("unhandled node should be returned as-is")
	}
}

// TestCSEOptimizeNilExpr tests CSE on nil expression
func TestCSEOptimizeNilExpr(t *testing.T) {
	opt := NewCSEOptimizer()
	result := opt.optimizeExpr(nil)
	if result != nil {
		t.Error("expected nil result for nil expression")
	}
}

// TestCSEOptimizeSubExprs tests CSE sub-expression optimization
func TestCSEOptimizeSubExprs(t *testing.T) {
	tok := token.Token{Type: token.INT, Literal: "1"}
	opt := NewCSEOptimizer()

	// Test prefix expression
	prefix := &ast.PrefixExpression{
		Token:    tok,
		Operator: "-",
		Right:    &ast.IntegerLiteral{Token: tok, Value: 5},
	}
	result := opt.optimizeSubExprs(prefix)
	_, ok := result.(*ast.PrefixExpression)
	if !ok {
		t.Fatalf("expected PrefixExpression, got %T", result)
	}

	// Test hash literal
	hash := &ast.HashLiteral{
		Token: tok,
		Pairs: []ast.HashPair{
			{Key: &ast.StringLiteral{Token: tok, Value: "a"}, Value: &ast.IntegerLiteral{Token: tok, Value: 1}},
		},
	}
	result = opt.optimizeSubExprs(hash)
	_, ok = result.(*ast.HashLiteral)
	if !ok {
		t.Fatalf("expected HashLiteral, got %T", result)
	}

	// Test index expression
	idx := &ast.IndexExpression{
		Token: tok,
		Left:  &ast.Identifier{Token: tok, Value: "arr"},
		Index: &ast.IntegerLiteral{Token: tok, Value: 0},
	}
	result = opt.optimizeSubExprs(idx)
	_, ok = result.(*ast.IndexExpression)
	if !ok {
		t.Fatalf("expected IndexExpression, got %T", result)
	}

	// Test lambda expression
	lambda := &ast.LambdaExpression{
		Token:      tok,
		Parameters: []*ast.Identifier{{Token: tok, Value: "x"}},
		Body:       &ast.IntegerLiteral{Token: tok, Value: 1},
	}
	result = opt.optimizeSubExprs(lambda)
	_, ok = result.(*ast.LambdaExpression)
	if !ok {
		t.Fatalf("expected LambdaExpression, got %T", result)
	}

	// Test pipe expression
	pipe := &ast.PipeExpression{
		Token: tok,
		Left:  &ast.Identifier{Token: tok, Value: "x"},
		Right: &ast.Identifier{Token: tok, Value: "f"},
	}
	result = opt.optimizeSubExprs(pipe)
	_, ok = result.(*ast.PipeExpression)
	if !ok {
		t.Fatalf("expected PipeExpression, got %T", result)
	}

	// Test effect pipe expression
	effPipe := &ast.EffectPipeExpression{
		Token: tok,
		Left:  &ast.Identifier{Token: tok, Value: "x"},
		Right: &ast.Identifier{Token: tok, Value: "f"},
	}
	result = opt.optimizeSubExprs(effPipe)
	_, ok = result.(*ast.EffectPipeExpression)
	if !ok {
		t.Fatalf("expected EffectPipeExpression, got %T", result)
	}

	// Test match expression
	matchExpr := &ast.MatchExpression{
		Token:   tok,
		Subject: &ast.Identifier{Token: tok, Value: "x"},
		Cases: []*ast.MatchCase{
			{Token: tok, Pattern: &ast.IntegerLiteral{Token: tok, Value: 1}, Body: &ast.IntegerLiteral{Token: tok, Value: 10}},
		},
	}
	result = opt.optimizeSubExprs(matchExpr)
	_, ok = result.(*ast.MatchExpression)
	if !ok {
		t.Fatalf("expected MatchExpression, got %T", result)
	}

	// Test effect handle expression
	handle := &ast.EffectHandleExpression{
		Token:       tok,
		Subject:     &ast.Identifier{Token: tok, Value: "x"},
		SuccessVar:  &ast.Identifier{Token: tok, Value: "v"},
		SuccessBody: &ast.Identifier{Token: tok, Value: "v"},
		FailureVar:  &ast.Identifier{Token: tok, Value: "e"},
		FailureBody: &ast.IntegerLiteral{Token: tok, Value: 0},
	}
	result = opt.optimizeSubExprs(handle)
	_, ok = result.(*ast.EffectHandleExpression)
	if !ok {
		t.Fatalf("expected EffectHandleExpression, got %T", result)
	}

	// Test success expression
	succ := &ast.SuccessExpression{Token: tok, Value: &ast.IntegerLiteral{Token: tok, Value: 1}}
	result = opt.optimizeSubExprs(succ)
	_, ok = result.(*ast.SuccessExpression)
	if !ok {
		t.Fatalf("expected SuccessExpression, got %T", result)
	}

	// Test failure expression
	fail := &ast.FailureExpression{Token: tok, Value: &ast.IntegerLiteral{Token: tok, Value: 1}}
	result = opt.optimizeSubExprs(fail)
	_, ok = result.(*ast.FailureExpression)
	if !ok {
		t.Fatalf("expected FailureExpression, got %T", result)
	}

	// Test member expression
	member := &ast.MemberExpression{
		Token:    tok,
		Object:   &ast.Identifier{Token: tok, Value: "obj"},
		Property: &ast.Identifier{Token: tok, Value: "prop"},
	}
	result = opt.optimizeSubExprs(member)
	_, ok = result.(*ast.MemberExpression)
	if !ok {
		t.Fatalf("expected MemberExpression, got %T", result)
	}

	// Test spread expression
	spread := &ast.SpreadExpression{Token: tok, Value: &ast.Identifier{Token: tok, Value: "arr"}}
	result = opt.optimizeSubExprs(spread)
	_, ok = result.(*ast.SpreadExpression)
	if !ok {
		t.Fatalf("expected SpreadExpression, got %T", result)
	}

	// Test interpolated string
	interp := &ast.InterpolatedString{
		Token: tok,
		Parts: []ast.Expression{
			&ast.StringLiteral{Token: tok, Value: "hello "},
			&ast.Identifier{Token: tok, Value: "name"},
		},
	}
	result = opt.optimizeSubExprs(interp)
	_, ok = result.(*ast.InterpolatedString)
	if !ok {
		t.Fatalf("expected InterpolatedString, got %T", result)
	}

	// Test default (identifier passthrough)
	ident := &ast.Identifier{Token: tok, Value: "x"}
	result = opt.optimizeSubExprs(ident)
	if result != ident {
		t.Error("identifier should pass through unchanged")
	}
}

// TestCSEDeduplicateArgs tests argument deduplication
func TestCSEDeduplicateArgs(t *testing.T) {
	opt := NewCSEOptimizer()
	tok := token.Token{Type: token.INT, Literal: "1"}

	// Single arg - no dedup needed
	args1 := []ast.Expression{&ast.IntegerLiteral{Token: tok, Value: 1}}
	result := opt.deduplicateArgs(args1)
	if len(result) != 1 {
		t.Errorf("expected 1 arg, got %d", len(result))
	}

	// Two different non-expensive args - no dedup
	args2 := []ast.Expression{
		&ast.IntegerLiteral{Token: tok, Value: 1},
		&ast.IntegerLiteral{Token: tok, Value: 2},
	}
	result = opt.deduplicateArgs(args2)
	if len(result) != 2 {
		t.Errorf("expected 2 args, got %d", len(result))
	}

	// Two identical expensive args - should trigger dedup path
	call := &ast.CallExpression{
		Token:     tok,
		Function:  &ast.Identifier{Token: tok, Value: "expensive"},
		Arguments: []ast.Expression{&ast.Identifier{Token: tok, Value: "x"}},
	}
	call2 := &ast.CallExpression{
		Token:     tok,
		Function:  &ast.Identifier{Token: tok, Value: "expensive"},
		Arguments: []ast.Expression{&ast.Identifier{Token: tok, Value: "x"}},
	}
	args3 := []ast.Expression{call, call2}
	result = opt.deduplicateArgs(args3)
	// The CSE transform is a placeholder that returns a copy
	if len(result) != 2 {
		t.Errorf("expected 2 args after dedup, got %d", len(result))
	}
}

// TestCSEInfixWithIdenticalPureSubexprs tests CSE on infix with equal pure subexprs
func TestCSEInfixWithIdenticalPureSubexprs(t *testing.T) {
	tok := token.Token{Type: token.PLUS, Literal: "+"}
	opt := NewCSEOptimizer()

	infix := &ast.InfixExpression{
		Token:    tok,
		Left:     &ast.Identifier{Token: tok, Value: "x"},
		Operator: "+",
		Right:    &ast.Identifier{Token: tok, Value: "x"},
	}

	result := opt.optimizeSubExprs(infix)
	_, ok := result.(*ast.InfixExpression)
	if !ok {
		t.Fatalf("expected InfixExpression, got %T", result)
	}
}

// TestCSEArrayDedup tests CSE dedup within array literals
func TestCSEArrayDedup(t *testing.T) {
	tok := token.Token{Type: token.LBRACKET, Literal: "["}
	opt := NewCSEOptimizer()

	arr := &ast.ArrayLiteral{
		Token: tok,
		Elements: []ast.Expression{
			&ast.IntegerLiteral{Token: tok, Value: 1},
			&ast.IntegerLiteral{Token: tok, Value: 2},
		},
	}

	result := opt.optimizeSubExprs(arr)
	arrResult, ok := result.(*ast.ArrayLiteral)
	if !ok {
		t.Fatalf("expected ArrayLiteral, got %T", result)
	}
	if len(arrResult.Elements) != 2 {
		t.Errorf("expected 2 elements, got %d", len(arrResult.Elements))
	}
}

// === Peephole Optimization Tests ===

// TestPeepholeOptimization tests bytecode-level optimizations
func TestPeepholeOptimization(t *testing.T) {
	// Test OpTrue + OpNot -> OpFalse
	instructions := bytecode.Instructions{}
	instructions = append(instructions, bytecode.Make(bytecode.OpTrue)...)
	instructions = append(instructions, bytecode.Make(bytecode.OpNot)...)

	optimized := PeepholeOptimize(instructions)

	if len(optimized) != 1 {
		t.Errorf("expected 1 instruction, got %d", len(optimized))
	}
	if bytecode.OpCode(optimized[0]) != bytecode.OpFalse {
		t.Errorf("expected OpFalse, got %v", bytecode.OpCode(optimized[0]))
	}
}

// TestPeepholeFalseNot tests OpFalse + OpNot -> OpTrue
func TestPeepholeFalseNot(t *testing.T) {
	instructions := bytecode.Instructions{}
	instructions = append(instructions, bytecode.Make(bytecode.OpFalse)...)
	instructions = append(instructions, bytecode.Make(bytecode.OpNot)...)

	optimized := PeepholeOptimize(instructions)

	if len(optimized) != 1 {
		t.Errorf("expected 1 instruction, got %d", len(optimized))
	}
	if bytecode.OpCode(optimized[0]) != bytecode.OpTrue {
		t.Errorf("expected OpTrue, got %v", bytecode.OpCode(optimized[0]))
	}
}

// TestPeepholeDoubleNegation tests removal of double negation
func TestPeepholeDoubleNegation(t *testing.T) {
	// Test OpNot + OpNot -> (removed)
	instructions := bytecode.Instructions{}
	instructions = append(instructions, bytecode.Make(bytecode.OpConstant, 0)...) // Push something first
	instructions = append(instructions, bytecode.Make(bytecode.OpNot)...)
	instructions = append(instructions, bytecode.Make(bytecode.OpNot)...)

	optimized := PeepholeOptimize(instructions)

	// Should have just the OpConstant
	if len(optimized) != 3 { // OpConstant takes 3 bytes
		t.Errorf("expected 3 bytes (OpConstant), got %d", len(optimized))
	}
}

// TestPeepholeDoubleNeg tests removal of double numeric negation
func TestPeepholeDoubleNeg(t *testing.T) {
	instructions := bytecode.Instructions{}
	instructions = append(instructions, bytecode.Make(bytecode.OpConstant, 0)...)
	instructions = append(instructions, bytecode.Make(bytecode.OpNeg)...)
	instructions = append(instructions, bytecode.Make(bytecode.OpNeg)...)

	optimized := PeepholeOptimize(instructions)

	if len(optimized) != 3 {
		t.Errorf("expected 3 bytes (OpConstant), got %d", len(optimized))
	}
}

// TestPeepholeDupPop tests removal of Dup + Pop
func TestPeepholeDupPop(t *testing.T) {
	instructions := bytecode.Instructions{}
	instructions = append(instructions, bytecode.Make(bytecode.OpConstant, 0)...)
	instructions = append(instructions, bytecode.Make(bytecode.OpDup)...)
	instructions = append(instructions, bytecode.Make(bytecode.OpPop)...)

	optimized := PeepholeOptimize(instructions)

	// Should have just the OpConstant
	if len(optimized) != 3 { // OpConstant takes 3 bytes
		t.Errorf("expected 3 bytes (OpConstant), got %d", len(optimized))
	}
}

// TestPeepholeConstantPop tests removal of OpConstant + OpPop
func TestPeepholeConstantPop(t *testing.T) {
	instructions := bytecode.Instructions{}
	instructions = append(instructions, bytecode.Make(bytecode.OpConstant, 0)...)
	instructions = append(instructions, bytecode.Make(bytecode.OpPop)...)

	optimized := PeepholeOptimize(instructions)

	if len(optimized) != 0 {
		t.Errorf("expected 0 bytes (dead constant removed), got %d", len(optimized))
	}
}

// TestPeepholeSetGetLocal tests OpSetLocal + OpGetLocal optimization
func TestPeepholeSetGetLocal(t *testing.T) {
	instructions := bytecode.Instructions{}
	instructions = append(instructions, bytecode.Make(bytecode.OpTrue)...)
	instructions = append(instructions, bytecode.Make(bytecode.OpSetLocal, 5)...)
	instructions = append(instructions, bytecode.Make(bytecode.OpGetLocal, 5)...)

	optimized := PeepholeOptimize(instructions)

	parsed := parseInstructions(optimized)
	found := false
	for _, inst := range parsed {
		if inst.opcode == bytecode.OpDup {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected OpDup in optimized output")
	}
}

// TestPeepholeSetGetGlobal tests OpSetGlobal + OpGetGlobal optimization
func TestPeepholeSetGetGlobal(t *testing.T) {
	instructions := bytecode.Instructions{}
	instructions = append(instructions, bytecode.Make(bytecode.OpTrue)...)
	instructions = append(instructions, bytecode.Make(bytecode.OpSetGlobal, 5)...)
	instructions = append(instructions, bytecode.Make(bytecode.OpGetGlobal, 5)...)

	optimized := PeepholeOptimize(instructions)

	parsed := parseInstructions(optimized)
	found := false
	for _, inst := range parsed {
		if inst.opcode == bytecode.OpDup {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected OpDup in optimized output")
	}
}

// TestPeepholeEmptyInstructions tests optimization of empty instructions
func TestPeepholeEmptyInstructions(t *testing.T) {
	optimized := PeepholeOptimize(bytecode.Instructions{})
	if len(optimized) != 0 {
		t.Errorf("expected 0 bytes, got %d", len(optimized))
	}
}

// TestPeepholeNoOptimization tests that non-matching patterns are preserved
func TestPeepholeNoOptimization(t *testing.T) {
	instructions := bytecode.Instructions{}
	instructions = append(instructions, bytecode.Make(bytecode.OpTrue)...)
	instructions = append(instructions, bytecode.Make(bytecode.OpPop)...)

	optimized := PeepholeOptimize(instructions)

	if len(optimized) != 2 {
		t.Errorf("expected 2 bytes, got %d", len(optimized))
	}
}

// TestPeepholeMultiplePasses tests that optimization applies multiple passes
func TestPeepholeMultiplePasses(t *testing.T) {
	// Create a sequence that requires multiple passes
	instructions := bytecode.Instructions{}
	instructions = append(instructions, bytecode.Make(bytecode.OpTrue)...)
	instructions = append(instructions, bytecode.Make(bytecode.OpNot)...) // -> OpFalse
	instructions = append(instructions, bytecode.Make(bytecode.OpNot)...) // -> OpTrue

	optimized := PeepholeOptimize(instructions)

	if len(optimized) != 1 {
		t.Errorf("expected 1 byte after multiple passes, got %d", len(optimized))
	}
	if bytecode.OpCode(optimized[0]) != bytecode.OpTrue {
		t.Errorf("expected OpTrue, got %v", bytecode.OpCode(optimized[0]))
	}
}

// === Optimizer Level Tests ===

// TestOptimizerLevels tests different optimization levels
func TestOptimizerLevels(t *testing.T) {
	input := "2 + 3;"
	program := parse(input)

	// O0 should not optimize
	opt := New(O0)
	result := opt.OptimizeAST(program)
	expr := getExpr(result)
	if _, ok := expr.(*ast.IntegerLiteral); ok {
		t.Error("O0 should not fold constants")
	}

	// O1 should fold constants
	program = parse(input) // Re-parse since optimization may modify AST
	opt = New(O1)
	result = opt.OptimizeAST(program)
	expr = getExpr(result)
	intLit, ok := expr.(*ast.IntegerLiteral)
	if !ok {
		t.Error("O1 should fold constants")
	} else if intLit.Value != 5 {
		t.Errorf("expected 5, got %d", intLit.Value)
	}
}

// TestOptimizerO2Level tests O2 optimization level
func TestOptimizerO2Level(t *testing.T) {
	input := "2 + 3;"
	program := parse(input)

	opt := New(O2)
	result := opt.OptimizeAST(program)
	expr := getExpr(result)
	intLit, ok := expr.(*ast.IntegerLiteral)
	if !ok {
		t.Fatal("O2 should fold constants")
	}
	if intLit.Value != 5 {
		t.Errorf("expected 5, got %d", intLit.Value)
	}
}

// TestOptimizeBytecodeO0 tests that O0 doesn't optimize bytecode
func TestOptimizeBytecodeO0(t *testing.T) {
	instructions := bytecode.Instructions{}
	instructions = append(instructions, bytecode.Make(bytecode.OpTrue)...)
	instructions = append(instructions, bytecode.Make(bytecode.OpNot)...)

	opt := New(O0)
	result := opt.OptimizeBytecode(instructions)

	if len(result) != len(instructions) {
		t.Errorf("O0 should not optimize bytecode, expected %d bytes, got %d", len(instructions), len(result))
	}
}

// TestOptimizeBytecodeO1 tests that O1 doesn't optimize bytecode
func TestOptimizeBytecodeO1(t *testing.T) {
	instructions := bytecode.Instructions{}
	instructions = append(instructions, bytecode.Make(bytecode.OpTrue)...)
	instructions = append(instructions, bytecode.Make(bytecode.OpNot)...)

	opt := New(O1)
	result := opt.OptimizeBytecode(instructions)

	if len(result) != len(instructions) {
		t.Errorf("O1 should not optimize bytecode, expected %d bytes, got %d", len(instructions), len(result))
	}
}

// TestOptimizeBytecodeO2 tests that O2 optimizes bytecode
func TestOptimizeBytecodeO2(t *testing.T) {
	instructions := bytecode.Instructions{}
	instructions = append(instructions, bytecode.Make(bytecode.OpTrue)...)
	instructions = append(instructions, bytecode.Make(bytecode.OpNot)...)

	opt := New(O2)
	result := opt.OptimizeBytecode(instructions)

	if len(result) != 1 {
		t.Errorf("O2 should optimize bytecode, expected 1 byte, got %d", len(result))
	}
}

// TestOptimizeProgram tests the convenience function
func TestOptimizeProgram(t *testing.T) {
	program := parse("2 + 3;")
	result := OptimizeProgram(program)
	expr := getExpr(result)

	intLit, ok := expr.(*ast.IntegerLiteral)
	if !ok {
		t.Fatal("OptimizeProgram should fold constants")
	}
	if intLit.Value != 5 {
		t.Errorf("expected 5, got %d", intLit.Value)
	}
}

// TestOptimizeAll tests the convenience function for full optimization
func TestOptimizeAll(t *testing.T) {
	program := parse("2 + 3;")
	instructions := bytecode.Instructions{}
	instructions = append(instructions, bytecode.Make(bytecode.OpTrue)...)
	instructions = append(instructions, bytecode.Make(bytecode.OpNot)...)

	resultAST, resultBytecode := OptimizeAll(program, instructions)

	// AST should be optimized
	expr := getExpr(resultAST)
	intLit, ok := expr.(*ast.IntegerLiteral)
	if !ok {
		t.Fatal("OptimizeAll should fold constants")
	}
	if intLit.Value != 5 {
		t.Errorf("expected 5, got %d", intLit.Value)
	}

	// Bytecode should be optimized
	if len(resultBytecode) != 1 {
		t.Errorf("expected 1 byte for optimized bytecode, got %d", len(resultBytecode))
	}
}

// TestFullOptimizationPipeline tests the complete optimization pipeline
func TestFullOptimizationPipeline(t *testing.T) {
	input := `
		x = 2 + 3;
		y = match true
			true => 10
			false => 20;
	`

	program := parse(input)
	opt := New(O1)
	optimized := opt.OptimizeAST(program)

	// Check that 2 + 3 was folded to 5
	if len(optimized.Statements) < 1 {
		t.Fatal("expected at least 1 statement")
	}
	assign, ok := optimized.Statements[0].(*ast.AssignmentStatement)
	if !ok {
		t.Fatal("expected AssignmentStatement")
	}
	intLit, ok := assign.Value.(*ast.IntegerLiteral)
	if !ok {
		t.Fatalf("expected IntegerLiteral, got %T", assign.Value)
	}
	if intLit.Value != 5 {
		t.Errorf("expected 5, got %d", intLit.Value)
	}

	// Check that match was simplified to 10
	if len(optimized.Statements) < 2 {
		t.Fatal("expected at least 2 statements")
	}
	assign2, ok := optimized.Statements[1].(*ast.AssignmentStatement)
	if !ok {
		t.Fatal("expected AssignmentStatement")
	}
	intLit2, ok := assign2.Value.(*ast.IntegerLiteral)
	if !ok {
		t.Fatalf("expected IntegerLiteral, got %T", assign2.Value)
	}
	if intLit2.Value != 10 {
		t.Errorf("expected 10, got %d", intLit2.Value)
	}
}

// TestPeepholeJumpToNextInstruction tests removal of jump to next instruction
func TestPeepholeJumpToNextInstruction(t *testing.T) {
	// Create a jump that points to the very next instruction
	jumpBytes := bytecode.Make(bytecode.OpJump, 3) // Offset 3 = right after OpJump (1 + 2 bytes operand)
	nextBytes := bytecode.Make(bytecode.OpTrue)

	instructions := bytecode.Instructions{}
	instructions = append(instructions, jumpBytes...)
	instructions = append(instructions, nextBytes...)

	optimized := PeepholeOptimize(instructions)

	// The jump should be removed, leaving just OpTrue
	if len(optimized) != 1 {
		t.Errorf("expected 1 byte (OpTrue only), got %d", len(optimized))
	}
}

// TestParseInstructionsAndSerialize tests round-trip parsing and serialization
func TestParseInstructionsAndSerialize(t *testing.T) {
	original := bytecode.Instructions{}
	original = append(original, bytecode.Make(bytecode.OpConstant, 42)...)
	original = append(original, bytecode.Make(bytecode.OpTrue)...)
	original = append(original, bytecode.Make(bytecode.OpPop)...)

	parsed := parseInstructions(original)
	if len(parsed) != 3 {
		t.Fatalf("expected 3 instructions, got %d", len(parsed))
	}

	serialized := serializeInstructions(parsed)
	if len(serialized) != len(original) {
		t.Errorf("expected %d bytes, got %d", len(original), len(serialized))
	}
}

// TestFixJumpOffsets tests jump offset recalculation
func TestFixJumpOffsets(t *testing.T) {
	// Empty input
	result := fixJumpOffsets(nil)
	if result != nil {
		t.Error("expected nil for nil input")
	}

	result = fixJumpOffsets([]instruction{})
	if len(result) != 0 {
		t.Errorf("expected 0 instructions, got %d", len(result))
	}
}

// Avoid unused import warning
var _ = token.Token{}
