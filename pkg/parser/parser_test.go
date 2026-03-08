package parser

import (
	"strings"
	"testing"

	"github.com/ytnobody/calcium-lang/pkg/ast"
	"github.com/ytnobody/calcium-lang/pkg/lexer"
)

func TestAssignmentStatements(t *testing.T) {
	input := `
x = 5;
y = 10;
name = "hello";
`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 3 {
		t.Fatalf("program.Statements does not contain 3 statements. got=%d",
			len(program.Statements))
	}

	tests := []struct {
		expectedName string
	}{
		{"x"},
		{"y"},
		{"name"},
	}

	for i, tt := range tests {
		stmt := program.Statements[i]
		if !testAssignmentStatement(t, stmt, tt.expectedName) {
			return
		}
	}
}

func testAssignmentStatement(t *testing.T, s ast.Statement, name string) bool {
	assignStmt, ok := s.(*ast.AssignmentStatement)
	if !ok {
		t.Errorf("s not *ast.AssignmentStatement. got=%T", s)
		return false
	}

	if assignStmt.Name.Value != name {
		t.Errorf("assignStmt.Name.Value not '%s'. got=%s", name, assignStmt.Name.Value)
		return false
	}

	return true
}

func TestFunctionDeclaration(t *testing.T) {
	input := `func add(a, b) = a + b;`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 1 {
		t.Fatalf("program.Statements does not contain 1 statement. got=%d",
			len(program.Statements))
	}

	stmt, ok := program.Statements[0].(*ast.FunctionDeclaration)
	if !ok {
		t.Fatalf("program.Statements[0] is not *ast.FunctionDeclaration. got=%T",
			program.Statements[0])
	}

	if stmt.Name.Value != "add" {
		t.Errorf("function name wrong. expected=add, got=%s", stmt.Name.Value)
	}

	if len(stmt.Parameters) != 2 {
		t.Errorf("function parameters wrong. expected=2, got=%d", len(stmt.Parameters))
	}

	if stmt.IsEffect {
		t.Errorf("function should not be effect function")
	}
}

func TestEffectFunctionDeclaration(t *testing.T) {
	input := `func! save(data) = data;`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 1 {
		t.Fatalf("program.Statements does not contain 1 statement. got=%d",
			len(program.Statements))
	}

	stmt, ok := program.Statements[0].(*ast.FunctionDeclaration)
	if !ok {
		t.Fatalf("program.Statements[0] is not *ast.FunctionDeclaration. got=%T",
			program.Statements[0])
	}

	if stmt.Name.Value != "save" {
		t.Errorf("function name wrong. expected=save, got=%s", stmt.Name.Value)
	}

	if !stmt.IsEffect {
		t.Errorf("function should be effect function")
	}
}

func TestConstraintDeclaration(t *testing.T) {
	input := `constraint Positive(n) = n > 0;`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 1 {
		t.Fatalf("program.Statements does not contain 1 statement. got=%d",
			len(program.Statements))
	}

	stmt, ok := program.Statements[0].(*ast.ConstraintDeclaration)
	if !ok {
		t.Fatalf("program.Statements[0] is not *ast.ConstraintDeclaration. got=%T",
			program.Statements[0])
	}

	if stmt.Name.Value != "Positive" {
		t.Errorf("constraint name wrong. expected=Positive, got=%s", stmt.Name.Value)
	}
}

func TestUseStatement(t *testing.T) {
	tests := []struct {
		input    string
		parts    []string
		isEffect bool
	}{
		{`use core.io!;`, []string{"core", "io"}, true},
		{`use math.utils;`, []string{"math", "utils"}, false},
	}

	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := New(l)
		program := p.ParseProgram()
		checkParserErrors(t, p)

		stmt, ok := program.Statements[0].(*ast.UseStatement)
		if !ok {
			t.Fatalf("statement is not *ast.UseStatement. got=%T",
				program.Statements[0])
		}

		if len(stmt.Path.Parts) != len(tt.parts) {
			t.Errorf("path parts wrong. expected=%d, got=%d",
				len(tt.parts), len(stmt.Path.Parts))
		}

		for i, part := range tt.parts {
			if stmt.Path.Parts[i] != part {
				t.Errorf("path part[%d] wrong. expected=%s, got=%s",
					i, part, stmt.Path.Parts[i])
			}
		}

		if stmt.IsEffect != tt.isEffect {
			t.Errorf("isEffect wrong. expected=%v, got=%v",
				tt.isEffect, stmt.IsEffect)
		}
	}
}

func TestRemoteUseStatement(t *testing.T) {
	// Test author/module format
	t.Run("author/module format", func(t *testing.T) {
		tests := []struct {
			input    string
			author   string
			name     string
			isEffect bool
		}{
			{`use ytnobody/json;`, "ytnobody", "json", false},
			{`use ytnobody/json!;`, "ytnobody", "json", true},
			{`use author/module_name;`, "author", "module_name", false},
		}

		for _, tt := range tests {
			l := lexer.New(tt.input)
			p := New(l)
			program := p.ParseProgram()
			checkParserErrors(t, p)

			stmt, ok := program.Statements[0].(*ast.UseStatement)
			if !ok {
				t.Fatalf("statement is not *ast.UseStatement. got=%T",
					program.Statements[0])
			}

			if !stmt.Path.IsRemote {
				t.Errorf("path should be remote")
			}

			if stmt.Path.Author != tt.author {
				t.Errorf("author wrong. expected=%s, got=%s",
					tt.author, stmt.Path.Author)
			}

			if stmt.Path.Name != tt.name {
				t.Errorf("name wrong. expected=%s, got=%s",
					tt.name, stmt.Path.Name)
			}

			if stmt.IsEffect != tt.isEffect {
				t.Errorf("isEffect wrong. expected=%v, got=%v",
					tt.isEffect, stmt.IsEffect)
			}

			// Check String() method
			expected := tt.author + "/" + tt.name
			if stmt.Path.String() != expected {
				t.Errorf("String() wrong. expected=%s, got=%s",
					expected, stmt.Path.String())
			}
		}
	})

	// Test URL format
	t.Run("URL format", func(t *testing.T) {
		tests := []struct {
			input    string
			url      string
			isEffect bool
		}{
			{`use "github.com/ytnobody/json-calcium";`, "github.com/ytnobody/json-calcium", false},
			{`use "github.com/ytnobody/json-calcium"!;`, "github.com/ytnobody/json-calcium", true},
		}

		for _, tt := range tests {
			l := lexer.New(tt.input)
			p := New(l)
			program := p.ParseProgram()
			checkParserErrors(t, p)

			stmt, ok := program.Statements[0].(*ast.UseStatement)
			if !ok {
				t.Fatalf("statement is not *ast.UseStatement. got=%T",
					program.Statements[0])
			}

			if !stmt.Path.IsRemote {
				t.Errorf("path should be remote")
			}

			if stmt.Path.RawURL != tt.url {
				t.Errorf("RawURL wrong. expected=%s, got=%s",
					tt.url, stmt.Path.RawURL)
			}

			if stmt.IsEffect != tt.isEffect {
				t.Errorf("isEffect wrong. expected=%v, got=%v",
					tt.isEffect, stmt.IsEffect)
			}

			// Check String() method
			if stmt.Path.String() != tt.url {
				t.Errorf("String() wrong. expected=%s, got=%s",
					tt.url, stmt.Path.String())
			}
		}
	})
}

