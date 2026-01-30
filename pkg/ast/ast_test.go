package ast

import (
	"testing"

	"github.com/ytnobody/calcium-lang/pkg/token"
)

// Helper to create token
func tok(t token.TokenType, lit string) token.Token {
	return token.Token{Type: t, Literal: lit, Line: 1, Column: 1}
}

// =============================================================================
// Program
// =============================================================================

func TestProgram(t *testing.T) {
	t.Run("TokenLiteral with statements", func(t *testing.T) {
		stmt := &ExpressionStatement{
			Token:      tok(token.IDENT, "foo"),
			Expression: &Identifier{Token: tok(token.IDENT, "foo"), Value: "foo"},
		}
		prog := &Program{Statements: []Statement{stmt}}
		if got := prog.TokenLiteral(); got != "foo" {
			t.Errorf("TokenLiteral() = %q, want %q", got, "foo")
		}
	})

	t.Run("TokenLiteral empty program", func(t *testing.T) {
		prog := &Program{Statements: []Statement{}}
		if got := prog.TokenLiteral(); got != "" {
			t.Errorf("TokenLiteral() = %q, want empty", got)
		}
	})
}

// =============================================================================
// Statements
// =============================================================================

func TestExpressionStatement(t *testing.T) {
	stmt := &ExpressionStatement{
		Token:      tok(token.IDENT, "x"),
		Expression: &Identifier{Token: tok(token.IDENT, "x"), Value: "x"},
	}
	stmt.statementNode() // marker method
	if got := stmt.TokenLiteral(); got != "x" {
		t.Errorf("TokenLiteral() = %q, want %q", got, "x")
	}
}

func TestAssignmentStatement(t *testing.T) {
	stmt := &AssignmentStatement{
		Token: tok(token.IDENT, "x"),
		Name:  &Identifier{Token: tok(token.IDENT, "x"), Value: "x"},
		Value: &IntegerLiteral{Token: tok(token.INT, "42"), Value: 42},
	}
	stmt.statementNode()
	if got := stmt.TokenLiteral(); got != "x" {
		t.Errorf("TokenLiteral() = %q, want %q", got, "x")
	}
}

func TestArrayDestructuringStatement(t *testing.T) {
	stmt := &ArrayDestructuringStatement{
		Token: tok(token.LBRACKET, "["),
		Names: []*Identifier{
			{Token: tok(token.IDENT, "a"), Value: "a"},
			{Token: tok(token.IDENT, "b"), Value: "b"},
		},
		Value: &Identifier{Token: tok(token.IDENT, "arr"), Value: "arr"},
	}
	stmt.statementNode()
	if got := stmt.TokenLiteral(); got != "[" {
		t.Errorf("TokenLiteral() = %q, want %q", got, "[")
	}
}

func TestHeadTailDestructuringStatement(t *testing.T) {
	stmt := &HeadTailDestructuringStatement{
		Token: tok(token.LBRACKET, "["),
		Head:  &Identifier{Token: tok(token.IDENT, "head"), Value: "head"},
		Tail:  &Identifier{Token: tok(token.IDENT, "tail"), Value: "tail"},
		Value: &Identifier{Token: tok(token.IDENT, "arr"), Value: "arr"},
	}
	stmt.statementNode()
	if got := stmt.TokenLiteral(); got != "[" {
		t.Errorf("TokenLiteral() = %q, want %q", got, "[")
	}
}

func TestFunctionDeclaration(t *testing.T) {
	t.Run("normal function", func(t *testing.T) {
		stmt := &FunctionDeclaration{
			Token: tok(token.FUNC, "func"),
			Name:  &Identifier{Token: tok(token.IDENT, "add"), Value: "add"},
			Parameters: []*Identifier{
				{Token: tok(token.IDENT, "a"), Value: "a"},
				{Token: tok(token.IDENT, "b"), Value: "b"},
			},
			Constraints: nil,
			Body:        &IntegerLiteral{Token: tok(token.INT, "0"), Value: 0},
			IsEffect:    false,
		}
		stmt.statementNode()
		if got := stmt.TokenLiteral(); got != "func" {
			t.Errorf("TokenLiteral() = %q, want %q", got, "func")
		}
	})

	t.Run("effect function", func(t *testing.T) {
		stmt := &FunctionDeclaration{
			Token:      tok(token.FUNC_EFFECT, "func!"),
			Name:       &Identifier{Token: tok(token.IDENT, "print"), Value: "print"},
			Parameters: []*Identifier{},
			IsEffect:   true,
		}
		stmt.statementNode()
		if got := stmt.TokenLiteral(); got != "func!" {
			t.Errorf("TokenLiteral() = %q, want %q", got, "func!")
		}
	})
}

