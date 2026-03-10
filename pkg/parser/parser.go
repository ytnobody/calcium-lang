package parser

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ytnobody/calcium-lang/pkg/ast"
	"github.com/ytnobody/calcium-lang/pkg/lexer"
	"github.com/ytnobody/calcium-lang/pkg/token"
	"github.com/ytnobody/calcium-lang/pkg/types"
)

// Precedence levels
const (
	_ int = iota
	LOWEST
	LAMBDA      // =>
	PIPE        // |> !>
	OR          // ||
	AND         // &&
	EQUALS      // == !=
	LESSGREATER // < > <= >=
	SUM         // + -
	PRODUCT     // * / %
	POWER       // **
	PREFIX      // -x !x
	CALL        // func(x)
	INDEX       // array[index]
	MEMBER      // obj.prop
	SPREAD      // expr...
)

var precedences = map[token.TokenType]int{
	token.PIPE:        PIPE,
	token.PROP_PIPE:   PIPE, // |>? for error propagation
	token.EFFECT_PIPE: PIPE,
	token.EFFECT_END:  PIPE, // !? for result handling
	token.OR:          OR,
	token.AND:         AND,
	token.EQ:          EQUALS,
	token.NOT_EQ:      EQUALS,
	token.LT:          LESSGREATER,
	token.GT:          LESSGREATER,
	token.LT_EQ:       LESSGREATER,
	token.GT_EQ:       LESSGREATER,
	token.PLUS:        SUM,
	token.MINUS:       SUM,
	token.ASTERISK:    PRODUCT,
	token.SLASH:       PRODUCT,
	token.PERCENT:     PRODUCT,
	token.POWER:       POWER,
	token.LPAREN:      CALL,
	token.LBRACKET:    INDEX,
	token.DOT:         MEMBER,
	token.SPREAD:      SPREAD,
	token.QUESTION:    SPREAD, // ? for constraint check (same precedence as spread)
	token.ARROW:       LAMBDA, // => for lambda
	token.TILDE:       EQUALS, // ~ for regex match
}

type (
	prefixParseFn func() ast.Expression
	infixParseFn  func(ast.Expression) ast.Expression
)

// Parser parses tokens into an AST
type Parser struct {
	l      *lexer.Lexer
	input  string // Source input for error messages
	errors []string

	curToken  token.Token
	peekToken token.Token

	prefixParseFns map[token.TokenType]prefixParseFn
	infixParseFns  map[token.TokenType]infixParseFn

	lookahead []token.Token // Buffer for lookahead tokens (used in destructuring detection)
}

// New creates a new Parser
func New(l *lexer.Lexer) *Parser {
	p := &Parser{
		l:      l,
		input:  l.Input(),
		errors: []string{},
	}

	p.prefixParseFns = make(map[token.TokenType]prefixParseFn)
	p.registerPrefix(token.IDENT, p.parseIdentifier)
	p.registerPrefix(token.INT, p.parseIntegerLiteral)
	p.registerPrefix(token.FLOAT, p.parseFloatLiteral)
	p.registerPrefix(token.STRING, p.parseStringLiteral)
	p.registerPrefix(token.STRING_TEMPLATE_START, p.parseInterpolatedString)
	p.registerPrefix(token.TRUE, p.parseBooleanLiteral)
	p.registerPrefix(token.FALSE, p.parseBooleanLiteral)
	p.registerPrefix(token.BANG, p.parsePrefixExpression)
	p.registerPrefix(token.MINUS, p.parsePrefixExpression)
	p.registerPrefix(token.LPAREN, p.parseGroupedOrLambda)
	p.registerPrefix(token.LBRACKET, p.parseArrayLiteral)
	p.registerPrefix(token.LBRACE, p.parseHashLiteral)
	p.registerPrefix(token.MATCH, p.parseMatchExpression)
	p.registerPrefix(token.UNDERSCORE, p.parseWildcard)
	p.registerPrefix(token.SUCCESS, p.parseSuccessExpression)
	p.registerPrefix(token.FAILURE, p.parseFailureExpression)
	p.registerPrefix(token.RETURN, p.parseReturnExpression)
	p.registerPrefix(token.DO, p.parseDoExpression)
	// Built-in functions as identifiers
	p.registerPrefix(token.MAP, p.parseBuiltinAsIdentifier)
	p.registerPrefix(token.FILTER, p.parseBuiltinAsIdentifier)
	p.registerPrefix(token.REDUCE, p.parseBuiltinAsIdentifier)
	p.registerPrefix(token.HAS, p.parseBuiltinAsIdentifier)
	p.registerPrefix(token.KEYS, p.parseBuiltinAsIdentifier)
	p.registerPrefix(token.VALUES, p.parseBuiltinAsIdentifier)
	p.registerPrefix(token.HASH, p.parseBuiltinAsIdentifier)
	p.registerPrefix(token.LEN, p.parseBuiltinAsIdentifier)
	p.registerPrefix(token.REGEX, p.parseRegexLiteral)

	p.infixParseFns = make(map[token.TokenType]infixParseFn)
	p.registerInfix(token.PLUS, p.parseInfixExpression)
	p.registerInfix(token.MINUS, p.parseInfixExpression)
	p.registerInfix(token.ASTERISK, p.parseInfixExpression)
	p.registerInfix(token.SLASH, p.parseInfixExpression)
	p.registerInfix(token.PERCENT, p.parseInfixExpression)
	p.registerInfix(token.POWER, p.parseInfixExpression)
	p.registerInfix(token.EQ, p.parseInfixExpression)
	p.registerInfix(token.NOT_EQ, p.parseInfixExpression)
	p.registerInfix(token.LT, p.parseInfixExpression)
	p.registerInfix(token.GT, p.parseInfixExpression)
	p.registerInfix(token.LT_EQ, p.parseInfixExpression)
	p.registerInfix(token.GT_EQ, p.parseInfixExpression)
	p.registerInfix(token.AND, p.parseInfixExpression)
	p.registerInfix(token.OR, p.parseInfixExpression)
	p.registerInfix(token.LPAREN, p.parseCallExpression)
	p.registerInfix(token.LBRACKET, p.parseIndexExpression)
	p.registerInfix(token.DOT, p.parseMemberExpression)
	p.registerInfix(token.PIPE, p.parsePipeExpression)
	p.registerInfix(token.PROP_PIPE, p.parseErrorPropPipeExpression)
	p.registerInfix(token.EFFECT_PIPE, p.parseEffectPipeExpression)
	p.registerInfix(token.SPREAD, p.parseSpreadExpression)
	p.registerInfix(token.QUESTION, p.parseConstraintCheckExpression)
	p.registerInfix(token.EFFECT_END, p.parseEffectHandleExpression)
	p.registerInfix(token.ARROW, p.parseLambdaFromIdent)
	p.registerInfix(token.TILDE, p.parseInfixExpression)

	// Read two tokens, so curToken and peekToken are both set
	p.nextToken()
	p.nextToken()

	return p
}

func (p *Parser) registerPrefix(tokenType token.TokenType, fn prefixParseFn) {
	p.prefixParseFns[tokenType] = fn
}