func TestIntegerLiteralExpression(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"5;", 5},
		{"10;", 10},
		{"0xFF;", 255},
		{"0b1010;", 10},
	}

	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := New(l)
		program := p.ParseProgram()
		checkParserErrors(t, p)

		if len(program.Statements) != 1 {
			t.Fatalf("program has not enough statements. got=%d",
				len(program.Statements))
		}
		stmt, ok := program.Statements[0].(*ast.ExpressionStatement)
		if !ok {
			t.Fatalf("program.Statements[0] is not ast.ExpressionStatement. got=%T",
				program.Statements[0])
		}

		literal, ok := stmt.Expression.(*ast.IntegerLiteral)
		if !ok {
			t.Fatalf("exp not *ast.IntegerLiteral. got=%T", stmt.Expression)
		}
		if literal.Value != tt.expected {
			t.Errorf("literal.Value not %d. got=%d", tt.expected, literal.Value)
		}
	}
}

func TestPrefixExpressions(t *testing.T) {
	prefixTests := []struct {
		input    string
		operator string
		value    interface{}
	}{
		{"!true;", "!", true},
		{"!false;", "!", false},
		{"-15;", "-", int64(15)},
	}

	for _, tt := range prefixTests {
		l := lexer.New(tt.input)
		p := New(l)
		program := p.ParseProgram()
		checkParserErrors(t, p)

		if len(program.Statements) != 1 {
			t.Fatalf("program.Statements does not contain 1 statement. got=%d",
				len(program.Statements))
		}

		stmt, ok := program.Statements[0].(*ast.ExpressionStatement)
		if !ok {
			t.Fatalf("program.Statements[0] is not ast.ExpressionStatement. got=%T",
				program.Statements[0])
		}

		exp, ok := stmt.Expression.(*ast.PrefixExpression)
		if !ok {
			t.Fatalf("stmt is not ast.PrefixExpression. got=%T", stmt.Expression)
		}
		if exp.Operator != tt.operator {
			t.Fatalf("exp.Operator is not '%s'. got=%s",
				tt.operator, exp.Operator)
		}
	}
}

func TestInfixExpressions(t *testing.T) {
	infixTests := []struct {
		input      string
		leftValue  interface{}
		operator   string
		rightValue interface{}
	}{
		{"5 + 5;", 5, "+", 5},
		{"5 - 5;", 5, "-", 5},
		{"5 * 5;", 5, "*", 5},
		{"5 / 5;", 5, "/", 5},
		{"5 % 5;", 5, "%", 5},
		{"5 ** 2;", 5, "**", 2},
		{"5 > 5;", 5, ">", 5},
		{"5 < 5;", 5, "<", 5},
		{"5 == 5;", 5, "==", 5},
		{"5 != 5;", 5, "!=", 5},
		{"5 >= 5;", 5, ">=", 5},
		{"5 <= 5;", 5, "<=", 5},
		{"true && false;", true, "&&", false},
		{"true || false;", true, "||", false},
	}

	for _, tt := range infixTests {
		l := lexer.New(tt.input)
		p := New(l)
		program := p.ParseProgram()
		checkParserErrors(t, p)

		if len(program.Statements) != 1 {
			t.Fatalf("program.Statements does not contain 1 statement. got=%d",
				len(program.Statements))
		}

		stmt, ok := program.Statements[0].(*ast.ExpressionStatement)
		if !ok {
			t.Fatalf("program.Statements[0] is not ast.ExpressionStatement. got=%T",
				program.Statements[0])
		}

		exp, ok := stmt.Expression.(*ast.InfixExpression)
		if !ok {
			t.Fatalf("exp is not ast.InfixExpression. got=%T", stmt.Expression)
		}

		if exp.Operator != tt.operator {
			t.Fatalf("exp.Operator is not '%s'. got=%s",
				tt.operator, exp.Operator)
		}
	}
}

func TestOperatorPrecedenceParsing(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"-a * b;", "((-a) * b)"},
		{"!-a;", "(!(-a))"},
		{"a + b + c;", "((a + b) + c)"},
		{"a + b - c;", "((a + b) - c)"},
		{"a * b * c;", "((a * b) * c)"},
		{"a * b / c;", "((a * b) / c)"},
		{"a + b / c;", "(a + (b / c))"},
		{"a + b * c + d / e - f;", "(((a + (b * c)) + (d / e)) - f)"},
		{"5 > 4 == 3 < 4;", "((5 > 4) == (3 < 4))"},
		{"3 + 4 * 5 == 3 * 1 + 4 * 5;", "((3 + (4 * 5)) == ((3 * 1) + (4 * 5)))"},
		{"true && false || true;", "((true && false) || true)"},
		{"a ** b ** c;", "(a ** (b ** c))"},
		// Chained comparisons: a <= b <= c becomes (a <= b) && (b <= c)
		{"0 <= x <= 100;", "((0 <= x) && (x <= 100))"},
		{"1 < x < 10;", "((1 < x) && (x < 10))"},
		{"a > b > c;", "((a > b) && (b > c))"},
		{"a >= b >= c;", "((a >= b) && (b >= c))"},
	}

	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := New(l)
		program := p.ParseProgram()
		checkParserErrors(t, p)

		actual := programToString(program)
		if actual != tt.expected {
			t.Errorf("expected=%q, got=%q", tt.expected, actual)
		}
	}
}

func TestArrayLiteral(t *testing.T) {
	input := `[1, 2, 3];`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	array, ok := stmt.Expression.(*ast.ArrayLiteral)
	if !ok {
		t.Fatalf("exp not ast.ArrayLiteral. got=%T", stmt.Expression)
	}

	if len(array.Elements) != 3 {
		t.Fatalf("array.Elements wrong. expected=3, got=%d", len(array.Elements))
	}

	if array.IsConcat {
		t.Errorf("array.IsConcat should be false")
	}
}

func TestArrayConcatLiteral(t *testing.T) {
	// Test simple space-separated concat: [a b c]
	// Note: [[1, 2] [3, 4]] has ambiguity with index syntax and requires special handling
	input := `[a b c];`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	array, ok := stmt.Expression.(*ast.ArrayLiteral)
	if !ok {
		t.Fatalf("exp not ast.ArrayLiteral. got=%T", stmt.Expression)
	}

	if len(array.Elements) != 3 {
		t.Fatalf("array.Elements wrong. expected=3, got=%d", len(array.Elements))
	}

	if !array.IsConcat {
		t.Errorf("array.IsConcat should be true")
	}

	// Verify elements are identifiers
	for i, elem := range array.Elements {
		ident, ok := elem.(*ast.Identifier)
		if !ok {
			t.Errorf("element[%d] not ast.Identifier. got=%T", i, elem)
		}
		expected := string(rune('a' + i))
		if ident.Value != expected {
			t.Errorf("element[%d] wrong. expected=%s, got=%s", i, expected, ident.Value)
		}
	}
}

