package ast

import (
	"strings"

	"github.com/ytnobody/calcium-lang/pkg/token"
)

// Node is the base interface for all AST nodes
type Node interface {
	TokenLiteral() string
}

// Statement represents a statement node
type Statement interface {
	Node
	statementNode()
}

// Expression represents an expression node
type Expression interface {
	Node
	expressionNode()
}

// Program is the root node of every AST
type Program struct {
	Statements []Statement
}

func (p *Program) TokenLiteral() string {
	if len(p.Statements) > 0 {
		return p.Statements[0].TokenLiteral()
	}
	return ""
}

// ---------------------------------------------------------------------
// Statements
// ---------------------------------------------------------------------

// ExpressionStatement wraps an expression as a statement
type ExpressionStatement struct {
	Token      token.Token // the first token of the expression
	Expression Expression
}

func (es *ExpressionStatement) statementNode()       {}
func (es *ExpressionStatement) TokenLiteral() string { return es.Token.Literal }

// AssignmentStatement represents variable binding: x = expr;
type AssignmentStatement struct {
	Token token.Token // the IDENT token
	Name  *Identifier
	Value Expression
}

func (as *AssignmentStatement) statementNode()       {}
func (as *AssignmentStatement) TokenLiteral() string { return as.Token.Literal }

// ArrayDestructuringStatement represents: [a, b, c] = arr;
type ArrayDestructuringStatement struct {
	Token token.Token   // the '[' token
	Names []*Identifier // variables to bind
	Value Expression    // the array expression
}

func (ad *ArrayDestructuringStatement) statementNode()       {}
func (ad *ArrayDestructuringStatement) TokenLiteral() string { return ad.Token.Literal }

// HeadTailDestructuringStatement represents: [head | tail] = arr;
type HeadTailDestructuringStatement struct {
	Token token.Token // the '[' token
	Head  *Identifier // first element
	Tail  *Identifier // rest of array
	Value Expression  // the array expression
}

func (ht *HeadTailDestructuringStatement) statementNode()       {}
func (ht *HeadTailDestructuringStatement) TokenLiteral() string { return ht.Token.Literal }

// FunctionDeclaration represents: func name(params) = body;
type FunctionDeclaration struct {
	Token       token.Token // the FUNC token
	Name        *Identifier
	Parameters  []*Identifier
	Constraints []*Identifier // constraint for each parameter (nil if none)
	Body        Expression
	IsEffect    bool // true for func! (side-effect function)
}

func (fd *FunctionDeclaration) statementNode()       {}
func (fd *FunctionDeclaration) TokenLiteral() string { return fd.Token.Literal }

// ConstraintDeclaration represents: constraint Name(params) = expr;
type ConstraintDeclaration struct {
	Token      token.Token // the CONSTRAINT token
	Name       *Identifier
	Parameters []*Identifier
	Body       Expression
}

func (cd *ConstraintDeclaration) statementNode()       {}
func (cd *ConstraintDeclaration) TokenLiteral() string { return cd.Token.Literal }

// NamespaceDeclaration represents: namespace name;
type NamespaceDeclaration struct {
	Token token.Token // the NAMESPACE token
	Name  *ModulePath
}

func (nd *NamespaceDeclaration) statementNode()       {}
func (nd *NamespaceDeclaration) TokenLiteral() string { return nd.Token.Literal }

// UseStatement represents: use module.path;
type UseStatement struct {
	Token    token.Token // the USE token
	Path     *ModulePath
	IsEffect bool // true if ending with !
}

func (us *UseStatement) statementNode()       {}
func (us *UseStatement) TokenLiteral() string { return us.Token.Literal }

// ---------------------------------------------------------------------
// Expressions
// ---------------------------------------------------------------------

// Identifier represents an identifier
type Identifier struct {
	Token token.Token // the IDENT token
	Value string
}

func (i *Identifier) expressionNode()      {}
func (i *Identifier) TokenLiteral() string { return i.Token.Literal }

// ModulePath represents a dotted module path: core.io
// Also supports remote module formats:
// - author/module (e.g., ytnobody/json)
// - "github.com/author/repo" (URL format)
type ModulePath struct {
	Token    token.Token // the first IDENT token
	Parts    []string
	IsRemote bool   // true for author/module or URL format
	Author   string // "ytnobody" for remote modules (author/module format)
	Name     string // "json" for remote modules (author/module format)
	RawURL   string // "github.com/author/repo" for URL format
}

// String returns the string representation of the module path
func (mp *ModulePath) String() string {
	if mp.RawURL != "" {
		return mp.RawURL
	}
	if mp.IsRemote {
		return mp.Author + "/" + mp.Name
	}
	return strings.Join(mp.Parts, ".")
}

func (mp *ModulePath) expressionNode()      {}
func (mp *ModulePath) TokenLiteral() string { return mp.Token.Literal }

