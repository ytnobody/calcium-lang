// Package formatter implements the Calcium source code formatter.
// It parses the source code into an AST and then pretty-prints it
// with a consistent style.
//
// Note: Since comments are stripped during lexing, they are not
// preserved in the formatted output.
package formatter

import (
	"fmt"
	"strings"

	"github.com/ytnobody/calcium-lang/pkg/ast"
	"github.com/ytnobody/calcium-lang/pkg/lexer"
	"github.com/ytnobody/calcium-lang/pkg/parser"
)

const indentUnit = "  " // 2 spaces

// Formatter pretty-prints a Calcium AST back to source code.
type Formatter struct {
	depth int
}

// New creates a new Formatter.
func New() *Formatter {
	return &Formatter{}
}

// Format parses src and returns the formatted source code.
// Returns an error if the source cannot be parsed.
func Format(src string) (string, error) {
	l := lexer.New(src)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		return "", fmt.Errorf("parse errors:\n%s", strings.Join(p.Errors(), "\n"))
	}

	f := New()
	return f.FormatProgram(program), nil
}

// indent returns the current indentation string.
func (f *Formatter) indent() string {
	return strings.Repeat(indentUnit, f.depth)
}

// FormatProgram formats an entire program.
func (f *Formatter) FormatProgram(program *ast.Program) string {
	var sb strings.Builder

	for i, stmt := range program.Statements {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(f.formatStatement(stmt))
	}

	// Ensure file ends with a newline
	result := sb.String()
	if len(result) > 0 && !strings.HasSuffix(result, "\n") {
		result += "\n"
	}
	return result
}

// formatStatement formats a single statement.
func (f *Formatter) formatStatement(stmt ast.Statement) string {
	switch s := stmt.(type) {
	case *ast.UseStatement:
		return f.formatUseStatement(s)
	case *ast.NamespaceDeclaration:
		return f.formatNamespaceDeclaration(s)
	case *ast.AssignmentStatement:
		return f.formatAssignmentStatement(s)
	case *ast.ArrayDestructuringStatement:
		return f.formatArrayDestructuringStatement(s)
	case *ast.HeadTailDestructuringStatement:
		return f.formatHeadTailDestructuringStatement(s)
	case *ast.FunctionDeclaration:
		return f.formatFunctionDeclaration(s)
	case *ast.ConstraintDeclaration:
		return f.formatConstraintDeclaration(s)
	case *ast.TypeDeclaration:
		return f.formatTypeDeclaration(s)
	case *ast.ExpressionStatement:
		return f.formatExpressionStatement(s)
	default:
		return fmt.Sprintf("/* unknown statement: %T */", stmt)
	}
}

func (f *Formatter) formatUseStatement(s *ast.UseStatement) string {
	path := f.formatModulePath(s.Path)
	if s.IsEffect {
		return f.indent() + "use " + path + "!;\n"
	}
	return f.indent() + "use " + path + ";\n"
}

func (f *Formatter) formatNamespaceDeclaration(s *ast.NamespaceDeclaration) string {
	return f.indent() + "namespace " + f.formatModulePath(s.Name) + ";\n"
}

func (f *Formatter) formatAssignmentStatement(s *ast.AssignmentStatement) string {
	val := f.formatExpression(s.Value)
	return f.indent() + s.Name.Value + " = " + val + ";\n"
}

func (f *Formatter) formatArrayDestructuringStatement(s *ast.ArrayDestructuringStatement) string {
	names := make([]string, len(s.Names))
	for i, n := range s.Names {
		names[i] = n.Value
	}
	val := f.formatExpression(s.Value)
	return f.indent() + "[" + strings.Join(names, ", ") + "] = " + val + ";\n"
}

func (f *Formatter) formatHeadTailDestructuringStatement(s *ast.HeadTailDestructuringStatement) string {
	val := f.formatExpression(s.Value)
	return f.indent() + "[" + s.Head.Value + " | " + s.Tail.Value + "] = " + val + ";\n"
}

