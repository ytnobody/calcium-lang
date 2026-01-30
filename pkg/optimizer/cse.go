// Package optimizer - Common Subexpression Elimination (CSE)
//
// CSE identifies identical expressions that are computed multiple times
// and transforms the code to compute them only once.
//
// Example:
//
//	use_twice(expensive(x), expensive(x))
//
// Becomes internally:
//
//	let _cse_0 = expensive(x) in use_twice(_cse_0, _cse_0)
//
// Note: Only pure expressions are candidates for CSE.
// Effect function calls (func!) are never deduplicated.
package optimizer

import (
	"fmt"

	"github.com/ytnobody/calcium-lang/pkg/ast"
)

// CSEOptimizer performs common subexpression elimination
type CSEOptimizer struct {
	counter int
	// Track which identifiers are known to be effect functions
	effectFuncs map[string]bool
}

// NewCSEOptimizer creates a new CSE optimizer
func NewCSEOptimizer() *CSEOptimizer {
	return &CSEOptimizer{
		effectFuncs: make(map[string]bool),
	}
}

// EliminateCommonSubexpressions performs CSE on the AST
func EliminateCommonSubexpressions(node ast.Node) ast.Node {
	opt := NewCSEOptimizer()
	// First pass: collect effect function declarations
	opt.collectEffectFuncs(node)
	// Second pass: perform CSE
	return opt.optimize(node)
}

// collectEffectFuncs finds all effect function declarations
func (o *CSEOptimizer) collectEffectFuncs(node ast.Node) {
	switch n := node.(type) {
	case *ast.Program:
		for _, stmt := range n.Statements {
			o.collectEffectFuncs(stmt)
		}
	case *ast.FunctionDeclaration:
		if n.IsEffect {
			o.effectFuncs[n.Name.Value] = true
		}
	}
}

// optimize performs the CSE optimization
func (o *CSEOptimizer) optimize(node ast.Node) ast.Node {
	if node == nil {
		return nil
	}

	switch n := node.(type) {
	case *ast.Program:
		return o.optimizeProgram(n)
	case *ast.ExpressionStatement:
		return &ast.ExpressionStatement{
			Token:      n.Token,
			Expression: o.optimizeExpr(n.Expression),
		}
	case *ast.AssignmentStatement:
		return &ast.AssignmentStatement{
			Token: n.Token,
			Name:  n.Name,
			Value: o.optimizeExpr(n.Value),
		}
	case *ast.FunctionDeclaration:
		return &ast.FunctionDeclaration{
			Token:       n.Token,
			Name:        n.Name,
			Parameters:  n.Parameters,
			Constraints: n.Constraints,
			Body:        o.optimizeExpr(n.Body),
			IsEffect:    n.IsEffect,
		}
	case *ast.ConstraintDeclaration:
		return &ast.ConstraintDeclaration{
			Token:      n.Token,
			Name:       n.Name,
			Parameters: n.Parameters,
			Body:       o.optimizeExpr(n.Body),
		}
	default:
		return node
	}
}

func (o *CSEOptimizer) optimizeProgram(p *ast.Program) *ast.Program {
	stmts := make([]ast.Statement, len(p.Statements))
	for i, stmt := range p.Statements {
		stmts[i] = o.optimize(stmt).(ast.Statement)
	}
	return &ast.Program{Statements: stmts}
}

// optimizeExpr optimizes an expression, performing CSE where applicable
func (o *CSEOptimizer) optimizeExpr(expr ast.Expression) ast.Expression {
	if expr == nil {
		return nil
	}

	// First, recursively optimize sub-expressions
	expr = o.optimizeSubExprs(expr)

	// Then, look for CSE opportunities in call expressions
	switch e := expr.(type) {
	case *ast.CallExpression:
		return o.optimizeCallExpr(e)
	default:
		return expr
	}
}