func TestConstraintDeclaration(t *testing.T) {
	stmt := &ConstraintDeclaration{
		Token: tok(token.CONSTRAINT, "constraint"),
		Name:  &Identifier{Token: tok(token.IDENT, "Positive"), Value: "Positive"},
		Parameters: []*Identifier{
			{Token: tok(token.IDENT, "n"), Value: "n"},
		},
		Body: &BooleanLiteral{Token: tok(token.TRUE, "true"), Value: true},
	}
	stmt.statementNode()
	if got := stmt.TokenLiteral(); got != "constraint" {
		t.Errorf("TokenLiteral() = %q, want %q", got, "constraint")
	}
}

func TestNamespaceDeclaration(t *testing.T) {
	stmt := &NamespaceDeclaration{
		Token: tok(token.NAMESPACE, "namespace"),
		Name: &ModulePath{
			Token: tok(token.IDENT, "core"),
			Parts: []string{"core", "io"},
		},
	}
	stmt.statementNode()
	if got := stmt.TokenLiteral(); got != "namespace" {
		t.Errorf("TokenLiteral() = %q, want %q", got, "namespace")
	}
}

func TestUseStatement(t *testing.T) {
	t.Run("normal use", func(t *testing.T) {
		stmt := &UseStatement{
			Token: tok(token.USE, "use"),
			Path: &ModulePath{
				Token: tok(token.IDENT, "core"),
				Parts: []string{"core", "io"},
			},
			IsEffect: false,
		}
		stmt.statementNode()
		if got := stmt.TokenLiteral(); got != "use" {
			t.Errorf("TokenLiteral() = %q, want %q", got, "use")
		}
	})

	t.Run("effect use", func(t *testing.T) {
		stmt := &UseStatement{
			Token:    tok(token.USE, "use"),
			IsEffect: true,
		}
		stmt.statementNode()
		if got := stmt.TokenLiteral(); got != "use" {
			t.Errorf("TokenLiteral() = %q, want %q", got, "use")
		}
	})
}

// =============================================================================
// Expressions - Basic
// =============================================================================

func TestIdentifier(t *testing.T) {
	expr := &Identifier{Token: tok(token.IDENT, "foo"), Value: "foo"}
	expr.expressionNode()
	if got := expr.TokenLiteral(); got != "foo" {
		t.Errorf("TokenLiteral() = %q, want %q", got, "foo")
	}
}

func TestModulePath(t *testing.T) {
	t.Run("local module", func(t *testing.T) {
		mp := &ModulePath{
			Token:    tok(token.IDENT, "core"),
			Parts:    []string{"core", "io"},
			IsRemote: false,
		}
		mp.expressionNode()
		if got := mp.TokenLiteral(); got != "core" {
			t.Errorf("TokenLiteral() = %q, want %q", got, "core")
		}
		if got := mp.String(); got != "core.io" {
			t.Errorf("String() = %q, want %q", got, "core.io")
		}
	})

	t.Run("remote module author/name", func(t *testing.T) {
		mp := &ModulePath{
			Token:    tok(token.IDENT, "ytnobody"),
			IsRemote: true,
			Author:   "ytnobody",
			Name:     "json",
		}
		mp.expressionNode()
		if got := mp.String(); got != "ytnobody/json" {
			t.Errorf("String() = %q, want %q", got, "ytnobody/json")
		}
	})

	t.Run("remote module URL", func(t *testing.T) {
		mp := &ModulePath{
			Token:    tok(token.STRING, "github.com/author/repo"),
			RawURL:   "github.com/author/repo",
			IsRemote: true,
		}
		mp.expressionNode()
		if got := mp.String(); got != "github.com/author/repo" {
			t.Errorf("String() = %q, want %q", got, "github.com/author/repo")
		}
	})
}

// =============================================================================
// Expressions - Literals
// =============================================================================

func TestIntegerLiteral(t *testing.T) {
	expr := &IntegerLiteral{Token: tok(token.INT, "42"), Value: 42}
	expr.expressionNode()
	if got := expr.TokenLiteral(); got != "42" {
		t.Errorf("TokenLiteral() = %q, want %q", got, "42")
	}
}