func (f *Formatter) formatFunctionDeclaration(s *ast.FunctionDeclaration) string {
	var sb strings.Builder

	if s.IsEffect {
		sb.WriteString(f.indent() + "func! ")
	} else {
		sb.WriteString(f.indent() + "func ")
	}
	sb.WriteString(s.Name.Value)
	sb.WriteString("(")

	params := make([]string, len(s.Parameters))
	for i, p := range s.Parameters {
		if i < len(s.Constraints) && s.Constraints[i] != nil {
			params[i] = p.Value + ": " + s.Constraints[i].Value + "?"
		} else {
			params[i] = p.Value
		}
	}
	sb.WriteString(strings.Join(params, ", "))
	sb.WriteString(") =")

	// Format the body
	body := f.formatFunctionBody(s.Body)
	sb.WriteString(body)
	sb.WriteString(";\n")

	return sb.String()
}

// formatFunctionBody formats the body of a function or constraint.
// For match expressions, it formats on a new line with indentation.
// For other expressions, it puts on the same line with a space.
func (f *Formatter) formatFunctionBody(expr ast.Expression) string {
	switch e := expr.(type) {
	case *ast.MatchExpression:
		return f.formatMatchExpressionBody(e)
	case *ast.DoExpression:
		return " " + f.formatExpression(expr)
	default:
		return " " + f.formatExpression(e)
	}
}

func (f *Formatter) formatConstraintDeclaration(s *ast.ConstraintDeclaration) string {
	params := make([]string, len(s.Parameters))
	for i, p := range s.Parameters {
		params[i] = p.Value
	}
	body := f.formatExpression(s.Body)
	return f.indent() + "constraint " + s.Name.Value + "(" + strings.Join(params, ", ") + ") = " + body + ";\n"
}

func (f *Formatter) formatTypeDeclaration(s *ast.TypeDeclaration) string {
	var sb strings.Builder
	sb.WriteString(f.indent() + "type " + s.Name.Value + " =")

	if len(s.Variants) == 0 {
		sb.WriteString(";\n")
		return sb.String()
	}

	for i, v := range s.Variants {
		if i > 0 {
			sb.WriteString(" |")
		}
		sb.WriteString(" " + v.Name)
		if len(v.Fields) > 0 {
			sb.WriteString("(" + strings.Join(v.Fields, ", ") + ")")
		}
	}
	sb.WriteString(";\n")
	return sb.String()
}

func (f *Formatter) formatExpressionStatement(s *ast.ExpressionStatement) string {
	expr := f.formatExpression(s.Expression)
	return f.indent() + expr + ";\n"
}

// formatExpression formats an expression node.
func (f *Formatter) formatExpression(expr ast.Expression) string {
	if expr == nil {
		return ""
	}

	switch e := expr.(type) {
	case *ast.Identifier:
		return e.Value
	case *ast.WildcardExpression:
		return "_"
	case *ast.IntegerLiteral:
		return e.Token.Literal
	case *ast.FloatLiteral:
		return e.Token.Literal
	case *ast.StringLiteral:
		return `"` + escapeString(e.Value) + `"`
	case *ast.InterpolatedString:
		return f.formatInterpolatedString(e)
	case *ast.RegexLiteral:
		return e.String()
	case *ast.BooleanLiteral:
		if e.Value {
			return "true"
		}
		return "false"
	case *ast.TupleLiteral:
		return f.formatTupleLiteral(e)
	case *ast.ArrayLiteral:
		return f.formatArrayLiteral(e)
	case *ast.HashLiteral:
		return f.formatHashLiteral(e)
	case *ast.PrefixExpression:
		return e.Operator + f.formatExpression(e.Right)
	case *ast.InfixExpression:
		return f.formatInfixExpression(e)
	case *ast.CallExpression:
		return f.formatCallExpression(e)
	case *ast.IndexExpression:
		return f.formatExpression(e.Left) + "[" + f.formatExpression(e.Index) + "]"
	case *ast.MemberExpression:
		return f.formatExpression(e.Object) + "." + e.Property.Value
	case *ast.LambdaExpression:
		return f.formatLambdaExpression(e)
	case *ast.PipeExpression:
		return f.formatExpression(e.Left) + " |> " + f.formatExpression(e.Right)
	case *ast.EffectPipeExpression:
		return f.formatExpression(e.Left) + " !> " + f.formatExpression(e.Right)
	case *ast.ErrorPropPipeExpression:
		return f.formatExpression(e.Left) + " |>? " + f.formatExpression(e.Right)
	case *ast.SpreadExpression:
		return f.formatExpression(e.Value) + "..."
	case *ast.MatchExpression:
		return f.formatMatchExpression(e)
	case *ast.EffectHandleExpression:
		return f.formatEffectHandleExpression(e)
	case *ast.SuccessExpression:
		return "success(" + f.formatExpression(e.Value) + ")"
	case *ast.FailureExpression:
		return "failure(" + f.formatExpression(e.Value) + ")"
	case *ast.ReturnExpression:
		return "return " + f.formatExpression(e.Value)
	case *ast.ConstraintCheckExpression:
		return f.formatExpression(e.Constraint) + "?"
	case *ast.DoExpression:
		return f.formatDoExpression(e)
	case *ast.ConstructorPattern:
		return f.formatConstructorPattern(e)
	case *ast.ModulePath:
		return f.formatModulePath(e)
	case *ast.StayExpression:
		return f.formatStayExpression(e)
	default:
		return fmt.Sprintf("/* unknown expr: %T */", expr)
	}
}