func (p *Parser) registerInfix(tokenType token.TokenType, fn infixParseFn) {
	p.infixParseFns[tokenType] = fn
}

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	// Check lookahead buffer first
	if len(p.lookahead) > 0 {
		p.peekToken = p.lookahead[0]
		p.lookahead = p.lookahead[1:]
	} else {
		p.peekToken = p.l.NextToken()
	}
}

// Errors returns the list of parsing errors
func (p *Parser) Errors() []string {
	return p.errors
}

// getSourceLine returns the source line at the given line number (1-indexed)
func (p *Parser) getSourceLine(line int) string {
	lines := strings.Split(p.input, "\n")
	if line < 1 || line > len(lines) {
		return ""
	}
	return lines[line-1]
}

// formatErrorWithContext creates a detailed error message with source context
func (p *Parser) formatErrorWithContext(line, column int, message string) string {
	sourceLine := p.getSourceLine(line)
	if sourceLine == "" {
		return fmt.Sprintf("line %d, column %d: %s", line, column, message)
	}

	// Build caret indicator with underline
	caretLine := strings.Repeat(" ", column-1) + "^"

	return fmt.Sprintf("line %d, column %d:\n  %s\n  %s\n%s", line, column, sourceLine, caretLine, message)
}

// formatErrorWithHint creates a detailed error message with source context and a help hint
func (p *Parser) formatErrorWithHint(line, column int, message, hint string) string {
	base := p.formatErrorWithContext(line, column, message)
	if hint != "" {
		return base + "\nhint: " + hint
	}
	return base
}

func (p *Parser) peekError(t token.TokenType) {
	hint := p.getExpectationHint(t, p.peekToken)
	msg := p.formatErrorWithHint(
		p.peekToken.Line,
		p.peekToken.Column,
		fmt.Sprintf("expected '%s' but found '%s'", t, p.peekToken.Type),
		hint,
	)
	p.errors = append(p.errors, msg)
}

// getExpectationHint returns a helpful hint based on what was expected vs what was found
func (p *Parser) getExpectationHint(expected token.TokenType, found token.Token) string {
	switch expected {
	case token.RPAREN:
		return "you may have an unclosed '(' — check for matching parentheses"
	case token.RBRACKET:
		return "you may have an unclosed '[' — check for matching brackets"
	case token.RBRACE:
		return "you may have an unclosed '{' — check for matching braces"
	case token.ASSIGN:
		if found.Type == token.EQ {
			return "use '=' for assignment, not '==' (which is comparison)"
		}
		return "variable binding requires '=' after the name"
	case token.LBRACE:
		if found.Type == token.IDENT {
			return "a block '{' ... '}' is expected here"
		}
	case token.ARROW:
		return "lambda syntax requires '=>' (e.g., (x) => x + 1)"
	}
	return ""
}

func (p *Parser) noPrefixParseFnError(t token.TokenType) {
	errMsg := fmt.Sprintf("unexpected token '%s'", t)
	hint := ""

	switch t {
	case token.IDENT:
		// Check for keyword typo
		if suggestion := token.SuggestKeyword(p.curToken.Literal, 2); suggestion != "" {
			errMsg += fmt.Sprintf(" (did you mean '%s'?)", suggestion)
		}
	case token.ASSIGN:
		hint = "assignment '=' cannot appear here — did you forget a variable name?"
	case token.RBRACE:
		hint = "unexpected '}' — you may have an extra closing brace"
	case token.RBRACKET:
		hint = "unexpected ']' — you may have an extra closing bracket"
	case token.RPAREN:
		hint = "unexpected ')' — you may have an extra closing parenthesis"
	case token.COMMA:
		hint = "unexpected ',' — check for a missing expression before the comma"
	case token.SEMICOLON:
		hint = "unexpected ';' — there may be an empty statement"
	case token.EOF:
		errMsg = "unexpected end of input"
		hint = "the expression or statement may be incomplete"
	}

	msg := p.formatErrorWithHint(
		p.curToken.Line,
		p.curToken.Column,
		errMsg,
		hint,
	)
	p.errors = append(p.errors, msg)
}

func (p *Parser) curTokenIs(t token.TokenType) bool {
	return p.curToken.Type == t
}

func (p *Parser) peekTokenIs(t token.TokenType) bool {
	return p.peekToken.Type == t
}

// isEndToken checks if a token is the contextual "end" keyword.
// "end" is treated as a contextual keyword (not a reserved word) so it can still
// be used as an identifier (e.g., parameter names like "start, end") outside do blocks.
func (p *Parser) isEndToken(tok token.Token) bool {
	return (tok.Type == token.END) || (tok.Type == token.IDENT && tok.Literal == "end")
}

func (p *Parser) expectPeek(t token.TokenType) bool {
	if p.peekTokenIs(t) {
		p.nextToken()
		return true
	}
	p.peekError(t)
	return false
}

func (p *Parser) peekPrecedence() int {
	if p, ok := precedences[p.peekToken.Type]; ok {
		return p
	}
	return LOWEST
}

func (p *Parser) curPrecedence() int {
	if p, ok := precedences[p.curToken.Type]; ok {
		return p
	}
	return LOWEST
}

// ParseProgram parses the entire program
func (p *Parser) ParseProgram() *ast.Program {
	program := &ast.Program{}
	program.Statements = []ast.Statement{}

	for !p.curTokenIs(token.EOF) {
		stmt := p.parseStatement()
		if stmt != nil {
			program.Statements = append(program.Statements, stmt)
		}
		p.nextToken()
	}

	return program
}

func (p *Parser) parseStatement() ast.Statement {
	switch p.curToken.Type {
	case token.FUNC, token.FUNC_EFFECT:
		return p.parseFunctionDeclaration()
	case token.CONSTRAINT:
		return p.parseConstraintDeclaration()
	case token.NAMESPACE:
		return p.parseNamespaceDeclaration()
	case token.USE:
		return p.parseUseStatement()
	case token.TYPE:
		return p.parseTypeDeclaration()
	case token.IDENT:
		// Check if this is an assignment: ident = expr
		if p.peekTokenIs(token.ASSIGN) {
			return p.parseAssignmentStatement()
		}
		// Check if this is a typed assignment: ident: TypeName = expr
		if p.peekTokenIs(token.COLON) {
			return p.parseTypedAssignmentStatement()
		}
		return p.parseExpressionStatement()
	case token.LBRACKET:
		// Could be array destructuring or array literal expression
		if p.isDestructuringPattern() {
			return p.parseDestructuringStatement()
		}
		return p.parseExpressionStatement()
	default:
		return p.parseExpressionStatement()
	}
}

