package optimizer

import (
	"github.com/ytnobody/calcium-lang/pkg/ast"
)

// EliminateDeadCode removes unreachable and unnecessary code from the AST.
func EliminateDeadCode(node ast.Node) ast.Node {
	return eliminateDeadCode(node)
}

func eliminateDeadCode(node ast.Node) ast.Node {
	if node == nil {
		return nil
	}

	switch n := node.(type) {
	case *ast.Program:
		return eliminateDeadCodeProgram(n)
	case *ast.ExpressionStatement:
		return eliminateDeadCodeExpressionStatement(n)
	case *ast.AssignmentStatement:
		return eliminateDeadCodeAssignmentStatement(n)
	case *ast.FunctionDeclaration:
		return eliminateDeadCodeFunctionDeclaration(n)
	case *ast.ConstraintDeclaration:
		return eliminateDeadCodeConstraintDeclaration(n)
	case *ast.ArrayDestructuringStatement:
		return eliminateDeadCodeArrayDestructuringStatement(n)
	case *ast.HeadTailDestructuringStatement:
		return eliminateDeadCodeHeadTailDestructuringStatement(n)
	case *ast.MatchExpression:
		return eliminateDeadCodeMatchExpression(n)
	case *ast.InfixExpression:
		return eliminateDeadCodeInfixExpression(n)
	case *ast.PrefixExpression:
		return eliminateDeadCodePrefixExpression(n)
	case *ast.ArrayLiteral:
		return eliminateDeadCodeArrayLiteral(n)
	case *ast.HashLiteral:
		return eliminateDeadCodeHashLiteral(n)
	case *ast.IndexExpression:
		return eliminateDeadCodeIndexExpression(n)
	case *ast.CallExpression:
		return eliminateDeadCodeCallExpression(n)
	case *ast.LambdaExpression:
		return eliminateDeadCodeLambdaExpression(n)
	case *ast.PipeExpression:
		return eliminateDeadCodePipeExpression(n)
	case *ast.EffectPipeExpression:
		return eliminateDeadCodeEffectPipeExpression(n)
	case *ast.EffectHandleExpression:
		return eliminateDeadCodeEffectHandleExpression(n)
	case *ast.SuccessExpression:
		return eliminateDeadCodeSuccessExpression(n)
	case *ast.FailureExpression:
		return eliminateDeadCodeFailureExpression(n)
	case *ast.ReturnExpression:
		return eliminateDeadCodeReturnExpression(n)
	case *ast.MemberExpression:
		return eliminateDeadCodeMemberExpression(n)
	case *ast.SpreadExpression:
		return eliminateDeadCodeSpreadExpression(n)
	case *ast.InterpolatedString:
		return eliminateDeadCodeInterpolatedString(n)
	case *ast.StayExpression:
		return eliminateDeadCodeStayExpression(n)
	default:
		return node
	}
}

func eliminateDeadCodeProgram(p *ast.Program) *ast.Program {
	stmts := make([]ast.Statement, 0, len(p.Statements))
	for _, stmt := range p.Statements {
		optimized := eliminateDeadCode(stmt).(ast.Statement)
		stmts = append(stmts, optimized)
	}
	return &ast.Program{Statements: stmts}
}

func eliminateDeadCodeExpressionStatement(es *ast.ExpressionStatement) *ast.ExpressionStatement {
	return &ast.ExpressionStatement{
		Token:      es.Token,
		Expression: eliminateDeadCode(es.Expression).(ast.Expression),
	}
}

func eliminateDeadCodeAssignmentStatement(as *ast.AssignmentStatement) *ast.AssignmentStatement {
	return &ast.AssignmentStatement{
		Token: as.Token,
		Name:  as.Name,
		Value: eliminateDeadCode(as.Value).(ast.Expression),
	}
}

