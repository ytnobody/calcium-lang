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

	// String interpolation tokens
	STRING_TEMPLATE_START  TokenType = "STRING_TEMPLATE_START"  // "...${
	STRING_TEMPLATE_MIDDLE TokenType = "STRING_TEMPLATE_MIDDLE" // }...${
	STRING_TEMPLATE_END    TokenType = "STRING_TEMPLATE_END"    // }..."

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
	PROP_PIPE   TokenType = "|>?" // Error propagation pipe: left |>? func
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
	FUNC        TokenType = "FUNC"
	FUNC_EFFECT TokenType = "FUNC!"
	MATCH       TokenType = "MATCH"
	CONSTRAINT  TokenType = "CONSTRAINT"
	NAMESPACE   TokenType = "NAMESPACE"
	USE         TokenType = "USE"
	TRUE        TokenType = "TRUE"
	FALSE       TokenType = "FALSE"
	SUCCESS     TokenType = "SUCCESS"
	FAILURE     TokenType = "FAILURE"
	RETURN      TokenType = "RETURN"
	IN          TokenType = "IN"
	DO          TokenType = "DO"
	END         TokenType = "END"

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
	"do":         DO,
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

// GetAllKeywords returns all keyword strings for spelling suggestions
func GetAllKeywords() []string {
	result := make([]string, 0, len(keywords))
	for k := range keywords {
		result = append(result, k)
	}
	return result
}

// levenshteinDistance calculates the edit distance between two strings
func levenshteinDistance(a, b string) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	matrix := make([][]int, len(a)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(b)+1)
		matrix[i][0] = i
	}
	for j := 0; j <= len(b); j++ {
		matrix[0][j] = j
	}

	for i := 1; i <= len(a); i++ {
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			del := matrix[i-1][j] + 1
			ins := matrix[i][j-1] + 1
			sub := matrix[i-1][j-1] + cost
			if del < ins {
				if del < sub {
					matrix[i][j] = del
				} else {
					matrix[i][j] = sub
				}
			} else {
				if ins < sub {
					matrix[i][j] = ins
				} else {
					matrix[i][j] = sub
				}
			}
		}
	}

	return matrix[len(a)][len(b)]
}

// SuggestKeyword finds a similar keyword for a misspelled identifier
// Returns empty string if no similar keyword found
func SuggestKeyword(ident string, maxDistance int) string {
	bestMatch := ""
	bestDistance := maxDistance + 1

	for keyword := range keywords {
		distance := levenshteinDistance(ident, keyword)
		if distance <= maxDistance && distance < bestDistance {
			bestDistance = distance
			bestMatch = keyword
		}
	}

	return bestMatch
}
