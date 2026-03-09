package compiler

import (
	"fmt"

	"github.com/ytnobody/calcium-lang/pkg/ast"
	"github.com/ytnobody/calcium-lang/pkg/token"
)

// constraintInfo stores a constraint declaration for static evaluation
type constraintInfo struct {
	Name       string
	Parameters []*ast.Identifier
	Body       ast.Expression
}

// funcConstraintInfo stores constraint information for a function's parameters
type funcConstraintInfo struct {
	ParamNames      []string
	ParamConstraint []string // constraint name per parameter ("" if none)
	ReturnConstr    string   // return constraint name ("" if none)
}

// staticConstraintChecker performs compile-time constraint validation
type staticConstraintChecker struct {
	constraints    map[string]*constraintInfo    // constraint name -> definition
	funcConstraint map[string]*funcConstraintInfo // function name -> param constraint info
}

func newStaticConstraintChecker() *staticConstraintChecker {
	return &staticConstraintChecker{
		constraints:    make(map[string]*constraintInfo),
		funcConstraint: make(map[string]*funcConstraintInfo),
	}
}

// registerConstraint registers a constraint definition for static checking
func (sc *staticConstraintChecker) registerConstraint(cd *ast.ConstraintDeclaration) {
	sc.constraints[cd.Name.Value] = &constraintInfo{
		Name:       cd.Name.Value,
		Parameters: cd.Parameters,
		Body:       cd.Body,
	}
}

// registerFunction registers a function's constraint metadata
func (sc *staticConstraintChecker) registerFunction(fd *ast.FunctionDeclaration) {
	info := &funcConstraintInfo{
		ParamNames:      make([]string, len(fd.Parameters)),
		ParamConstraint: make([]string, len(fd.Parameters)),
	}

	hasAny := false

	for i, p := range fd.Parameters {
		info.ParamNames[i] = p.Value
		// Explicit constraints take priority
		if i < len(fd.Constraints) && fd.Constraints[i] != nil {
			info.ParamConstraint[i] = fd.Constraints[i].Value
			hasAny = true
		} else if i < len(fd.ParamTypes) && fd.ParamTypes[i] != nil {
			// Fall back to type annotation as constraint (e.g., Int, String, etc.)
			info.ParamConstraint[i] = fd.ParamTypes[i].Name
			hasAny = true
		}
	}

	// Return constraint: explicit ReturnConstraint or ReturnType annotation
	if fd.ReturnConstraint != nil {
		info.ReturnConstr = fd.ReturnConstraint.Value
		hasAny = true
	} else if fd.ReturnType != nil {
		info.ReturnConstr = fd.ReturnType.Name
		hasAny = true
	}

	if !hasAny {
		return
	}

	sc.funcConstraint[fd.Name.Value] = info
}

// checkCallExpression checks if a function call with literal arguments violates constraints
// Returns a list of error messages (empty if no violations or can't check statically)
func (sc *staticConstraintChecker) checkCallExpression(funcName string, args []ast.Expression, _ token.Token) []string {
	info, ok := sc.funcConstraint[funcName]
	if !ok {
		return nil
	}

	var errors []string

	for i, arg := range args {
		if i >= len(info.ParamConstraint) {
			break
		}

		constraintName := info.ParamConstraint[i]
		if constraintName == "" {
			continue
		}

		// Check if the argument is a literal value
		litValue, isLiteral := extractLiteralValue(arg)
		if !isLiteral {
			continue
		}

		// Try to check built-in type constraints first
		if result, ok := checkBuiltinTypeConstraint(constraintName, arg); ok {
			if !result {
				errors = append(errors, fmt.Sprintf(
					"constraint `%s` violated: argument %d (`%s`) with value %s does not satisfy constraint `%s`",
					constraintName, i+1, info.ParamNames[i], litValue, constraintName,
				))
			}
			continue
		}

		// Try to evaluate user-defined constraint
		cInfo, ok := sc.constraints[constraintName]
		if !ok {
			continue // constraint not found, skip static check
		}

		result, ok := sc.evaluateConstraint(cInfo, arg)
		if !ok {
			continue // can't evaluate statically, skip
		}

		if !result {
			errors = append(errors, fmt.Sprintf(
				"constraint `%s` violated: argument %d (`%s`) with value %s does not satisfy constraint `%s`",
				constraintName, i+1, info.ParamNames[i], litValue, constraintName,
			))
		}
	}

	return errors
}