func TestLambdaExpression(t *testing.T) {
	tests := []struct {
		input          string
		expectedParams []string
	}{
		{"x => x * 2;", []string{"x"}},
		{"(x) => x * 2;", []string{"x"}},
		{"(x, y) => x + y;", []string{"x", "y"}},
		{"() => 42;", []string{}},
	}

	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := New(l)
		program := p.ParseProgram()
		checkParserErrors(t, p)

		stmt := program.Statements[0].(*ast.ExpressionStatement)
		lambda, ok := stmt.Expression.(*ast.LambdaExpression)
		if !ok {
			t.Fatalf("exp not ast.LambdaExpression. got=%T for input %s",
				stmt.Expression, tt.input)
		}

		if len(lambda.Parameters) != len(tt.expectedParams) {
			t.Errorf("parameters wrong. expected=%d, got=%d for input %s",
				len(tt.expectedParams), len(lambda.Parameters), tt.input)
		}

		for i, param := range tt.expectedParams {
			if lambda.Parameters[i].Value != param {
				t.Errorf("parameter[%d] wrong. expected=%s, got=%s",
					i, param, lambda.Parameters[i].Value)
			}
		}
	}
}

func TestPipeExpression(t *testing.T) {
	input := `x |> double |> add(1);`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	pipe, ok := stmt.Expression.(*ast.PipeExpression)
	if !ok {
		t.Fatalf("exp not ast.PipeExpression. got=%T", stmt.Expression)
	}

	// Should be: (x |> double) |> add(1)
	leftPipe, ok := pipe.Left.(*ast.PipeExpression)
	if !ok {
		t.Fatalf("pipe.Left not ast.PipeExpression. got=%T", pipe.Left)
	}

	_, ok = leftPipe.Left.(*ast.Identifier)
	if !ok {
		t.Fatalf("leftPipe.Left not ast.Identifier. got=%T", leftPipe.Left)
	}
}

func TestErrorPropPipeExpression(t *testing.T) {
	t.Run("simple propagation", func(t *testing.T) {
		input := `x |>? validate;`

		l := lexer.New(input)
		p := New(l)
		program := p.ParseProgram()
		checkParserErrors(t, p)

		stmt := program.Statements[0].(*ast.ExpressionStatement)
		pipe, ok := stmt.Expression.(*ast.ErrorPropPipeExpression)
		if !ok {
			t.Fatalf("exp not ast.ErrorPropPipeExpression. got=%T", stmt.Expression)
		}

		ident, ok := pipe.Left.(*ast.Identifier)
		if !ok {
			t.Fatalf("pipe.Left not ast.Identifier. got=%T", pipe.Left)
		}
		if ident.Value != "x" {
			t.Fatalf("pipe.Left.Value not 'x'. got=%q", ident.Value)
		}

		right, ok := pipe.Right.(*ast.Identifier)
		if !ok {
			t.Fatalf("pipe.Right not ast.Identifier. got=%T", pipe.Right)
		}
		if right.Value != "validate" {
			t.Fatalf("pipe.Right.Value not 'validate'. got=%q", right.Value)
		}
	})

	t.Run("chained propagation", func(t *testing.T) {
		input := `x |>? validate |>? save;`

		l := lexer.New(input)
		p := New(l)
		program := p.ParseProgram()
		checkParserErrors(t, p)

		stmt := program.Statements[0].(*ast.ExpressionStatement)
		// Should be: (x |>? validate) |>? save
		pipe, ok := stmt.Expression.(*ast.ErrorPropPipeExpression)
		if !ok {
			t.Fatalf("exp not ast.ErrorPropPipeExpression. got=%T", stmt.Expression)
		}

		_, ok = pipe.Left.(*ast.ErrorPropPipeExpression)
		if !ok {
			t.Fatalf("pipe.Left not ast.ErrorPropPipeExpression. got=%T", pipe.Left)
		}
	})

	t.Run("with extra arguments", func(t *testing.T) {
		input := `x |>? validate(schema);`

		l := lexer.New(input)
		p := New(l)
		program := p.ParseProgram()
		checkParserErrors(t, p)

		stmt := program.Statements[0].(*ast.ExpressionStatement)
		pipe, ok := stmt.Expression.(*ast.ErrorPropPipeExpression)
		if !ok {
			t.Fatalf("exp not ast.ErrorPropPipeExpression. got=%T", stmt.Expression)
		}

		_, ok = pipe.Right.(*ast.CallExpression)
		if !ok {
			t.Fatalf("pipe.Right not ast.CallExpression. got=%T", pipe.Right)
		}
	})
}

func TestSpreadExpression(t *testing.T) {
	input := `[1, 2, 3]... |> add;`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	pipe, ok := stmt.Expression.(*ast.PipeExpression)
	if !ok {
		t.Fatalf("exp not ast.PipeExpression. got=%T", stmt.Expression)
	}

	spread, ok := pipe.Left.(*ast.SpreadExpression)
	if !ok {
		t.Fatalf("pipe.Left not ast.SpreadExpression. got=%T", pipe.Left)
	}

	_, ok = spread.Value.(*ast.ArrayLiteral)
	if !ok {
		t.Fatalf("spread.Value not ast.ArrayLiteral. got=%T", spread.Value)
	}
}

func TestMatchExpression(t *testing.T) {
	input := `match x
		0 => "zero"
		n > 0 => "positive"
		_ => "negative";`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	match, ok := stmt.Expression.(*ast.MatchExpression)
	if !ok {
		t.Fatalf("exp not ast.MatchExpression. got=%T", stmt.Expression)
	}

	if len(match.Cases) != 3 {
		t.Fatalf("match.Cases wrong. expected=3, got=%d", len(match.Cases))
	}

	// Last case should be default
	if !match.Cases[2].IsDefault {
		t.Errorf("last case should be default")
	}
}

func TestMatchGuardClause(t *testing.T) {
	input := `match x
		n if n > 0 => "positive"
		_ => "other";`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	match, ok := stmt.Expression.(*ast.MatchExpression)
	if !ok {
		t.Fatalf("exp not ast.MatchExpression. got=%T", stmt.Expression)
	}

	if len(match.Cases) != 2 {
		t.Fatalf("match.Cases wrong. expected=2, got=%d", len(match.Cases))
	}

	// First case should have a guard
	if match.Cases[0].Guard == nil {
		t.Errorf("first case should have a guard clause")
	}

	// First case pattern should be identifier "n"
	ident, ok := match.Cases[0].Pattern.(*ast.Identifier)
	if !ok {
		t.Fatalf("first case pattern not ast.Identifier. got=%T", match.Cases[0].Pattern)
	}
	if ident.Value != "n" {
		t.Errorf("pattern identifier wrong. expected='n', got=%q", ident.Value)
	}

	// Second case should be default with no guard
	if !match.Cases[1].IsDefault {
		t.Errorf("second case should be default")
	}
	if match.Cases[1].Guard != nil {
		t.Errorf("default case should not have a guard")
	}
}

func TestTupleLiteral(t *testing.T) {
	input := `(1, "hello", true);`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	tuple, ok := stmt.Expression.(*ast.TupleLiteral)
	if !ok {
		t.Fatalf("exp not ast.TupleLiteral. got=%T", stmt.Expression)
	}

	if len(tuple.Elements) != 3 {
		t.Fatalf("tuple.Elements wrong. expected=3, got=%d", len(tuple.Elements))
	}
}

