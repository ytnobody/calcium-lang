// Package ops holds the constant evaluation of Calcium's operators.
//
// Two passes fold expressions at compile time: the optimiser (constant folding) and
// the compiler's static constraint checker. They must agree, or a program's
// constraint can be "violated" according to one and not the other, so the operator
// table lives here instead of being written twice.
package ops

import (
	"math"

	"github.com/ytnobody/calcium-lang/pkg/ast"
	"github.com/ytnobody/calcium-lang/pkg/token"
)

// IntInfix folds an integer operator, or returns nil when the result is not
// statically known: division or modulo by zero, and a shift outside 0..63 (the VM
// rejects those, so guessing here would hide the error).
func IntInfix(op string, left, right int64, tok token.Token) ast.Expression {
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
	case "**":
		return &ast.IntegerLiteral{Token: tok, Value: int64(math.Pow(float64(left), float64(right)))}
	case "&":
		return &ast.IntegerLiteral{Token: tok, Value: left & right}
	case "|":
		return &ast.IntegerLiteral{Token: tok, Value: left | right}
	case "^":
		return &ast.IntegerLiteral{Token: tok, Value: left ^ right}
	case "<<", ">>":
		if right < 0 || right > 63 {
			return nil
		}
		if op == "<<" {
			return &ast.IntegerLiteral{Token: tok, Value: left << uint(right)}
		}
		// Arithmetic shift, as in Go and in the VM.
		return &ast.IntegerLiteral{Token: tok, Value: left >> uint(right)}
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

// FloatInfix is IntInfix for float operands.
func FloatInfix(op string, left, right float64, tok token.Token) ast.Expression {
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

// Prefix folds a unary operator: `-` and `^` on integer literals, `-` on a float and
// `!` on a boolean. Anything else is left alone.
func Prefix(op string, right ast.Expression, tok token.Token) ast.Expression {
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
	case "^":
		if intLit, ok := right.(*ast.IntegerLiteral); ok {
			return &ast.IntegerLiteral{Token: tok, Value: ^intLit.Value}
		}
	}
	return nil
}
