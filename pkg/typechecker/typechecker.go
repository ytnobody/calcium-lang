// Package typechecker implements an optional, gradual type checker for calcium-lang.
//
// The type checker is non-mandatory: programs without type annotations continue to
// work exactly as before. When annotations are present, the checker verifies that
// the annotated types are consistent with the inferred types of the expressions.
//
// Under gradual typing, the special type "Any" is compatible with every other type,
// so mixing typed and untyped code is always safe from the checker's perspective.
//
// Usage:
//
//	errs := typechecker.Check(program)
//	if len(errs) > 0 {
//	    for _, e := range errs { fmt.Println(e) }
//	}
package typechecker

import (
	"fmt"

	"github.com/ytnobody/calcium-lang/pkg/ast"
	"github.com/ytnobody/calcium-lang/pkg/types"
)

// TypeError describes a type mismatch found during checking.
type TypeError struct {
	Line    int
	Column  int
	Message string
}

func (e TypeError) Error() string {
	return fmt.Sprintf("[type error] line %d, col %d: %s", e.Line, e.Column, e.Message)
}

// Checker holds type environment state while walking the AST.
type Checker struct {
	errors []TypeError
	// env maps variable names to their declared (annotated) types.
	env map[string]types.TypeExpr
	// funcReturnTypes maps function names to their declared return types.
	funcReturnTypes map[string]types.TypeExpr
}

// New creates a fresh Checker.
func New() *Checker {
	return &Checker{
		env:             make(map[string]types.TypeExpr),
		funcReturnTypes: make(map[string]types.TypeExpr),
	}
}

// Check runs the gradual type checker on the given program and returns any
// type errors found. An empty slice means no type errors were detected.
// Calling Check does NOT affect runtime behavior.
func Check(program *ast.Program) []TypeError {
	c := New()
	c.checkProgram(program)
	return c.errors
}

func (c *Checker) addError(line, col int, msg string) {
	c.errors = append(c.errors, TypeError{Line: line, Column: col, Message: msg})
}

// ---- program / statement walking ----

func (c *Checker) checkProgram(program *ast.Program) {
	for _, stmt := range program.Statements {
		c.checkStatement(stmt)
	}
}

func (c *Checker) checkStatement(stmt ast.Statement) {
	switch s := stmt.(type) {
	case *ast.AssignmentStatement:
		c.checkAssignment(s)
	case *ast.FunctionDeclaration:
		c.checkFunctionDeclaration(s)
	case *ast.ExpressionStatement:
		c.inferExpr(s.Expression)
	// Other statement kinds (namespace, use, type, destructuring) currently
	// have no type annotation support; skip silently.
	}
}

// ---- assignment checking ----

func (c *Checker) checkAssignment(s *ast.AssignmentStatement) {
	if s.TypeAnnot == nil {
		// No annotation — infer and record type for future references.
		inferred := c.inferExpr(s.Value)
		c.env[s.Name.Value] = inferred
		return
	}

	// Annotated assignment: verify the value type is compatible.
	annotType := typeFromAnnotation(s.TypeAnnot)
	inferred := c.inferExpr(s.Value)

	if !annotType.IsCompatibleWith(inferred) {
		tok := s.TypeAnnot.Token
		c.addError(tok.Line, tok.Column, fmt.Sprintf(
			"type mismatch for %q: declared %s but expression has type %s",
			s.Name.Value, annotType, inferred,
		))
	}

	// Record the declared type (not the inferred one) so subsequent uses of the
	// variable are checked against the annotation.
	c.env[s.Name.Value] = annotType
}

// ---- function declaration checking ----

func (c *Checker) checkFunctionDeclaration(fd *ast.FunctionDeclaration) {
	// Record parameter types in a child environment.
	childEnv := make(map[string]types.TypeExpr)
	for k, v := range c.env {
		childEnv[k] = v
	}

	for i, param := range fd.Parameters {
		var paramType types.TypeExpr = types.Any
		if i < len(fd.ParamTypes) && fd.ParamTypes[i] != nil {
			paramType = typeFromAnnotation(fd.ParamTypes[i])
		}
		childEnv[param.Value] = paramType
	}

	// Register return type before checking body (allows recursive calls).
	var retType types.TypeExpr = types.Any
	if fd.ReturnType != nil {
		retType = typeFromAnnotation(fd.ReturnType)
	}
	c.funcReturnTypes[fd.Name.Value] = retType

	// Check body in child environment.
	saved := c.env
	c.env = childEnv
	bodyType := c.inferExpr(fd.Body)
	c.env = saved

	// Verify return type.
	if fd.ReturnType != nil && !retType.IsCompatibleWith(bodyType) {
		tok := fd.ReturnType.Token
		c.addError(tok.Line, tok.Column, fmt.Sprintf(
			"return type mismatch for function %q: declared %s but body has type %s",
			fd.Name.Value, retType, bodyType,
		))
	}
}