func TestTupleVsLambda(t *testing.T) {
	// (x, y) => x + y should still be a lambda, not a tuple
	input := `(x, y) => x + y;`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	_, ok := stmt.Expression.(*ast.LambdaExpression)
	if !ok {
		t.Fatalf("exp not ast.LambdaExpression. got=%T", stmt.Expression)
	}
}

func TestCallExpression(t *testing.T) {
	input := `add(1, 2 * 3, 4 + 5);`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	call, ok := stmt.Expression.(*ast.CallExpression)
	if !ok {
		t.Fatalf("exp not ast.CallExpression. got=%T", stmt.Expression)
	}

	if len(call.Arguments) != 3 {
		t.Fatalf("call.Arguments wrong. expected=3, got=%d", len(call.Arguments))
	}
}

func TestMemberExpression(t *testing.T) {
	input := `io.say;`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	member, ok := stmt.Expression.(*ast.MemberExpression)
	if !ok {
		t.Fatalf("exp not ast.MemberExpression. got=%T", stmt.Expression)
	}

	obj, ok := member.Object.(*ast.Identifier)
	if !ok {
		t.Fatalf("member.Object not ast.Identifier. got=%T", member.Object)
	}

	if obj.Value != "io" {
		t.Errorf("object wrong. expected=io, got=%s", obj.Value)
	}

	if member.Property.Value != "say" {
		t.Errorf("property wrong. expected=say, got=%s", member.Property.Value)
	}
}

func checkParserErrors(t *testing.T, p *Parser) {
	errors := p.Errors()
	if len(errors) == 0 {
		return
	}

	t.Errorf("parser has %d errors", len(errors))
	for _, msg := range errors {
		t.Errorf("parser error: %q", msg)
	}
	t.FailNow()
}

// Helper function to convert AST back to string for precedence testing
func programToString(program *ast.Program) string {
	if len(program.Statements) == 0 {
		return ""
	}
	stmt, ok := program.Statements[0].(*ast.ExpressionStatement)
	if !ok {
		return ""
	}
	return exprToString(stmt.Expression)
}

func exprToString(exp ast.Expression) string {
	switch e := exp.(type) {
	case *ast.Identifier:
		return e.Value
	case *ast.IntegerLiteral:
		return e.Token.Literal
	case *ast.BooleanLiteral:
		return e.Token.Literal
	case *ast.PrefixExpression:
		return "(" + e.Operator + exprToString(e.Right) + ")"
	case *ast.InfixExpression:
		return "(" + exprToString(e.Left) + " " + e.Operator + " " + exprToString(e.Right) + ")"
	default:
		return ""
	}
}

func TestErrorMessagesWithHints(t *testing.T) {
	t.Run("unclosed parenthesis hint", func(t *testing.T) {
		input := `(1 + 2;`
		l := lexer.New(input)
		p := New(l)
		p.ParseProgram()
		errors := p.Errors()
		if len(errors) == 0 {
			t.Fatal("expected parse errors, got none")
		}
		found := false
		for _, e := range errors {
			if containsString(e, "unclosed '('") || containsString(e, "parenthes") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected hint about unclosed parenthesis, got: %v", errors)
		}
	})

	t.Run("unexpected EOF hint", func(t *testing.T) {
		input := `1 +`
		l := lexer.New(input)
		p := New(l)
		p.ParseProgram()
		errors := p.Errors()
		if len(errors) == 0 {
			t.Fatal("expected parse errors, got none")
		}
		found := false
		for _, e := range errors {
			if containsString(e, "end of input") || containsString(e, "incomplete") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected hint about unexpected end of input, got: %v", errors)
		}
	})

	t.Run("source context in error", func(t *testing.T) {
		input := "x = 1;\ny = (2 + 3;\n"
		l := lexer.New(input)
		p := New(l)
		p.ParseProgram()
		errors := p.Errors()
		if len(errors) == 0 {
			t.Fatal("expected parse errors, got none")
		}
		// Error messages should contain source context (caret indicator)
		found := false
		for _, e := range errors {
			if containsString(e, "^") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected caret indicator in error message, got: %v", errors)
		}
	})

	t.Run("formatErrorWithHint adds hint", func(t *testing.T) {
		l := lexer.New("x = 1;")
		p := New(l)
		msg := p.formatErrorWithHint(1, 1, "some error", "try this fix")
		if !containsString(msg, "hint: try this fix") {
			t.Errorf("expected hint in message, got: %s", msg)
		}
	})

	t.Run("formatErrorWithHint without hint", func(t *testing.T) {
		l := lexer.New("x = 1;")
		p := New(l)
		msg := p.formatErrorWithHint(1, 1, "some error", "")
		if containsString(msg, "hint:") {
			t.Errorf("expected no hint in message, got: %s", msg)
		}
	})
}

func containsString(s, substr string) bool {
	return strings.Contains(s, substr)
}

// ---- gradual typing tests ----

func TestTypedAssignmentStatement(t *testing.T) {
	input := `x: Int = 42;`
	l := lexer.New(input)
	p := New(l)
	prog := p.ParseProgram()
	checkParserErrors(t, p)

	if len(prog.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(prog.Statements))
	}

	stmt, ok := prog.Statements[0].(*ast.AssignmentStatement)
	if !ok {
		t.Fatalf("expected *ast.AssignmentStatement, got %T", prog.Statements[0])
	}

	if stmt.Name.Value != "x" {
		t.Errorf("expected name 'x', got %q", stmt.Name.Value)
	}

	if stmt.TypeAnnot == nil {
		t.Fatal("expected type annotation, got nil")
	}

	if stmt.TypeAnnot.Name != "Int" {
		t.Errorf("expected type annotation 'Int', got %q", stmt.TypeAnnot.Name)
	}
}

func TestTypedAssignmentMultiple(t *testing.T) {
	input := `
x: Int = 1;
name: String = "hello";
flag: Bool = true;
`
	l := lexer.New(input)
	p := New(l)
	prog := p.ParseProgram()
	checkParserErrors(t, p)

	if len(prog.Statements) != 3 {
		t.Fatalf("expected 3 statements, got %d", len(prog.Statements))
	}

	expected := []struct {
		name     string
		typeName string
	}{
		{"x", "Int"},
		{"name", "String"},
		{"flag", "Bool"},
	}

	for i, ex := range expected {
		stmt, ok := prog.Statements[i].(*ast.AssignmentStatement)
		if !ok {
			t.Fatalf("[%d] expected *ast.AssignmentStatement, got %T", i, prog.Statements[i])
		}
		if stmt.Name.Value != ex.name {
			t.Errorf("[%d] expected name %q, got %q", i, ex.name, stmt.Name.Value)
		}
		if stmt.TypeAnnot == nil {
			t.Fatalf("[%d] expected type annotation, got nil", i)
		}
		if stmt.TypeAnnot.Name != ex.typeName {
			t.Errorf("[%d] expected type %q, got %q", i, ex.typeName, stmt.TypeAnnot.Name)
		}
	}
}

func TestUntypedAssignmentUnchanged(t *testing.T) {
	// Plain assignment without type annotation should still work.
	input := `x = 42;`
	l := lexer.New(input)
	p := New(l)
	prog := p.ParseProgram()
	checkParserErrors(t, p)

	stmt, ok := prog.Statements[0].(*ast.AssignmentStatement)
	if !ok {
		t.Fatalf("expected *ast.AssignmentStatement, got %T", prog.Statements[0])
	}
	if stmt.TypeAnnot != nil {
		t.Errorf("expected no type annotation, got %q", stmt.TypeAnnot.Name)
	}
}

