package lexer

import (
	"github.com/ytnobody/calcium-lang/pkg/token"
)

type Lexer struct {
	input        string
	position     int             // current position in input (points to current char)
	readPosition int             // current reading position in input (after current char)
	ch           byte            // current char under examination
	line         int             // current line number
	column       int             // current column number
	prevToken    token.TokenType // previous token type for regex/division disambiguation

	// String interpolation state
	inStringInterp   bool // true when inside a string interpolation
	interpBraceDepth int  // track nested braces inside ${...}
}

// New creates a new Lexer
func New(input string) *Lexer {
	l := &Lexer{input: input, line: 1, column: 0}
	l.readChar()
	return l
}

func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.readPosition]
	}
	l.position = l.readPosition
	l.readPosition++

	if l.ch == '\n' {
		l.line++
		l.column = 0
	} else {
		l.column++
	}
}

func (l *Lexer) peekChar() byte {
	if l.readPosition >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition]
}

func (l *Lexer) peekCharN(n int) byte {
	pos := l.readPosition + n - 1
	if pos >= len(l.input) {
		return 0
	}
	return l.input[pos]
}

// NextToken returns the next token from the input
func (l *Lexer) NextToken() token.Token {
	var tok token.Token

	l.skipWhitespaceAndComments()

	tok.Line = l.line
	tok.Column = l.column

	switch l.ch {
	case '=':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.EQ, Literal: string(ch) + string(l.ch), Line: tok.Line, Column: tok.Column}
		} else if l.peekChar() == '>' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.ARROW, Literal: string(ch) + string(l.ch), Line: tok.Line, Column: tok.Column}
		} else {
			tok = newToken(token.ASSIGN, l.ch, tok.Line, tok.Column)
		}
	case '+':
		tok = newToken(token.PLUS, l.ch, tok.Line, tok.Column)
	case '-':
		tok = newToken(token.MINUS, l.ch, tok.Line, tok.Column)
	case '*':
		if l.peekChar() == '*' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.POWER, Literal: string(ch) + string(l.ch), Line: tok.Line, Column: tok.Column}
		} else {
			tok = newToken(token.ASTERISK, l.ch, tok.Line, tok.Column)
		}
	case '/':
		if l.isRegexContext() {
			tok.Type = token.REGEX
			tok.Literal = l.readRegex()
			l.prevToken = tok.Type
			return tok
		}
		tok = newToken(token.SLASH, l.ch, tok.Line, tok.Column)
	case '%':
		tok = newToken(token.PERCENT, l.ch, tok.Line, tok.Column)
	case '!':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.NOT_EQ, Literal: string(ch) + string(l.ch), Line: tok.Line, Column: tok.Column}
		} else if l.peekChar() == '>' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.EFFECT_PIPE, Literal: string(ch) + string(l.ch), Line: tok.Line, Column: tok.Column}
		} else if l.peekChar() == '?' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.EFFECT_END, Literal: string(ch) + string(l.ch), Line: tok.Line, Column: tok.Column}
		} else {
			tok = newToken(token.BANG, l.ch, tok.Line, tok.Column)
		}
	case '<':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.LT_EQ, Literal: string(ch) + string(l.ch), Line: tok.Line, Column: tok.Column}
		} else {
			tok = newToken(token.LT, l.ch, tok.Line, tok.Column)
		}
	case '>':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.GT_EQ, Literal: string(ch) + string(l.ch), Line: tok.Line, Column: tok.Column}
		} else {
			tok = newToken(token.GT, l.ch, tok.Line, tok.Column)
		}
	case '&':
		if l.peekChar() == '&' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.AND, Literal: string(ch) + string(l.ch), Line: tok.Line, Column: tok.Column}
		} else {
			tok = newToken(token.ILLEGAL, l.ch, tok.Line, tok.Column)
		}
	case '|':
		if l.peekChar() == '|' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.OR, Literal: string(ch) + string(l.ch), Line: tok.Line, Column: tok.Column}
		} else if l.peekChar() == '>' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.PIPE, Literal: string(ch) + string(l.ch), Line: tok.Line, Column: tok.Column}
		} else {
			tok = newToken(token.PIPE_CHAR, l.ch, tok.Line, tok.Column)
		}
	case '.':
		if l.peekChar() == '.' && l.peekCharN(2) == '.' {
			l.readChar()
			l.readChar()
			tok = token.Token{Type: token.SPREAD, Literal: "...", Line: tok.Line, Column: tok.Column}
		} else {
			tok = newToken(token.DOT, l.ch, tok.Line, tok.Column)
		}
	case ',':
		tok = newToken(token.COMMA, l.ch, tok.Line, tok.Column)
	case ';':
		tok = newToken(token.SEMICOLON, l.ch, tok.Line, tok.Column)
	case ':':
		tok = newToken(token.COLON, l.ch, tok.Line, tok.Column)
	case '(':
		tok = newToken(token.LPAREN, l.ch, tok.Line, tok.Column)
	case ')':
		tok = newToken(token.RPAREN, l.ch, tok.Line, tok.Column)
	case '[':
		tok = newToken(token.LBRACKET, l.ch, tok.Line, tok.Column)
	case ']':
		tok = newToken(token.RBRACKET, l.ch, tok.Line, tok.Column)
	case '{':
		if l.inStringInterp {
			l.interpBraceDepth++
		}
		tok = newToken(token.LBRACE, l.ch, tok.Line, tok.Column)
	case '?':
		tok = newToken(token.QUESTION, l.ch, tok.Line, tok.Column)
	case '}':
		if l.inStringInterp && l.interpBraceDepth == 0 {
			// End of interpolation expression, continue reading string
			l.readChar() // consume '}'
			// Check what comes next
			stringPart, hasMore := l.readStringInterpolationPart()
			if hasMore {
				tok.Type = token.STRING_TEMPLATE_MIDDLE
				tok.Literal = stringPart
			} else {
				tok.Type = token.STRING_TEMPLATE_END
				tok.Literal = stringPart
				l.inStringInterp = false
			}
			l.prevToken = tok.Type
			return tok
		}
		if l.inStringInterp {
			l.interpBraceDepth--
		}
		tok = newToken(token.RBRACE, l.ch, tok.Line, tok.Column)
	case '_':
		if isLetter(l.peekChar()) || isDigit(l.peekChar()) {
			tok.Literal = l.readIdentifier()
			tok.Type = token.LookupIdent(tok.Literal)
			l.prevToken = tok.Type
			return tok
		}
		tok = newToken(token.UNDERSCORE, l.ch, tok.Line, tok.Column)
	case '"':
		// Check if this is an interpolated string or regular string
		stringContent, hasInterpolation := l.peekStringContent()
		if hasInterpolation {
			// Start of interpolated string: read until first ${
			tok.Type = token.STRING_TEMPLATE_START
			tok.Literal = l.readStringUntilInterpolation()
			l.prevToken = tok.Type
			l.inStringInterp = true
			l.interpBraceDepth = 0
			return tok
		}
		tok.Type = token.STRING
		tok.Literal = l.readString()
		tok.Line = tok.Line
		tok.Column = tok.Column
		l.prevToken = tok.Type
		_ = stringContent // unused, just for peeking
		return tok
	case 0:
		tok.Literal = ""
		tok.Type = token.EOF
	default:
		if isLetter(l.ch) {
			tok.Literal = l.readIdentifier()
			tok.Type = token.LookupIdent(tok.Literal)
			// Check for func! (identifier followed by !)
			if tok.Literal == "func" && l.ch == '!' {
				l.readChar()
				tok.Literal = "func!"
				tok.Type = token.FUNC_EFFECT
			}
			l.prevToken = tok.Type
			return tok
		} else if isDigit(l.ch) {
			tok.Literal, tok.Type = l.readNumber()
			l.prevToken = tok.Type
			return tok
		} else {
			tok = newToken(token.ILLEGAL, l.ch, tok.Line, tok.Column)
		}
	}

	l.readChar()
	l.prevToken = tok.Type
	return tok
}