func TestFloatLiteral(t *testing.T) {
	expr := &FloatLiteral{Token: tok(token.FLOAT, "3.14"), Value: 3.14}
	expr.expressionNode()
	if got := expr.TokenLiteral(); got != "3.14" {
		t.Errorf("TokenLiteral() = %q, want %q", got, "3.14")
	}
}

func TestStringLiteral(t *testing.T) {
	expr := &StringLiteral{Token: tok(token.STRING, "hello"), Value: "hello"}
	expr.expressionNode()
	if got := expr.TokenLiteral(); got != "hello" {
		t.Errorf("TokenLiteral() = %q, want %q", got, "hello")
	}
}

func TestInterpolatedString(t *testing.T) {
	expr := &InterpolatedString{
		Token: tok(token.STRING_TEMPLATE_START, "Hello, "),
		Parts: []Expression{
			&StringLiteral{Token: tok(token.STRING, "Hello, "), Value: "Hello, "},
			&Identifier{Token: tok(token.IDENT, "name"), Value: "name"},
			&StringLiteral{Token: tok(token.STRING, "!"), Value: "!"},
		},
	}
	expr.expressionNode()
	if got := expr.TokenLiteral(); got != "Hello, " {
		t.Errorf("TokenLiteral() = %q, want %q", got, "Hello, ")
	}
}

func TestRegexLiteral(t *testing.T) {
	expr := &RegexLiteral{
		Token:   tok(token.REGEX, "/[a-z]+/i"),
		Pattern: "[a-z]+",
		Flags:   "i",
	}
	expr.expressionNode()
	if got := expr.TokenLiteral(); got != "/[a-z]+/i" {
		t.Errorf("TokenLiteral() = %q, want %q", got, "/[a-z]+/i")
	}
	if got := expr.String(); got != "/[a-z]+/i" {
		t.Errorf("String() = %q, want %q", got, "/[a-z]+/i")
	}
}

func TestBooleanLiteral(t *testing.T) {
	t.Run("true", func(t *testing.T) {
		expr := &BooleanLiteral{Token: tok(token.TRUE, "true"), Value: true}
		expr.expressionNode()
		if got := expr.TokenLiteral(); got != "true" {
			t.Errorf("TokenLiteral() = %q, want %q", got, "true")
		}
	})

	t.Run("false", func(t *testing.T) {
		expr := &BooleanLiteral{Token: tok(token.FALSE, "false"), Value: false}
		expr.expressionNode()
		if got := expr.TokenLiteral(); got != "false" {
			t.Errorf("TokenLiteral() = %q, want %q", got, "false")
		}
	})
}

// =============================================================================
// Expressions - Collections
// =============================================================================

func TestArrayLiteral(t *testing.T) {
	t.Run("comma-separated", func(t *testing.T) {
		expr := &ArrayLiteral{
			Token: tok(token.LBRACKET, "["),
			Elements: []Expression{
				&IntegerLiteral{Token: tok(token.INT, "1"), Value: 1},
				&IntegerLiteral{Token: tok(token.INT, "2"), Value: 2},
			},
			IsConcat: false,
		}
		expr.expressionNode()
		if got := expr.TokenLiteral(); got != "[" {
			t.Errorf("TokenLiteral() = %q, want %q", got, "[")
		}
	})

	t.Run("space-separated concatenation", func(t *testing.T) {
		expr := &ArrayLiteral{
			Token:    tok(token.LBRACKET, "["),
			Elements: []Expression{},
			IsConcat: true,
		}
		expr.expressionNode()
		if got := expr.TokenLiteral(); got != "[" {
			t.Errorf("TokenLiteral() = %q, want %q", got, "[")
		}
	})
}

func TestHashLiteral(t *testing.T) {
	expr := &HashLiteral{
		Token: tok(token.LBRACE, "{"),
		Pairs: []HashPair{
			{
				Key:   &Identifier{Token: tok(token.IDENT, "name"), Value: "name"},
				Value: &StringLiteral{Token: tok(token.STRING, "Alice"), Value: "Alice"},
			},
			{
				Key:   &Identifier{Token: tok(token.IDENT, "age"), Value: "age"},
				Value: &IntegerLiteral{Token: tok(token.INT, "30"), Value: 30},
			},
		},
	}
	expr.expressionNode()
	if got := expr.TokenLiteral(); got != "{" {
		t.Errorf("TokenLiteral() = %q, want %q", got, "{")
	}
}