// IntegerLiteral represents an integer
type IntegerLiteral struct {
	Token token.Token
	Value int64
}

func (il *IntegerLiteral) expressionNode()      {}
func (il *IntegerLiteral) TokenLiteral() string { return il.Token.Literal }

// FloatLiteral represents a floating-point number
type FloatLiteral struct {
	Token token.Token
	Value float64
}

func (fl *FloatLiteral) expressionNode()      {}
func (fl *FloatLiteral) TokenLiteral() string { return fl.Token.Literal }

// StringLiteral represents a string
type StringLiteral struct {
	Token token.Token
	Value string
}

func (sl *StringLiteral) expressionNode()      {}
func (sl *StringLiteral) TokenLiteral() string { return sl.Token.Literal }

// InterpolatedString represents a string with embedded expressions: "Hello, ${name}!"
type InterpolatedString struct {
	Token token.Token  // the STRING_TEMPLATE_START token
	Parts []Expression // alternating StringLiteral and expressions
}

func (is *InterpolatedString) expressionNode()      {}
func (is *InterpolatedString) TokenLiteral() string { return is.Token.Literal }

// RegexLiteral represents /pattern/flags
type RegexLiteral struct {
	Token   token.Token
	Pattern string // The regex pattern without slashes
	Flags   string // Optional flags like 'i', 'm', 's'
}

func (rl *RegexLiteral) expressionNode()      {}
func (rl *RegexLiteral) TokenLiteral() string { return rl.Token.Literal }
func (rl *RegexLiteral) String() string       { return "/" + rl.Pattern + "/" + rl.Flags }

// BooleanLiteral represents true or false
type BooleanLiteral struct {
	Token token.Token
	Value bool
}

func (bl *BooleanLiteral) expressionNode()      {}
func (bl *BooleanLiteral) TokenLiteral() string { return bl.Token.Literal }

// ArrayLiteral represents [elem1, elem2] or [elem1 elem2]
type ArrayLiteral struct {
	Token    token.Token // the '[' token
	Elements []Expression
	IsConcat bool // true if space-separated (concatenation mode)
}

func (al *ArrayLiteral) expressionNode()      {}
func (al *ArrayLiteral) TokenLiteral() string { return al.Token.Literal }

// HashPair represents a key-value pair in a hash literal
type HashPair struct {
	Key   Expression
	Value Expression
}

// HashLiteral represents {key: value, ...}
type HashLiteral struct {
	Token token.Token // the '{' token
	Pairs []HashPair  // Ordered key-value pairs
}

func (hl *HashLiteral) expressionNode()      {}
func (hl *HashLiteral) TokenLiteral() string { return hl.Token.Literal }

// PrefixExpression represents !expr, -expr
type PrefixExpression struct {
	Token    token.Token // the prefix token, e.g., !
	Operator string
	Right    Expression
}

func (pe *PrefixExpression) expressionNode()      {}
func (pe *PrefixExpression) TokenLiteral() string { return pe.Token.Literal }

// InfixExpression represents expr op expr
type InfixExpression struct {
	Token    token.Token // the operator token
	Left     Expression
	Operator string
	Right    Expression
}

func (ie *InfixExpression) expressionNode()      {}
func (ie *InfixExpression) TokenLiteral() string { return ie.Token.Literal }

// CallExpression represents function(args)
type CallExpression struct {
	Token     token.Token // the '(' token
	Function  Expression  // Identifier or member expression
	Arguments []Expression
}

func (ce *CallExpression) expressionNode()      {}
func (ce *CallExpression) TokenLiteral() string { return ce.Token.Literal }

// IndexExpression represents array[index]
type IndexExpression struct {
	Token token.Token // the '[' token
	Left  Expression
	Index Expression
}

func (ie *IndexExpression) expressionNode()      {}
func (ie *IndexExpression) TokenLiteral() string { return ie.Token.Literal }

// MemberExpression represents obj.property
type MemberExpression struct {
	Token    token.Token // the '.' token
	Object   Expression
	Property *Identifier
}

func (me *MemberExpression) expressionNode()      {}
func (me *MemberExpression) TokenLiteral() string { return me.Token.Literal }

// LambdaExpression represents (params) => expr or x => expr
type LambdaExpression struct {
	Token      token.Token // the '=>' token
	Parameters []*Identifier
	Body       Expression
}

func (le *LambdaExpression) expressionNode()      {}
func (le *LambdaExpression) TokenLiteral() string { return le.Token.Literal }

// PipeExpression represents expr |> func
type PipeExpression struct {
	Token token.Token // the '|>' token
	Left  Expression
	Right Expression
}

func (pe *PipeExpression) expressionNode()      {}
func (pe *PipeExpression) TokenLiteral() string { return pe.Token.Literal }

// EffectPipeExpression represents expr !> func!
type EffectPipeExpression struct {
	Token token.Token // the '!>' token
	Left  Expression
	Right Expression
}

