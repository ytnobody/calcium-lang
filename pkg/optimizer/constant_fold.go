package optimizer

import (
	"math"

	"github.com/ytnobody/calcium-lang/pkg/ast"
	"github.com/ytnobody/calcium-lang/pkg/token"
)

// FoldConstants performs constant folding optimization on the AST.
// It evaluates expressions with constant operands at compile time.
func FoldConstants(node ast.Node) ast.Node {
	return foldNode(node)
}

func foldNode(node ast.Node) ast.Node {
	if node == nil {
		return nil
	}

	switch n := node.(type) {
	case *ast.Program:
		return foldProgram(n)
	case *ast.ExpressionStatement:
		return foldExpressionStatement(n)
	case *ast.AssignmentStatement:
		return foldAssignmentStatement(n)
	case *ast.FunctionDeclaration:
		return foldFunctionDeclaration(n)
	case *ast.ConstraintDeclaration:
		return foldConstraintDeclaration(n)
	case *ast.ArrayDestructuringStatement:
		return foldArrayDestructuringStatement(n)
	case *ast.HeadTailDestructuringStatement:
		return foldHeadTailDestructuringStatement(n)
	case *ast.InfixExpression:
		return foldInfixExpression(n)
	case *ast.PrefixExpression:
		return foldPrefixExpression(n)
	case *ast.ArrayLiteral:
		return foldArrayLiteral(n)
	case *ast.HashLiteral:
		return foldHashLiteral(n)
	case *ast.IndexExpression:
		return foldIndexExpression(n)
	case *ast.CallExpression:
		return foldCallExpression(n)
	case *ast.LambdaExpression:
		return foldLambdaExpression(n)
	case *ast.PipeExpression:
		return foldPipeExpression(n)
	case *ast.EffectPipeExpression:
		return foldEffectPipeExpression(n)
	case *ast.MatchExpression:
		return foldMatchExpression(n)
	case *ast.EffectHandleExpression:
		return foldEffectHandleExpression(n)
	case *ast.SuccessExpression:
		return foldSuccessExpression(n)
	case *ast.FailureExpression:
		return foldFailureExpression(n)
	case *ast.ReturnExpression:
		return foldReturnExpression(n)
	case *ast.MemberExpression:
		return foldMemberExpression(n)
	case *ast.SpreadExpression:
		return foldSpreadExpression(n)
	case *ast.InterpolatedString:
		return foldInterpolatedString(n)
	case *ast.StayExpression:
		return foldStayExpression(n)
	default:
		// Literals and identifiers are already constant or cannot be folded
		return node
	}
}

func foldProgram(p *ast.Program) *ast.Program {
	stmts := make([]ast.Statement, len(p.Statements))
	for i, stmt := range p.Statements {
		stmts[i] = foldNode(stmt).(ast.Statement)
	}
	return &ast.Program{Statements: stmts}
}

func foldExpressionStatement(es *ast.ExpressionStatement) *ast.ExpressionStatement {
	return &ast.ExpressionStatement{
		Token:      es.Token,
		Expression: foldNode(es.Expression).(ast.Expression),
	}
}

func foldAssignmentStatement(as *ast.AssignmentStatement) *ast.AssignmentStatement {
	return &ast.AssignmentStatement{
		Token: as.Token,
		Name:  as.Name,
		Value: foldNode(as.Value).(ast.Expression),
	}
}

func foldFunctionDeclaration(fd *ast.FunctionDeclaration) *ast.FunctionDeclaration {
	return &ast.FunctionDeclaration{
		Token:       fd.Token,
		Name:        fd.Name,
		Parameters:  fd.Parameters,
		Constraints: fd.Constraints,
		Body:        foldNode(fd.Body).(ast.Expression),
		IsEffect:    fd.IsEffect,
	}
}

func foldConstraintDeclaration(cd *ast.ConstraintDeclaration) *ast.ConstraintDeclaration {
	return &ast.ConstraintDeclaration{
		Token:      cd.Token,
		Name:       cd.Name,
		Parameters: cd.Parameters,
		Body:       foldNode(cd.Body).(ast.Expression),
	}
}

func foldArrayDestructuringStatement(ad *ast.ArrayDestructuringStatement) *ast.ArrayDestructuringStatement {
	return &ast.ArrayDestructuringStatement{
		Token: ad.Token,
		Names: ad.Names,
		Value: foldNode(ad.Value).(ast.Expression),
	}
}