// evaluateConstraint evaluates a constraint body with a literal argument substituted
// Returns (result, canEvaluate)
func (sc *staticConstraintChecker) evaluateConstraint(cInfo *constraintInfo, arg ast.Expression) (bool, bool) {
	if len(cInfo.Parameters) != 1 {
		return false, false // only single-parameter constraints for now
	}

	paramName := cInfo.Parameters[0].Value
	env := map[string]ast.Expression{paramName: arg}

	result := staticEval(cInfo.Body, env, sc)
	if result == nil {
		return false, false
	}

	boolResult, ok := result.(*ast.BooleanLiteral)
	if !ok {
		return false, false
	}

	return boolResult.Value, true
}

// extractLiteralValue returns a string representation of a literal, and whether it is a literal
func extractLiteralValue(expr ast.Expression) (string, bool) {
	switch e := expr.(type) {
	case *ast.IntegerLiteral:
		return fmt.Sprintf("%d", e.Value), true
	case *ast.FloatLiteral:
		return fmt.Sprintf("%g", e.Value), true
	case *ast.StringLiteral:
		return fmt.Sprintf("%q", e.Value), true
	case *ast.BooleanLiteral:
		return fmt.Sprintf("%t", e.Value), true
	case *ast.PrefixExpression:
		// Handle negative literals like -1
		if e.Operator == "-" {
			if inner, ok := extractLiteralValue(e.Right); ok {
				return "-" + inner, true
			}
		}
		return "", false
	default:
		return "", false
	}
}

// checkBuiltinTypeConstraint checks if a literal satisfies a built-in type constraint
// Returns (result, isBuiltinConstraint)
func checkBuiltinTypeConstraint(constraintName string, arg ast.Expression) (bool, bool) {
	switch constraintName {
	case "Int":
		_, ok := arg.(*ast.IntegerLiteral)
		if !ok {
			// Check for negative int: PrefixExpression with "-" and IntegerLiteral
			if prefix, ok := arg.(*ast.PrefixExpression); ok && prefix.Operator == "-" {
				_, ok = prefix.Right.(*ast.IntegerLiteral)
				return ok, true
			}
		}
		return ok, true
	case "Float":
		_, ok := arg.(*ast.FloatLiteral)
		if !ok {
			if prefix, ok := arg.(*ast.PrefixExpression); ok && prefix.Operator == "-" {
				_, ok = prefix.Right.(*ast.FloatLiteral)
				return ok, true
			}
		}
		return ok, true
	case "String":
		_, ok := arg.(*ast.StringLiteral)
		return ok, true
	case "Bool":
		_, ok := arg.(*ast.BooleanLiteral)
		return ok, true
	case "Number":
		_, isInt := arg.(*ast.IntegerLiteral)
		_, isFloat := arg.(*ast.FloatLiteral)
		if !isInt && !isFloat {
			if prefix, ok := arg.(*ast.PrefixExpression); ok && prefix.Operator == "-" {
				_, isInt = prefix.Right.(*ast.IntegerLiteral)
				_, isFloat = prefix.Right.(*ast.FloatLiteral)
			}
		}
		return isInt || isFloat, true
	case "Null":
		// null literal - we don't have a NullLiteral AST type typically
		return false, false
	case "Array":
		_, ok := arg.(*ast.ArrayLiteral)
		return ok, true
	case "Hash":
		_, ok := arg.(*ast.HashLiteral)
		return ok, true
	case "Tuple":
		_, ok := arg.(*ast.TupleLiteral)
		return ok, true
	case "Function":
		_, ok := arg.(*ast.LambdaExpression)
		return ok, true
	default:
		return false, false
	}
}