func (f *Formatter) formatModulePath(mp *ast.ModulePath) string {
	if mp == nil {
		return ""
	}
	return mp.String()
}

func (f *Formatter) formatInterpolatedString(e *ast.InterpolatedString) string {
	var sb strings.Builder
	sb.WriteString(`"`)

	for _, part := range e.Parts {
		switch p := part.(type) {
		case *ast.StringLiteral:
			sb.WriteString(escapeStringRaw(p.Value))
		default:
			sb.WriteString("${")
			sb.WriteString(f.formatExpression(p))
			sb.WriteString("}")
		}
	}
	sb.WriteString(`"`)
	return sb.String()
}

func (f *Formatter) formatTupleLiteral(e *ast.TupleLiteral) string {
	elems := make([]string, len(e.Elements))
	for i, el := range e.Elements {
		elems[i] = f.formatExpression(el)
	}
	return "(" + strings.Join(elems, ", ") + ")"
}

func (f *Formatter) formatArrayLiteral(e *ast.ArrayLiteral) string {
	if len(e.Elements) == 0 {
		return "[]"
	}
	elems := make([]string, len(e.Elements))
	for i, el := range e.Elements {
		elems[i] = f.formatExpression(el)
	}
	if e.IsConcat {
		return "[" + strings.Join(elems, " ") + "]"
	}
	return "[" + strings.Join(elems, ", ") + "]"
}

func (f *Formatter) formatHashLiteral(e *ast.HashLiteral) string {
	if len(e.Pairs) == 0 {
		return "{}"
	}

	pairs := make([]string, len(e.Pairs))
	for i, pair := range e.Pairs {
		k := f.formatExpression(pair.Key)
		v := f.formatExpression(pair.Value)
		pairs[i] = k + ": " + v
	}

	// If there's only one pair or the result is short, keep on one line
	oneLine := "{ " + strings.Join(pairs, ", ") + " }"
	if len(pairs) <= 3 && len(oneLine) <= 60 {
		return oneLine
	}

	// Otherwise, multi-line
	f.depth++
	ind := f.indent()
	f.depth--
	outerInd := f.indent()

	var sb strings.Builder
	sb.WriteString("{\n")
	for i, pair := range pairs {
		sb.WriteString(ind + pair)
		if i < len(pairs)-1 {
			sb.WriteString(",")
		}
		sb.WriteString("\n")
	}
	sb.WriteString(outerInd + "}")
	return sb.String()
}

func (f *Formatter) formatInfixExpression(e *ast.InfixExpression) string {
	left := f.formatExpression(e.Left)
	right := f.formatExpression(e.Right)
	return left + " " + e.Operator + " " + right
}

func (f *Formatter) formatCallExpression(e *ast.CallExpression) string {
	fn := f.formatExpression(e.Function)
	args := make([]string, len(e.Arguments))
	for i, arg := range e.Arguments {
		args[i] = f.formatExpression(arg)
	}
	return fn + "(" + strings.Join(args, ", ") + ")"
}