func newToken(tokenType token.TokenType, ch byte, line, column int) token.Token {
	return token.Token{Type: tokenType, Literal: string(ch), Line: line, Column: column}
}

// isRegexContext determines if the current '/' should start a regex literal
// based on the previous token
func (l *Lexer) isRegexContext() bool {
	switch l.prevToken {
	case token.IDENT, token.INT, token.FLOAT, token.STRING, token.REGEX,
		token.RPAREN, token.RBRACKET, token.RBRACE,
		token.TRUE, token.FALSE, token.UNDERSCORE:
		// After a value, '/' is division
		return false
	default:
		// After operators, punctuation, keywords, or at start, '/' is regex
		return true
	}
}

// readRegex reads a regex literal /pattern/flags
func (l *Lexer) readRegex() string {
	startPos := l.position
	l.readChar() // consume opening '/'

	// Read pattern until unescaped '/'
	for {
		if l.ch == 0 {
			break
		}
		if l.ch == '\\' {
			l.readChar() // skip escape
			if l.ch != 0 {
				l.readChar() // skip escaped char
			}
			continue
		}
		if l.ch == '/' {
			break
		}
		l.readChar()
	}

	l.readChar() // consume closing '/'

	// Read flags (i, m, s, g)
	for l.ch == 'i' || l.ch == 'm' || l.ch == 's' || l.ch == 'g' {
		l.readChar()
	}

	return l.input[startPos:l.position]
}

