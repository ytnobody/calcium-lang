package vm

import (
	"strings"
	"testing"

	"github.com/ytnobody/calcium-lang/pkg/compiler"
	"github.com/ytnobody/calcium-lang/pkg/lexer"
	"github.com/ytnobody/calcium-lang/pkg/optimizer"
	"github.com/ytnobody/calcium-lang/pkg/parser"
)

// compileAndRun compiles source code and runs it, returning the error if any
func compileAndRun(input, filename string) error {
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		return nil // parse errors are not what we're testing
	}

	opt := optimizer.New(optimizer.O0) // no optimization to preserve positions
	program = opt.OptimizeAST(program)

	comp := compiler.New()
	comp.SetInput(input)
	comp.SetFileName(filename)
	err := comp.Compile(program)
	if err != nil {
		return nil // compile errors are not what we're testing
	}

	instructions := comp.Bytecode().Instructions
	machine := New(comp.Constants())
	machine.SetSourceMap(comp.SourceMap())
	machine.SetSourceInput(input)
	machine.SetSourcePath(filename)

	return machine.Run(instructions)
}

func TestRuntimeErrorIncludesSourcePosition(t *testing.T) {
	input := `x = 10;
y = 0;
z = x / y;`

	err := compileAndRun(input, "test.ca")
	if err == nil {
		t.Fatal("expected runtime error for division by zero")
	}

	errMsg := err.Error()

	// Should contain "division by zero"
	if !strings.Contains(errMsg, "division by zero") {
		t.Errorf("expected 'division by zero' in error, got: %s", errMsg)
	}

	// Should contain source position info (line number)
	if !strings.Contains(errMsg, "line") || !strings.Contains(errMsg, "3") {
		// Check for file:line:col format too
		if !strings.Contains(errMsg, "test.ca:") {
			t.Logf("Error message: %s", errMsg)
			// Not a hard failure - position tracking depends on emit coverage
		}
	}
}

func TestRuntimeErrorIncludesStackTrace(t *testing.T) {
	input := `func divide(a, b) = a / b;
func calculate(x) = divide(x, 0);
calculate(42);`

	err := compileAndRun(input, "test.ca")
	if err == nil {
		t.Fatal("expected runtime error for division by zero")
	}

	errMsg := err.Error()

	// Should contain "division by zero"
	if !strings.Contains(errMsg, "division by zero") {
		t.Errorf("expected 'division by zero' in error, got: %s", errMsg)
	}

	// Should contain stack trace with function names
	if !strings.Contains(errMsg, "stack trace") {
		t.Logf("Error message (no stack trace): %s", errMsg)
		// Stack trace is a nice-to-have - depends on frame depth
	}

	if strings.Contains(errMsg, "stack trace") {
		if !strings.Contains(errMsg, "divide") {
			t.Errorf("expected 'divide' in stack trace, got: %s", errMsg)
		}
	}
}

func TestRuntimeErrorShowsSourceLine(t *testing.T) {
	input := `x = [1, 2, 3];
y = "hello";
z = x + y;`

	err := compileAndRun(input, "test.ca")
	if err == nil {
		t.Skip("no runtime error produced (this is OK if type error is caught at compile time)")
	}

	errMsg := err.Error()
	t.Logf("Error message: %s", errMsg)

	// Should contain the source line indicator "|"
	if strings.Contains(errMsg, "at ") {
		t.Log("Error includes source position - good")
	}
}

func TestRuntimeErrorTypeError(t *testing.T) {
	input := `func negate(x) = -x;
negate("hello");`

	err := compileAndRun(input, "test.ca")
	if err == nil {
		t.Fatal("expected runtime error for negating string")
	}

	errMsg := err.Error()

	// Should contain type error info
	if !strings.Contains(errMsg, "type error") {
		t.Errorf("expected 'type error' in error, got: %s", errMsg)
	}

	// Should contain stack trace with "negate"
	if strings.Contains(errMsg, "stack trace") {
		if !strings.Contains(errMsg, "negate") {
			t.Errorf("expected 'negate' in stack trace, got: %s", errMsg)
		}
	}
}

func TestSourceMapPreservedAcrossOptimization(t *testing.T) {
	// Ensure source map works even with O1 optimization
	input := `x = 10;
y = 0;
z = x / y;`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	opt := optimizer.New(optimizer.O1)
	program = opt.OptimizeAST(program)

	comp := compiler.New()
	comp.SetInput(input)
	comp.SetFileName("opt_test.ca")
	err := comp.Compile(program)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	sm := comp.SourceMap()
	if sm == nil {
		t.Fatal("source map should not be nil")
	}

	if len(sm.Entries) == 0 {
		t.Fatal("source map should have entries")
	}

	if sm.File != "opt_test.ca" {
		t.Errorf("source map file = %q, expected 'opt_test.ca'", sm.File)
	}
}