func TestFunctionDeclarationWithTypes(t *testing.T) {
	input := `func add(a: Int, b: Int): Int = a + b;`
	l := lexer.New(input)
	p := New(l)
	prog := p.ParseProgram()
	checkParserErrors(t, p)

	if len(prog.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(prog.Statements))
	}

	fd, ok := prog.Statements[0].(*ast.FunctionDeclaration)
	if !ok {
		t.Fatalf("expected *ast.FunctionDeclaration, got %T", prog.Statements[0])
	}

	if fd.Name.Value != "add" {
		t.Errorf("expected function name 'add', got %q", fd.Name.Value)
	}

	if len(fd.Parameters) != 2 {
		t.Fatalf("expected 2 parameters, got %d", len(fd.Parameters))
	}

	if len(fd.ParamTypes) != 2 {
		t.Fatalf("expected 2 param type annotations, got %d", len(fd.ParamTypes))
	}

	if fd.ParamTypes[0] == nil || fd.ParamTypes[0].Name != "Int" {
		t.Errorf("expected param 0 type 'Int', got %v", fd.ParamTypes[0])
	}
	if fd.ParamTypes[1] == nil || fd.ParamTypes[1].Name != "Int" {
		t.Errorf("expected param 1 type 'Int', got %v", fd.ParamTypes[1])
	}

	if fd.ReturnType == nil {
		t.Fatal("expected return type annotation, got nil")
	}
	if fd.ReturnType.Name != "Int" {
		t.Errorf("expected return type 'Int', got %q", fd.ReturnType.Name)
	}
}

func TestFunctionDeclarationWithTypedParamsNoReturn(t *testing.T) {
	input := `func double(n: Int) = n + n;`
	l := lexer.New(input)
	p := New(l)
	prog := p.ParseProgram()
	checkParserErrors(t, p)

	fd, ok := prog.Statements[0].(*ast.FunctionDeclaration)
	if !ok {
		t.Fatalf("expected *ast.FunctionDeclaration, got %T", prog.Statements[0])
	}

	if fd.ReturnType != nil {
		t.Error("expected no return type annotation")
	}
	if len(fd.ParamTypes) != 1 || fd.ParamTypes[0] == nil {
		t.Fatalf("expected 1 non-nil param type annotation, got %v", fd.ParamTypes)
	}
	if fd.ParamTypes[0].Name != "Int" {
		t.Errorf("expected param type 'Int', got %q", fd.ParamTypes[0].Name)
	}
}

func TestConstraintAnnotationUnchanged(t *testing.T) {
	// Constraints with '?' should still work as before.
	input := `
constraint Positive(x) = x > 0;
func square(n: Positive?) = n * n;
`
	l := lexer.New(input)
	p := New(l)
	prog := p.ParseProgram()
	checkParserErrors(t, p)

	if len(prog.Statements) != 2 {
		t.Fatalf("expected 2 statements, got %d", len(prog.Statements))
	}

	fd, ok := prog.Statements[1].(*ast.FunctionDeclaration)
	if !ok {
		t.Fatalf("expected *ast.FunctionDeclaration, got %T", prog.Statements[1])
	}

	// Should have a constraint, not a type annotation
	if len(fd.Constraints) != 1 {
		t.Fatalf("expected 1 constraint, got %d", len(fd.Constraints))
	}
	if fd.Constraints[0] == nil {
		t.Error("expected non-nil constraint")
	}
	if len(fd.ParamTypes) != 1 || fd.ParamTypes[0] != nil {
		t.Error("expected no type annotation for constrained param")
	}
}

// ---- Additional tests for uncovered code paths ----

func TestFloatLiteralExpression(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{"3.14;", 3.14},
		{"0.5;", 0.5},
		{"100.0;", 100.0},
	}

	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := New(l)
		program := p.ParseProgram()
		checkParserErrors(t, p)

		if len(program.Statements) != 1 {
			t.Fatalf("program has not enough statements. got=%d", len(program.Statements))
		}
		stmt, ok := program.Statements[0].(*ast.ExpressionStatement)
		if !ok {
			t.Fatalf("program.Statements[0] is not ast.ExpressionStatement. got=%T",
				program.Statements[0])
		}
		literal, ok := stmt.Expression.(*ast.FloatLiteral)
		if !ok {
			t.Fatalf("exp not *ast.FloatLiteral. got=%T", stmt.Expression)
		}
		if literal.Value != tt.expected {
			t.Errorf("literal.Value not %f. got=%f", tt.expected, literal.Value)
		}
	}
}

func TestStringLiteral(t *testing.T) {
	input := `"hello world";`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	str, ok := stmt.Expression.(*ast.StringLiteral)
	if !ok {
		t.Fatalf("exp not *ast.StringLiteral. got=%T", stmt.Expression)
	}
	if str.Value != "hello world" {
		t.Errorf("str.Value not %q. got=%q", "hello world", str.Value)
	}
}

func TestInterpolatedString(t *testing.T) {
	input := `"hello ${name} world";`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	is, ok := stmt.Expression.(*ast.InterpolatedString)
	if !ok {
		t.Fatalf("exp not *ast.InterpolatedString. got=%T", stmt.Expression)
	}
	if len(is.Parts) < 3 {
		t.Fatalf("expected at least 3 parts, got %d", len(is.Parts))
	}
}

func TestRegexLiteral(t *testing.T) {
	input := `/hello/gi;`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	regex, ok := stmt.Expression.(*ast.RegexLiteral)
	if !ok {
		t.Fatalf("exp not *ast.RegexLiteral. got=%T", stmt.Expression)
	}
	if regex.Pattern != "hello" {
		t.Errorf("regex.Pattern not %q. got=%q", "hello", regex.Pattern)
	}
	if regex.Flags != "gi" {
		t.Errorf("regex.Flags not %q. got=%q", "gi", regex.Flags)
	}
}

func TestHashLiteral(t *testing.T) {
	t.Run("empty hash", func(t *testing.T) {
		input := `{};`
		l := lexer.New(input)
		p := New(l)
		program := p.ParseProgram()
		checkParserErrors(t, p)

		stmt := program.Statements[0].(*ast.ExpressionStatement)
		hash, ok := stmt.Expression.(*ast.HashLiteral)
		if !ok {
			t.Fatalf("exp not *ast.HashLiteral. got=%T", stmt.Expression)
		}
		if len(hash.Pairs) != 0 {
			t.Errorf("expected 0 pairs, got %d", len(hash.Pairs))
		}
	})

	t.Run("hash with pairs", func(t *testing.T) {
		input := `{name: "alice", age: 30};`
		l := lexer.New(input)
		p := New(l)
		program := p.ParseProgram()
		checkParserErrors(t, p)

		stmt := program.Statements[0].(*ast.ExpressionStatement)
		hash, ok := stmt.Expression.(*ast.HashLiteral)
		if !ok {
			t.Fatalf("exp not *ast.HashLiteral. got=%T", stmt.Expression)
		}
		if len(hash.Pairs) != 2 {
			t.Fatalf("expected 2 pairs, got %d", len(hash.Pairs))
		}
	})
}