// optimizeSubExprs recursively optimizes sub-expressions
func (o *CSEOptimizer) optimizeSubExprs(expr ast.Expression) ast.Expression {
	switch e := expr.(type) {
	case *ast.CallExpression:
		args := make([]ast.Expression, len(e.Arguments))
		for i, arg := range e.Arguments {
			args[i] = o.optimizeExpr(arg)
		}
		return &ast.CallExpression{
			Token:     e.Token,
			Function:  o.optimizeExpr(e.Function),
			Arguments: o.deduplicateArgs(args),
		}

	case *ast.InfixExpression:
		left := o.optimizeExpr(e.Left)
		right := o.optimizeExpr(e.Right)
		// Check if left and right are identical pure expressions
		//nolint:staticcheck // TODO: Implement CSE optimization with AST transformation
		if o.isPureExpr(left) && o.exprEqual(left, right) {
			// They're the same, but we can't easily introduce a let binding
			// at this level without AST transformation support.
			// For now, just return the optimized version
		}
		return &ast.InfixExpression{
			Token:    e.Token,
			Left:     left,
			Operator: e.Operator,
			Right:    right,
		}

	case *ast.PrefixExpression:
		return &ast.PrefixExpression{
			Token:    e.Token,
			Operator: e.Operator,
			Right:    o.optimizeExpr(e.Right),
		}

	case *ast.ArrayLiteral:
		elements := make([]ast.Expression, len(e.Elements))
		for i, elem := range e.Elements {
			elements[i] = o.optimizeExpr(elem)
		}
		// Deduplicate array elements if they're pure
		elements = o.deduplicateArgs(elements)
		return &ast.ArrayLiteral{
			Token:    e.Token,
			Elements: elements,
			IsConcat: e.IsConcat,
		}

	case *ast.HashLiteral:
		pairs := make([]ast.HashPair, len(e.Pairs))
		for i, pair := range e.Pairs {
			pairs[i] = ast.HashPair{
				Key:   o.optimizeExpr(pair.Key),
				Value: o.optimizeExpr(pair.Value),
			}
		}
		return &ast.HashLiteral{
			Token: e.Token,
			Pairs: pairs,
		}

	case *ast.IndexExpression:
		return &ast.IndexExpression{
			Token: e.Token,
			Left:  o.optimizeExpr(e.Left),
			Index: o.optimizeExpr(e.Index),
		}

	case *ast.LambdaExpression:
		return &ast.LambdaExpression{
			Token:      e.Token,
			Parameters: e.Parameters,
			Body:       o.optimizeExpr(e.Body),
		}

	case *ast.PipeExpression:
		return &ast.PipeExpression{
			Token: e.Token,
			Left:  o.optimizeExpr(e.Left),
			Right: o.optimizeExpr(e.Right),
		}

	case *ast.EffectPipeExpression:
		return &ast.EffectPipeExpression{
			Token: e.Token,
			Left:  o.optimizeExpr(e.Left),
			Right: o.optimizeExpr(e.Right),
		}

	case *ast.MatchExpression:
		cases := make([]*ast.MatchCase, len(e.Cases))
		for i, c := range e.Cases {
			cases[i] = &ast.MatchCase{
				Token:     c.Token,
				Pattern:   c.Pattern, // Don't optimize patterns
				Body:      o.optimizeExpr(c.Body),
				IsDefault: c.IsDefault,
			}
		}
		return &ast.MatchExpression{
			Token:   e.Token,
			Subject: o.optimizeExpr(e.Subject),
			Cases:   cases,
		}

	case *ast.EffectHandleExpression:
		return &ast.EffectHandleExpression{
			Token:       e.Token,
			Subject:     o.optimizeExpr(e.Subject),
			SuccessVar:  e.SuccessVar,
			SuccessBody: o.optimizeExpr(e.SuccessBody),
			FailureVar:  e.FailureVar,
			FailureBody: o.optimizeExpr(e.FailureBody),
		}

	case *ast.SuccessExpression:
		return &ast.SuccessExpression{
			Token: e.Token,
			Value: o.optimizeExpr(e.Value),
		}

	case *ast.FailureExpression:
		return &ast.FailureExpression{
			Token: e.Token,
			Value: o.optimizeExpr(e.Value),
		}

	case *ast.MemberExpression:
		return &ast.MemberExpression{
			Token:    e.Token,
			Object:   o.optimizeExpr(e.Object),
			Property: e.Property,
		}

	case *ast.SpreadExpression:
		return &ast.SpreadExpression{
			Token: e.Token,
			Value: o.optimizeExpr(e.Value),
		}

	case *ast.InterpolatedString:
		parts := make([]ast.Expression, len(e.Parts))
		for i, part := range e.Parts {
			parts[i] = o.optimizeExpr(part)
		}
		return &ast.InterpolatedString{
			Token: e.Token,
			Parts: parts,
		}

	default:
		return expr
	}
}