// staticEval evaluates an AST expression with a given variable environment
// Returns nil if the expression can't be statically evaluated
func staticEval(expr ast.Expression, env map[string]ast.Expression, sc *staticConstraintChecker) ast.Expression {
	if expr == nil {
		return nil
	}

	switch e := expr.(type) {
	case *ast.IntegerLiteral, *ast.FloatLiteral, *ast.StringLiteral, *ast.BooleanLiteral:
		return e

	case *ast.Identifier:
		// Look up variable in environment
		if env != nil {
			if val, ok := env[e.Value]; ok {
				// Normalize the value (e.g., PrefixExpression "-" IntegerLiteral → IntegerLiteral with negative value)
				return normalizeExpr(val)
			}
		}
		return nil // unknown variable, can't evaluate

	case *ast.PrefixExpression:
		right := staticEval(e.Right, env, sc)
		if right == nil {
			return nil
		}
		return evalStaticPrefix(e.Operator, right, e.Token)

	case *ast.InfixExpression:
		// Short-circuit for && and ||
		if e.Operator == "&&" {
			left := staticEval(e.Left, env, sc)
			if left == nil {
				return nil
			}
			if lb, ok := left.(*ast.BooleanLiteral); ok {
				if !lb.Value {
					return &ast.BooleanLiteral{Token: e.Token, Value: false}
				}
				right := staticEval(e.Right, env, sc)
				if right == nil {
					return nil
				}
				if rb, ok := right.(*ast.BooleanLiteral); ok {
					return &ast.BooleanLiteral{Token: e.Token, Value: rb.Value}
				}
				return nil
			}
			return nil
		}
		if e.Operator == "||" {
			left := staticEval(e.Left, env, sc)
			if left == nil {
				return nil
			}
			if lb, ok := left.(*ast.BooleanLiteral); ok {
				if lb.Value {
					return &ast.BooleanLiteral{Token: e.Token, Value: true}
				}
				right := staticEval(e.Right, env, sc)
				if right == nil {
					return nil
				}
				if rb, ok := right.(*ast.BooleanLiteral); ok {
					return &ast.BooleanLiteral{Token: e.Token, Value: rb.Value}
				}
				return nil
			}
			return nil
		}

		left := staticEval(e.Left, env, sc)
		right := staticEval(e.Right, env, sc)
		if left == nil || right == nil {
			return nil
		}
		return evalStaticInfix(e.Operator, left, right, e.Token)

	case *ast.CallExpression:
		// Check if it's a constraint call like Positive(x) inside another constraint
		if ident, ok := e.Function.(*ast.Identifier); ok {
			if cInfo, ok := sc.constraints[ident.Value]; ok {
				if len(e.Arguments) == 1 && len(cInfo.Parameters) == 1 {
					argVal := staticEval(e.Arguments[0], env, sc)
					if argVal != nil {
						innerEnv := map[string]ast.Expression{
							cInfo.Parameters[0].Value: argVal,
						}
						return staticEval(cInfo.Body, innerEnv, sc)
					}
				}
			}
		}
		return nil

	default:
		return nil
	}
}