// isDestructuringPattern checks if current position is an array destructuring pattern
// Pattern: [ ident, ident, ... ] = or [ ident | ident ] =
// We need to distinguish from array literals like [a b c] or [a, b, c]
func (p *Parser) isDestructuringPattern() bool {
	if !p.curTokenIs(token.LBRACKET) {
		return false
	}

	// Peek at next token - should be an identifier for destructuring
	if !p.peekTokenIs(token.IDENT) {
		return false
	}

	// Scan ahead to determine if this matches destructuring pattern
	// We need to find: [ IDENT (,|) ... ] =
	// Use a temporary scan of the lexer without consuming tokens

	// Save tokens we'll scan
	tokens := []token.Token{p.curToken, p.peekToken}

	// Scan ahead using the lexer directly
	for {
		tok := p.l.NextToken()
		tokens = append(tokens, tok)

		// Check for pattern completion
		if tok.Type == token.RBRACKET {
			// Check if next is ASSIGN
			next := p.l.NextToken()
			tokens = append(tokens, next)

			// Restore: we need to "unread" these tokens
			// Since we can't unread, we'll store them in a buffer
			// For now, use a simpler approach: check the pattern we found

			isDestructuring := next.Type == token.ASSIGN

			// We need to make the lexer available for these tokens
			// Store them in parser's lookahead buffer
			p.lookahead = tokens[2:] // Skip curToken and peekToken which are already stored
			return isDestructuring
		}

		// Valid destructuring pattern continues with COMMA or PIPE_CHAR
		if tok.Type == token.COMMA || tok.Type == token.PIPE_CHAR {
			// Continue scanning - need to find IDENT after
			continue
		}

		if tok.Type == token.IDENT {
			// Continue scanning
			continue
		}

		// Invalid token for destructuring - it's an expression
		p.lookahead = tokens[2:]
		return false
	}
}

func (p *Parser) parseDestructuringStatement() ast.Statement {
	tok := p.curToken // the '[' token

	p.nextToken() // consume '['

	// First identifier
	if !p.curTokenIs(token.IDENT) {
		// Not actually destructuring, this shouldn't happen if isDestructuringPattern was correct
		p.errors = append(p.errors, p.formatErrorWithContext(
			p.curToken.Line, p.curToken.Column,
			"expected identifier in destructuring pattern"))
		return nil
	}

	firstName := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	// Check for head|tail pattern: [ head | tail ]
	if p.peekTokenIs(token.PIPE_CHAR) {
		p.nextToken() // consume first ident
		p.nextToken() // consume '|'

		if !p.curTokenIs(token.IDENT) {
			p.errors = append(p.errors, p.formatErrorWithContext(
				p.curToken.Line, p.curToken.Column,
				"expected identifier after '|' in destructuring"))
			return nil
		}

		tailName := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

		if !p.expectPeek(token.RBRACKET) {
			return nil
		}

		if !p.expectPeek(token.ASSIGN) {
			return nil
		}

		p.nextToken() // move to value expression
		value := p.parseExpression(LOWEST)

		if p.peekTokenIs(token.SEMICOLON) {
			p.nextToken()
		}

		return &ast.HeadTailDestructuringStatement{
			Token: tok,
			Head:  firstName,
			Tail:  tailName,
			Value: value,
		}
	}

	// Otherwise, it's array destructuring: [ a, b, c ]
	names := []*ast.Identifier{firstName}

	for p.peekTokenIs(token.COMMA) {
		p.nextToken() // consume current ident
		p.nextToken() // consume ','

		if !p.curTokenIs(token.IDENT) {
			p.errors = append(p.errors, p.formatErrorWithContext(
				p.curToken.Line, p.curToken.Column,
				"expected identifier in destructuring pattern"))
			return nil
		}

		names = append(names, &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal})
	}

	if !p.expectPeek(token.RBRACKET) {
		return nil
	}

	if !p.expectPeek(token.ASSIGN) {
		// This wasn't a destructuring pattern after all
		// This is an edge case - the pattern looked like destructuring but isn't
		// For now, report an error
		p.errors = append(p.errors, p.formatErrorWithContext(
			p.curToken.Line, p.curToken.Column,
			"expected '=' after destructuring pattern"))
		return nil
	}

	p.nextToken() // move to value expression
	value := p.parseExpression(LOWEST)

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	return &ast.ArrayDestructuringStatement{
		Token: tok,
		Names: names,
		Value: value,
	}
}