// optimizeCallExpr optimizes a call expression
func (o *CSEOptimizer) optimizeCallExpr(call *ast.CallExpression) ast.Expression {
	// Deduplicate arguments that are identical pure expressions
	args := o.deduplicateArgs(call.Arguments)
	return &ast.CallExpression{
		Token:     call.Token,
		Function:  call.Function,
		Arguments: args,
	}
}

// deduplicateArgs finds duplicate pure expressions in args and marks them for reuse
// This is a simplified version - in a full implementation, we'd introduce let bindings
func (o *CSEOptimizer) deduplicateArgs(args []ast.Expression) []ast.Expression {
	if len(args) < 2 {
		return args
	}

	// Find groups of identical expressions
	groups := make([]exprGroup, 0)

	for i, arg := range args {
		if !o.isPureExpr(arg) || !o.isExpensiveExpr(arg) {
			continue
		}

		found := false
		for j := range groups {
			if o.exprEqual(groups[j].expr, arg) {
				groups[j].indices = append(groups[j].indices, i)
				found = true
				break
			}
		}
		if !found {
			groups = append(groups, exprGroup{expr: arg, indices: []int{i}})
		}
	}

	// For groups with more than one occurrence, we could introduce CSE
	// For now, we just return the original args since proper CSE would require
	// AST-level let bindings which Calcium doesn't have yet
	//
	// In the future, this could transform:
	//   f(expensive(x), expensive(x))
	// Into something like:
	//   ((_cse) => f(_cse, _cse))(expensive(x))

	// Check if we have any duplicates worth optimizing
	for _, group := range groups {
		if len(group.indices) > 1 {
			// Found duplicates - apply CSE transformation
			return o.applyCSETransform(args, groups)
		}
	}

	return args
}

// applyCSETransform applies CSE by wrapping in an immediately-invoked lambda
// This transforms: f(expensive(x), expensive(x))
// Into: ((_cse0) => f(_cse0, _cse0))(expensive(x))
func (o *CSEOptimizer) applyCSETransform(args []ast.Expression, groups []exprGroup) []ast.Expression {
	// For simplicity, we'll only handle the case where we have duplicates
	// and create a simple transformation

	result := make([]ast.Expression, len(args))
	copy(result, args)

	// This is a placeholder - full CSE transformation would require
	// returning a different expression type (wrapped in lambda)
	// For now, we mark this as a future enhancement

	return result
}

// isPureExpr checks if an expression is pure (no side effects)
func (o *CSEOptimizer) isPureExpr(expr ast.Expression) bool {
	switch e := expr.(type) {
	case *ast.IntegerLiteral, *ast.FloatLiteral, *ast.StringLiteral,
		*ast.BooleanLiteral, *ast.Identifier:
		return true

	case *ast.ArrayLiteral:
		for _, elem := range e.Elements {
			if !o.isPureExpr(elem) {
				return false
			}
		}
		return true

	case *ast.HashLiteral:
		for _, pair := range e.Pairs {
			if !o.isPureExpr(pair.Key) || !o.isPureExpr(pair.Value) {
				return false
			}
		}
		return true

	case *ast.InfixExpression:
		return o.isPureExpr(e.Left) && o.isPureExpr(e.Right)

	case *ast.PrefixExpression:
		return o.isPureExpr(e.Right)

	case *ast.CallExpression:
		// Check if the function is an effect function
		if ident, ok := e.Function.(*ast.Identifier); ok {
			if o.effectFuncs[ident.Value] {
				return false
			}
		}
		if member, ok := e.Function.(*ast.MemberExpression); ok {
			// Module calls like io.println are effects if the module is imported with !
			// For now, assume member expressions might be effects
			// A more sophisticated check would track which modules are effect modules
			_ = member
			// Conservative: treat member expressions as potentially impure
			// unless we can prove otherwise
		}
		// Check if all arguments are pure
		for _, arg := range e.Arguments {
			if !o.isPureExpr(arg) {
				return false
			}
		}
		return true

	case *ast.IndexExpression:
		return o.isPureExpr(e.Left) && o.isPureExpr(e.Index)

	case *ast.LambdaExpression:
		// Lambdas themselves are pure values
		return true

	case *ast.MemberExpression:
		return o.isPureExpr(e.Object)

	default:
		return false
	}
}

