// Package eval provides a high-level API for evaluating Calcium source code.
// It is primarily used by the Web Playground WASM module and testing utilities.
package eval

import (
	"fmt"
	"io"
	"strings"

	"github.com/ytnobody/calcium-lang/pkg/compiler"
	"github.com/ytnobody/calcium-lang/pkg/lexer"
	"github.com/ytnobody/calcium-lang/pkg/optimizer"
	"github.com/ytnobody/calcium-lang/pkg/parser"
	"github.com/ytnobody/calcium-lang/pkg/vm"
)

// Result holds the result of evaluating Calcium source code.
type Result struct {
	// Output contains the text written via print/println during execution.
	Output string
	// Error is non-nil if evaluation failed at any stage.
	Error error
}

// Eval evaluates the given Calcium source code and returns a Result.
// Output from print/println is captured and returned in Result.Output.
// If w is non-nil, output is also written to w.
func Eval(source string) Result {
	var sb strings.Builder
	err := evalInto(source, &sb)
	return Result{
		Output: sb.String(),
		Error:  err,
	}
}

// EvalInto evaluates the given Calcium source code and writes output to w.
func EvalInto(source string, w io.Writer) error {
	return evalInto(source, w)
}

func evalInto(source string, w io.Writer) error {
	// Parse
	l := lexer.New(source)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		return fmt.Errorf("parse error:\n%s", strings.Join(p.Errors(), "\n"))
	}

	// Optimize AST
	opt := optimizer.New(optimizer.O1)
	program = opt.OptimizeAST(program)

	// Compile
	c := compiler.New()
	c.SetInput(source)
	c.SetFileName("<playground>")
	if err := c.Compile(program); err != nil {
		return fmt.Errorf("compile error:\n%s", err.Error())
	}

	// Optimize bytecode
	instructions := opt.OptimizeBytecode(c.Bytecode().Instructions)

	// Run
	machine := vm.New(c.Constants())
	machine.SetSourceMap(c.SourceMap())
	machine.SetSourceInput(source)
	machine.SetOutput(w)

	if err := machine.Run(instructions); err != nil {
		return fmt.Errorf("runtime error:\n%s", err.Error())
	}

	return nil
}
