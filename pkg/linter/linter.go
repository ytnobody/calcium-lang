// Package linter performs static analysis of Calcium source programs. It is
// the engine behind the `calcium lint` command.
//
// It reports three categories of findings:
//   - parse errors (the program cannot be parsed at all)
//   - type errors (gradual type annotations that conflict with usage)
//   - static compile errors (static constraint violations, unknown symbols,
//     invalid structure, etc.)
//
// Usage:
//
//	res := linter.Lint(source)
//	for _, f := range res.Findings {
//	    fmt.Printf("%d:%d [%s] %s\n", f.Line, f.Column, f.Rule, f.Message)
//	}
package linter

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/ytnobody/calcium-lang/pkg/compiler"
	"github.com/ytnobody/calcium-lang/pkg/lexer"
	"github.com/ytnobody/calcium-lang/pkg/parser"
	"github.com/ytnobody/calcium-lang/pkg/typechecker"
)

// Severity describes how severe a finding is.
type Severity int

const (
	// SeverityError indicates the program will not compile or run correctly.
	SeverityError Severity = iota
	// SeverityWarning indicates a possible problem that does not stop compilation.
	SeverityWarning
	// SeverityInfo indicates an informational remark.
	SeverityInfo
)

func (s Severity) String() string {
	switch s {
	case SeverityError:
		return "error"
	case SeverityWarning:
		return "warning"
	case SeverityInfo:
		return "info"
	}
	return "unknown"
}

// Finding is a single diagnostic produced by Lint.
type Finding struct {
	Severity Severity
	// Rule names the analysis that produced the finding: "parse", "type",
	// "static" or "compile".
	Rule string
	// Line and Column are 1-based source positions; 0 when unknown.
	Line   int
	Column int
	// Message is a human-readable description.
	Message string
}

// Result is the outcome of linting a single source file.
type Result struct {
	Findings []Finding
}

// positionHeader is the prefix parser error messages use to carry the source
// position, e.g. "line 12, column 5:".
var positionHeader = regexp.MustCompile(`^line (\d+), column (\d+):`)

// Lint runs all static analyses on the given Calcium source and returns the
// resulting findings. An empty Findings slice means nothing was flagged.
func Lint(source string) Result {
	var findings []Finding

	l := lexer.New(source)
	p := parser.New(l)
	program := p.ParseProgram()

	if parseErrs := p.Errors(); len(parseErrs) > 0 {
		// Parse failed: further analyses are meaningless. The parse error
		// strings already carry the source position.
		for _, e := range parseErrs {
			line, col, msg := extractPosition(e)
			findings = append(findings, Finding{
				Severity: SeverityError,
				Rule:     "parse",
				Line:     line,
				Column:   col,
				Message:  msg,
			})
		}
		return Result{Findings: findings}
	}

	// Gradual type checking (annotations + inference).
	for _, e := range typechecker.Check(program) {
		findings = append(findings, Finding{
			Severity: SeverityError,
			Rule:     "type",
			Line:     e.Line,
			Column:   e.Column,
			Message:  e.Message,
		})
	}

	// Compile to surface static constraint violations and other compile errors
	// (constant propagation, function aliases, return constraints, etc.).
	comp := compiler.New()
	comp.SetInput(source)
	if err := comp.Compile(program); err != nil {
		findings = append(findings, compileFindings(err)...)
	}

	return Result{Findings: findings}
}

// compileFindings converts a compiler error (which may wrap several) into
// individual findings.
func compileFindings(err error) []Finding {
	var out []Finding
	var walk func(error)
	walk = func(e error) {
		switch t := e.(type) {
		case *compiler.CompileErrors:
			for _, sub := range t.Errors {
				walk(sub)
			}
		case *compiler.CompileError:
			out = append(out, Finding{
				Severity: SeverityError,
				Rule:     "static",
				Line:     t.Line,
				Column:   t.Column,
				Message:  t.Message,
			})
		default:
			out = append(out, Finding{
				Severity: SeverityError,
				Rule:     "compile",
				Message:  e.Error(),
			})
		}
	}
	walk(err)
	return out
}

// extractPosition pulls the line/column header off a parser error message and
// returns the position plus the remaining message text.
func extractPosition(msg string) (line, col int, rest string) {
	m := positionHeader.FindStringSubmatch(msg)
	if m == nil {
		return 0, 0, msg
	}
	line, _ = strconv.Atoi(m[1])
	col, _ = strconv.Atoi(m[2])
	rest = strings.TrimSpace(msg[len(m[0]):])
	return line, col, rest
}