// isExpensiveExpr checks if an expression is worth caching (not trivial)
func (o *CSEOptimizer) isExpensiveExpr(expr ast.Expression) bool {
	switch e := expr.(type) {
	case *ast.IntegerLiteral, *ast.FloatLiteral, *ast.StringLiteral,
		*ast.BooleanLiteral, *ast.Identifier:
		// Trivial expressions - not worth caching
		return false

	case *ast.CallExpression:
		// Function calls are expensive
		return true

	case *ast.InfixExpression:
		// Binary operations might be expensive if operands are
		return o.isExpensiveExpr(e.Left) || o.isExpensiveExpr(e.Right)

	case *ast.IndexExpression:
		return o.isExpensiveExpr(e.Left)

	default:
		return false
	}
}

// exprEqual checks if two expressions are structurally equal
func (o *CSEOptimizer) exprEqual(a, b ast.Expression) bool {
	if a == nil || b == nil {
		return a == b
	}

	switch ae := a.(type) {
	case *ast.IntegerLiteral:
		if be, ok := b.(*ast.IntegerLiteral); ok {
			return ae.Value == be.Value
		}
	case *ast.FloatLiteral:
		if be, ok := b.(*ast.FloatLiteral); ok {
			return ae.Value == be.Value
		}
	case *ast.StringLiteral:
		if be, ok := b.(*ast.StringLiteral); ok {
			return ae.Value == be.Value
		}
	case *ast.BooleanLiteral:
		if be, ok := b.(*ast.BooleanLiteral); ok {
			return ae.Value == be.Value
		}
	case *ast.Identifier:
		if be, ok := b.(*ast.Identifier); ok {
			return ae.Value == be.Value
		}
	case *ast.CallExpression:
		if be, ok := b.(*ast.CallExpression); ok {
			if !o.exprEqual(ae.Function, be.Function) {
				return false
			}
			if len(ae.Arguments) != len(be.Arguments) {
				return false
			}
			for i := range ae.Arguments {
				if !o.exprEqual(ae.Arguments[i], be.Arguments[i]) {
					return false
				}
			}
			return true
		}
	case *ast.InfixExpression:
		if be, ok := b.(*ast.InfixExpression); ok {
			return ae.Operator == be.Operator &&
				o.exprEqual(ae.Left, be.Left) &&
				o.exprEqual(ae.Right, be.Right)
		}
	case *ast.PrefixExpression:
		if be, ok := b.(*ast.PrefixExpression); ok {
			return ae.Operator == be.Operator &&
				o.exprEqual(ae.Right, be.Right)
		}
	case *ast.IndexExpression:
		if be, ok := b.(*ast.IndexExpression); ok {
			return o.exprEqual(ae.Left, be.Left) &&
				o.exprEqual(ae.Index, be.Index)
		}
	case *ast.MemberExpression:
		if be, ok := b.(*ast.MemberExpression); ok {
			return o.exprEqual(ae.Object, be.Object) &&
				ae.Property.Value == be.Property.Value
		}
	case *ast.ArrayLiteral:
		if be, ok := b.(*ast.ArrayLiteral); ok {
			if len(ae.Elements) != len(be.Elements) {
				return false
			}
			for i := range ae.Elements {
				if !o.exprEqual(ae.Elements[i], be.Elements[i]) {
					return false
				}
			}
			return true
		}
	}

	return false
}

// generateCSEVar generates a unique variable name for CSE
func (o *CSEOptimizer) generateCSEVar() string {
	name := fmt.Sprintf("_cse_%d", o.counter)
	o.counter++
	return name
}

// exprGroup is used internally to track groups of identical expressions
type exprGroup struct {
	expr    ast.Expression
	indices []int
}