func (l *Lexer) readIdentifier() string {
	position := l.position
	for isLetter(l.ch) || isDigit(l.ch) {
		l.readChar()
	}
	return l.input[position:l.position]
}

func (l *Lexer) readNumber() (string, token.TokenType) {
	position := l.position
	tokenType := token.INT

	// Handle hex (0x) and binary (0b)
	if l.ch == '0' {
		if l.peekChar() == 'x' || l.peekChar() == 'X' {
			l.readChar() // consume '0'
			l.readChar() // consume 'x'
			for isHexDigit(l.ch) || l.ch == '_' {
				l.readChar()
			}
			return l.input[position:l.position], token.INT
		} else if l.peekChar() == 'b' || l.peekChar() == 'B' {
			l.readChar() // consume '0'
			l.readChar() // consume 'b'
			for l.ch == '0' || l.ch == '1' || l.ch == '_' {
				l.readChar()
			}
			return l.input[position:l.position], token.INT
		}
	}

	// Read integer part
	for isDigit(l.ch) || l.ch == '_' {
		l.readChar()
	}

	// Check for decimal point
	if l.ch == '.' && isDigit(l.peekChar()) {
		tokenType = token.FLOAT
		l.readChar() // consume '.'
		for isDigit(l.ch) || l.ch == '_' {
			l.readChar()
		}
	}

	// Check for exponent (scientific notation)
	if l.ch == 'e' || l.ch == 'E' {
		tokenType = token.FLOAT
		l.readChar() // consume 'e'
		if l.ch == '+' || l.ch == '-' {
			l.readChar()
		}
		for isDigit(l.ch) {
			l.readChar()
		}
	}

	return l.input[position:l.position], tokenType
}

func (l *Lexer) readString() string {
	position := l.position + 1 // skip opening quote
	for {
		l.readChar()
		if l.ch == '"' || l.ch == 0 {
			break
		}
		if l.ch == '\\' {
			l.readChar() // skip escaped character
		}
	}
	result := l.input[position:l.position]
	l.readChar() // consume closing quote
	return processEscapeSequences(result)
}

