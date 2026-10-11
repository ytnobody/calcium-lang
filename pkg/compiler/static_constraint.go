package compiler

import (
	"fmt"
	"math"
	"strings"
	"unicode/utf8"

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
	constraints    map[string]*constraintInfo     // constraint name -> definition
	funcConstraint map[string]*funcConstraintInfo // function name -> param constraint info

	// constants tracks identifier -> literal for simple constant propagation
	// (e.g. x = 5; f(x)). Referenced during static constraint checking.
	constants  map[string]ast.Expression
	constStack []map[string]ast.Expression // backups for function scopes

	// funcAlias tracks simple function aliases (g = f) so indirect call sites
	// (g(0)) can be checked against the aliased function's constraints.
	funcAlias map[string]string
}

func newStaticConstraintChecker() *staticConstraintChecker {
	return &staticConstraintChecker{
		constraints:    make(map[string]*constraintInfo),
		funcConstraint: make(map[string]*funcConstraintInfo),
		constants:      make(map[string]ast.Expression),
		funcAlias:      make(map[string]string),
	}
}

// enterFunc snapshots the current constant set before compiling a function
// body so local assignments do not leak and params are not mistaken for globals.
func (sc *staticConstraintChecker) enterFunc() {
	backup := make(map[string]ast.Expression, len(sc.constants))
	for k, v := range sc.constants {
		backup[k] = v
	}
	sc.constStack = append(sc.constStack, backup)
}

// leaveFunc restores the constant set captured at enterFunc.
func (sc *staticConstraintChecker) leaveFunc() {
	if len(sc.constStack) == 0 {
		return
	}
	sc.constants = sc.constStack[len(sc.constStack)-1]
	sc.constStack = sc.constStack[:len(sc.constStack)-1]
}

// trackConst records an identifier as a constant literal when assignable.
func (sc *staticConstraintChecker) trackConst(name string, expr ast.Expression) {
	if expr == nil {
		delete(sc.constants, name)
		return
	}
	if _, ok := extractLiteralValue(expr); ok {
		sc.constants[name] = expr
	} else if id, ok := expr.(*ast.Identifier); ok {
		// If the assigned name references a function, record a function alias
		// (g = f) so indirect call sites can be checked.
		if _, isFunc := sc.funcConstraint[id.Value]; isFunc {
			sc.funcAlias[name] = id.Value
			delete(sc.constants, name)
			return
		}
		// Otherwise propagate chained constant assignments: a = b (b known).
		if c, ok := sc.constants[id.Value]; ok {
			sc.constants[name] = c
			sc.funcAlias[name] = sc.funcAlias[id.Value]
		} else {
			delete(sc.constants, name)
			delete(sc.funcAlias, name)
		}
	} else {
		delete(sc.constants, name)
		delete(sc.funcAlias, name)
	}
}