func eliminateDeadCodeFunctionDeclaration(fd *ast.FunctionDeclaration) *ast.FunctionDeclaration {
	return &ast.FunctionDeclaration{
		Token:                fd.Token,
		Name:                 fd.Name,
		Parameters:           fd.Parameters,
		Constraints:          fd.Constraints,
		ParamTypes:           fd.ParamTypes,
		CompositeConstraints: fd.CompositeConstraints,
		ReturnType:           fd.ReturnType,
		ReturnConstraint:     fd.ReturnConstraint,
		Body:                 eliminateDeadCode(fd.Body).(ast.Expression),
		IsEffect:             fd.IsEffect,
	}
}

func eliminateDeadCodeConstraintDeclaration(cd *ast.ConstraintDeclaration) *ast.ConstraintDeclaration {
	return &ast.ConstraintDeclaration{
		Token:      cd.Token,
		Name:       cd.Name,
		Parameters: cd.Parameters,
		Body:       eliminateDeadCode(cd.Body).(ast.Expression),
	}
}

func eliminateDeadCodeArrayDestructuringStatement(ad *ast.ArrayDestructuringStatement) *ast.ArrayDestructuringStatement {
	return &ast.ArrayDestructuringStatement{
		Token: ad.Token,
		Names: ad.Names,
		Value: eliminateDeadCode(ad.Value).(ast.Expression),
	}
}

func eliminateDeadCodeHeadTailDestructuringStatement(ht *ast.HeadTailDestructuringStatement) *ast.HeadTailDestructuringStatement {
	return &ast.HeadTailDestructuringStatement{
		Token: ht.Token,
		Head:  ht.Head,
		Tail:  ht.Tail,
		Value: eliminateDeadCode(ht.Value).(ast.Expression),
	}
}

func eliminateDeadCodeMatchExpression(me *ast.MatchExpression) ast.Expression {
	subject := eliminateDeadCode(me.Subject).(ast.Expression)

	// Fold a match on a constant subject when the outcome is decidable soundly.
	if folded := foldConstantMatch(me.Cases, subject); folded != nil {
		return folded
	}

	// Otherwise, optimize all branches
	cases := make([]*ast.MatchCase, len(me.Cases))
	for i, c := range me.Cases {
		cases[i] = &ast.MatchCase{
			Token:     c.Token,
			Pattern:   eliminateDeadCode(c.Pattern).(ast.Expression),
			Guard:     eliminateDeadCodeGuard(c.Guard),
			Body:      eliminateDeadCode(c.Body).(ast.Expression),
			IsDefault: c.IsDefault,
		}
	}

	return &ast.MatchExpression{
		Token:   me.Token,
		Subject: subject,
		Cases:   cases,
	}
}

func eliminateDeadCodeInfixExpression(ie *ast.InfixExpression) ast.Expression {
	left := eliminateDeadCode(ie.Left).(ast.Expression)
	right := eliminateDeadCode(ie.Right).(ast.Expression)

	// Short-circuit evaluation for && with constant left operand
	if ie.Operator == "&&" {
		if boolLit, ok := left.(*ast.BooleanLiteral); ok {
			if !boolLit.Value {
				// false && anything => false
				return &ast.BooleanLiteral{Token: ie.Token, Value: false}
			}
			// true && x => x
			return right
		}
	}

	// Short-circuit evaluation for || with constant left operand
	if ie.Operator == "||" {
		if boolLit, ok := left.(*ast.BooleanLiteral); ok {
			if boolLit.Value {
				// true || anything => true
				return &ast.BooleanLiteral{Token: ie.Token, Value: true}
			}
			// false || x => x
			return right
		}
	}

	return &ast.InfixExpression{
		Token:    ie.Token,
		Left:     left,
		Operator: ie.Operator,
		Right:    right,
	}
}

func eliminateDeadCodePrefixExpression(pe *ast.PrefixExpression) *ast.PrefixExpression {
	return &ast.PrefixExpression{
		Token:    pe.Token,
		Operator: pe.Operator,
		Right:    eliminateDeadCode(pe.Right).(ast.Expression),
	}
}