func TestIndexExpression(t *testing.T) {
	input := `arr[1];`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	idx, ok := stmt.Expression.(*ast.IndexExpression)
	if !ok {
		t.Fatalf("exp not *ast.IndexExpression. got=%T", stmt.Expression)
	}

	ident, ok := idx.Left.(*ast.Identifier)
	if !ok {
		t.Fatalf("idx.Left not *ast.Identifier. got=%T", idx.Left)
	}
	if ident.Value != "arr" {
		t.Errorf("expected 'arr', got %q", ident.Value)
	}

	index, ok := idx.Index.(*ast.IntegerLiteral)
	if !ok {
		t.Fatalf("idx.Index not *ast.IntegerLiteral. got=%T", idx.Index)
	}
	if index.Value != 1 {
		t.Errorf("expected index 1, got %d", index.Value)
	}
}

func TestTypeDeclaration(t *testing.T) {
	t.Run("simple variants", func(t *testing.T) {
		input := `type Color = Red | Green | Blue;`
		l := lexer.New(input)
		p := New(l)
		program := p.ParseProgram()
		checkParserErrors(t, p)

		if len(program.Statements) != 1 {
			t.Fatalf("expected 1 statement, got %d", len(program.Statements))
		}

		td, ok := program.Statements[0].(*ast.TypeDeclaration)
		if !ok {
			t.Fatalf("expected *ast.TypeDeclaration, got %T", program.Statements[0])
		}
		if td.Name.Value != "Color" {
			t.Errorf("expected type name 'Color', got %q", td.Name.Value)
		}
		if len(td.Variants) != 3 {
			t.Fatalf("expected 3 variants, got %d", len(td.Variants))
		}
		expected := []string{"Red", "Green", "Blue"}
		for i, v := range td.Variants {
			if v.Name != expected[i] {
				t.Errorf("variant[%d] expected %q, got %q", i, expected[i], v.Name)
			}
		}
	})

	t.Run("variants with fields", func(t *testing.T) {
		input := `type Shape = Circle(radius) | Rect(width, height);`
		l := lexer.New(input)
		p := New(l)
		program := p.ParseProgram()
		checkParserErrors(t, p)

		td := program.Statements[0].(*ast.TypeDeclaration)
		if len(td.Variants) != 2 {
			t.Fatalf("expected 2 variants, got %d", len(td.Variants))
		}
		if len(td.Variants[0].Fields) != 1 {
			t.Fatalf("expected 1 field for Circle, got %d", len(td.Variants[0].Fields))
		}
		if td.Variants[0].Fields[0] != "radius" {
			t.Errorf("expected field 'radius', got %q", td.Variants[0].Fields[0])
		}
		if len(td.Variants[1].Fields) != 2 {
			t.Fatalf("expected 2 fields for Rect, got %d", len(td.Variants[1].Fields))
		}
	})
}

func TestNamespaceDeclaration(t *testing.T) {
	input := `namespace math.utils;`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}

	ns, ok := program.Statements[0].(*ast.NamespaceDeclaration)
	if !ok {
		t.Fatalf("expected *ast.NamespaceDeclaration, got %T", program.Statements[0])
	}
	if len(ns.Name.Parts) != 2 {
		t.Fatalf("expected 2 parts, got %d", len(ns.Name.Parts))
	}
	if ns.Name.Parts[0] != "math" || ns.Name.Parts[1] != "utils" {
		t.Errorf("expected [math, utils], got %v", ns.Name.Parts)
	}
}

func TestDoExpression(t *testing.T) {
	t.Run("simple do block", func(t *testing.T) {
		input := `do x = 1; x + 1 end;`
		l := lexer.New(input)
		p := New(l)
		program := p.ParseProgram()
		checkParserErrors(t, p)

		stmt := program.Statements[0].(*ast.ExpressionStatement)
		doExpr, ok := stmt.Expression.(*ast.DoExpression)
		if !ok {
			t.Fatalf("exp not *ast.DoExpression. got=%T", stmt.Expression)
		}
		if len(doExpr.Statements) != 1 {
			t.Errorf("expected 1 intermediate statement, got %d", len(doExpr.Statements))
		}
		if doExpr.FinalExpression == nil {
			t.Fatal("expected final expression, got nil")
		}
	})

	t.Run("do block with just expression", func(t *testing.T) {
		input := `do 42 end;`
		l := lexer.New(input)
		p := New(l)
		program := p.ParseProgram()
		checkParserErrors(t, p)

		stmt := program.Statements[0].(*ast.ExpressionStatement)
		doExpr, ok := stmt.Expression.(*ast.DoExpression)
		if !ok {
			t.Fatalf("exp not *ast.DoExpression. got=%T", stmt.Expression)
		}
		if doExpr.FinalExpression == nil {
			t.Fatal("expected final expression, got nil")
		}
	})

	t.Run("unclosed do block", func(t *testing.T) {
		input := `do 42`
		l := lexer.New(input)
		p := New(l)
		p.ParseProgram()
		errors := p.Errors()
		if len(errors) == 0 {
			t.Fatal("expected parse errors for unclosed do block")
		}
	})

	t.Run("empty do block", func(t *testing.T) {
		input := `do end;`
		l := lexer.New(input)
		p := New(l)
		p.ParseProgram()
		errors := p.Errors()
		if len(errors) == 0 {
			t.Fatal("expected parse errors for empty do block")
		}
	})
}

func TestReturnExpression(t *testing.T) {
	input := `return 42;`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	ret, ok := stmt.Expression.(*ast.ReturnExpression)
	if !ok {
		t.Fatalf("exp not *ast.ReturnExpression. got=%T", stmt.Expression)
	}
	if ret.Value == nil {
		t.Fatal("expected return value, got nil")
	}
}

func TestSuccessExpression(t *testing.T) {
	input := `success(42);`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	succ, ok := stmt.Expression.(*ast.SuccessExpression)
	if !ok {
		t.Fatalf("exp not *ast.SuccessExpression. got=%T", stmt.Expression)
	}
	if succ.Value == nil {
		t.Fatal("expected success value, got nil")
	}
}

func TestFailureExpression(t *testing.T) {
	input := `failure("error");`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	fail, ok := stmt.Expression.(*ast.FailureExpression)
	if !ok {
		t.Fatalf("exp not *ast.FailureExpression. got=%T", stmt.Expression)
	}
	if fail.Value == nil {
		t.Fatal("expected failure value, got nil")
	}
}

func TestWildcardExpression(t *testing.T) {
	input := `_;`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	_, ok := stmt.Expression.(*ast.WildcardExpression)
	if !ok {
		t.Fatalf("exp not *ast.WildcardExpression. got=%T", stmt.Expression)
	}
}

func TestEffectPipeExpression(t *testing.T) {
	input := `x !> save;`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	pipe, ok := stmt.Expression.(*ast.EffectPipeExpression)
	if !ok {
		t.Fatalf("exp not *ast.EffectPipeExpression. got=%T", stmt.Expression)
	}
	if pipe.Left == nil || pipe.Right == nil {
		t.Fatal("expected left and right for effect pipe")
	}
}