// processEscapeSequences converts escape sequences in a string to their actual characters
func processEscapeSequences(s string) string {
	var result []byte
	i := 0
	for i < len(s) {
		if s[i] == '\\' && i+1 < len(s) {
			switch s[i+1] {
			case 'n':
				result = append(result, '\n')
				i += 2
			case 'r':
				result = append(result, '\r')
				i += 2
			case 't':
				result = append(result, '\t')
				i += 2
			case '\\':
				result = append(result, '\\')
				i += 2
			case '"':
				result = append(result, '"')
				i += 2
			case '\'':
				result = append(result, '\'')
				i += 2
			default:
				// Unknown escape, keep as-is
				result = append(result, s[i])
				i++
			}
		} else {
			result = append(result, s[i])
			i++
		}
	}
	return string(result)
}

// peekStringContent checks if a string contains interpolation markers
func (l *Lexer) peekStringContent() (string, bool) {
	pos := l.position + 1 // skip opening quote
	hasInterpolation := false
	for pos < len(l.input) {
		if l.input[pos] == '"' {
			break
		}
		if l.input[pos] == '\\' {
			pos += 2 // skip escape sequence
			continue
		}
		if l.input[pos] == '$' && pos+1 < len(l.input) && l.input[pos+1] == '{' {
			hasInterpolation = true
			break
		}
		pos++
	}
	return l.input[l.position+1 : pos], hasInterpolation
}

// readStringUntilInterpolation reads a string until the first ${
func (l *Lexer) readStringUntilInterpolation() string {
	l.readChar() // skip opening quote
	position := l.position
	for {
		if l.ch == 0 || l.ch == '"' {
			break
		}
		if l.ch == '\\' {
			l.readChar() // skip escape char
			l.readChar() // skip escaped char
			continue
		}
		if l.ch == '$' && l.peekChar() == '{' {
			// Found ${, stop here
			result := l.input[position:l.position]
			l.readChar() // consume '$'
			l.readChar() // consume '{'
			return processEscapeSequences(result)
		}
		l.readChar()
	}
	// Should not reach here in normal cases
	return processEscapeSequences(l.input[position:l.position])
}

// readStringInterpolationPart reads the string part after } in an interpolation
// Returns the string content and whether there are more interpolations
func (l *Lexer) readStringInterpolationPart() (string, bool) {
	position := l.position
	for {
		if l.ch == 0 {
			break
		}
		if l.ch == '"' {
			// End of string
			result := l.input[position:l.position]
			l.readChar() // consume closing quote
			return processEscapeSequences(result), false
		}
		if l.ch == '\\' {
			l.readChar() // skip escape char
			l.readChar() // skip escaped char
			continue
		}
		if l.ch == '$' && l.peekChar() == '{' {
			// Another interpolation
			result := l.input[position:l.position]
			l.readChar() // consume '$'
			l.readChar() // consume '{'
			return processEscapeSequences(result), true
		}
		l.readChar()
	}
	return processEscapeSequences(l.input[position:l.position]), false
}

func (l *Lexer) skipWhitespaceAndComments() {
	for {
		// Skip whitespace
		for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
			l.readChar()
		}

		// Check for comments
		if l.ch == '/' {
			if l.peekChar() == '/' {
				// Single-line comment - skip to end of line
				for l.ch != '\n' && l.ch != 0 {
					l.readChar()
				}
				continue // Check for more whitespace/comments
			} else if l.peekChar() == '*' {
				// Multi-line comment
				l.readChar() // consume '/'
				l.readChar() // consume '*'
				for {
					if l.ch == 0 {
						break
					}
					if l.ch == '*' && l.peekChar() == '/' {
						l.readChar() // consume '*'
						l.readChar() // consume '/'
						break
					}
					l.readChar()
				}
				continue // Check for more whitespace/comments
			}
		}

		// No more whitespace or comments
		break
	}
}

func isLetter(ch byte) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_'
}

func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}

func isHexDigit(ch byte) bool {
	return isDigit(ch) || 'a' <= ch && ch <= 'f' || 'A' <= ch && ch <= 'F'
}

// Input returns the source input string
func (l *Lexer) Input() string {
	return l.input
}