// ---- expression type inference ----

// inferExpr infers the type of an expression under gradual typing rules.
// When the type cannot be determined statically (e.g., the expression involves
// unannotated variables), it returns Any.
func (c *Checker) inferExpr(expr ast.Expression) types.TypeExpr {
	if expr == nil {
		return types.Any
	}
	switch e := expr.(type) {
	case *ast.IntegerLiteral:
		return types.Int
	case *ast.FloatLiteral:
		return types.Float
	case *ast.StringLiteral:
		return types.Str
	case *ast.InterpolatedString:
		return types.Str
	case *ast.BooleanLiteral:
		return types.Bool
	case *ast.RegexLiteral:
		return types.Rx
	case *ast.ArrayLiteral:
		return types.Arr
	case *ast.HashLiteral:
		return types.Hsh
	case *ast.TupleLiteral:
		return types.Tup
	case *ast.SuccessExpression:
		return types.Succ
	case *ast.FailureExpression:
		return types.Fail
	case *ast.Identifier:
		if t, ok := c.env[e.Value]; ok {
			return t
		}
		return types.Any
	case *ast.InfixExpression:
		return c.inferInfix(e)
	case *ast.PrefixExpression:
		return c.inferPrefix(e)
	case *ast.CallExpression:
		return c.inferCall(e)
	case *ast.LambdaExpression:
		return types.Fn
	case *ast.DoExpression:
		return c.inferDo(e)
	case *ast.MatchExpression:
		return types.Any // result type depends on branches; treated as Any for now
	default:
		return types.Any
	}
}

func (c *Checker) inferInfix(e *ast.InfixExpression) types.TypeExpr {
	left := c.inferExpr(e.Left)
	right := c.inferExpr(e.Right)

	switch e.Operator {
	case "+", "-", "*", "/", "%", "**":
		// Numeric operators: if both operands are Int, result is Int;
		// if either is Float, result is Float; otherwise Any.
		if isInt(left) && isInt(right) {
			return types.Int
		}
		if isFloat(left) || isFloat(right) {
			return types.Float
		}
		// String concatenation with +
		if e.Operator == "+" && isStr(left) && isStr(right) {
			return types.Str
		}
		return types.Any
	case "==", "!=", "<", ">", "<=", ">=", "&&", "||":
		return types.Bool
	default:
		return types.Any
	}
}

func (c *Checker) inferPrefix(e *ast.PrefixExpression) types.TypeExpr {
	operand := c.inferExpr(e.Right)
	switch e.Operator {
	case "!":
		return types.Bool
	case "-":
		if isInt(operand) {
			return types.Int
		}
		if isFloat(operand) {
			return types.Float
		}
		return types.Any
	}
	return types.Any
}

func (c *Checker) inferCall(e *ast.CallExpression) types.TypeExpr {
	// If calling a known named function, return its declared return type.
	if ident, ok := e.Function.(*ast.Identifier); ok {
		if rt, ok := c.funcReturnTypes[ident.Value]; ok {
			return rt
		}
	}
	return types.Any
}

func (c *Checker) inferDo(e *ast.DoExpression) types.TypeExpr {
	// Walk intermediate statements, then return type of final expression.
	savedEnv := c.env
	childEnv := make(map[string]types.TypeExpr)
	for k, v := range c.env {
		childEnv[k] = v
	}
	c.env = childEnv
	for _, stmt := range e.Statements {
		c.checkStatement(stmt)
	}
	result := c.inferExpr(e.FinalExpression)
	c.env = savedEnv
	return result
}

// ---- helpers ----

func typeFromAnnotation(a *ast.TypeAnnotation) types.TypeExpr {
	if a == nil {
		return types.Any
	}
	t := types.FromName(a.Name)
	if t == nil {
		return types.Any
	}
	return t
}

func isInt(t types.TypeExpr) bool {
	_, ok := t.(*types.IntType)
	return ok
}

func isFloat(t types.TypeExpr) bool {
	_, ok := t.(*types.FloatType)
	return ok
}

func isStr(t types.TypeExpr) bool {
	_, ok := t.(*types.StringType)
	return ok
}