func (p *Parser) parseExpressionStatement() *ast.ExpressionStatement {
	stmt := &ast.ExpressionStatement{Token: p.curToken}
	stmt.Expression = p.parseExpression(LOWEST)

	// Handle !? error handling after an expression
	if p.peekTokenIs(token.EFFECT_END) {
		p.nextToken()
		stmt.Expression = p.parseEffectHandleExpression(stmt.Expression)
	}

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseAssignmentStatement() *ast.AssignmentStatement {
	stmt := &ast.AssignmentStatement{Token: p.curToken}
	stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(token.ASSIGN) {
		return nil
	}

	p.nextToken()
	stmt.Value = p.parseExpression(LOWEST)

	// Handle !? error handling
	if p.peekTokenIs(token.EFFECT_END) {
		p.nextToken()
		stmt.Value = p.parseEffectHandleExpression(stmt.Value)
	}

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

// parseTypedAssignmentStatement parses: ident: TypeName = expr
// This is a type-annotated variable binding introduced by gradual typing.
// If the token after ':' is not a built-in type name, it falls back to a
// plain expression statement (preserving backward compatibility with constraint
// notation in other contexts).
func (p *Parser) parseTypedAssignmentStatement() *ast.AssignmentStatement {
	stmt := &ast.AssignmentStatement{Token: p.curToken}
	stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	// consume ':'
	p.nextToken()

	// get type name
	p.nextToken()
	typeTok := p.curToken
	typeName := typeTok.Literal

	// Only accept known built-in type names; otherwise treat as expression statement
	if !types.IsBuiltinTypeName(typeName) {
		// Not a type annotation — fall back to expression statement
		// We can't easily "unread" tokens so we report an error.
		p.errors = append(p.errors, p.formatErrorWithContext(
			typeTok.Line, typeTok.Column,
			fmt.Sprintf("unknown type name %q in type annotation; expected a built-in type (Int, Float, String, Bool, Null, Array, Hash, Tuple, Func, Regex, Any, Success, Failure)", typeName),
		))
		return nil
	}

	stmt.TypeAnnot = &ast.TypeAnnotation{Token: typeTok, Name: typeName}

	if !p.expectPeek(token.ASSIGN) {
		return nil
	}

	p.nextToken()
	stmt.Value = p.parseExpression(LOWEST)

	// Handle !? error handling
	if p.peekTokenIs(token.EFFECT_END) {
		p.nextToken()
		stmt.Value = p.parseEffectHandleExpression(stmt.Value)
	}

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseFunctionDeclaration() *ast.FunctionDeclaration {
	stmt := &ast.FunctionDeclaration{Token: p.curToken}
	stmt.IsEffect = p.curToken.Type == token.FUNC_EFFECT

	if !p.expectPeek(token.IDENT) {
		return nil
	}

	stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(token.LPAREN) {
		return nil
	}

	stmt.Parameters, stmt.Constraints, stmt.ParamTypes = p.parseFunctionParametersWithConstraints()

	// Optional return type/constraint annotation: ): ReturnType =
	// Syntax: func add(a: Int, b: Int): Int = a + b
	// With constraint: func add(a: Positive, b: Positive): Positive = a + b
	if p.peekTokenIs(token.COLON) {
		p.nextToken() // consume ':'
		p.nextToken() // get return type/constraint name
		retTypeTok := p.curToken
		retTypeName := retTypeTok.Literal

		// If it's a known built-in type name and NOT followed by '?', treat as type annotation.
		if types.IsBuiltinTypeName(retTypeName) && !p.peekTokenIs(token.QUESTION) {
			stmt.ReturnType = &ast.TypeAnnotation{Token: retTypeTok, Name: retTypeName}
		} else {
			// Otherwise treat as a return constraint (user-defined constraint name)
			stmt.ReturnConstraint = &ast.Identifier{Token: retTypeTok, Value: retTypeName}
			if p.peekTokenIs(token.QUESTION) {
				p.nextToken() // consume '?'
			}
		}
	}

	if !p.expectPeek(token.ASSIGN) {
		return nil
	}

	p.nextToken()
	stmt.Body = p.parseExpression(LOWEST)

	// Handle !? error handling in function body
	if p.peekTokenIs(token.EFFECT_END) {
		p.nextToken()
		stmt.Body = p.parseEffectHandleExpression(stmt.Body)
	}

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseFunctionParameters() []*ast.Identifier {
	params, _, _ := p.parseFunctionParametersWithConstraints()
	return params
}

// parseFunctionParametersWithConstraints parses function parameters, returning:
//   - the list of parameter identifiers
//   - the list of constraint annotations (nil element = no constraint for that param)
//   - the list of type annotations (nil element = no type annotation for that param)
//
// Each parameter may carry a type annotation OR a constraint (not both):
//   - param: Int          → type annotation (built-in type name, no trailing '?')
//   - param: Positive?    → constraint (trailing '?')
//   - param: Positive     → constraint (user-defined name without '?', backward compat)
func (p *Parser) parseFunctionParametersWithConstraints() ([]*ast.Identifier, []*ast.Identifier, []*ast.TypeAnnotation) {
	identifiers := []*ast.Identifier{}
	constraints := []*ast.Identifier{}
	typeAnnots := []*ast.TypeAnnotation{}

	if p.peekTokenIs(token.RPAREN) {
		p.nextToken()
		return identifiers, constraints, typeAnnots
	}

	p.nextToken()

	ident, constraint, typeAnnot := p.parseParameterWithConstraint()
	identifiers = append(identifiers, ident)
	constraints = append(constraints, constraint)
	typeAnnots = append(typeAnnots, typeAnnot)

	for p.peekTokenIs(token.COMMA) {
		p.nextToken()
		p.nextToken()
		ident, constraint, typeAnnot := p.parseParameterWithConstraint()
		identifiers = append(identifiers, ident)
		constraints = append(constraints, constraint)
		typeAnnots = append(typeAnnots, typeAnnot)
	}

	if !p.expectPeek(token.RPAREN) {
		return nil, nil, nil
	}

	return identifiers, constraints, typeAnnots
}

// parseParameterWithConstraint parses a single parameter, which may have:
//   - a type annotation:  param: Int
//   - a constraint:       param: Positive?  or  param: Positive
//   - nothing:            param
func (p *Parser) parseParameterWithConstraint() (*ast.Identifier, *ast.Identifier, *ast.TypeAnnotation) {
	ident := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	// Check for annotation after ':'
	if p.peekTokenIs(token.COLON) {
		p.nextToken() // consume ':'
		p.nextToken() // get annotation name

		annotTok := p.curToken
		annotName := annotTok.Literal

		// If it's a known built-in type name and NOT followed by '?', treat as type annotation.
		if types.IsBuiltinTypeName(annotName) && !p.peekTokenIs(token.QUESTION) {
			typeAnnot := &ast.TypeAnnotation{Token: annotTok, Name: annotName}
			return ident, nil, typeAnnot
		}

		// Otherwise treat as a constraint (legacy behaviour)
		constraint := &ast.Identifier{Token: annotTok, Value: annotName}
		if p.peekTokenIs(token.QUESTION) {
			p.nextToken() // consume '?'
		}
		return ident, constraint, nil
	}

	return ident, nil, nil
}

func (p *Parser) parseConstraintDeclaration() *ast.ConstraintDeclaration {
	stmt := &ast.ConstraintDeclaration{Token: p.curToken}

	if !p.expectPeek(token.IDENT) {
		return nil
	}

	stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(token.LPAREN) {
		return nil
	}

	stmt.Parameters = p.parseFunctionParameters()

	if !p.expectPeek(token.ASSIGN) {
		return nil
	}

	p.nextToken()
	stmt.Body = p.parseExpression(LOWEST)

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseTypeDeclaration() *ast.TypeDeclaration {
	stmt := &ast.TypeDeclaration{Token: p.curToken}

	// Expect type name (must start with uppercase by convention)
	if !p.expectPeek(token.IDENT) {
		return nil
	}
	stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	// Expect =
	if !p.expectPeek(token.ASSIGN) {
		return nil
	}

	// Parse variants: Variant1(field1, field2) | Variant2 | Variant3(field)
	stmt.Variants = []*ast.VariantDef{}
	for {
		p.nextToken()
		variant := &ast.VariantDef{}

		if !p.curTokenIs(token.IDENT) {
			p.errors = append(p.errors, p.formatErrorWithHint(
				p.curToken.Line, p.curToken.Column,
				fmt.Sprintf("expected variant name, got '%s'", p.curToken.Type),
				"type variants must start with a name (e.g., type Color = Red | Green | Blue)",
			))
			return nil
		}
		variant.Name = p.curToken.Literal

		// Check if variant has fields: Variant(field1, field2)
		if p.peekTokenIs(token.LPAREN) {
			p.nextToken() // consume (
			variant.Fields = []string{}
			if !p.peekTokenIs(token.RPAREN) {
				p.nextToken()
				variant.Fields = append(variant.Fields, p.curToken.Literal)
				for p.peekTokenIs(token.COMMA) {
					p.nextToken() // consume ,
					p.nextToken() // move to next field
					variant.Fields = append(variant.Fields, p.curToken.Literal)
				}
			}
			if !p.expectPeek(token.RPAREN) {
				return nil
			}
		}

		stmt.Variants = append(stmt.Variants, variant)

		// Check for | (more variants) or end
		if p.peekTokenIs(token.PIPE_CHAR) {
			p.nextToken() // consume |
		} else {
			break
		}
	}

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseNamespaceDeclaration() *ast.NamespaceDeclaration {
	stmt := &ast.NamespaceDeclaration{Token: p.curToken}

	p.nextToken()
	stmt.Name = p.parseModulePath()

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseUseStatement() *ast.UseStatement {
	stmt := &ast.UseStatement{Token: p.curToken}

	p.nextToken()

	// Case 1: String literal -> URL format
	// use "github.com/author/repo";
	if p.curTokenIs(token.STRING) {
		stmt.Path = &ast.ModulePath{
			Token:    p.curToken,
			Parts:    []string{p.curToken.Literal},
			IsRemote: true,
			RawURL:   p.curToken.Literal,
		}
	} else if p.curTokenIs(token.IDENT) && p.peekTokenIs(token.SLASH) {
		// Case 2: IDENT/IDENT -> author/module format
		// use ytnobody/json;
		stmt.Path = p.parseRemoteModulePath()
	} else {
		// Case 3: IDENT.IDENT -> local/stdlib format
		// use core.io;
		stmt.Path = p.parseModulePath()
	}

	// Check for trailing ! (effect module)
	if p.peekTokenIs(token.BANG) {
		p.nextToken()
		stmt.IsEffect = true
	}

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

// parseRemoteModulePath parses author/module format
func (p *Parser) parseRemoteModulePath() *ast.ModulePath {
	path := &ast.ModulePath{Token: p.curToken, IsRemote: true}
	path.Author = p.curToken.Literal // "ytnobody"

	p.nextToken()                  // consume '/'
	p.nextToken()                  // move to module name
	path.Name = p.curToken.Literal // "json"
	path.Parts = []string{path.Author, path.Name}

	return path
}

func (p *Parser) parseModulePath() *ast.ModulePath {
	path := &ast.ModulePath{Token: p.curToken}
	path.Parts = []string{p.curToken.Literal}

	for p.peekTokenIs(token.DOT) {
		p.nextToken() // consume '.'
		p.nextToken() // consume next identifier
		path.Parts = append(path.Parts, p.curToken.Literal)
	}

	return path
}

func (p *Parser) parseExpression(precedence int) ast.Expression {
	prefix := p.prefixParseFns[p.curToken.Type]
	if prefix == nil {
		p.noPrefixParseFnError(p.curToken.Type)
		return nil
	}
	leftExp := prefix()

	for !p.peekTokenIs(token.SEMICOLON) && precedence < p.peekPrecedence() {
		infix := p.infixParseFns[p.peekToken.Type]
		if infix == nil {
			return leftExp
		}

		p.nextToken()
		leftExp = infix(leftExp)
	}

	return leftExp
}

func (p *Parser) parseIdentifier() ast.Expression {
	return &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
}

func (p *Parser) parseBuiltinAsIdentifier() ast.Expression {
	// Treat built-in keywords (map, filter, etc.) as identifiers
	return &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
}

func (p *Parser) parseIntegerLiteral() ast.Expression {
	lit := &ast.IntegerLiteral{Token: p.curToken}

	literal := p.curToken.Literal
	var value int64
	var err error

	// Handle different bases
	if len(literal) > 2 {
		switch literal[0:2] {
		case "0x", "0X":
			value, err = strconv.ParseInt(literal[2:], 16, 64)
		case "0b", "0B":
			value, err = strconv.ParseInt(literal[2:], 2, 64)
		default:
			value, err = strconv.ParseInt(literal, 10, 64)
		}
	} else {
		value, err = strconv.ParseInt(literal, 10, 64)
	}

	if err != nil {
		p.errors = append(p.errors, p.formatErrorWithContext(
			p.curToken.Line, p.curToken.Column,
			fmt.Sprintf("could not parse %q as integer", literal)))
		return nil
	}

	lit.Value = value
	return lit
}

func (p *Parser) parseFloatLiteral() ast.Expression {
	lit := &ast.FloatLiteral{Token: p.curToken}

	value, err := strconv.ParseFloat(p.curToken.Literal, 64)
	if err != nil {
		p.errors = append(p.errors, p.formatErrorWithContext(
			p.curToken.Line, p.curToken.Column,
			fmt.Sprintf("could not parse %q as float", p.curToken.Literal)))
		return nil
	}

	lit.Value = value
	return lit
}

func (p *Parser) parseStringLiteral() ast.Expression {
	return &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}
}

func (p *Parser) parseInterpolatedString() ast.Expression {
	is := &ast.InterpolatedString{Token: p.curToken}
	is.Parts = []ast.Expression{}

	// Always add the first string part (even if empty) to ensure string concatenation
	is.Parts = append(is.Parts, &ast.StringLiteral{
		Token: p.curToken,
		Value: p.curToken.Literal,
	})

	// Now we're after the first ${, parse expressions and string parts
	for {
		p.nextToken()

		// Parse the expression inside ${...}
		expr := p.parseExpression(LOWEST)
		if expr != nil {
			is.Parts = append(is.Parts, expr)
		}

		// After expression, we should have STRING_TEMPLATE_MIDDLE or STRING_TEMPLATE_END
		if p.peekTokenIs(token.STRING_TEMPLATE_MIDDLE) {
			p.nextToken()
			// Add the string part between } and ${
			is.Parts = append(is.Parts, &ast.StringLiteral{
				Token: p.curToken,
				Value: p.curToken.Literal,
			})
			// Continue to parse next expression
		} else if p.peekTokenIs(token.STRING_TEMPLATE_END) {
			p.nextToken()
			// Add the final string part
			is.Parts = append(is.Parts, &ast.StringLiteral{
				Token: p.curToken,
				Value: p.curToken.Literal,
			})
			break
		} else {
			// Unexpected token
			p.errors = append(p.errors, p.formatErrorWithContext(
				p.peekToken.Line, p.peekToken.Column,
				fmt.Sprintf("expected '}' in string interpolation, got '%s'", p.peekToken.Type)))
			return nil
		}
	}

	return is
}

func (p *Parser) parseRegexLiteral() ast.Expression {
	// Token.Literal contains the full regex: /pattern/flags
	literal := p.curToken.Literal

	// Parse pattern and flags from literal
	// Format: /pattern/flags
	if len(literal) < 2 || literal[0] != '/' {
		p.errors = append(p.errors, p.formatErrorWithContext(
			p.curToken.Line, p.curToken.Column,
			fmt.Sprintf("invalid regex literal %q", literal)))
		return nil
	}

	// Find the closing slash (handle escaped slashes)
	pattern := ""
	flags := ""
	i := 1 // skip opening /
	for i < len(literal) {
		if literal[i] == '\\' && i+1 < len(literal) {
			pattern += string(literal[i]) + string(literal[i+1])
			i += 2
			continue
		}
		if literal[i] == '/' {
			// Found closing slash, rest is flags
			flags = literal[i+1:]
			break
		}
		pattern += string(literal[i])
		i++
	}

	return &ast.RegexLiteral{
		Token:   p.curToken,
		Pattern: pattern,
		Flags:   flags,
	}
}

func (p *Parser) parseBooleanLiteral() ast.Expression {
	return &ast.BooleanLiteral{
		Token: p.curToken,
		Value: p.curToken.Type == token.TRUE,
	}
}

func (p *Parser) parsePrefixExpression() ast.Expression {
	expression := &ast.PrefixExpression{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
	}

	p.nextToken()
	expression.Right = p.parseExpression(PREFIX)

	return expression
}

func (p *Parser) parseInfixExpression(left ast.Expression) ast.Expression {
	expression := &ast.InfixExpression{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
		Left:     left,
	}

	precedence := p.curPrecedence()
	// Right associative for power operator
	if p.curToken.Type == token.POWER {
		precedence--
	}

	isComparison := p.isComparisonOperator(p.curToken.Type)

	p.nextToken()
	expression.Right = p.parseExpression(precedence)

	// Handle chained comparisons: a <= b <= c becomes (a <= b) && (b <= c)
	if isComparison && p.isComparisonOperator(p.peekToken.Type) {
		// Save the middle expression (b) for the next comparison
		middle := expression.Right

		// Move to next comparison operator
		p.nextToken()
		nextOp := p.curToken

		// Parse the right side of the chain
		p.nextToken()
		nextRight := p.parseExpression(precedence)

		// Create: (a op1 b) && (b op2 c)
		rightComparison := &ast.InfixExpression{
			Token:    nextOp,
			Operator: nextOp.Literal,
			Left:     middle,
			Right:    nextRight,
		}

		// Combine with AND
		combined := &ast.InfixExpression{
			Token:    token.Token{Type: token.AND, Literal: "&&"},
			Operator: "&&",
			Left:     expression,
			Right:    rightComparison,
		}

		// Handle further chaining: a <= b <= c <= d
		for p.isComparisonOperator(p.peekToken.Type) {
			middle = nextRight
			p.nextToken()
			nextOp = p.curToken
			p.nextToken()
			nextRight = p.parseExpression(precedence)

			rightComparison = &ast.InfixExpression{
				Token:    nextOp,
				Operator: nextOp.Literal,
				Left:     middle,
				Right:    nextRight,
			}

			combined = &ast.InfixExpression{
				Token:    token.Token{Type: token.AND, Literal: "&&"},
				Operator: "&&",
				Left:     combined,
				Right:    rightComparison,
			}
		}

		return combined
	}

	return expression
}

// isOrderingOperator returns true for ordering comparisons that can be chained
// (e.g., 0 <= x <= 100). Excludes == and != as they don't participate in chaining.
func (p *Parser) isComparisonOperator(t token.TokenType) bool {
	switch t {
	case token.LT, token.GT, token.LT_EQ, token.GT_EQ:
		return true
	default:
		return false
	}
}

func (p *Parser) parseGroupedOrLambda() ast.Expression {
	// Could be grouped expression (expr) or lambda (params) => expr
	tok := p.curToken

	// Empty params: () => expr
	if p.peekTokenIs(token.RPAREN) {
		p.nextToken() // consume ')'
		if p.peekTokenIs(token.ARROW) {
			p.nextToken() // consume '=>'
			return p.parseLambdaBody(tok, []*ast.Identifier{})
		}
		// Empty parentheses are invalid as expression
		p.errors = append(p.errors, p.formatErrorWithContext(
			tok.Line, tok.Column,
			"unexpected empty parentheses"))
		return nil
	}

	p.nextToken()
	firstExpr := p.parseExpression(LOWEST)

	// Check if it's a lambda: (x) => or (x, y) =>
	if p.peekTokenIs(token.RPAREN) {
		p.nextToken() // consume ')'
		if p.peekTokenIs(token.ARROW) {
			// It's a lambda
			p.nextToken() // consume '=>'
			params := p.exprToIdentifiers(firstExpr)
			if params == nil {
				return nil
			}
			return p.parseLambdaBody(tok, params)
		}
		// It's a grouped expression
		return firstExpr
	}

	// Check for comma (lambda with multiple params OR tuple literal)
	if p.peekTokenIs(token.COMMA) {
		// Collect all comma-separated expressions
		elements := []ast.Expression{firstExpr}

		for p.peekTokenIs(token.COMMA) {
			p.nextToken() // consume ','
			p.nextToken()
			elements = append(elements, p.parseExpression(LOWEST))
		}

		if !p.expectPeek(token.RPAREN) {
			return nil
		}

		// If followed by =>, it's a lambda
		if p.peekTokenIs(token.ARROW) {
			p.nextToken() // consume '=>'
			params := []*ast.Identifier{}
			for _, elem := range elements {
				ident := p.exprToIdentifiers(elem)
				if ident == nil {
					return nil
				}
				params = append(params, ident...)
			}
			return p.parseLambdaBody(tok, params)
		}

		// Otherwise it's a tuple literal
		return &ast.TupleLiteral{Token: tok, Elements: elements}
	}

	// It's a grouped expression, but we need to continue parsing
	// This handles cases like (a + b) which have operators after first expr
	// Return the grouped result
	if !p.expectPeek(token.RPAREN) {
		return nil
	}
	return firstExpr
}

func (p *Parser) parseLambdaBody(tok token.Token, params []*ast.Identifier) ast.Expression {
	lambda := &ast.LambdaExpression{
		Token:      tok,
		Parameters: params,
	}
	p.nextToken()
	// If the body starts with '{', parse it as a block body (sequence of statements)
	if p.curTokenIs(token.LBRACE) {
		lambda.Body = p.parseBraceBlockExpression()
	} else {
		lambda.Body = p.parseExpression(LOWEST)
	}
	return lambda
}

func (p *Parser) parseLambdaFromIdent(left ast.Expression) ast.Expression {
	// Handle: x => expr
	tok := p.curToken

	ident, ok := left.(*ast.Identifier)
	if !ok {
		p.errors = append(p.errors, p.formatErrorWithContext(
			tok.Line, tok.Column,
			"expected identifier before '=>'"))
		return nil
	}

	params := []*ast.Identifier{ident}

	lambda := &ast.LambdaExpression{
		Token:      tok,
		Parameters: params,
	}
	p.nextToken()
	// If the body starts with '{', parse it as a block body (sequence of statements)
	if p.curTokenIs(token.LBRACE) {
		lambda.Body = p.parseBraceBlockExpression()
	} else {
		lambda.Body = p.parseExpression(LOWEST)
	}
	return lambda
}

// parseBraceBlockExpression parses a block body enclosed in braces: { stmt1; stmt2; ...; lastExpr }
// Used for block-body lambdas: (x) => { stmt1; stmt2; lastExpr }
// Reuses DoExpression structure: intermediate statements are popped, last expression is the return value.
func (p *Parser) parseBraceBlockExpression() ast.Expression {
	expr := &ast.DoExpression{Token: p.curToken}
	expr.Statements = []ast.Statement{}

	// Advance past '{'
	p.nextToken()

	for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
		stmt := p.parseStatement()
		if stmt != nil {
			expr.Statements = append(expr.Statements, stmt)
		}
		if p.peekTokenIs(token.RBRACE) {
			p.nextToken()
			break
		}
		p.nextToken()
	}

	if p.curTokenIs(token.EOF) {
		p.errors = append(p.errors, p.formatErrorWithContext(
			p.curToken.Line, p.curToken.Column,
			"unexpected end of file inside block body, expected '}'"))
		return nil
	}

	// If block is empty, create a null-returning do expression
	if len(expr.Statements) == 0 {
		// Use integer 0 as a placeholder for null return from empty block
		expr.FinalExpression = &ast.IntegerLiteral{Token: p.curToken, Value: 0}
		return expr
	}

	// The last statement provides the return value of the block
	last := expr.Statements[len(expr.Statements)-1]
	if exprStmt, ok := last.(*ast.ExpressionStatement); ok {
		expr.Statements = expr.Statements[:len(expr.Statements)-1]
		expr.FinalExpression = exprStmt.Expression
	} else {
		// Last statement is an assignment or declaration; return 0 as placeholder
		expr.FinalExpression = &ast.IntegerLiteral{Token: p.curToken, Value: 0}
	}

	return expr
}

func (p *Parser) exprToIdentifiers(expr ast.Expression) []*ast.Identifier {
	switch e := expr.(type) {
	case *ast.Identifier:
		return []*ast.Identifier{e}
	default:
		p.errors = append(p.errors, p.formatErrorWithContext(
			p.curToken.Line, p.curToken.Column,
			"expected identifier in lambda parameters"))
		return nil
	}
}

func (p *Parser) parseArrayLiteral() ast.Expression {
	array := &ast.ArrayLiteral{Token: p.curToken}
	array.Elements = []ast.Expression{}

	if p.peekTokenIs(token.RBRACKET) {
		p.nextToken()
		return array
	}

	p.nextToken()
	firstElem := p.parseExpression(LOWEST)
	array.Elements = append(array.Elements, firstElem)

	// Check separator: comma or space
	if p.peekTokenIs(token.COMMA) {
		// Comma-separated: individual elements
		array.IsConcat = false
		for p.peekTokenIs(token.COMMA) {
			p.nextToken()
			p.nextToken()
			array.Elements = append(array.Elements, p.parseExpression(LOWEST))
		}
	} else if !p.peekTokenIs(token.RBRACKET) {
		// Space-separated: concatenation mode
		array.IsConcat = true
		for !p.peekTokenIs(token.RBRACKET) && !p.peekTokenIs(token.EOF) {
			p.nextToken()
			array.Elements = append(array.Elements, p.parseExpression(LOWEST))
		}
	}

	if !p.expectPeek(token.RBRACKET) {
		return nil
	}

	return array
}

func (p *Parser) parseHashLiteral() ast.Expression {
	hash := &ast.HashLiteral{Token: p.curToken}
	hash.Pairs = []ast.HashPair{}

	// Empty hash: {}
	if p.peekTokenIs(token.RBRACE) {
		p.nextToken()
		return hash
	}

	// Parse first key-value pair
	p.nextToken()
	pair := p.parseHashPair()
	if pair == nil {
		return nil
	}
	hash.Pairs = append(hash.Pairs, *pair)

	// Parse remaining pairs
	for p.peekTokenIs(token.COMMA) {
		p.nextToken() // consume ','
		p.nextToken() // move to key
		pair := p.parseHashPair()
		if pair == nil {
			return nil
		}
		hash.Pairs = append(hash.Pairs, *pair)
	}

	if !p.expectPeek(token.RBRACE) {
		return nil
	}

	return hash
}

func (p *Parser) parseHashPair() *ast.HashPair {
	// Key can be identifier (treated as string) or expression
	var key ast.Expression

	// If it's an identifier followed by colon, treat identifier as string key
	if p.curTokenIs(token.IDENT) && p.peekTokenIs(token.COLON) {
		key = &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}
	} else {
		key = p.parseExpression(LOWEST)
	}

	if !p.expectPeek(token.COLON) {
		return nil
	}

	p.nextToken()
	value := p.parseExpression(LOWEST)

	return &ast.HashPair{Key: key, Value: value}
}

func (p *Parser) parseCallExpression(function ast.Expression) ast.Expression {
	// Check if this is async.stay(state) { body }
	if me, ok := function.(*ast.MemberExpression); ok {
		if ident, ok := me.Object.(*ast.Identifier); ok {
			if ident.Value == "async" && me.Property.Value == "stay" {
				return p.parseStayExpression(me.Token)
			}
		}
	}

	exp := &ast.CallExpression{Token: p.curToken, Function: function}
	exp.Arguments = p.parseExpressionList(token.RPAREN)
	return exp
}

// parseStayExpression parses async.stay(state) { body }
func (p *Parser) parseStayExpression(tok token.Token) ast.Expression {
	stay := &ast.StayExpression{Token: tok}
	stay.StateInit = []ast.StayStatePair{}

	// We're at '(' - parse state initialization pairs
	// Format: async.stay(name1: value1, name2: value2) { body }
	if !p.peekTokenIs(token.RPAREN) {
		p.nextToken()
		pair := p.parseStayStatePair()
		if pair != nil {
			stay.StateInit = append(stay.StateInit, *pair)
		}

		for p.peekTokenIs(token.COMMA) {
			p.nextToken() // consume ','
			p.nextToken() // move to key
			pair := p.parseStayStatePair()
			if pair != nil {
				stay.StateInit = append(stay.StateInit, *pair)
			}
		}
	}

	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	// Expect '{' for body
	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	// Parse body statements until '}'
	stay.Body = []ast.Statement{}
	p.nextToken()

	for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
		stmt := p.parseStatement()
		if stmt != nil {
			stay.Body = append(stay.Body, stmt)
		}
		p.nextToken()
	}

	return stay
}

// parseStayStatePair parses a state initialization pair: name: value
func (p *Parser) parseStayStatePair() *ast.StayStatePair {
	if !p.curTokenIs(token.IDENT) {
		p.errors = append(p.errors, p.formatErrorWithContext(
			p.curToken.Line, p.curToken.Column,
			fmt.Sprintf("expected identifier in stay state, got '%s'", p.curToken.Type)))
		return nil
	}

	name := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(token.COLON) {
		return nil
	}

	p.nextToken()
	value := p.parseExpression(LOWEST)

	return &ast.StayStatePair{Name: name, Value: value}
}

func (p *Parser) parseExpressionList(end token.TokenType) []ast.Expression {
	list := []ast.Expression{}

	if p.peekTokenIs(end) {
		p.nextToken()
		return list
	}

	p.nextToken()
	list = append(list, p.parseExpression(LOWEST))

	for p.peekTokenIs(token.COMMA) {
		p.nextToken()
		p.nextToken()
		list = append(list, p.parseExpression(LOWEST))
	}

	if !p.expectPeek(end) {
		return nil
	}

	return list
}

func (p *Parser) parseIndexExpression(left ast.Expression) ast.Expression {
	exp := &ast.IndexExpression{Token: p.curToken, Left: left}

	p.nextToken()
	exp.Index = p.parseExpression(LOWEST)

	if !p.expectPeek(token.RBRACKET) {
		return nil
	}

	return exp
}

func (p *Parser) parseMemberExpression(left ast.Expression) ast.Expression {
	exp := &ast.MemberExpression{Token: p.curToken, Object: left}

	p.nextToken()
	exp.Property = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	return exp
}

func (p *Parser) parsePipeExpression(left ast.Expression) ast.Expression {
	exp := &ast.PipeExpression{Token: p.curToken, Left: left}

	precedence := p.curPrecedence()
	p.nextToken()
	exp.Right = p.parseExpression(precedence)

	return exp
}

func (p *Parser) parseEffectPipeExpression(left ast.Expression) ast.Expression {
	exp := &ast.EffectPipeExpression{Token: p.curToken, Left: left}

	precedence := p.curPrecedence()
	p.nextToken()
	exp.Right = p.parseExpression(precedence)

	return exp
}

func (p *Parser) parseErrorPropPipeExpression(left ast.Expression) ast.Expression {
	exp := &ast.ErrorPropPipeExpression{Token: p.curToken, Left: left}

	precedence := p.curPrecedence()
	p.nextToken()
	exp.Right = p.parseExpression(precedence)

	return exp
}

func (p *Parser) parseSpreadExpression(left ast.Expression) ast.Expression {
	return &ast.SpreadExpression{Token: p.curToken, Value: left}
}

func (p *Parser) parseConstraintCheckExpression(left ast.Expression) ast.Expression {
	return &ast.ConstraintCheckExpression{Token: p.curToken, Constraint: left}
}

func (p *Parser) parseMatchExpression() ast.Expression {
	exp := &ast.MatchExpression{Token: p.curToken}

	p.nextToken()
	exp.Subject = p.parseExpression(LOWEST)

	exp.Cases = []*ast.MatchCase{}

	// Parse cases until we hit a semicolon or EOF
	for !p.peekTokenIs(token.SEMICOLON) && !p.peekTokenIs(token.EOF) {
		p.nextToken()
		matchCase := p.parseMatchCase()
		if matchCase != nil {
			exp.Cases = append(exp.Cases, matchCase)
		}
	}

	return exp
}

func (p *Parser) parseMatchCase() *ast.MatchCase {
	mc := &ast.MatchCase{Token: p.curToken}

	// Check for wildcard
	if p.curTokenIs(token.UNDERSCORE) {
		mc.IsDefault = true
		mc.Pattern = &ast.WildcardExpression{Token: p.curToken}
	} else if p.curTokenIs(token.IDENT) && p.peekTokenIs(token.LPAREN) && isUpperCase(p.curToken.Literal) {
		// Constructor pattern: Name(x, y) where Name starts with uppercase
		mc.Pattern = p.parseConstructorPattern()
	} else {
		// Parse pattern/condition - use LAMBDA precedence to stop before =>
		mc.Pattern = p.parseExpression(LAMBDA)
	}

	// Check for guard clause: pattern if condition => body
	// "if" is a contextual keyword (not reserved), so it appears as IDENT with literal "if"
	if p.peekTokenIs(token.IDENT) && p.peekToken.Literal == "if" {
		p.nextToken() // consume "if"
		p.nextToken() // move to guard expression
		mc.Guard = p.parseExpression(LAMBDA)
	}

	// Expect =>
	if !p.expectPeek(token.ARROW) {
		return nil
	}
	mc.Token = p.curToken

	p.nextToken()
	// Parse body - also use LAMBDA to allow chained match cases
	mc.Body = p.parseExpression(LAMBDA)

	return mc
}

func (p *Parser) parseConstructorPattern() ast.Expression {
	cp := &ast.ConstructorPattern{
		Token: p.curToken,
		Name:  p.curToken.Literal,
	}

	p.nextToken() // consume (
	cp.Fields = []*ast.Identifier{}

	if !p.peekTokenIs(token.RPAREN) {
		p.nextToken()
		cp.Fields = append(cp.Fields, &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal})
		for p.peekTokenIs(token.COMMA) {
			p.nextToken() // consume ,
			p.nextToken() // move to next field
			cp.Fields = append(cp.Fields, &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal})
		}
	}

	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	return cp
}

func isUpperCase(s string) bool {
	if len(s) == 0 {
		return false
	}
	return s[0] >= 'A' && s[0] <= 'Z'
}

func (p *Parser) parseEffectHandleExpression(subject ast.Expression) ast.Expression {
	exp := &ast.EffectHandleExpression{Token: p.curToken, Subject: subject}

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	// Parse success and failure cases
	for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
		p.nextToken()

		if p.curTokenIs(token.SUCCESS) {
			if !p.expectPeek(token.LPAREN) {
				return nil
			}
			p.nextToken()
			exp.SuccessVar = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
			if !p.expectPeek(token.RPAREN) {
				return nil
			}
			if !p.expectPeek(token.ARROW) {
				return nil
			}
			p.nextToken()
			exp.SuccessBody = p.parseExpression(LOWEST)
		} else if p.curTokenIs(token.FAILURE) {
			if !p.expectPeek(token.LPAREN) {
				return nil
			}
			p.nextToken()
			exp.FailureVar = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
			if !p.expectPeek(token.RPAREN) {
				return nil
			}
			if !p.expectPeek(token.ARROW) {
				return nil
			}
			p.nextToken()
			exp.FailureBody = p.parseExpression(LOWEST)
		} else if p.curTokenIs(token.RBRACE) {
			break
		}
	}

	if !p.curTokenIs(token.RBRACE) {
		p.expectPeek(token.RBRACE)
	}

	return exp
}

func (p *Parser) parseWildcard() ast.Expression {
	return &ast.WildcardExpression{Token: p.curToken}
}

func (p *Parser) parseSuccessExpression() ast.Expression {
	exp := &ast.SuccessExpression{Token: p.curToken}

	if !p.expectPeek(token.LPAREN) {
		return nil
	}

	p.nextToken()
	exp.Value = p.parseExpression(LOWEST)

	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	return exp
}

func (p *Parser) parseFailureExpression() ast.Expression {
	exp := &ast.FailureExpression{Token: p.curToken}

	if !p.expectPeek(token.LPAREN) {
		return nil
	}

	p.nextToken()
	exp.Value = p.parseExpression(LOWEST)

	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	return exp
}

func (p *Parser) parseReturnExpression() ast.Expression {
	exp := &ast.ReturnExpression{Token: p.curToken}

	p.nextToken()
	exp.Value = p.parseExpression(LOWEST)

	return exp
}

// parseDoExpression parses a do...end block expression.
// Syntax: do <stmt1>; <stmt2>; ... <finalExpr> end
// The block evaluates each statement in sequence and returns the value of the final expression.
func (p *Parser) parseDoExpression() ast.Expression {
	expr := &ast.DoExpression{Token: p.curToken}
	expr.Statements = []ast.Statement{}

	// Advance past 'do' to the first token of the block body
	p.nextToken()

	for !p.isEndToken(p.curToken) && !p.curTokenIs(token.EOF) {
		stmt := p.parseStatement()
		if stmt != nil {
			expr.Statements = append(expr.Statements, stmt)
		}
		// Check if the next token is 'end' or EOF (block terminator)
		if p.isEndToken(p.peekToken) || p.peekTokenIs(token.EOF) {
			p.nextToken() // advance to 'end' or EOF
			break
		}
		p.nextToken() // advance to next statement
	}

	// If we exited because curToken == EOF, the block was not closed.
	if p.curTokenIs(token.EOF) {
		p.errors = append(p.errors, p.formatErrorWithContext(
			p.curToken.Line, p.curToken.Column,
			"unexpected end of file inside do block, expected 'end'"))
		return nil
	}

	// curToken should be 'end' now
	if len(expr.Statements) == 0 {
		p.errors = append(p.errors, p.formatErrorWithContext(
			p.curToken.Line, p.curToken.Column,
			"do block must contain at least one expression"))
		return nil
	}

	// The last statement must be an expression statement (provides the return value)
	last := expr.Statements[len(expr.Statements)-1]
	exprStmt, ok := last.(*ast.ExpressionStatement)
	if !ok {
		p.errors = append(p.errors, p.formatErrorWithContext(
			p.curToken.Line, p.curToken.Column,
			"last item in do block must be an expression (not an assignment or declaration)"))
		return nil
	}

	// Split: intermediate statements stay in Statements, final expression in FinalExpression
	expr.Statements = expr.Statements[:len(expr.Statements)-1]
	expr.FinalExpression = exprStmt.Expression

	return expr
}
