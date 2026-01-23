package token

type TokenType string

type Token struct {
	Type    TokenType
	Literal string
	Line    int
	Column  int
}

const (
	// Special tokens
	ILLEGAL TokenType = "ILLEGAL"
	EOF     TokenType = "EOF"

	// Identifiers and literals
	IDENT  TokenType = "IDENT"  // x, foo, bar
	INT    TokenType = "INT"    // 123
	FLOAT  TokenType = "FLOAT"  // 3.14
	STRING TokenType = "STRING" // "hello"
	REGEX  TokenType = "REGEX"  // /pattern/flags

	// Operators
	ASSIGN   TokenType = "="
	PLUS     TokenType = "+"
	MINUS    TokenType = "-"
	ASTERISK TokenType = "*"
	SLASH    TokenType = "/"
	PERCENT  TokenType = "%"
	POWER    TokenType = "**"
	BANG     TokenType = "!"

	// Comparison
	EQ     TokenType = "=="
	NOT_EQ TokenType = "!="
	LT     TokenType = "<"
	GT     TokenType = ">"
	LT_EQ  TokenType = "<="
	GT_EQ  TokenType = ">="

	// Logical
	AND TokenType = "&&"
	OR  TokenType = "||"

	// Pipelines
	PIPE        TokenType = "|>"
	EFFECT_PIPE TokenType = "!>"
	EFFECT_END  TokenType = "!?"

	// Arrow
	ARROW TokenType = "=>"

	// Spread
	SPREAD TokenType = "..."

	// Delimiters
	COMMA     TokenType = ","
	SEMICOLON TokenType = ";"
	COLON     TokenType = ":"
	DOT       TokenType = "."
	PIPE_CHAR TokenType = "|"
	QUESTION  TokenType = "?"

	LPAREN   TokenType = "("
	RPAREN   TokenType = ")"
	LBRACKET TokenType = "["
	RBRACKET TokenType = "]"
	LBRACE   TokenType = "{"
	RBRACE   TokenType = "}"

	// Keywords
	FUNC       TokenType = "FUNC"
	FUNC_EFFECT TokenType = "FUNC!"
	MATCH      TokenType = "MATCH"
	CONSTRAINT TokenType = "CONSTRAINT"
	NAMESPACE  TokenType = "NAMESPACE"
	USE        TokenType = "USE"
	TRUE       TokenType = "TRUE"
	FALSE      TokenType = "FALSE"
	SUCCESS    TokenType = "SUCCESS"
	FAILURE    TokenType = "FAILURE"
	RETURN     TokenType = "RETURN"
	IN         TokenType = "IN"

	// Built-in function keywords
	MAP    TokenType = "MAP"
	FILTER TokenType = "FILTER"
	REDUCE TokenType = "REDUCE"
	HAS    TokenType = "HAS"
	KEYS   TokenType = "KEYS"
	VALUES TokenType = "VALUES"
	HASH   TokenType = "HASH"
	LEN    TokenType = "LEN"

	// Wildcard
	UNDERSCORE TokenType = "_"
)

var keywords = map[string]TokenType{
	"func":       FUNC,
	"func!":      FUNC_EFFECT,
	"match":      MATCH,
	"constraint": CONSTRAINT,
	"namespace":  NAMESPACE,
	"use":        USE,
	"true":       TRUE,
	"false":      FALSE,
	"success":    SUCCESS,
	"failure":    FAILURE,
	"return":     RETURN,
	"in":         IN,
	"map":        MAP,
	"filter":     FILTER,
	"reduce":     REDUCE,
	"has":        HAS,
	"keys":       KEYS,
	"values":     VALUES,
	"hash":       HASH,
	"len":        LEN,
}

// LookupIdent checks if the given identifier is a keyword
func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}

// String returns the string representation of the token type
func (t TokenType) String() string {
	return string(t)
}
