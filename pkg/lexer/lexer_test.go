package lexer

import (
	"testing"

	"github.com/example/calcium/pkg/token"
)

func TestNextToken(t *testing.T) {
	input := `
	x = 10;
	y = 3.14;
	name = "hello";

	func add(a, b) = a + b;
	func! save(data) = data;

	result = [1, 2, 3]
		|> map(x => x * 2)
		|> filter(x => x > 2);

	match x
		0 => "zero"
		n > 0 => "positive"
		_ => "negative";

	data !> save !? {
		success(r) => r
		failure(e) => e
	};

	arr... |> add;
	foo(1, [2, 3]...);

	0xFF;
	0b1010;
	1.5e10;

	x == y;
	x != y;
	x <= y;
	x >= y;
	x && y;
	x || y;
	x ** 2;

	constraint Positive(n) = n > 0;
	use core.io!;
	`

	tests := []struct {
		expectedType    token.TokenType
		expectedLiteral string
	}{
		// x = 10;
		{token.IDENT, "x"},
		{token.ASSIGN, "="},
		{token.INT, "10"},
		{token.SEMICOLON, ";"},

		// y = 3.14;
		{token.IDENT, "y"},
		{token.ASSIGN, "="},
		{token.FLOAT, "3.14"},
		{token.SEMICOLON, ";"},

		// name = "hello";
		{token.IDENT, "name"},
		{token.ASSIGN, "="},
		{token.STRING, "hello"},
		{token.SEMICOLON, ";"},

		// func add(a, b) = a + b;
		{token.FUNC, "func"},
		{token.IDENT, "add"},
		{token.LPAREN, "("},
		{token.IDENT, "a"},
		{token.COMMA, ","},
		{token.IDENT, "b"},
		{token.RPAREN, ")"},
		{token.ASSIGN, "="},
		{token.IDENT, "a"},
		{token.PLUS, "+"},
		{token.IDENT, "b"},
		{token.SEMICOLON, ";"},

		// func! save(data) = data;
		{token.FUNC_EFFECT, "func!"},
		{token.IDENT, "save"},
		{token.LPAREN, "("},
		{token.IDENT, "data"},
		{token.RPAREN, ")"},
		{token.ASSIGN, "="},
		{token.IDENT, "data"},
		{token.SEMICOLON, ";"},

		// result = [1, 2, 3]
		{token.IDENT, "result"},
		{token.ASSIGN, "="},
		{token.LBRACKET, "["},
		{token.INT, "1"},
		{token.COMMA, ","},
		{token.INT, "2"},
		{token.COMMA, ","},
		{token.INT, "3"},
		{token.RBRACKET, "]"},

		// |> map(x => x * 2)
		{token.PIPE, "|>"},
		{token.MAP, "map"},
		{token.LPAREN, "("},
		{token.IDENT, "x"},
		{token.ARROW, "=>"},
		{token.IDENT, "x"},
		{token.ASTERISK, "*"},
		{token.INT, "2"},
		{token.RPAREN, ")"},

		// |> filter(x => x > 2);
		{token.PIPE, "|>"},
		{token.FILTER, "filter"},
		{token.LPAREN, "("},
		{token.IDENT, "x"},
		{token.ARROW, "=>"},
		{token.IDENT, "x"},
		{token.GT, ">"},
		{token.INT, "2"},
		{token.RPAREN, ")"},
		{token.SEMICOLON, ";"},

		// match x
		{token.MATCH, "match"},
		{token.IDENT, "x"},

		// 0 => "zero"
		{token.INT, "0"},
		{token.ARROW, "=>"},
		{token.STRING, "zero"},

		// n > 0 => "positive"
		{token.IDENT, "n"},
		{token.GT, ">"},
		{token.INT, "0"},
		{token.ARROW, "=>"},
		{token.STRING, "positive"},

		// _ => "negative";
		{token.UNDERSCORE, "_"},
		{token.ARROW, "=>"},
		{token.STRING, "negative"},
		{token.SEMICOLON, ";"},

		// data !> save !? {
		{token.IDENT, "data"},
		{token.EFFECT_PIPE, "!>"},
		{token.IDENT, "save"},
		{token.EFFECT_END, "!?"},
		{token.LBRACE, "{"},

		// success(r) => r
		{token.SUCCESS, "success"},
		{token.LPAREN, "("},
		{token.IDENT, "r"},
		{token.RPAREN, ")"},
		{token.ARROW, "=>"},
		{token.IDENT, "r"},

		// failure(e) => e
		{token.FAILURE, "failure"},
		{token.LPAREN, "("},
		{token.IDENT, "e"},
		{token.RPAREN, ")"},
		{token.ARROW, "=>"},
		{token.IDENT, "e"},

		// };
		{token.RBRACE, "}"},
		{token.SEMICOLON, ";"},

		// arr... |> add;
		{token.IDENT, "arr"},
		{token.SPREAD, "..."},
		{token.PIPE, "|>"},
		{token.IDENT, "add"},
		{token.SEMICOLON, ";"},

		// foo(1, [2, 3]...);
		{token.IDENT, "foo"},
		{token.LPAREN, "("},
		{token.INT, "1"},
		{token.COMMA, ","},
		{token.LBRACKET, "["},
		{token.INT, "2"},
		{token.COMMA, ","},
		{token.INT, "3"},
		{token.RBRACKET, "]"},
		{token.SPREAD, "..."},
		{token.RPAREN, ")"},
		{token.SEMICOLON, ";"},

		// 0xFF;
		{token.INT, "0xFF"},
		{token.SEMICOLON, ";"},

		// 0b1010;
		{token.INT, "0b1010"},
		{token.SEMICOLON, ";"},

		// 1.5e10;
		{token.FLOAT, "1.5e10"},
		{token.SEMICOLON, ";"},

		// x == y;
		{token.IDENT, "x"},
		{token.EQ, "=="},
		{token.IDENT, "y"},
		{token.SEMICOLON, ";"},

		// x != y;
		{token.IDENT, "x"},
		{token.NOT_EQ, "!="},
		{token.IDENT, "y"},
		{token.SEMICOLON, ";"},

		// x <= y;
		{token.IDENT, "x"},
		{token.LT_EQ, "<="},
		{token.IDENT, "y"},
		{token.SEMICOLON, ";"},

		// x >= y;
		{token.IDENT, "x"},
		{token.GT_EQ, ">="},
		{token.IDENT, "y"},
		{token.SEMICOLON, ";"},

		// x && y;
		{token.IDENT, "x"},
		{token.AND, "&&"},
		{token.IDENT, "y"},
		{token.SEMICOLON, ";"},

		// x || y;
		{token.IDENT, "x"},
		{token.OR, "||"},
		{token.IDENT, "y"},
		{token.SEMICOLON, ";"},

		// x ** 2;
		{token.IDENT, "x"},
		{token.POWER, "**"},
		{token.INT, "2"},
		{token.SEMICOLON, ";"},

		// constraint Positive(n) = n > 0;
		{token.CONSTRAINT, "constraint"},
		{token.IDENT, "Positive"},
		{token.LPAREN, "("},
		{token.IDENT, "n"},
		{token.RPAREN, ")"},
		{token.ASSIGN, "="},
		{token.IDENT, "n"},
		{token.GT, ">"},
		{token.INT, "0"},
		{token.SEMICOLON, ";"},

		// use core.io!;
		{token.USE, "use"},
		{token.IDENT, "core"},
		{token.DOT, "."},
		{token.IDENT, "io"},
		{token.BANG, "!"},
		{token.SEMICOLON, ";"},

		{token.EOF, ""},
	}

	l := New(input)

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q (literal=%q)",
				i, tt.expectedType, tok.Type, tok.Literal)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestComments(t *testing.T) {
	input := `
	// single line comment
	x = 10;
	/* multi-line
	   comment */
	y = 20;
	`

	l := New(input)

	tests := []struct {
		expectedType    token.TokenType
		expectedLiteral string
	}{
		{token.IDENT, "x"},
		{token.ASSIGN, "="},
		{token.INT, "10"},
		{token.SEMICOLON, ";"},
		{token.IDENT, "y"},
		{token.ASSIGN, "="},
		{token.INT, "20"},
		{token.SEMICOLON, ";"},
		{token.EOF, ""},
	}

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestStringEscape(t *testing.T) {
	input := `"hello\nworld" "tab\there"`

	l := New(input)

	tok := l.NextToken()
	if tok.Type != token.STRING {
		t.Fatalf("expected STRING, got %q", tok.Type)
	}
	if tok.Literal != `hello\nworld` {
		t.Fatalf("expected %q, got %q", `hello\nworld`, tok.Literal)
	}

	tok = l.NextToken()
	if tok.Type != token.STRING {
		t.Fatalf("expected STRING, got %q", tok.Type)
	}
	if tok.Literal != `tab\there` {
		t.Fatalf("expected %q, got %q", `tab\there`, tok.Literal)
	}
}

func TestRegexLiteral(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`/hello/`, `/hello/`},
		{`/hello/i`, `/hello/i`},
		{`/^[a-z]+$/`, `/^[a-z]+$/`},
		{`/\d{3}-\d{4}/`, `/\d{3}-\d{4}/`},
		{`/https?:\/\//`, `/https?:\/\//`},
		{`/pattern/ims`, `/pattern/ims`},
	}

	for _, tt := range tests {
		l := New(tt.input)
		tok := l.NextToken()

		if tok.Type != token.REGEX {
			t.Errorf("expected REGEX, got %q for input %q", tok.Type, tt.input)
		}
		if tok.Literal != tt.expected {
			t.Errorf("expected literal %q, got %q", tt.expected, tok.Literal)
		}
	}
}