func (ep *EffectPipeExpression) expressionNode()      {}
func (ep *EffectPipeExpression) TokenLiteral() string { return ep.Token.Literal }

// ErrorPropPipeExpression represents expr |>? func
// If func(expr) returns failure(e), the enclosing function returns failure(e) immediately.
// If func(expr) returns success(v), the expression evaluates to v (unwrapped).
type ErrorPropPipeExpression struct {
	Token token.Token // the '|>?' token
	Left  Expression
	Right Expression
}

func (ep *ErrorPropPipeExpression) expressionNode()      {}
func (ep *ErrorPropPipeExpression) TokenLiteral() string { return ep.Token.Literal }

// SpreadExpression represents expr...
type SpreadExpression struct {
	Token token.Token // the '...' token
	Value Expression
}

func (se *SpreadExpression) expressionNode()      {}
func (se *SpreadExpression) TokenLiteral() string { return se.Token.Literal }

// MatchExpression represents match expr case1 => result1 case2 => result2;
type MatchExpression struct {
	Token   token.Token // the MATCH token
	Subject Expression
	Cases   []*MatchCase
}

func (me *MatchExpression) expressionNode()      {}
func (me *MatchExpression) TokenLiteral() string { return me.Token.Literal }

// MatchCase represents a single case in a match expression
type MatchCase struct {
	Token     token.Token // the '=>' token
	Pattern   Expression  // pattern or condition
	Guard     Expression  // optional guard condition (nil if no guard); e.g. "if x > 0"
	Body      Expression
	IsDefault bool // true if pattern is _
	IsBinding bool // true if pattern is a variable binding (identifier captures subject value)
}

// EffectHandleExpression represents !? { success(r) => ... failure(e) => ... }
type EffectHandleExpression struct {
	Token       token.Token // the '!?' token
	Subject     Expression
	SuccessVar  *Identifier
	SuccessBody Expression
	FailureVar  *Identifier
	FailureBody Expression
}

func (eh *EffectHandleExpression) expressionNode()      {}
func (eh *EffectHandleExpression) TokenLiteral() string { return eh.Token.Literal }

// SuccessExpression represents success(value)
type SuccessExpression struct {
	Token token.Token // the SUCCESS token
	Value Expression
}

func (se *SuccessExpression) expressionNode()      {}
func (se *SuccessExpression) TokenLiteral() string { return se.Token.Literal }

// FailureExpression represents failure(error)
type FailureExpression struct {
	Token token.Token // the FAILURE token
	Value Expression
}

func (fe *FailureExpression) expressionNode()      {}
func (fe *FailureExpression) TokenLiteral() string { return fe.Token.Literal }

// WildcardExpression represents _
type WildcardExpression struct {
	Token token.Token // the '_' token
}

func (we *WildcardExpression) expressionNode()      {}
func (we *WildcardExpression) TokenLiteral() string { return we.Token.Literal }

// ReturnExpression represents return expr (for explicit returns in function bodies)
type ReturnExpression struct {
	Token token.Token // the RETURN token
	Value Expression
}

func (re *ReturnExpression) expressionNode()      {}
func (re *ReturnExpression) TokenLiteral() string { return re.Token.Literal }

// ConstraintCheckExpression represents Constraint? (constraint validation)
type ConstraintCheckExpression struct {
	Token      token.Token // the '?' token
	Constraint Expression  // the constraint to check (usually an Identifier)
}

func (cc *ConstraintCheckExpression) expressionNode()      {}
func (cc *ConstraintCheckExpression) TokenLiteral() string { return cc.Token.Literal }

// DoExpression represents a do...end block expression.
// It contains a sequence of statements followed by a final expression whose
// value is the result of the entire block.
// Syntax: do <stmt1>; <stmt2>; ... <finalExpr> end
type DoExpression struct {
	Token           token.Token // the DO token
	Statements      []Statement // intermediate statements (assignments, expr statements)
	FinalExpression Expression  // the last expression whose value is returned
}

func (de *DoExpression) expressionNode()      {}
func (de *DoExpression) TokenLiteral() string { return de.Token.Literal }

// ConstrainedParameter represents a parameter with constraint: param: Constraint?
type ConstrainedParameter struct {
	Name       *Identifier
	Constraint *Identifier
}

// StayStatePair represents a state variable initialization in async.stay
type StayStatePair struct {
	Name  *Identifier
	Value Expression
}

// StayExpression represents async.stay(state) { body }
type StayExpression struct {
	Token     token.Token     // the 'async' or 'stay' token
	StateInit []StayStatePair // State initialization: results: [], count: 0
	Body      []Statement     // Statements inside the stay block
}

func (se *StayExpression) expressionNode()      {}
func (se *StayExpression) TokenLiteral() string { return se.Token.Literal }
