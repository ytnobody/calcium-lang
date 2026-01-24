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
		expected interface{}
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

// Avoid unused import warning
var _ = token.Token{}