// =============================================================================
// Expressions - Operations
// =============================================================================

func TestPrefixExpression(t *testing.T) {
	t.Run("negation", func(t *testing.T) {
		expr := &PrefixExpression{
			Token:    tok(token.MINUS, "-"),
			Operator: "-",
			Right:    &IntegerLiteral{Token: tok(token.INT, "5"), Value: 5},
		}
		expr.expressionNode()
		if got := expr.TokenLiteral(); got != "-" {
			t.Errorf("TokenLiteral() = %q, want %q", got, "-")
		}
	})

	t.Run("not", func(t *testing.T) {
		expr := &PrefixExpression{
			Token:    tok(token.BANG, "!"),
			Operator: "!",
			Right:    &BooleanLiteral{Token: tok(token.TRUE, "true"), Value: true},
		}
		expr.expressionNode()
		if got := expr.TokenLiteral(); got != "!" {
			t.Errorf("TokenLiteral() = %q, want %q", got, "!")
		}
	})
}

func TestInfixExpression(t *testing.T) {
	operators := []struct {
		tok token.TokenType
		op  string
	}{
		{token.PLUS, "+"},
		{token.MINUS, "-"},
		{token.ASTERISK, "*"},
		{token.SLASH, "/"},
		{token.PERCENT, "%"},
		{token.POWER, "**"},
		{token.EQ, "=="},
		{token.NOT_EQ, "!="},
		{token.LT, "<"},
		{token.GT, ">"},
		{token.LT_EQ, "<="},
		{token.GT_EQ, ">="},
		{token.AND, "&&"},
		{token.OR, "||"},
	}

	for _, op := range operators {
		t.Run(op.op, func(t *testing.T) {
			expr := &InfixExpression{
				Token:    tok(op.tok, op.op),
				Left:     &IntegerLiteral{Token: tok(token.INT, "1"), Value: 1},
				Operator: op.op,
				Right:    &IntegerLiteral{Token: tok(token.INT, "2"), Value: 2},
			}
			expr.expressionNode()
			if got := expr.TokenLiteral(); got != op.op {
				t.Errorf("TokenLiteral() = %q, want %q", got, op.op)
			}
		})
	}
}

func TestCallExpression(t *testing.T) {
	expr := &CallExpression{
		Token:    tok(token.LPAREN, "("),
		Function: &Identifier{Token: tok(token.IDENT, "add"), Value: "add"},
		Arguments: []Expression{
			&IntegerLiteral{Token: tok(token.INT, "1"), Value: 1},
			&IntegerLiteral{Token: tok(token.INT, "2"), Value: 2},
		},
	}
	expr.expressionNode()
	if got := expr.TokenLiteral(); got != "(" {
		t.Errorf("TokenLiteral() = %q, want %q", got, "(")
	}
}

func TestIndexExpression(t *testing.T) {
	expr := &IndexExpression{
		Token: tok(token.LBRACKET, "["),
		Left:  &Identifier{Token: tok(token.IDENT, "arr"), Value: "arr"},
		Index: &IntegerLiteral{Token: tok(token.INT, "0"), Value: 0},
	}
	expr.expressionNode()
	if got := expr.TokenLiteral(); got != "[" {
		t.Errorf("TokenLiteral() = %q, want %q", got, "[")
	}
}

func TestMemberExpression(t *testing.T) {
	expr := &MemberExpression{
		Token:    tok(token.DOT, "."),
		Object:   &Identifier{Token: tok(token.IDENT, "obj"), Value: "obj"},
		Property: &Identifier{Token: tok(token.IDENT, "field"), Value: "field"},
	}
	expr.expressionNode()
	if got := expr.TokenLiteral(); got != "." {
		t.Errorf("TokenLiteral() = %q, want %q", got, ".")
	}
}

// =============================================================================
// Expressions - Functions and Pipes
// =============================================================================