func foldHeadTailDestructuringStatement(ht *ast.HeadTailDestructuringStatement) *ast.HeadTailDestructuringStatement {
	return &ast.HeadTailDestructuringStatement{
		Token: ht.Token,
		Head:  ht.Head,
		Tail:  ht.Tail,
		Value: foldNode(ht.Value).(ast.Expression),
	}
}

func foldInfixExpression(ie *ast.InfixExpression) ast.Expression {
	left := foldNode(ie.Left).(ast.Expression)
	right := foldNode(ie.Right).(ast.Expression)

	// Try to evaluate constant expressions
	if result := evalConstantInfix(ie.Operator, left, right, ie.Token); result != nil {
		return result
	}

	return &ast.InfixExpression{
		Token:    ie.Token,
		Left:     left,
		Operator: ie.Operator,
		Right:    right,
	}
}

func foldPrefixExpression(pe *ast.PrefixExpression) ast.Expression {
	right := foldNode(pe.Right).(ast.Expression)

	// Try to evaluate constant expressions
	if result := evalConstantPrefix(pe.Operator, right, pe.Token); result != nil {
		return result
	}

	return &ast.PrefixExpression{
		Token:    pe.Token,
		Operator: pe.Operator,
		Right:    right,
	}
}

func foldArrayLiteral(al *ast.ArrayLiteral) *ast.ArrayLiteral {
	elements := make([]ast.Expression, len(al.Elements))
	for i, elem := range al.Elements {
		elements[i] = foldNode(elem).(ast.Expression)
	}
	return &ast.ArrayLiteral{
		Token:    al.Token,
		Elements: elements,
		IsConcat: al.IsConcat,
	}
}

func foldHashLiteral(hl *ast.HashLiteral) *ast.HashLiteral {
	pairs := make([]ast.HashPair, len(hl.Pairs))
	for i, pair := range hl.Pairs {
		pairs[i] = ast.HashPair{
			Key:   foldNode(pair.Key).(ast.Expression),
			Value: foldNode(pair.Value).(ast.Expression),
		}
	}
	return &ast.HashLiteral{
		Token: hl.Token,
		Pairs: pairs,
	}
}

func foldIndexExpression(ie *ast.IndexExpression) ast.Expression {
	left := foldNode(ie.Left).(ast.Expression)
	index := foldNode(ie.Index).(ast.Expression)

	// Fold constant array/string indexing
	if arr, ok := left.(*ast.ArrayLiteral); ok {
		if idx, ok := index.(*ast.IntegerLiteral); ok {
			i := int(idx.Value)
			if i >= 0 && i < len(arr.Elements) {
				return arr.Elements[i]
			}
		}
	}

	if str, ok := left.(*ast.StringLiteral); ok {
		if idx, ok := index.(*ast.IntegerLiteral); ok {
			i := int(idx.Value)
			runes := []rune(str.Value)
			if i >= 0 && i < len(runes) {
				return &ast.StringLiteral{
					Token: ie.Token,
					Value: string(runes[i]),
				}
			}
		}
	}

	return &ast.IndexExpression{
		Token: ie.Token,
		Left:  left,
		Index: index,
	}
}

func foldCallExpression(ce *ast.CallExpression) *ast.CallExpression {
	args := make([]ast.Expression, len(ce.Arguments))
	for i, arg := range ce.Arguments {
		args[i] = foldNode(arg).(ast.Expression)
	}
	return &ast.CallExpression{
		Token:     ce.Token,
		Function:  foldNode(ce.Function).(ast.Expression),
		Arguments: args,
	}
}

func foldLambdaExpression(le *ast.LambdaExpression) *ast.LambdaExpression {
	return &ast.LambdaExpression{
		Token:      le.Token,
		Parameters: le.Parameters,
		Body:       foldNode(le.Body).(ast.Expression),
	}
}

func foldPipeExpression(pe *ast.PipeExpression) *ast.PipeExpression {
	return &ast.PipeExpression{
		Token: pe.Token,
		Left:  foldNode(pe.Left).(ast.Expression),
		Right: foldNode(pe.Right).(ast.Expression),
	}
}

func foldEffectPipeExpression(ep *ast.EffectPipeExpression) *ast.EffectPipeExpression {
	return &ast.EffectPipeExpression{
		Token: ep.Token,
		Left:  foldNode(ep.Left).(ast.Expression),
		Right: foldNode(ep.Right).(ast.Expression),
	}
}