// normalizeExpr reduces a literal expression to its simplest form
// e.g., PrefixExpression{"-", IntegerLiteral{1}} → IntegerLiteral{-1}
func normalizeExpr(expr ast.Expression) ast.Expression {
	switch e := expr.(type) {
	case *ast.IntegerLiteral, *ast.FloatLiteral, *ast.StringLiteral, *ast.BooleanLiteral:
		return e
	case *ast.PrefixExpression:
		if e.Operator == "-" {
			if intLit, ok := e.Right.(*ast.IntegerLiteral); ok {
				return &ast.IntegerLiteral{Token: e.Token, Value: -intLit.Value}
			}
			if floatLit, ok := e.Right.(*ast.FloatLiteral); ok {
				return &ast.FloatLiteral{Token: e.Token, Value: -floatLit.Value}
			}
		}
		if e.Operator == "!" {
			if boolLit, ok := e.Right.(*ast.BooleanLiteral); ok {
				return &ast.BooleanLiteral{Token: e.Token, Value: !boolLit.Value}
			}
		}
		return expr
	default:
		return expr
	}
}

// evalStaticPrefix evaluates a prefix operation on a static value
func evalStaticPrefix(op string, right ast.Expression, tok token.Token) ast.Expression {
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

// evalStaticInfix evaluates a binary operation on two static values
func evalStaticInfix(op string, left, right ast.Expression, tok token.Token) ast.Expression {
	// Integer operations
	leftInt, leftIsInt := left.(*ast.IntegerLiteral)
	rightInt, rightIsInt := right.(*ast.IntegerLiteral)

	if leftIsInt && rightIsInt {
		return evalStaticIntegerInfix(op, leftInt.Value, rightInt.Value, tok)
	}

	// Float / mixed operations
	leftFloat, leftIsFloat := left.(*ast.FloatLiteral)
	rightFloat, rightIsFloat := right.(*ast.FloatLiteral)

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
		return evalStaticFloatInfix(op, leftNum, rightNum, tok)
	}

	// String comparison
	leftStr, leftIsStr := left.(*ast.StringLiteral)
	rightStr, rightIsStr := right.(*ast.StringLiteral)
	if leftIsStr && rightIsStr {
		switch op {
		case "==":
			return &ast.BooleanLiteral{Token: tok, Value: leftStr.Value == rightStr.Value}
		case "!=":
			return &ast.BooleanLiteral{Token: tok, Value: leftStr.Value != rightStr.Value}
		}
	}

	// Boolean operations
	leftBool, leftIsBool := left.(*ast.BooleanLiteral)
	rightBool, rightIsBool := right.(*ast.BooleanLiteral)
	if leftIsBool && rightIsBool {
		switch op {
		case "==":
			return &ast.BooleanLiteral{Token: tok, Value: leftBool.Value == rightBool.Value}
		case "!=":
			return &ast.BooleanLiteral{Token: tok, Value: leftBool.Value != rightBool.Value}
		case "&&":
			return &ast.BooleanLiteral{Token: tok, Value: leftBool.Value && rightBool.Value}
		case "||":
			return &ast.BooleanLiteral{Token: tok, Value: leftBool.Value || rightBool.Value}
		}
	}

	return nil
}

func evalStaticIntegerInfix(op string, left, right int64, tok token.Token) ast.Expression {
	switch op {
	case "+":
		return &ast.IntegerLiteral{Token: tok, Value: left + right}
	case "-":
		return &ast.IntegerLiteral{Token: tok, Value: left - right}
	case "*":
		return &ast.IntegerLiteral{Token: tok, Value: left * right}
	case "/":
		if right == 0 {
			return nil
		}
		return &ast.IntegerLiteral{Token: tok, Value: left / right}
	case "%":
		if right == 0 {
			return nil
		}
		return &ast.IntegerLiteral{Token: tok, Value: left % right}
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

func evalStaticFloatInfix(op string, left, right float64, tok token.Token) ast.Expression {
	switch op {
	case "+":
		return &ast.FloatLiteral{Token: tok, Value: left + right}
	case "-":
		return &ast.FloatLiteral{Token: tok, Value: left - right}
	case "*":
		return &ast.FloatLiteral{Token: tok, Value: left * right}
	case "/":
		if right == 0 {
			return nil
		}
		return &ast.FloatLiteral{Token: tok, Value: left / right}
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