func eliminateDeadCodeArrayLiteral(al *ast.ArrayLiteral) *ast.ArrayLiteral {
	elements := make([]ast.Expression, len(al.Elements))
	for i, elem := range al.Elements {
		elements[i] = eliminateDeadCode(elem).(ast.Expression)
	}
	return &ast.ArrayLiteral{
		Token:    al.Token,
		Elements: elements,
		IsConcat: al.IsConcat,
	}
}

func eliminateDeadCodeHashLiteral(hl *ast.HashLiteral) *ast.HashLiteral {
	pairs := make([]ast.HashPair, len(hl.Pairs))
	for i, pair := range hl.Pairs {
		pairs[i] = ast.HashPair{
			Key:   eliminateDeadCode(pair.Key).(ast.Expression),
			Value: eliminateDeadCode(pair.Value).(ast.Expression),
		}
	}
	return &ast.HashLiteral{
		Token: hl.Token,
		Pairs: pairs,
	}
}

func eliminateDeadCodeIndexExpression(ie *ast.IndexExpression) *ast.IndexExpression {
	return &ast.IndexExpression{
		Token: ie.Token,
		Left:  eliminateDeadCode(ie.Left).(ast.Expression),
		Index: eliminateDeadCode(ie.Index).(ast.Expression),
	}
}

func eliminateDeadCodeCallExpression(ce *ast.CallExpression) *ast.CallExpression {
	args := make([]ast.Expression, len(ce.Arguments))
	for i, arg := range ce.Arguments {
		args[i] = eliminateDeadCode(arg).(ast.Expression)
	}
	return &ast.CallExpression{
		Token:     ce.Token,
		Function:  eliminateDeadCode(ce.Function).(ast.Expression),
		Arguments: args,
	}
}

func eliminateDeadCodeLambdaExpression(le *ast.LambdaExpression) *ast.LambdaExpression {
	return &ast.LambdaExpression{
		Token:      le.Token,
		Parameters: le.Parameters,
		Body:       eliminateDeadCode(le.Body).(ast.Expression),
	}
}

func eliminateDeadCodePipeExpression(pe *ast.PipeExpression) *ast.PipeExpression {
	return &ast.PipeExpression{
		Token: pe.Token,
		Left:  eliminateDeadCode(pe.Left).(ast.Expression),
		Right: eliminateDeadCode(pe.Right).(ast.Expression),
	}
}

func eliminateDeadCodeEffectPipeExpression(ep *ast.EffectPipeExpression) *ast.EffectPipeExpression {
	return &ast.EffectPipeExpression{
		Token:       ep.Token,
		Left:        eliminateDeadCode(ep.Left).(ast.Expression),
		Right:       eliminateDeadCode(ep.Right).(ast.Expression),
		AllowSecret: ep.AllowSecret,
	}
}

func eliminateDeadCodeEffectHandleExpression(eh *ast.EffectHandleExpression) *ast.EffectHandleExpression {
	return &ast.EffectHandleExpression{
		Token:       eh.Token,
		Subject:     eliminateDeadCode(eh.Subject).(ast.Expression),
		SuccessVar:  eh.SuccessVar,
		SuccessBody: eliminateDeadCode(eh.SuccessBody).(ast.Expression),
		FailureVar:  eh.FailureVar,
		FailureBody: eliminateDeadCode(eh.FailureBody).(ast.Expression),
	}
}

func eliminateDeadCodeSuccessExpression(se *ast.SuccessExpression) *ast.SuccessExpression {
	return &ast.SuccessExpression{
		Token: se.Token,
		Value: eliminateDeadCode(se.Value).(ast.Expression),
	}
}

func eliminateDeadCodeFailureExpression(fe *ast.FailureExpression) *ast.FailureExpression {
	return &ast.FailureExpression{
		Token: fe.Token,
		Value: eliminateDeadCode(fe.Value).(ast.Expression),
	}
}

func eliminateDeadCodeReturnExpression(re *ast.ReturnExpression) *ast.ReturnExpression {
	return &ast.ReturnExpression{
		Token: re.Token,
		Value: eliminateDeadCode(re.Value).(ast.Expression),
	}
}

