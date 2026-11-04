package optimizer

import "github.com/ytnobody/calcium-lang/pkg/ast"

// withGuard passes a match case's guard through an optimizer pass, preserving nil.
//
// A guard is optional: `x => body` has none, while `x if x > 0 => body` does.
// Dropping a nil guard is not a simplification - turning `x if x > 0 => body` into
// `x => body` changes the program. Three passes rewrite match cases, so the rule
// lives here instead of being re-implemented (and possibly forgotten) in each.
func withGuard(pass func(ast.Node) ast.Node, guard ast.Expression) ast.Expression {
	if guard == nil {
		return nil
	}
	return pass(guard).(ast.Expression)
}