func foldMatchExpression(me *ast.MatchExpression) *ast.MatchExpression {
	cases := make([]*ast.MatchCase, len(me.Cases))
	for i, c := range me.Cases {
		cases[i] = &ast.MatchCase{
			Token:     c.Token,
			Pattern:   foldNode(c.Pattern).(ast.Expression),
			Body:      foldNode(c.Body).(ast.Expression),
			IsDefault: c.IsDefault,
		}
	}
	return &ast.MatchExpression{
		Token:   me.Token,
		Subject: foldNode(me.Subject).(ast.Expression),
		Cases:   cases,
	}
}

func foldEffectHandleExpression(eh *ast.EffectHandleExpression) *ast.EffectHandleExpression {
	return &ast.EffectHandleExpression{
		Token:       eh.Token,
		Subject:     foldNode(eh.Subject).(ast.Expression),
		SuccessVar:  eh.SuccessVar,
		SuccessBody: foldNode(eh.SuccessBody).(ast.Expression),
		FailureVar:  eh.FailureVar,
		FailureBody: foldNode(eh.FailureBody).(ast.Expression),
	}
}

func foldSuccessExpression(se *ast.SuccessExpression) *ast.SuccessExpression {
	return &ast.SuccessExpression{
		Token: se.Token,
		Value: foldNode(se.Value).(ast.Expression),
	}
}

func foldFailureExpression(fe *ast.FailureExpression) *ast.FailureExpression {
	return &ast.FailureExpression{
		Token: fe.Token,
		Value: foldNode(fe.Value).(ast.Expression),
	}
}

func foldReturnExpression(re *ast.ReturnExpression) *ast.ReturnExpression {
	return &ast.ReturnExpression{
		Token: re.Token,
		Value: foldNode(re.Value).(ast.Expression),
	}
}

func foldMemberExpression(me *ast.MemberExpression) *ast.MemberExpression {
	return &ast.MemberExpression{
		Token:    me.Token,
		Object:   foldNode(me.Object).(ast.Expression),
		Property: me.Property,
	}
}

func foldSpreadExpression(se *ast.SpreadExpression) *ast.SpreadExpression {
	return &ast.SpreadExpression{
		Token: se.Token,
		Value: foldNode(se.Value).(ast.Expression),
	}
}

func foldInterpolatedString(is *ast.InterpolatedString) ast.Expression {
	parts := make([]ast.Expression, len(is.Parts))
	allStrings := true
	for i, part := range is.Parts {
		parts[i] = foldNode(part).(ast.Expression)
		if _, ok := parts[i].(*ast.StringLiteral); !ok {
			allStrings = false
		}
	}

	// If all parts are string literals, concatenate them
	if allStrings {
		var result string
		for _, part := range parts {
			result += part.(*ast.StringLiteral).Value
		}
		return &ast.StringLiteral{
			Token: is.Token,
			Value: result,
		}
	}

	return &ast.InterpolatedString{
		Token: is.Token,
		Parts: parts,
	}
}

func foldStayExpression(se *ast.StayExpression) *ast.StayExpression {
	stateInit := make([]ast.StayStatePair, len(se.StateInit))
	for i, pair := range se.StateInit {
		stateInit[i] = ast.StayStatePair{
			Name:  pair.Name,
			Value: foldNode(pair.Value).(ast.Expression),
		}
	}

	body := make([]ast.Statement, len(se.Body))
	for i, stmt := range se.Body {
		body[i] = foldNode(stmt).(ast.Statement)
	}

	return &ast.StayExpression{
		Token:     se.Token,
		StateInit: stateInit,
		Body:      body,
	}
}

// evalConstantInfix evaluates a binary operation on constant operands
func evalConstantInfix(op string, left, right ast.Expression, tok token.Token) ast.Expression {
	// Integer operations
	leftInt, leftIsInt := left.(*ast.IntegerLiteral)
	rightInt, rightIsInt := right.(*ast.IntegerLiteral)

	if leftIsInt && rightIsInt {
		return evalIntegerInfix(op, leftInt.Value, rightInt.Value, tok)
	}

	// Float operations
	leftFloat, leftIsFloat := left.(*ast.FloatLiteral)
	rightFloat, rightIsFloat := right.(*ast.FloatLiteral)

	// Mixed int/float
	var leftNum, rightNum float64
	var hasFloat bool

	if leftIsInt {
		leftNum = float64(leftInt.Value)
	} else if leftIsFloat {
		leftNum = leftFloat.Value
		hasFloat = true
	}

	if rightIsInt {
		rightNum = float64(rightInt.Value)
	} else if rightIsFloat {
		rightNum = rightFloat.Value
		hasFloat = true
	}

	if hasFloat && (leftIsInt || leftIsFloat) && (rightIsInt || rightIsFloat) {
		return evalFloatInfix(op, leftNum, rightNum, tok)
	}

	// String concatenation
	leftStr, leftIsStr := left.(*ast.StringLiteral)
	rightStr, rightIsStr := right.(*ast.StringLiteral)

	if leftIsStr && rightIsStr && op == "+" {
		return &ast.StringLiteral{
			Token: tok,
			Value: leftStr.Value + rightStr.Value,
		}
	}

	// Boolean operations
	leftBool, leftIsBool := left.(*ast.BooleanLiteral)
	rightBool, rightIsBool := right.(*ast.BooleanLiteral)

	if leftIsBool && rightIsBool {
		return evalBooleanInfix(op, leftBool.Value, rightBool.Value, tok)
	}

	return nil
}