func TestConstraintCheckExpression(t *testing.T) {
	input := `x?;`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	check, ok := stmt.Expression.(*ast.ConstraintCheckExpression)
	if !ok {
		t.Fatalf("exp not *ast.ConstraintCheckExpression. got=%T", stmt.Expression)
	}
	if check.Constraint == nil {
		t.Fatal("expected constraint")
	}
}

func TestEffectHandleExpression(t *testing.T) {
	input := `x !? { success(v) => v failure(e) => 0 };`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	handle, ok := stmt.Expression.(*ast.EffectHandleExpression)
	if !ok {
		t.Fatalf("exp not *ast.EffectHandleExpression. got=%T", stmt.Expression)
	}
	if handle.SuccessVar == nil {
		t.Error("expected success var")
	}
	if handle.FailureVar == nil {
		t.Error("expected failure var")
	}
	if handle.SuccessBody == nil {
		t.Error("expected success body")
	}
	if handle.FailureBody == nil {
		t.Error("expected failure body")
	}
}

func TestBuiltinAsIdentifier(t *testing.T) {
	builtins := []string{"map", "filter", "reduce", "has", "keys", "values", "len"}
	for _, name := range builtins {
		input := name + "(x);"
		l := lexer.New(input)
		p := New(l)
		program := p.ParseProgram()
		checkParserErrors(t, p)

		stmt := program.Statements[0].(*ast.ExpressionStatement)
		call, ok := stmt.Expression.(*ast.CallExpression)
		if !ok {
			t.Fatalf("for %s: exp not *ast.CallExpression. got=%T", name, stmt.Expression)
		}
		ident, ok := call.Function.(*ast.Identifier)
		if !ok {
			t.Fatalf("for %s: function not *ast.Identifier. got=%T", name, call.Function)
		}
		if ident.Value != name {
			t.Errorf("expected %q, got %q", name, ident.Value)
		}
	}
}

func TestConstructorPatternInMatch(t *testing.T) {
	input := `match shape
		Circle(r) => r
		Rect(w, h) => w
		_ => 0;`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	match, ok := stmt.Expression.(*ast.MatchExpression)
	if !ok {
		t.Fatalf("exp not *ast.MatchExpression. got=%T", stmt.Expression)
	}

	if len(match.Cases) != 3 {
		t.Fatalf("expected 3 cases, got %d", len(match.Cases))
	}

	// First case: Circle(r)
	cp, ok := match.Cases[0].Pattern.(*ast.ConstructorPattern)
	if !ok {
		t.Fatalf("case 0 pattern not *ast.ConstructorPattern. got=%T", match.Cases[0].Pattern)
	}
	if cp.Name != "Circle" {
		t.Errorf("expected constructor name 'Circle', got %q", cp.Name)
	}
	if len(cp.Fields) != 1 {
		t.Fatalf("expected 1 field, got %d", len(cp.Fields))
	}

	// Second case: Rect(w, h)
	cp2, ok := match.Cases[1].Pattern.(*ast.ConstructorPattern)
	if !ok {
		t.Fatalf("case 1 pattern not *ast.ConstructorPattern. got=%T", match.Cases[1].Pattern)
	}
	if len(cp2.Fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(cp2.Fields))
	}
}

func TestArrayDestructuringStatement(t *testing.T) {
	input := `[a, b, c] = [1, 2, 3];`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}

	destr, ok := program.Statements[0].(*ast.ArrayDestructuringStatement)
	if !ok {
		t.Fatalf("expected *ast.ArrayDestructuringStatement, got %T", program.Statements[0])
	}
	if len(destr.Names) != 3 {
		t.Fatalf("expected 3 names, got %d", len(destr.Names))
	}
	expected := []string{"a", "b", "c"}
	for i, n := range destr.Names {
		if n.Value != expected[i] {
			t.Errorf("name[%d] expected %q, got %q", i, expected[i], n.Value)
		}
	}
}

func TestHeadTailDestructuringStatement(t *testing.T) {
	input := `[head | tail] = [1, 2, 3];`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}

	destr, ok := program.Statements[0].(*ast.HeadTailDestructuringStatement)
	if !ok {
		t.Fatalf("expected *ast.HeadTailDestructuringStatement, got %T", program.Statements[0])
	}
	if destr.Head.Value != "head" {
		t.Errorf("expected head 'head', got %q", destr.Head.Value)
	}
	if destr.Tail.Value != "tail" {
		t.Errorf("expected tail 'tail', got %q", destr.Tail.Value)
	}
}

func TestEmptyArray(t *testing.T) {
	input := `[];`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	array, ok := stmt.Expression.(*ast.ArrayLiteral)
	if !ok {
		t.Fatalf("exp not *ast.ArrayLiteral. got=%T", stmt.Expression)
	}
	if len(array.Elements) != 0 {
		t.Errorf("expected 0 elements, got %d", len(array.Elements))
	}
}

func TestLambdaWithBlockBody(t *testing.T) {
	t.Run("single ident lambda with block", func(t *testing.T) {
		input := `x => { x + 1 };`
		l := lexer.New(input)
		p := New(l)
		program := p.ParseProgram()
		checkParserErrors(t, p)

		stmt := program.Statements[0].(*ast.ExpressionStatement)
		lambda, ok := stmt.Expression.(*ast.LambdaExpression)
		if !ok {
			t.Fatalf("exp not *ast.LambdaExpression. got=%T", stmt.Expression)
		}
		if len(lambda.Parameters) != 1 {
			t.Errorf("expected 1 parameter, got %d", len(lambda.Parameters))
		}
		// Body should be a DoExpression (reused for block)
		_, ok = lambda.Body.(*ast.DoExpression)
		if !ok {
			t.Fatalf("lambda body not *ast.DoExpression. got=%T", lambda.Body)
		}
	})

	t.Run("parenthesized lambda with block", func(t *testing.T) {
		input := `(x, y) => { x + y };`
		l := lexer.New(input)
		p := New(l)
		program := p.ParseProgram()
		checkParserErrors(t, p)

		stmt := program.Statements[0].(*ast.ExpressionStatement)
		lambda, ok := stmt.Expression.(*ast.LambdaExpression)
		if !ok {
			t.Fatalf("exp not *ast.LambdaExpression. got=%T", stmt.Expression)
		}
		if len(lambda.Parameters) != 2 {
			t.Errorf("expected 2 parameters, got %d", len(lambda.Parameters))
		}
	})

	t.Run("empty block body", func(t *testing.T) {
		input := `() => {};`
		l := lexer.New(input)
		p := New(l)
		program := p.ParseProgram()
		checkParserErrors(t, p)

		stmt := program.Statements[0].(*ast.ExpressionStatement)
		lambda, ok := stmt.Expression.(*ast.LambdaExpression)
		if !ok {
			t.Fatalf("exp not *ast.LambdaExpression. got=%T", stmt.Expression)
		}
		doExpr, ok := lambda.Body.(*ast.DoExpression)
		if !ok {
			t.Fatalf("lambda body not *ast.DoExpression. got=%T", lambda.Body)
		}
		// Empty block should still have a final expression (placeholder)
		if doExpr.FinalExpression == nil {
			t.Error("expected final expression in empty block")
		}
	})

	t.Run("block with assignments", func(t *testing.T) {
		input := `(x) => { y = x + 1; y };`
		l := lexer.New(input)
		p := New(l)
		program := p.ParseProgram()
		checkParserErrors(t, p)

		stmt := program.Statements[0].(*ast.ExpressionStatement)
		lambda, ok := stmt.Expression.(*ast.LambdaExpression)
		if !ok {
			t.Fatalf("exp not *ast.LambdaExpression. got=%T", stmt.Expression)
		}
		doExpr, ok := lambda.Body.(*ast.DoExpression)
		if !ok {
			t.Fatalf("lambda body not *ast.DoExpression. got=%T", lambda.Body)
		}
		if len(doExpr.Statements) != 1 {
			t.Errorf("expected 1 intermediate statement, got %d", len(doExpr.Statements))
		}
	})
}