func TestRegexVsDivision(t *testing.T) {
	// Test context detection: regex vs division
	tests := []struct {
		input    string
		tokens   []token.TokenType
	}{
		// Regex contexts
		{`x = /pat/`, []token.TokenType{token.IDENT, token.ASSIGN, token.REGEX}},
		{`f(/pat/)`, []token.TokenType{token.IDENT, token.LPAREN, token.REGEX, token.RPAREN}},
		{`[/pat/]`, []token.TokenType{token.LBRACKET, token.REGEX, token.RBRACKET}},

		// Division contexts
		{`x / y`, []token.TokenType{token.IDENT, token.SLASH, token.IDENT}},
		{`10 / 2`, []token.TokenType{token.INT, token.SLASH, token.INT}},
		{`(x) / y`, []token.TokenType{token.LPAREN, token.IDENT, token.RPAREN, token.SLASH, token.IDENT}},
		{`arr[0] / 2`, []token.TokenType{token.IDENT, token.LBRACKET, token.INT, token.RBRACKET, token.SLASH, token.INT}},
	}

	for _, tt := range tests {
		l := New(tt.input)
		for i, expectedType := range tt.tokens {
			tok := l.NextToken()
			if tok.Type != expectedType {
				t.Errorf("input %q, token %d: expected %q, got %q", tt.input, i, expectedType, tok.Type)
			}
		}
	}
}