func evalIntegerInfix(op string, left, right int64, tok token.Token) ast.Expression {
	switch op {
	case "+":
		return &ast.IntegerLiteral{Token: tok, Value: left + right}
	case "-":
		return &ast.IntegerLiteral{Token: tok, Value: left - right}
	case "*":
		return &ast.IntegerLiteral{Token: tok, Value: left * right}
	case "/":
		if right == 0 {
			return nil // Don't fold division by zero
		}
		return &ast.IntegerLiteral{Token: tok, Value: left / right}
	case "%":
		if right == 0 {
			return nil // Don't fold modulo by zero
		}
		return &ast.IntegerLiteral{Token: tok, Value: left % right}
	case "**":
		return &ast.IntegerLiteral{Token: tok, Value: int64(math.Pow(float64(left), float64(right)))}
	case "<":
		return &ast.BooleanLiteral{Token: tok, Value: left < right}
	case ">":
		return &ast.BooleanLiteral{Token: tok, Value: left > right}
	case "<=":
		return &ast.BooleanLiteral{Token: tok, Value: left <= right}
	case ">=":
		return &ast.BooleanLiteral{Token: tok, Value: left >= right}
	case "==":
		return &ast.BooleanLiteral{Token: tok, Value: left == right}
	case "!=":
		return &ast.BooleanLiteral{Token: tok, Value: left != right}
	}
	return nil
}

func evalFloatInfix(op string, left, right float64, tok token.Token) ast.Expression {
	switch op {
	case "+":
		return &ast.FloatLiteral{Token: tok, Value: left + right}
	case "-":
		return &ast.FloatLiteral{Token: tok, Value: left - right}
	case "*":
		return &ast.FloatLiteral{Token: tok, Value: left * right}
	case "/":
		if right == 0 {
			return nil // Don't fold division by zero
		}
		return &ast.FloatLiteral{Token: tok, Value: left / right}
	case "**":
		return &ast.FloatLiteral{Token: tok, Value: math.Pow(left, right)}
	case "<":
		return &ast.BooleanLiteral{Token: tok, Value: left < right}
	case ">":
		return &ast.BooleanLiteral{Token: tok, Value: left > right}
	case "<=":
		return &ast.BooleanLiteral{Token: tok, Value: left <= right}
	case ">=":
		return &ast.BooleanLiteral{Token: tok, Value: left >= right}
	case "==":
		return &ast.BooleanLiteral{Token: tok, Value: left == right}
	case "!=":
		return &ast.BooleanLiteral{Token: tok, Value: left != right}
	}
	return nil
}

func evalBooleanInfix(op string, left, right bool, tok token.Token) ast.Expression {
	switch op {
	case "&&":
		return &ast.BooleanLiteral{Token: tok, Value: left && right}
	case "||":
		return &ast.BooleanLiteral{Token: tok, Value: left || right}
	case "==":
		return &ast.BooleanLiteral{Token: tok, Value: left == right}
	case "!=":
		return &ast.BooleanLiteral{Token: tok, Value: left != right}
	}
	return nil
}

// evalConstantPrefix evaluates a unary operation on a constant operand
func evalConstantPrefix(op string, right ast.Expression, tok token.Token) ast.Expression {
	switch op {
	case "-":
		if intLit, ok := right.(*ast.IntegerLiteral); ok {
			return &ast.IntegerLiteral{Token: tok, Value: -intLit.Value}
		}
		if floatLit, ok := right.(*ast.FloatLiteral); ok {
			return &ast.FloatLiteral{Token: tok, Value: -floatLit.Value}
		}
	case "!":
		if boolLit, ok := right.(*ast.BooleanLiteral); ok {
			return &ast.BooleanLiteral{Token: tok, Value: !boolLit.Value}
		}
	}
	return nil
}