func TestLambdaExpression(t *testing.T) {
	t.Run("single param", func(t *testing.T) {
		expr := &LambdaExpression{
			Token: tok(token.ARROW, "=>"),
			Parameters: []*Identifier{
				{Token: tok(token.IDENT, "x"), Value: "x"},
			},
			Body: &Identifier{Token: tok(token.IDENT, "x"), Value: "x"},
		}
		expr.expressionNode()
		if got := expr.TokenLiteral(); got != "=>" {
			t.Errorf("TokenLiteral() = %q, want %q", got, "=>")
		}
	})

	t.Run("multi params", func(t *testing.T) {
		expr := &LambdaExpression{
			Token: tok(token.ARROW, "=>"),
			Parameters: []*Identifier{
				{Token: tok(token.IDENT, "a"), Value: "a"},
				{Token: tok(token.IDENT, "b"), Value: "b"},
			},
			Body: &InfixExpression{
				Token:    tok(token.PLUS, "+"),
				Left:     &Identifier{Token: tok(token.IDENT, "a"), Value: "a"},
				Operator: "+",
				Right:    &Identifier{Token: tok(token.IDENT, "b"), Value: "b"},
			},
		}
		expr.expressionNode()
		if got := expr.TokenLiteral(); got != "=>" {
			t.Errorf("TokenLiteral() = %q, want %q", got, "=>")
		}
	})
}

func TestPipeExpression(t *testing.T) {
	expr := &PipeExpression{
		Token: tok(token.PIPE, "|>"),
		Left:  &IntegerLiteral{Token: tok(token.INT, "5"), Value: 5},
		Right: &Identifier{Token: tok(token.IDENT, "double"), Value: "double"},
	}
	expr.expressionNode()
	if got := expr.TokenLiteral(); got != "|>" {
		t.Errorf("TokenLiteral() = %q, want %q", got, "|>")
	}
}

func TestEffectPipeExpression(t *testing.T) {
	expr := &EffectPipeExpression{
		Token: tok(token.EFFECT_PIPE, "!>"),
		Left:  &StringLiteral{Token: tok(token.STRING, "hello"), Value: "hello"},
		Right: &Identifier{Token: tok(token.IDENT, "print"), Value: "print"},
	}
	expr.expressionNode()
	if got := expr.TokenLiteral(); got != "!>" {
		t.Errorf("TokenLiteral() = %q, want %q", got, "!>")
	}
}

func TestSpreadExpression(t *testing.T) {
	expr := &SpreadExpression{
		Token: tok(token.SPREAD, "..."),
		Value: &Identifier{Token: tok(token.IDENT, "arr"), Value: "arr"},
	}
	expr.expressionNode()
	if got := expr.TokenLiteral(); got != "..." {
		t.Errorf("TokenLiteral() = %q, want %q", got, "...")
	}
}

// =============================================================================
// Expressions - Match
// =============================================================================

func TestMatchExpression(t *testing.T) {
	expr := &MatchExpression{
		Token:   tok(token.MATCH, "match"),
		Subject: &Identifier{Token: tok(token.IDENT, "x"), Value: "x"},
		Cases: []*MatchCase{
			{
				Token:     tok(token.ARROW, "=>"),
				Pattern:   &IntegerLiteral{Token: tok(token.INT, "1"), Value: 1},
				Body:      &StringLiteral{Token: tok(token.STRING, "one"), Value: "one"},
				IsDefault: false,
			},
			{
				Token:     tok(token.ARROW, "=>"),
				Pattern:   &WildcardExpression{Token: tok(token.UNDERSCORE, "_")},
				Body:      &StringLiteral{Token: tok(token.STRING, "other"), Value: "other"},
				IsDefault: true,
			},
		},
	}
	expr.expressionNode()
	if got := expr.TokenLiteral(); got != "match" {
		t.Errorf("TokenLiteral() = %q, want %q", got, "match")
	}
}

// =============================================================================
// Expressions - Effect Handling
// =============================================================================

func TestEffectHandleExpression(t *testing.T) {
	expr := &EffectHandleExpression{
		Token:       tok(token.EFFECT_END, "!?"),
		Subject:     &Identifier{Token: tok(token.IDENT, "result"), Value: "result"},
		SuccessVar:  &Identifier{Token: tok(token.IDENT, "v"), Value: "v"},
		SuccessBody: &Identifier{Token: tok(token.IDENT, "v"), Value: "v"},
		FailureVar:  &Identifier{Token: tok(token.IDENT, "e"), Value: "e"},
		FailureBody: &StringLiteral{Token: tok(token.STRING, "error"), Value: "error"},
	}
	expr.expressionNode()
	if got := expr.TokenLiteral(); got != "!?" {
		t.Errorf("TokenLiteral() = %q, want %q", got, "!?")
	}
}