func TestNoPrefixParseFnErrors(t *testing.T) {
	tests := []struct {
		input       string
		expectError bool
	}{
		{`= 5;`, true},          // ASSIGN
		{`};`, true},            // RBRACE
		{`];`, true},            // RBRACKET
		{`);`, true},            // RPAREN
		{`, 5;`, true},          // COMMA
		{`;;`, true},            // SEMICOLON
	}

	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := New(l)
		p.ParseProgram()
		errors := p.Errors()
		if tt.expectError && len(errors) == 0 {
			t.Errorf("expected errors for input %q, got none", tt.input)
		}
	}
}

func TestGetExpectationHints(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Unclosed bracket
		{`[1, 2, 3;`, "unclosed"},
		// Unclosed brace
		{`{name: 1;`, "unclosed"},
	}

	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := New(l)
		p.ParseProgram()
		errors := p.Errors()
		if len(errors) == 0 {
			t.Errorf("expected errors for input %q", tt.input)
			continue
		}
		found := false
		for _, e := range errors {
			if containsString(e, tt.expected) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected error containing %q for input %q, got: %v",
				tt.expected, tt.input, errors)
		}
	}
}

func TestDoBlockWithLastStatementNotExpression(t *testing.T) {
	// do block where the last statement is an assignment
	input := `do x = 5 end;`
	l := lexer.New(input)
	p := New(l)
	p.ParseProgram()
	errors := p.Errors()
	if len(errors) == 0 {
		t.Fatal("expected error when do block last item is assignment")
	}
	found := false
	for _, e := range errors {
		if containsString(e, "must be an expression") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error about last item must be expression, got: %v", errors)
	}
}

func TestAssignmentWithEffectHandle(t *testing.T) {
	input := `x = getData() !? { success(v) => v failure(e) => 0 };`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt, ok := program.Statements[0].(*ast.AssignmentStatement)
	if !ok {
		t.Fatalf("expected *ast.AssignmentStatement, got %T", program.Statements[0])
	}

	handle, ok := stmt.Value.(*ast.EffectHandleExpression)
	if !ok {
		t.Fatalf("expected value to be *ast.EffectHandleExpression, got %T", stmt.Value)
	}
	if handle.SuccessVar == nil || handle.FailureVar == nil {
		t.Error("expected both success and failure vars")
	}
}

func TestTypedAssignmentWithUnknownType(t *testing.T) {
	input := `x: UnknownType = 42;`
	l := lexer.New(input)
	p := New(l)
	p.ParseProgram()
	errors := p.Errors()
	if len(errors) == 0 {
		t.Fatal("expected error for unknown type name")
	}
	found := false
	for _, e := range errors {
		if containsString(e, "unknown type name") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error about unknown type name, got: %v", errors)
	}
}

func TestFunctionDeclarationWithEffectHandle(t *testing.T) {
	input := `func safe(x) = getData(x) !? { success(v) => v failure(e) => 0 };`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	fd, ok := program.Statements[0].(*ast.FunctionDeclaration)
	if !ok {
		t.Fatalf("expected *ast.FunctionDeclaration, got %T", program.Statements[0])
	}

	_, ok = fd.Body.(*ast.EffectHandleExpression)
	if !ok {
		t.Fatalf("expected body to be *ast.EffectHandleExpression, got %T", fd.Body)
	}
}

func TestFunctionDeclarationWithUnknownReturnType(t *testing.T) {
	input := `func add(a, b): UnknownType = a + b;`
	l := lexer.New(input)
	p := New(l)
	p.ParseProgram()
	errors := p.Errors()
	if len(errors) == 0 {
		t.Fatal("expected error for unknown return type")
	}
	found := false
	for _, e := range errors {
		if containsString(e, "unknown return type") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error about unknown return type, got: %v", errors)
	}
}

func TestTypedAssignmentWithEffectHandle(t *testing.T) {
	input := `x: Int = getData() !? { success(v) => v failure(e) => 0 };`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt, ok := program.Statements[0].(*ast.AssignmentStatement)
	if !ok {
		t.Fatalf("expected *ast.AssignmentStatement, got %T", program.Statements[0])
	}
	if stmt.TypeAnnot == nil {
		t.Fatal("expected type annotation")
	}
	_, ok = stmt.Value.(*ast.EffectHandleExpression)
	if !ok {
		t.Fatalf("expected value to be *ast.EffectHandleExpression, got %T", stmt.Value)
	}
}

func TestExpressionStatementWithEffectHandle(t *testing.T) {
	input := `getData() !? { success(v) => v failure(e) => 0 };`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	_, ok := stmt.Expression.(*ast.EffectHandleExpression)
	if !ok {
		t.Fatalf("expected *ast.EffectHandleExpression, got %T", stmt.Expression)
	}
}

func TestGroupedExpression(t *testing.T) {
	input := `(5 + 3) * 2;`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	infix, ok := stmt.Expression.(*ast.InfixExpression)
	if !ok {
		t.Fatalf("exp not *ast.InfixExpression. got=%T", stmt.Expression)
	}
	if infix.Operator != "*" {
		t.Errorf("expected '*', got %q", infix.Operator)
	}
}

func TestEmptyParenthesesError(t *testing.T) {
	// () without => should be an error
	input := `() + 1;`
	l := lexer.New(input)
	p := New(l)
	p.ParseProgram()
	errors := p.Errors()
	if len(errors) == 0 {
		t.Fatal("expected errors for empty parentheses without arrow")
	}
}

func TestHashBuiltinAsIdentifier(t *testing.T) {
	// "hash" is also a builtin
	input := `hash(x);`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	call, ok := stmt.Expression.(*ast.CallExpression)
	if !ok {
		t.Fatalf("exp not *ast.CallExpression. got=%T", stmt.Expression)
	}
	ident, ok := call.Function.(*ast.Identifier)
	if !ok {
		t.Fatalf("function not *ast.Identifier. got=%T", call.Function)
	}
	if ident.Value != "hash" {
		t.Errorf("expected 'hash', got %q", ident.Value)
	}
}

func TestDoBlockMultipleStatements(t *testing.T) {
	input := `do
		x = 1;
		y = 2;
		x + y
	end;`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	doExpr, ok := stmt.Expression.(*ast.DoExpression)
	if !ok {
		t.Fatalf("exp not *ast.DoExpression. got=%T", stmt.Expression)
	}
	if len(doExpr.Statements) != 2 {
		t.Errorf("expected 2 intermediate statements, got %d", len(doExpr.Statements))
	}
	if doExpr.FinalExpression == nil {
		t.Fatal("expected final expression")
	}
}