func eliminateDeadCodeMemberExpression(me *ast.MemberExpression) *ast.MemberExpression {
	return &ast.MemberExpression{
		Token:    me.Token,
		Object:   eliminateDeadCode(me.Object).(ast.Expression),
		Property: me.Property,
	}
}

func eliminateDeadCodeSpreadExpression(se *ast.SpreadExpression) *ast.SpreadExpression {
	return &ast.SpreadExpression{
		Token: se.Token,
		Value: eliminateDeadCode(se.Value).(ast.Expression),
	}
}

func eliminateDeadCodeInterpolatedString(is *ast.InterpolatedString) *ast.InterpolatedString {
	parts := make([]ast.Expression, len(is.Parts))
	for i, part := range is.Parts {
		parts[i] = eliminateDeadCode(part).(ast.Expression)
	}
	return &ast.InterpolatedString{
		Token: is.Token,
		Parts: parts,
	}
}

func eliminateDeadCodeStayExpression(se *ast.StayExpression) *ast.StayExpression {
	stateInit := make([]ast.StayStatePair, len(se.StateInit))
	for i, pair := range se.StateInit {
		stateInit[i] = ast.StayStatePair{
			Name:  pair.Name,
			Value: eliminateDeadCode(pair.Value).(ast.Expression),
		}
	}

	body := make([]ast.Statement, len(se.Body))
	for i, stmt := range se.Body {
		body[i] = eliminateDeadCode(stmt).(ast.Statement)
	}

	return &ast.StayExpression{
		Token:     se.Token,
		StateInit: stateInit,
		Body:      body,
	}
}

// eliminateDeadCodeGuard folds a case guard, preserving nil. A guard is
// optional, and dropping it would turn `x if x > 0` into `x`.
func eliminateDeadCodeGuard(guard ast.Expression) ast.Expression {
	if guard == nil {
		return nil
	}
	return eliminateDeadCode(guard).(ast.Expression)
}

// foldConstantMatch selects the single arm a match on a constant subject must
// take, or returns nil when the match cannot be folded soundly.
//
// Folding is only sound when every arm pattern is a literal of the subject's
// kind (or a wildcard) and no arm has a guard. In particular `match true` is a
// common idiom for condition chains:
//
//	match true
//	  n % 15 == 0 => "FizzBuzz"
//	  true        => to_string(n)
//
// whose patterns are boolean *expressions*, not literals. Selecting the `true`
// arm there would be wrong, so such a match is left intact.
//
// Returning nil when no arm matches is also deliberate: the match must fail at
// run time, which only the VM's no-match handler can report.
func foldConstantMatch(cases []*ast.MatchCase, subject ast.Expression) ast.Expression {
	var comparable func(pattern ast.Expression) (matches bool, ok bool)

	switch s := subject.(type) {
	case *ast.BooleanLiteral:
		comparable = func(pattern ast.Expression) (bool, bool) {
			lit, ok := pattern.(*ast.BooleanLiteral)
			if !ok {
				return false, false
			}
			return lit.Value == s.Value, true
		}
	case *ast.IntegerLiteral:
		comparable = func(pattern ast.Expression) (bool, bool) {
			lit, ok := pattern.(*ast.IntegerLiteral)
			if !ok {
				return false, false
			}
			return lit.Value == s.Value, true
		}
	case *ast.StringLiteral:
		comparable = func(pattern ast.Expression) (bool, bool) {
			lit, ok := pattern.(*ast.StringLiteral)
			if !ok {
				return false, false
			}
			return lit.Value == s.Value, true
		}
	default:
		return nil
	}

	for _, c := range cases {
		if c.Guard != nil {
			return nil
		}
		if c.IsDefault {
			return eliminateDeadCode(c.Body).(ast.Expression)
		}
		matches, ok := comparable(c.Pattern)
		if !ok {
			return nil
		}
		if matches {
			return eliminateDeadCode(c.Body).(ast.Expression)
		}
	}

	return nil
}