// resolveArg substitutes a known constant for an identifier argument.
func (sc *staticConstraintChecker) resolveArg(arg ast.Expression) ast.Expression {
	if id, ok := arg.(*ast.Identifier); ok {
		if c, ok := sc.constants[id.Value]; ok {
			return c
		}
	}
	return arg
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

// checkReturnConstraint statically validates a function's return value when it
// is a literal. Returns an error message if the literal violates the return
// constraint, otherwise "".
func (sc *staticConstraintChecker) checkReturnConstraint(fd *ast.FunctionDeclaration) string {
	var constraintName string
	if fd.ReturnConstraint != nil {
		constraintName = fd.ReturnConstraint.Value
	} else if fd.ReturnType != nil {
		constraintName = fd.ReturnType.Name
	} else {
		return ""
	}

	body := fd.Body
	if body == nil {
		return ""
	}

	lit, _ := extractLiteralValue(body)

	if result, recognized := checkBuiltinTypeConstraint(constraintName, body); recognized {
		if !result {
			return fmt.Sprintf("return constraint `%s` violated: function returns %s which does not satisfy `%s`", constraintName, lit, constraintName)
		}
		return ""
	}
	if result, recognized := checkBuiltinCommonConstraint(constraintName, body); recognized {
		if !result {
			return fmt.Sprintf("return constraint `%s` violated: function returns %s which does not satisfy `%s`", constraintName, lit, constraintName)
		}
		return ""
	}
	if cInfo, ok := sc.constraints[constraintName]; ok {
		if result, canEval := sc.evaluateConstraint(cInfo, body); canEval && !result {
			return fmt.Sprintf("return constraint `%s` violated: function returns %s which does not satisfy `%s`", constraintName, lit, constraintName)
		}
	}
	return ""
}

// checkCallExpression checks if a function call with literal arguments violates constraints
// Returns a list of error messages (empty if no violations or can't check statically)
func (sc *staticConstraintChecker) checkCallExpression(funcName string, args []ast.Expression, _ token.Token) []string {
	// Resolve a simple function alias (g = f) to the aliased function.
	if _, ok := sc.funcConstraint[funcName]; !ok {
		if alias, ok := sc.funcAlias[funcName]; ok {
			funcName = alias
		}
	}
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

		// Resolve identifier arguments through known constants before checking.
		arg = sc.resolveArg(arg)

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

		// Check built-in common constraints (Positive, NonZero, NonEmpty, ...)
		if result, ok := checkBuiltinCommonConstraint(constraintName, arg); ok {
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

// checkBuiltinCommonConstraint statically evaluates the built-in common
// constraints (Positive, NonZero, NonEmpty, ...) against a literal argument.
// Returns (result, isRecognizedCommonConstraint).
func checkBuiltinCommonConstraint(constraintName string, arg ast.Expression) (bool, bool) {
	switch constraintName {
	case "Positive", "NonZero", "NonNegative", "Natural", "Finite":
		num, ok := extractNumericFloat(arg)
		if !ok {
			return false, false
		}
		switch constraintName {
		case "Positive":
			return num > 0, true
		case "NonZero":
			return num != 0, true
		case "NonNegative":
			return num >= 0, true
		case "Natural":
			return num >= 0 && num == math.Trunc(num), true
		case "Finite":
			return num == num && !math.IsInf(num, 0), true // literals are always finite
		}
	case "NonEmpty", "Empty":
		if str, ok := extractStringLiteral(arg); ok {
			switch constraintName {
			case "NonEmpty":
				return utf8.RuneCountInString(str) > 0, true
			case "Empty":
				return utf8.RuneCountInString(str) == 0, true
			}
		} else if arr, ok := extractArrayLiteral(arg); ok {
			switch constraintName {
			case "NonEmpty":
				return len(arr) > 0, true
			case "Empty":
				return len(arr) == 0, true
			}
		}
	case "Digits", "Lowercase", "Uppercase":
		str, ok := extractStringLiteral(arg)
		if !ok {
			return false, false
		}
		switch constraintName {
		case "Digits":
			if len(str) == 0 {
				return false, true
			}
			for _, r := range str {
				if r < '0' || r > '9' {
					return false, true
				}
			}
			return true, true
		case "Lowercase":
			return len(str) > 0 && str == strings.ToLower(str), true
		case "Uppercase":
			return len(str) > 0 && str == strings.ToUpper(str), true
		}
	}
	return false, false
}

// extractNumericFloat returns the float value of a numeric literal (incl. negatives).
func extractNumericFloat(expr ast.Expression) (float64, bool) {
	switch e := expr.(type) {
	case *ast.IntegerLiteral:
		return float64(e.Value), true
	case *ast.FloatLiteral:
		return e.Value, true
	case *ast.PrefixExpression:
		if e.Operator == "-" {
			if v, ok := extractNumericFloat(e.Right); ok {
				return -v, true
			}
		}
	}
	return 0, false
}

// extractStringLiteral returns the value of a string literal.
func extractStringLiteral(expr ast.Expression) (string, bool) {
	if s, ok := expr.(*ast.StringLiteral); ok {
		return s.Value, true
	}
	return "", false
}

// expandCompositeConstraints rewrites parameter annotations of the form
// `A & B` / `A | B` into synthetic named constraint declarations, so the
// existing static + runtime constraint machinery handles them unchanged.
// It may append generated statements to the program. Returns the generated
// statements.
func expandCompositeConstraints(node *ast.Program) []ast.Statement {
	var generated []ast.Statement
	counter := 0

	for _, s := range node.Statements {
		fd, ok := s.(*ast.FunctionDeclaration)
		if !ok {
			continue
		}
		for i, comp := range fd.CompositeConstraints {
			if comp == nil || i >= len(fd.Parameters) {
				continue
			}
			counter++
			op := comp.Operator
			logicalOp := "&&"
			namePrefix := "__comp_and"
			if op == "|" {
				logicalOp = "||"
				namePrefix = "__comp_or"
			}
			name := fmt.Sprintf("%s_%d", namePrefix, counter)
			paramName := fd.Parameters[i].Value

			// Build body: p0(x) <op> p1(x) <op> ...
			var body ast.Expression
			for _, part := range comp.Parts {
				callExpr := &ast.CallExpression{
					Token:     part.Token,
					Function:  &ast.Identifier{Token: part.Token, Value: part.Value},
					Arguments: []ast.Expression{&ast.Identifier{Token: part.Token, Value: paramName}},
				}
				if body == nil {
					body = callExpr
					continue
				}
				body = &ast.InfixExpression{
					Token:    part.Token,
					Left:     body,
					Operator: logicalOp,
					Right:    callExpr,
				}
			}

			synth := &ast.ConstraintDeclaration{
				Token:      comp.Token,
				Name:       &ast.Identifier{Token: comp.Token, Value: name},
				Parameters: []*ast.Identifier{&ast.Identifier{Token: comp.Token, Value: paramName}},
				Body:       body,
			}
			generated = append(generated, synth)

			// Point the parameter's constraint at the synthetic name.
			if len(fd.Constraints) <= i {
				for len(fd.Constraints) <= i {
					fd.Constraints = append(fd.Constraints, nil)
				}
			}
			fd.Constraints[i] = &ast.Identifier{Token: comp.Token, Value: name}
		}
	}
	return generated
}

// extractArrayLiteral returns the elements of an array literal.
func extractArrayLiteral(expr ast.Expression) ([]ast.Expression, bool) {
	if a, ok := expr.(*ast.ArrayLiteral); ok {
		return a.Elements, true
	}
	return nil, false
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
				return nil
			}
			// Otherwise evaluate a call to a built-in constraint / type checker
			// (e.g. Positive(x), Int(x)) against a literal argument.
			if len(e.Arguments) == 1 {
				argVal := staticEval(e.Arguments[0], env, sc)
				if argVal != nil {
					if res, ok := checkBuiltinTypeConstraint(ident.Value, argVal); ok {
						return &ast.BooleanLiteral{Token: e.Token, Value: res}
					}
					if res, ok := checkBuiltinCommonConstraint(ident.Value, argVal); ok {
						return &ast.BooleanLiteral{Token: e.Token, Value: res}
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
