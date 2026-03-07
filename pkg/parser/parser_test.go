package parser

import (
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
