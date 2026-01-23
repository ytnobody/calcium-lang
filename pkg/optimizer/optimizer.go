// Package optimizer provides optimization passes for the Calcium compiler.
// It includes AST-level optimizations (constant folding, dead code elimination)
// and bytecode-level optimizations (peephole optimization).
package optimizer

import (
	"github.com/example/calcium/pkg/ast"
	"github.com/example/calcium/pkg/bytecode"
)

// Level represents the optimization level
type Level int

const (
	// O0 disables all optimizations
	O0 Level = iota
	// O1 enables basic optimizations (constant folding + dead code elimination)
	O1
	// O2 enables all optimizations (O1 + peephole)
	O2
)

// Optimizer holds the optimization configuration
type Optimizer struct {
	Level Level
}

// New creates a new optimizer with the specified level
func New(level Level) *Optimizer {
	return &Optimizer{Level: level}
}

// OptimizeAST applies AST-level optimizations to the program.
// This should be called after parsing and before compilation.
func (o *Optimizer) OptimizeAST(program *ast.Program) *ast.Program {
	if o.Level == O0 {
		return program
	}

	// Apply constant folding first
	optimized := FoldConstants(program)

	// Then apply dead code elimination
	optimized = EliminateDeadCode(optimized)

	return optimized.(*ast.Program)
}

// OptimizeBytecode applies bytecode-level optimizations.
// This should be called after compilation.
func (o *Optimizer) OptimizeBytecode(instructions bytecode.Instructions) bytecode.Instructions {
	if o.Level < O2 {
		return instructions
	}

	return PeepholeOptimize(instructions)
}

// OptimizeProgram is a convenience function that optimizes at level O1 (AST only)
func OptimizeProgram(program *ast.Program) *ast.Program {
	opt := New(O1)
	return opt.OptimizeAST(program)
}

// OptimizeAll is a convenience function that optimizes at level O2 (AST + bytecode)
func OptimizeAll(program *ast.Program, instructions bytecode.Instructions) (*ast.Program, bytecode.Instructions) {
	opt := New(O2)
	optimizedAST := opt.OptimizeAST(program)
	optimizedBytecode := opt.OptimizeBytecode(instructions)
	return optimizedAST, optimizedBytecode
}