func TestSuccessExpression(t *testing.T) {
	expr := &SuccessExpression{
		Token: tok(token.SUCCESS, "success"),
		Value: &IntegerLiteral{Token: tok(token.INT, "42"), Value: 42},
	}
	expr.expressionNode()
	if got := expr.TokenLiteral(); got != "success" {
		t.Errorf("TokenLiteral() = %q, want %q", got, "success")
	}
}

func TestFailureExpression(t *testing.T) {
	expr := &FailureExpression{
		Token: tok(token.FAILURE, "failure"),
		Value: &StringLiteral{Token: tok(token.STRING, "error"), Value: "error"},
	}
	expr.expressionNode()
	if got := expr.TokenLiteral(); got != "failure" {
		t.Errorf("TokenLiteral() = %q, want %q", got, "failure")
	}
}

// =============================================================================
// Expressions - Special
// =============================================================================

func TestWildcardExpression(t *testing.T) {
	expr := &WildcardExpression{Token: tok(token.UNDERSCORE, "_")}
	expr.expressionNode()
	if got := expr.TokenLiteral(); got != "_" {
		t.Errorf("TokenLiteral() = %q, want %q", got, "_")
	}
}

func TestReturnExpression(t *testing.T) {
	expr := &ReturnExpression{
		Token: tok(token.RETURN, "return"),
		Value: &IntegerLiteral{Token: tok(token.INT, "0"), Value: 0},
	}
	expr.expressionNode()
	if got := expr.TokenLiteral(); got != "return" {
		t.Errorf("TokenLiteral() = %q, want %q", got, "return")
	}
}

func TestConstraintCheckExpression(t *testing.T) {
	expr := &ConstraintCheckExpression{
		Token:      tok(token.QUESTION, "?"),
		Constraint: &Identifier{Token: tok(token.IDENT, "Positive"), Value: "Positive"},
	}
	expr.expressionNode()
	if got := expr.TokenLiteral(); got != "?" {
		t.Errorf("TokenLiteral() = %q, want %q", got, "?")
	}
}

func TestConstrainedParameter(t *testing.T) {
	param := &ConstrainedParameter{
		Name:       &Identifier{Token: tok(token.IDENT, "n"), Value: "n"},
		Constraint: &Identifier{Token: tok(token.IDENT, "Positive"), Value: "Positive"},
	}
	// ConstrainedParameter is a helper struct, not a Node
	if param.Name.Value != "n" {
		t.Errorf("Name.Value = %q, want %q", param.Name.Value, "n")
	}
	if param.Constraint.Value != "Positive" {
		t.Errorf("Constraint.Value = %q, want %q", param.Constraint.Value, "Positive")
	}
}

// =============================================================================
// Expressions - Async
// =============================================================================

func TestStayExpression(t *testing.T) {
	expr := &StayExpression{
		Token: tok(token.IDENT, "async"),
		StateInit: []StayStatePair{
			{
				Name:  &Identifier{Token: tok(token.IDENT, "count"), Value: "count"},
				Value: &IntegerLiteral{Token: tok(token.INT, "0"), Value: 0},
			},
			{
				Name:  &Identifier{Token: tok(token.IDENT, "results"), Value: "results"},
				Value: &ArrayLiteral{Token: tok(token.LBRACKET, "["), Elements: []Expression{}},
			},
		},
		Body: []Statement{
			&ExpressionStatement{
				Token:      tok(token.IDENT, "count"),
				Expression: &Identifier{Token: tok(token.IDENT, "count"), Value: "count"},
			},
		},
	}
	expr.expressionNode()
	if got := expr.TokenLiteral(); got != "async" {
		t.Errorf("TokenLiteral() = %q, want %q", got, "async")
	}
}

func TestStayStatePair(t *testing.T) {
	pair := StayStatePair{
		Name:  &Identifier{Token: tok(token.IDENT, "x"), Value: "x"},
		Value: &IntegerLiteral{Token: tok(token.INT, "10"), Value: 10},
	}
	// StayStatePair is a helper struct, not a Node
	if pair.Name.Value != "x" {
		t.Errorf("Name.Value = %q, want %q", pair.Name.Value, "x")
	}
}