func (f *Formatter) formatLambdaExpression(e *ast.LambdaExpression) string {
	body := f.formatExpression(e.Body)

	if len(e.Parameters) == 1 {
		return e.Parameters[0].Value + " => " + body
	}

	params := make([]string, len(e.Parameters))
	for i, p := range e.Parameters {
		params[i] = p.Value
	}
	return "(" + strings.Join(params, ", ") + ") => " + body
}

// formatMatchExpression formats a match expression inline (used when match is itself an expression).
func (f *Formatter) formatMatchExpression(e *ast.MatchExpression) string {
	subject := f.formatExpression(e.Subject)
	var sb strings.Builder
	sb.WriteString("match " + subject)

	f.depth++
	ind := f.indent()
	f.depth--

	for _, c := range e.Cases {
		sb.WriteString("\n" + ind)
		sb.WriteString(f.formatMatchCase(c))
	}
	return sb.String()
}

// formatMatchExpressionBody formats a match expression as a function body.
// Places it on a new line from the `=` sign.
func (f *Formatter) formatMatchExpressionBody(e *ast.MatchExpression) string {
	subject := f.formatExpression(e.Subject)
	var sb strings.Builder
	sb.WriteString(" match " + subject)

	f.depth++
	ind := f.indent()
	f.depth--

	for _, c := range e.Cases {
		sb.WriteString("\n" + ind)
		sb.WriteString(f.formatMatchCase(c))
	}
	return sb.String()
}

func (f *Formatter) formatMatchCase(c *ast.MatchCase) string {
	var pattern string
	if c.IsDefault {
		pattern = "_"
	} else {
		pattern = f.formatExpression(c.Pattern)
	}

	if c.Guard != nil {
		pattern += " if " + f.formatExpression(c.Guard)
	}

	body := f.formatExpression(c.Body)
	return pattern + " => " + body
}

func (f *Formatter) formatEffectHandleExpression(e *ast.EffectHandleExpression) string {
	subject := f.formatExpression(e.Subject)

	f.depth++
	ind := f.indent()
	f.depth--

	successCase := ind + "success(" + e.SuccessVar.Value + ") => " + f.formatExpression(e.SuccessBody)
	failureCase := ind + "failure(" + e.FailureVar.Value + ") => " + f.formatExpression(e.FailureBody)

	return subject + " !? {\n" + successCase + "\n" + failureCase + "\n" + f.indent() + "}"
}

func (f *Formatter) formatDoExpression(e *ast.DoExpression) string {
	var sb strings.Builder
	sb.WriteString("do\n")

	f.depth++
	ind := f.indent()
	f.depth--

	for _, stmt := range e.Statements {
		// Re-format each statement with increased indent
		f.depth++
		formatted := f.formatStatement(stmt)
		f.depth--
		sb.WriteString(formatted)
	}

	if e.FinalExpression != nil {
		sb.WriteString(ind + f.formatExpression(e.FinalExpression) + "\n")
	}

	sb.WriteString(f.indent() + "end")
	return sb.String()
}

func (f *Formatter) formatConstructorPattern(e *ast.ConstructorPattern) string {
	if len(e.Fields) == 0 {
		return e.Name
	}
	fields := make([]string, len(e.Fields))
	for i, field := range e.Fields {
		fields[i] = field.Value
	}
	return e.Name + "(" + strings.Join(fields, ", ") + ")"
}

func (f *Formatter) formatStayExpression(e *ast.StayExpression) string {
	var sb strings.Builder
	sb.WriteString("async.stay(")

	pairs := make([]string, len(e.StateInit))
	for i, p := range e.StateInit {
		pairs[i] = p.Name.Value + ": " + f.formatExpression(p.Value)
	}
	sb.WriteString(strings.Join(pairs, ", "))
	sb.WriteString(") {\n")

	f.depth++
	for _, stmt := range e.Body {
		sb.WriteString(f.formatStatement(stmt))
	}
	f.depth--

	sb.WriteString(f.indent() + "}")
	return sb.String()
}

// escapeString escapes special characters in a string value for output.
func escapeString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\t", `\t`)
	s = strings.ReplaceAll(s, "\r", `\r`)
	return s
}

// escapeStringRaw escapes a raw string segment (already unescaped by parser).
func escapeStringRaw(s string) string {
	return escapeString(s)
}
