package compiler

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/ytnobody/calcium-lang/pkg/ast"
	"github.com/ytnobody/calcium-lang/pkg/bytecode"
	"github.com/ytnobody/calcium-lang/pkg/token"
	"github.com/ytnobody/calcium-lang/pkg/value"
)

// CompileError represents a compilation error with position information
type CompileError struct {
	Line    int
	Column  int
	Message string
	Source  string // The source line where the error occurred
}

func (e *CompileError) Error() string {
	if e.Source == "" {
		return fmt.Sprintf("line %d, column %d: %s", e.Line, e.Column, e.Message)
	}
	caretLine := strings.Repeat(" ", e.Column-1) + "^"
	return fmt.Sprintf("line %d, column %d:\n  %s\n  %s\n%s", e.Line, e.Column, e.Source, caretLine, e.Message)
}

// CompileErrors represents multiple compilation errors
type CompileErrors struct {
	Errors []error
}

func (e *CompileErrors) Error() string {
	if len(e.Errors) == 1 {
		return e.Errors[0].Error()
	}

	var sb strings.Builder
	for i, err := range e.Errors {
		if i > 0 {
			sb.WriteString("\n\n")
		}
		sb.WriteString(err.Error())
	}
	return sb.String()
}

// Compiler compiles AST to bytecode
type Compiler struct {
	instructions bytecode.Instructions
	constants    []value.Value

	// Symbol table for variable resolution
	symbolTable *SymbolTable

	// Scope management
	scopes     []CompilationScope
	scopeIndex int

	// Source input for error messages
	input string

	// Collected errors for multi-error reporting
	errors []error

	// Source map tracking: current position being compiled
	currentLine   int
	currentColumn int
	fileName      string

	// Source maps per scope (stack mirrors scopes)
	sourceMaps     []*bytecode.SourceMap
	sourceMapIndex int

	// compiledFunctionNames tracks which function names have been compiled at least once
	// in the current top-level scope. Used to detect overload definitions.
	compiledFunctionNames map[string]bool
}

// CompilationScope represents a compilation scope (function/global)
type CompilationScope struct {
	instructions        bytecode.Instructions
	lastInstruction     EmittedInstruction
	previousInstruction EmittedInstruction
	funcName            string // function name for this scope ("" for global)
}

// EmittedInstruction tracks emitted instructions
type EmittedInstruction struct {
	Opcode   bytecode.OpCode
	Position int
}

// Symbol represents a variable binding
type Symbol struct {
	Name     string
	Scope    SymbolScope
	Index    int
	IsEffect bool // true for func! or effect module members
}

// SymbolScope represents where a symbol is defined
type SymbolScope string

const (
	GlobalScope    SymbolScope = "GLOBAL"
	LocalScope     SymbolScope = "LOCAL"
	BuiltinScope   SymbolScope = "BUILTIN"
	FreeScope      SymbolScope = "FREE"
	StayStateScope SymbolScope = "STAY_STATE"
)

// SymbolTable manages variable bindings
type SymbolTable struct {
	Outer       *SymbolTable
	store       map[string]Symbol
	count       int
	FreeSymbols []Symbol // Free variables captured from outer scopes
}

// NewSymbolTable creates a new symbol table
func NewSymbolTable() *SymbolTable {
	return &SymbolTable{
		store: make(map[string]Symbol),
	}
}

// NewEnclosedSymbolTable creates a new enclosed symbol table
func NewEnclosedSymbolTable(outer *SymbolTable) *SymbolTable {
	s := NewSymbolTable()
	s.Outer = outer
	return s
}

// Define defines a new symbol
func (s *SymbolTable) Define(name string) Symbol {
	symbol := Symbol{Name: name, Index: s.count, IsEffect: false}
	if s.Outer == nil {
		symbol.Scope = GlobalScope
	} else {
		symbol.Scope = LocalScope
	}
	s.store[name] = symbol
	s.count++
	return symbol
}

// DefineOrGet returns the existing symbol if the name is already defined in the current scope,
// otherwise defines a new symbol. This is used to handle function overloading where multiple
// function declarations share the same name and thus the same symbol slot.
func (s *SymbolTable) DefineOrGet(name string) (Symbol, bool) {
	if existing, ok := s.store[name]; ok {
		return existing, false // false = already existed
	}
	sym := s.Define(name)
	return sym, true // true = newly defined
}

// DefineEffectOrGet is like DefineOrGet but for effect functions.
func (s *SymbolTable) DefineEffectOrGet(name string) (Symbol, bool) {
	if existing, ok := s.store[name]; ok {
		return existing, false
	}
	sym := s.DefineEffect(name)
	return sym, true
}

// DefineEffect defines a new effect symbol (for func! or effect module members)
func (s *SymbolTable) DefineEffect(name string) Symbol {
	symbol := Symbol{Name: name, Index: s.count, IsEffect: true}
	if s.Outer == nil {
		symbol.Scope = GlobalScope
	} else {
		symbol.Scope = LocalScope
	}
	s.store[name] = symbol
	s.count++
	return symbol
}

// DefineBuiltin defines a builtin symbol
func (s *SymbolTable) DefineBuiltin(index int, name string) Symbol {
	symbol := Symbol{Name: name, Scope: BuiltinScope, Index: index}
	s.store[name] = symbol
	return symbol
}

// DefineStayState defines a stay state variable
// The Index field stores the constant index of the key name
func (s *SymbolTable) DefineStayState(name string, keyConstIndex int) Symbol {
	symbol := Symbol{Name: name, Scope: StayStateScope, Index: keyConstIndex}
	s.store[name] = symbol
	return symbol
}

// Resolve looks up a symbol
func (s *SymbolTable) Resolve(name string) (Symbol, bool) {
	obj, ok := s.store[name]
	if !ok && s.Outer != nil {
		obj, ok = s.Outer.Resolve(name)
		if !ok {
			return obj, ok
		}
		// Global, builtin, and stay state symbols don't need to be captured as free variables
		// Stay state variables are always accessed via OpStayGetState from the current stay loop
		if obj.Scope == GlobalScope || obj.Scope == BuiltinScope || obj.Scope == StayStateScope {
			return obj, ok
		}
		// This is a free variable - capture it
		return s.defineFree(obj), true
	}
	return obj, ok
}

// defineFree adds a free variable to the symbol table
func (s *SymbolTable) defineFree(original Symbol) Symbol {
	s.FreeSymbols = append(s.FreeSymbols, original)
	symbol := Symbol{Name: original.Name, Index: len(s.FreeSymbols) - 1, Scope: FreeScope}
	s.store[original.Name] = symbol
	return symbol
}

// IsDefinedInCurrentScope checks if a name is defined in the current scope (not outer scopes)
func (s *SymbolTable) IsDefinedInCurrentScope(name string) bool {
	_, ok := s.store[name]
	return ok
}

// NumDefinitions returns the number of definitions
func (s *SymbolTable) NumDefinitions() int {
	return s.count
}

// New creates a new Compiler
func New() *Compiler {
	mainScope := CompilationScope{
		instructions:        bytecode.Instructions{},
		lastInstruction:     EmittedInstruction{},
		previousInstruction: EmittedInstruction{},
	}

	symbolTable := NewSymbolTable()

	mainSourceMap := bytecode.NewSourceMap()

	c := &Compiler{
		constants:             []value.Value{},
		symbolTable:           symbolTable,
		scopes:                []CompilationScope{mainScope},
		scopeIndex:            0,
		sourceMaps:            []*bytecode.SourceMap{mainSourceMap},
		sourceMapIndex:        0,
		compiledFunctionNames: make(map[string]bool),
	}

	// Define built-in functions
	builtins := []string{
		"len", "concat", "to_string", "get", "has", "head", "tail", "push", "range",
		"map", "filter", "reduce",
		"keys", "values", "hash_set", "hash_merge",
		// Primitive type constraint checkers (usable as Name? in constraint expressions)
		"Int", "Float", "String", "Bool", "Array", "Hash", "Tuple", "Null",
		"Function", "Number",
	}
	for i, name := range builtins {
		symbolTable.DefineBuiltin(i, name)
	}

	return c
}

// SetInput sets the source input for error messages
func (c *Compiler) SetInput(input string) {
	c.input = input
}

// addError adds an error to the collection
func (c *Compiler) addError(err error) {
	c.errors = append(c.errors, err)
}

// hasErrors returns true if there are any collected errors
func (c *Compiler) hasErrors() bool {
	return len(c.errors) > 0
}

// getErrors returns all collected errors as a single error
func (c *Compiler) getErrors() error {
	if len(c.errors) == 0 {
		return nil
	}
	if len(c.errors) == 1 {
		return c.errors[0]
	}
	return &CompileErrors{Errors: c.errors}
}

// getSourceLine returns the source line at the given line number (1-indexed)
func (c *Compiler) getSourceLine(line int) string {
	if c.input == "" {
		return ""
	}
	lines := strings.Split(c.input, "\n")
	if line < 1 || line > len(lines) {
		return ""
	}
	return lines[line-1]
}

// newCompileError creates a CompileError with source context
func (c *Compiler) newCompileError(tok token.Token, message string) *CompileError {
	return &CompileError{
		Line:    tok.Line,
		Column:  tok.Column,
		Message: message,
		Source:  c.getSourceLine(tok.Line),
	}
}

// newCompileErrorAt creates a CompileError at a specific position
func (c *Compiler) newCompileErrorAt(line, column int, message string) *CompileError {
	return &CompileError{
		Line:    line,
		Column:  column,
		Message: message,
		Source:  c.getSourceLine(line),
	}
}

// levenshteinDistance calculates the edit distance between two strings
func levenshteinDistance(a, b string) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	// Create matrix
	matrix := make([][]int, len(a)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(b)+1)
		matrix[i][0] = i
	}
	for j := 0; j <= len(b); j++ {
		matrix[0][j] = j
	}

	// Fill matrix
	for i := 1; i <= len(a); i++ {
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			matrix[i][j] = min(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[len(a)][len(b)]
}

func min(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

// findSimilarName finds the most similar name from a list of candidates
func findSimilarName(name string, candidates []string, maxDistance int) string {
	bestMatch := ""
	bestDistance := maxDistance + 1

	for _, candidate := range candidates {
		distance := levenshteinDistance(name, candidate)
		if distance <= maxDistance && distance < bestDistance {
			bestDistance = distance
			bestMatch = candidate
		}
	}

	return bestMatch
}

// getAllDefinedNames returns all defined names in the current scope chain
func (c *Compiler) getAllDefinedNames() []string {
	names := []string{}
	seen := make(map[string]bool)

	// Walk up the symbol table chain
	for st := c.symbolTable; st != nil; st = st.Outer {
		for name := range st.store {
			if !seen[name] {
				names = append(names, name)
				seen[name] = true
			}
		}
	}

	return names
}

// isEffectExpression checks if an expression refers to an effect function
// Returns: (isEffect bool, canDetermine bool)
// canDetermine is false if we can't statically determine the effect status
func (c *Compiler) isEffectExpression(expr ast.Expression) (bool, bool) {
	switch e := expr.(type) {
	case *ast.Identifier:
		// Direct identifier - check symbol table
		if symbol, ok := c.symbolTable.Resolve(e.Value); ok {
			return symbol.IsEffect, true
		}
		return false, false

	case *ast.MemberExpression:
		// Member access like io.say - check if the object is an effect module
		if ident, ok := e.Object.(*ast.Identifier); ok {
			if symbol, ok := c.symbolTable.Resolve(ident.Value); ok {
				// If the module is an effect module, its members are effect functions
				return symbol.IsEffect, true
			}
		}
		return false, false

	case *ast.CallExpression:
		// For call expressions in pipe, check the function
		return c.isEffectExpression(e.Function)

	default:
		// Lambda, etc. - can't determine statically
		return false, false
	}
}

// NewWithState creates a compiler with existing state
func NewWithState(symbolTable *SymbolTable, constants []value.Value) *Compiler {
	mainScope := CompilationScope{
		instructions:        bytecode.Instructions{},
		lastInstruction:     EmittedInstruction{},
		previousInstruction: EmittedInstruction{},
	}

	return &Compiler{
		constants:             constants,
		symbolTable:           symbolTable,
		scopes:                []CompilationScope{mainScope},
		scopeIndex:            0,
		sourceMaps:            []*bytecode.SourceMap{bytecode.NewSourceMap()},
		sourceMapIndex:        0,
		compiledFunctionNames: make(map[string]bool),
	}
}

// Compile compiles a program
func (c *Compiler) Compile(node ast.Node) error {
	switch node := node.(type) {
	case *ast.Program:
		// First pass: register all function and constraint declarations
		// This enables forward references (functions calling other functions defined later).
		// For overloaded functions (same name, different signatures), we register the name
		// only once so all overloads share the same symbol slot.
		for _, s := range node.Statements {
			switch stmt := s.(type) {
			case *ast.FunctionDeclaration:
				if stmt.IsEffect {
					c.symbolTable.DefineEffectOrGet(stmt.Name.Value)
				} else {
					c.symbolTable.DefineOrGet(stmt.Name.Value)
				}
			case *ast.ConstraintDeclaration:
				c.symbolTable.DefineOrGet(stmt.Name.Value)
			case *ast.TypeDeclaration:
				// Register all variant constructors
				for _, v := range stmt.Variants {
					c.symbolTable.DefineOrGet(v.Name)
				}
			}
		}

		// Second pass: compile all statements
		for _, s := range node.Statements {
			err := c.Compile(s)
			if err != nil {
				c.addError(err)
				// Continue to find more errors at statement level
			}
		}
		// Return collected errors if any
		if c.hasErrors() {
			return c.getErrors()
		}
		return nil

	case *ast.ExpressionStatement:
		c.setPosFromToken(node.Token)
		err := c.Compile(node.Expression)
		if err != nil {
			return err
		}
		c.emit(bytecode.OpPop)

	case *ast.AssignmentStatement:
		c.setPosFromToken(node.Token)
		// Check if variable already exists in current scope (reassignment is forbidden)
		if c.symbolTable.IsDefinedInCurrentScope(node.Name.Value) {
			return &CompileError{
				Line:    node.Token.Line,
				Column:  node.Token.Column,
				Message: fmt.Sprintf("variable `%s` is already bound and cannot be reassigned", node.Name.Value),
			}
		}
		// New variable
		symbol := c.symbolTable.Define(node.Name.Value)
		err := c.Compile(node.Value)
		if err != nil {
			return err
		}
		if symbol.Scope == GlobalScope {
			c.emit(bytecode.OpSetGlobal, symbol.Index)
		} else {
			c.emit(bytecode.OpSetLocal, symbol.Index)
		}

	case *ast.ArrayDestructuringStatement:
		// Compile: [a, b, c] = arr;
		// Strategy: evaluate arr, then extract elements by index
		err := c.Compile(node.Value)
		if err != nil {
			return err
		}

		// For each variable, duplicate array, push index, get element, assign
		for i, name := range node.Names {
			// Duplicate array on stack (except for last one)
			if i < len(node.Names)-1 {
				c.emit(bytecode.OpDup)
			}

			// Push index
			c.emit(bytecode.OpConstant, c.addConstant(value.Int(int64(i))))

			// Get element (handles null for out of bounds)
			c.emit(bytecode.OpIndex)

			// Define variable and assign (reassignment is forbidden)
			if c.symbolTable.IsDefinedInCurrentScope(name.Value) {
				return &CompileError{
					Line:    node.Token.Line,
					Column:  node.Token.Column,
					Message: fmt.Sprintf("variable `%s` is already bound and cannot be reassigned", name.Value),
				}
			}
			symbol := c.symbolTable.Define(name.Value)
			if symbol.Scope == GlobalScope {
				c.emit(bytecode.OpSetGlobal, symbol.Index)
			} else {
				c.emit(bytecode.OpSetLocal, symbol.Index)
			}
		}

	case *ast.HeadTailDestructuringStatement:
		// Compile: [head | tail] = arr;
		// Strategy: evaluate arr twice - once for head, once for tail

		// Get head (arr[0])
		err := c.Compile(node.Value)
		if err != nil {
			return err
		}
		c.emit(bytecode.OpConstant, c.addConstant(value.Int(0)))
		c.emit(bytecode.OpIndex)

		// Assign head
		headSymbol, ok := c.symbolTable.Resolve(node.Head.Value)
		if !ok {
			headSymbol = c.symbolTable.Define(node.Head.Value)
		}
		if headSymbol.Scope == GlobalScope {
			c.emit(bytecode.OpSetGlobal, headSymbol.Index)
		} else {
			c.emit(bytecode.OpSetLocal, headSymbol.Index)
		}

		// Get tail: push tail builtin, then array, then call
		c.emit(bytecode.OpBuiltin, 6) // tail is at index 6
		err = c.Compile(node.Value)
		if err != nil {
			return err
		}
		c.emit(bytecode.OpCall, 1)

		// Assign tail
		tailSymbol, ok := c.symbolTable.Resolve(node.Tail.Value)
		if !ok {
			tailSymbol = c.symbolTable.Define(node.Tail.Value)
		}
		if tailSymbol.Scope == GlobalScope {
			c.emit(bytecode.OpSetGlobal, tailSymbol.Index)
		} else {
			c.emit(bytecode.OpSetLocal, tailSymbol.Index)
		}

	case *ast.FunctionDeclaration:
		c.setPosFromToken(node.Token)
		// Resolve the function name (already registered in first pass for forward references)
		// If not found, define it (for nested scopes or REPL)
		symbol, ok := c.symbolTable.Resolve(node.Name.Value)
		if !ok {
			if node.IsEffect {
				symbol = c.symbolTable.DefineEffect(node.Name.Value)
			} else {
				symbol = c.symbolTable.Define(node.Name.Value)
			}
		}

		c.enterScopeWithName(node.Name.Value)

		// For recursion: the function can reference itself
		// We need to resolve the function name from the outer scope
		// The outer scope's symbol is still accessible

		// Define parameters as local variables
		for _, param := range node.Parameters {
			c.symbolTable.Define(param.Value)
		}

		// Check parameter constraints at function entry
		hasConstraints := false
		for i, constraint := range node.Constraints {
			if constraint != nil {
				hasConstraints = true
				paramSymbol, _ := c.symbolTable.Resolve(node.Parameters[i].Value)

				// Load parameter value
				c.emit(bytecode.OpGetLocal, paramSymbol.Index)

				// Load constraint function (from outer scope)
				constraintSymbol, ok := c.symbolTable.Resolve(constraint.Value)
				if !ok {
					msg := fmt.Sprintf("undefined constraint '%s'", constraint.Value)
					candidates := c.getAllDefinedNames()
					if similar := findSimilarName(constraint.Value, candidates, 2); similar != "" {
						msg += fmt.Sprintf(" (did you mean '%s'?)", similar)
					}
					return c.newCompileError(constraint.Token, msg)
				}
				c.loadSymbol(constraintSymbol)

				// Load parameter value again for the call
				c.emit(bytecode.OpGetLocal, paramSymbol.Index)

				// Call constraint(value)
				c.emit(bytecode.OpCall, 1)

				// Check constraint result - if false, return failure
				jumpIfTruePos := c.emit(bytecode.OpJumpIfTrue, 9999)

				// Constraint failed - return failure(value)
				c.emit(bytecode.OpPop) // pop the saved value
				c.emit(bytecode.OpGetLocal, paramSymbol.Index)
				c.emit(bytecode.OpWrapFailure)
				c.emit(bytecode.OpReturn)

				// Constraint passed - continue
				afterPos := len(c.currentInstructions())
				c.changeOperand(jumpIfTruePos, afterPos)
				c.emit(bytecode.OpPop) // pop the saved value
			}
		}

		// Also enforce type annotations as constraints when a constraint function
		// with the same name exists (builtin like Int, String, etc. or user-defined).
		// This allows: func sum(a: Int, b: Int) = a + b; to enforce Int constraint.
		for i, typeAnnot := range node.ParamTypes {
			if typeAnnot != nil && node.Constraints[i] == nil {
				// Only if no explicit constraint already set for this parameter
				constraintSymbol, ok := c.symbolTable.Resolve(typeAnnot.Name)
				if ok {
					hasConstraints = true
					paramSymbol, _ := c.symbolTable.Resolve(node.Parameters[i].Value)

					// Load parameter value (saved for failure message)
					c.emit(bytecode.OpGetLocal, paramSymbol.Index)

					// Load constraint function
					c.loadSymbol(constraintSymbol)

					// Load parameter value for the call
					c.emit(bytecode.OpGetLocal, paramSymbol.Index)

					// Call constraint(value)
					c.emit(bytecode.OpCall, 1)

					// Check constraint result - if false, return failure
					jumpIfTruePos := c.emit(bytecode.OpJumpIfTrue, 9999)

					// Constraint failed - return failure(value)
					c.emit(bytecode.OpPop) // pop the saved value
					c.emit(bytecode.OpGetLocal, paramSymbol.Index)
					c.emit(bytecode.OpWrapFailure)
					c.emit(bytecode.OpReturn)

					// Constraint passed - continue
					afterPos := len(c.currentInstructions())
					c.changeOperand(jumpIfTruePos, afterPos)
					c.emit(bytecode.OpPop) // pop the saved value
				}
			}
		}

		// Compile function body
		err := c.Compile(node.Body)
		if err != nil {
			return err
		}

		// Check return constraint if specified (explicit constraint or type annotation)
		// Determine the effective return constraint name:
		// 1. Explicit ReturnConstraint (e.g., func add(a, b): Positive = ...)
		// 2. ReturnType annotation with a resolvable constraint (e.g., func add(a, b): Int = ...)
		var returnConstraintName string
		var returnConstraintToken token.Token
		if node.ReturnConstraint != nil {
			returnConstraintName = node.ReturnConstraint.Value
			returnConstraintToken = node.ReturnConstraint.Token
		} else if node.ReturnType != nil {
			// Check if the return type name resolves to a constraint function
			if _, ok := c.symbolTable.Resolve(node.ReturnType.Name); ok {
				returnConstraintName = node.ReturnType.Name
				returnConstraintToken = node.ReturnType.Token
			}
		}

		hasReturnConstraint := returnConstraintName != ""
		if hasReturnConstraint {
			hasConstraints = true

			// Stack has: [return_value]
			// We need to: dup value, call constraint, check result

			// Store return value in a temporary local
			tmpSymbol := c.symbolTable.Define("__return_tmp__")
			c.emit(bytecode.OpSetLocal, tmpSymbol.Index)

			// Load constraint function (from outer scope)
			constraintSymbol, ok := c.symbolTable.Resolve(returnConstraintName)
			if !ok {
				msg := fmt.Sprintf("undefined return constraint '%s'", returnConstraintName)
				candidates := c.getAllDefinedNames()
				if similar := findSimilarName(returnConstraintName, candidates, 2); similar != "" {
					msg += fmt.Sprintf(" (did you mean '%s'?)", similar)
				}
				return c.newCompileError(returnConstraintToken, msg)
			}
			c.loadSymbol(constraintSymbol)

			// Load return value for the call
			c.emit(bytecode.OpGetLocal, tmpSymbol.Index)

			// Call constraint(return_value)
			c.emit(bytecode.OpCall, 1)

			// Check constraint result - if true, continue to wrap in success
			// OpJumpIfTrue pops the condition value from the stack
			jumpIfTruePos := c.emit(bytecode.OpJumpIfTrue, 9999)

			// Constraint failed - return failure(value)
			c.emit(bytecode.OpGetLocal, tmpSymbol.Index)
			c.emit(bytecode.OpWrapFailure)
			c.emit(bytecode.OpReturn)

			// Constraint passed - push the original value for wrapping in success
			afterPos := len(c.currentInstructions())
			c.changeOperand(jumpIfTruePos, afterPos)
			c.emit(bytecode.OpGetLocal, tmpSymbol.Index)
		}

		// For effect functions (func!), wrap return value in success
		// Also wrap if function has parameter constraints or return constraint (makes it return success/failure)
		if node.IsEffect || hasConstraints {
			c.emit(bytecode.OpWrapSuccess)
		}

		// Return the last value
		c.replaceLastPopWithReturn()

		numLocals := c.symbolTable.NumDefinitions()
		freeSymbols := c.symbolTable.FreeSymbols
		instructions, fnSourceMap := c.leaveScopeWithSourceMap()

		// Create function object
		fn := &value.Function{
			Name:           node.Name.Value,
			Parameters:     make([]string, len(node.Parameters)),
			ParamTypeNames: make([]string, len(node.Parameters)),
			Body:           instructions,
			NumLocals:      numLocals,
			IsEffect:       node.IsEffect || hasConstraints,
			SourceMap:      fnSourceMap,
		}
		for i, p := range node.Parameters {
			fn.Parameters[i] = p.Value
		}
		// Record parameter type annotations for overload resolution
		for i, pt := range node.ParamTypes {
			if pt != nil {
				fn.ParamTypeNames[i] = pt.Name
			}
		}

		fnIndex := c.addConstant(value.Func(fn))

		// Determine if this is an overloaded definition (same name compiled before)
		isOverload := c.compiledFunctionNames[node.Name.Value]
		c.compiledFunctionNames[node.Name.Value] = true

		if isOverload {
			// Load the existing value (closure or overloaded closure) from its symbol slot
			if symbol.Scope == GlobalScope {
				c.emit(bytecode.OpGetGlobal, symbol.Index)
			} else {
				c.emit(bytecode.OpGetLocal, symbol.Index)
			}
		}

		// Emit instructions to load free variables onto stack
		for _, s := range freeSymbols {
			c.loadSymbol(s)
		}
		c.emit(bytecode.OpClosure, fnIndex, len(freeSymbols))

		if isOverload {
			// Merge the existing function and new closure into an overloaded closure
			c.emit(bytecode.OpAddOverload)
		}

		// Store the function (or overloaded function) in the pre-defined symbol
		if symbol.Scope == GlobalScope {
			c.emit(bytecode.OpSetGlobal, symbol.Index)
		} else {
			c.emit(bytecode.OpSetLocal, symbol.Index)
		}

	case *ast.ConstraintDeclaration:
		c.setPosFromToken(node.Token)
		// Constraints are compiled like functions
		c.enterScopeWithName(node.Name.Value)

		for _, param := range node.Parameters {
			c.symbolTable.Define(param.Value)
		}

		err := c.Compile(node.Body)
		if err != nil {
			return err
		}

		c.replaceLastPopWithReturn()

		numLocals := c.symbolTable.NumDefinitions()
		freeSymbols := c.symbolTable.FreeSymbols
		instructions, constraintSourceMap := c.leaveScopeWithSourceMap()

		fn := &value.Function{
			Name:       node.Name.Value,
			Parameters: make([]string, len(node.Parameters)),
			Body:       instructions,
			NumLocals:  numLocals,
			IsEffect:   false,
			SourceMap:  constraintSourceMap,
		}
		for i, p := range node.Parameters {
			fn.Parameters[i] = p.Value
		}

		fnIndex := c.addConstant(value.Func(fn))

		// Emit instructions to load free variables onto stack
		for _, s := range freeSymbols {
			c.loadSymbol(s)
		}
		c.emit(bytecode.OpClosure, fnIndex, len(freeSymbols))

		symbol := c.symbolTable.Define(node.Name.Value)
		if symbol.Scope == GlobalScope {
			c.emit(bytecode.OpSetGlobal, symbol.Index)
		} else {
			c.emit(bytecode.OpSetLocal, symbol.Index)
		}

	case *ast.TypeDeclaration:
		// For each variant, create a constructor function or constant
		typeName := node.Name.Value
		for _, variant := range node.Variants {
			if len(variant.Fields) == 0 {
				// Zero-arity constructor: just create an ADT constant value
				adtVal := value.ADTVal(&value.ADT{
					TypeName: typeName,
					Tag:      variant.Name,
					Values:   []value.Value{},
				})
				constIdx := c.addConstant(adtVal)
				c.emit(bytecode.OpConstant, constIdx)
			} else {
				// N-arity constructor: create a function that constructs the ADT value
				c.enterScope()

				// Define parameters
				for _, field := range variant.Fields {
					c.symbolTable.Define(field)
				}

				// Emit OpConstructADT with tag name and arity
				tagIdx := c.addConstant(value.String(typeName + "." + variant.Name))
				c.emit(bytecode.OpConstructADT, tagIdx, len(variant.Fields))
				c.emit(bytecode.OpReturn)

				numLocals := c.symbolTable.NumDefinitions()
				instructions := c.leaveScope()

				fn := &value.Function{
					Name:       variant.Name,
					Parameters: make([]string, len(variant.Fields)),
					Body:       instructions,
					NumLocals:  numLocals,
					IsEffect:   false,
				}
				copy(fn.Parameters, variant.Fields)

				fnIndex := c.addConstant(value.Func(fn))
				c.emit(bytecode.OpClosure, fnIndex, 0)
			}

			symbol := c.symbolTable.Define(variant.Name)
			if symbol.Scope == GlobalScope {
				c.emit(bytecode.OpSetGlobal, symbol.Index)
			} else {
				c.emit(bytecode.OpSetLocal, symbol.Index)
			}
		}

	case *ast.IntegerLiteral:
		integer := value.Int(node.Value)
		c.emit(bytecode.OpConstant, c.addConstant(integer))

	case *ast.FloatLiteral:
		float := value.Float(node.Value)
		c.emit(bytecode.OpConstant, c.addConstant(float))

	case *ast.StringLiteral:
		str := value.String(node.Value)
		c.emit(bytecode.OpConstant, c.addConstant(str))

	case *ast.InterpolatedString:
		// Compile interpolated string as concatenation of parts
		// VM auto-converts non-strings to strings during concatenation
		if len(node.Parts) == 0 {
			// Empty interpolated string
			c.emit(bytecode.OpConstant, c.addConstant(value.String("")))
		} else {
			// Compile each part
			for i, part := range node.Parts {
				err := c.Compile(part)
				if err != nil {
					return err
				}

				// Concatenate with previous parts
				if i > 0 {
					c.emit(bytecode.OpAdd) // String concatenation (auto-converts)
				}
			}
		}

	case *ast.RegexLiteral:
		// Convert Calcium flags to Go regex flags
		goPattern := convertRegexFlags(node.Pattern, node.Flags)

		// Compile regex at compile time
		re, err := regexp.Compile(goPattern)
		if err != nil {
			return c.newCompileError(node.Token, fmt.Sprintf("invalid regex /%s/: %s", node.Pattern, err))
		}

		// Store compiled regex in constants
		regexVal := value.RegexVal(&value.Regex{
			Pattern: node.Pattern,
			Flags:   node.Flags,
			Re:      re,
		})
		c.emit(bytecode.OpConstant, c.addConstant(regexVal))

	case *ast.BooleanLiteral:
		if node.Value {
			c.emit(bytecode.OpTrue)
		} else {
			c.emit(bytecode.OpFalse)
		}

	case *ast.Identifier:
		c.setPosFromToken(node.Token)
		symbol, ok := c.symbolTable.Resolve(node.Value)
		if !ok {
			msg := fmt.Sprintf("undefined variable '%s'", node.Value)
			// Try to find a similar name for suggestion
			candidates := c.getAllDefinedNames()
			if similar := findSimilarName(node.Value, candidates, 2); similar != "" {
				msg += fmt.Sprintf(" (did you mean '%s'?)", similar)
			} else if similar := token.SuggestKeyword(node.Value, 2); similar != "" {
				// Also check for keyword typos
				msg += fmt.Sprintf(" (did you mean '%s'?)", similar)
			}
			return c.newCompileError(node.Token, msg)
		}
		c.loadSymbol(symbol)

	case *ast.PrefixExpression:
		err := c.Compile(node.Right)
		if err != nil {
			return err
		}

		switch node.Operator {
		case "-":
			c.emit(bytecode.OpNeg)
		case "!":
			c.emit(bytecode.OpNot)
		default:
			return c.newCompileError(node.Token, fmt.Sprintf("unknown prefix operator '%s'", node.Operator))
		}

	case *ast.InfixExpression:
		c.setPosFromToken(node.Token)
		// Handle short-circuit operators specially
		if node.Operator == "&&" {
			return c.compileAnd(node)
		}
		if node.Operator == "||" {
			return c.compileOr(node)
		}

		err := c.Compile(node.Left)
		if err != nil {
			return err
		}

		err = c.Compile(node.Right)
		if err != nil {
			return err
		}

		switch node.Operator {
		case "+":
			c.emit(bytecode.OpAdd)
		case "-":
			c.emit(bytecode.OpSub)
		case "*":
			c.emit(bytecode.OpMul)
		case "/":
			c.emit(bytecode.OpDiv)
		case "%":
			c.emit(bytecode.OpMod)
		case "**":
			c.emit(bytecode.OpPow)
		case "==":
			c.emit(bytecode.OpEqual)
		case "!=":
			c.emit(bytecode.OpNotEqual)
		case "<":
			c.emit(bytecode.OpLessThan)
		case ">":
			c.emit(bytecode.OpGreaterThan)
		case "<=":
			c.emit(bytecode.OpLessEqual)
		case ">=":
			c.emit(bytecode.OpGreaterEqual)
		case "~":
			c.emit(bytecode.OpRegexMatch)
		default:
			return c.newCompileError(node.Token, fmt.Sprintf("unknown infix operator '%s'", node.Operator))
		}

	case *ast.ArrayLiteral:
		for _, elem := range node.Elements {
			err := c.Compile(elem)
			if err != nil {
				return err
			}
		}
		if node.IsConcat {
			c.emit(bytecode.OpArrayConcat, len(node.Elements))
		} else {
			c.emit(bytecode.OpArray, len(node.Elements))
		}

	case *ast.TupleLiteral:
		for _, elem := range node.Elements {
			err := c.Compile(elem)
			if err != nil {
				return err
			}
		}
		c.emit(bytecode.OpTuple, len(node.Elements))

	case *ast.HashLiteral:
		// Compile each key-value pair
		for _, pair := range node.Pairs {
			err := c.Compile(pair.Key)
			if err != nil {
				return err
			}
			err = c.Compile(pair.Value)
			if err != nil {
				return err
			}
		}
		c.emit(bytecode.OpHash, len(node.Pairs))

	case *ast.IndexExpression:
		err := c.Compile(node.Left)
		if err != nil {
			return err
		}
		err = c.Compile(node.Index)
		if err != nil {
			return err
		}
		c.emit(bytecode.OpIndex)

	case *ast.CallExpression:
		c.setPosFromToken(node.Token)
		err := c.Compile(node.Function)
		if err != nil {
			return err
		}

		for _, arg := range node.Arguments {
			err := c.Compile(arg)
			if err != nil {
				return err
			}
		}

		c.emit(bytecode.OpCall, len(node.Arguments))

	case *ast.LambdaExpression:
		c.setPosFromToken(node.Token)
		c.enterScopeWithName("<lambda>")

		for _, param := range node.Parameters {
			c.symbolTable.Define(param.Value)
		}

		err := c.Compile(node.Body)
		if err != nil {
			return err
		}

		c.replaceLastPopWithReturn()

		numLocals := c.symbolTable.NumDefinitions()
		freeSymbols := c.symbolTable.FreeSymbols
		instructions, lambdaSourceMap := c.leaveScopeWithSourceMap()

		fn := &value.Function{
			Name:       "<lambda>",
			Parameters: make([]string, len(node.Parameters)),
			Body:       instructions,
			NumLocals:  numLocals,
			IsEffect:   false,
			SourceMap:  lambdaSourceMap,
		}
		for i, p := range node.Parameters {
			fn.Parameters[i] = p.Value
		}

		fnIndex := c.addConstant(value.Func(fn))

		// Emit instructions to load free variables onto stack
		for _, s := range freeSymbols {
			c.loadSymbol(s)
		}
		c.emit(bytecode.OpClosure, fnIndex, len(freeSymbols))

	case *ast.PipeExpression:
		// expr |> func is equivalent to func(expr)
		// Stack order for call: [func, arg1, arg2, ...]

		// Check if right side is an effect function - should use !> instead
		funcExpr := node.Right
		if call, ok := node.Right.(*ast.CallExpression); ok {
			funcExpr = call.Function
		}
		if isEffect, canDetermine := c.isEffectExpression(funcExpr); canDetermine && isEffect {
			return c.newCompileError(node.Token,
				"cannot use '|>' with effect function; use '!>' instead")
		}

		// Check if left side is a spread expression
		if spread, ok := node.Left.(*ast.SpreadExpression); ok {
			// arr... |> func -> call func with spread array elements
			// Compile the function first
			err := c.Compile(node.Right)
			if err != nil {
				return err
			}

			// Compile the array (not the spread, just the inner value)
			err = c.Compile(spread.Value)
			if err != nil {
				return err
			}

			// Use OpCallSpread to call with spread array
			c.emit(bytecode.OpCallSpread)
			return nil
		}

		// Check if right side is a constraint check: value |> Constraint?
		if cc, ok := node.Right.(*ast.ConstraintCheckExpression); ok {
			// Compile: value, constraint, value, call, wrap
			// Stack after:
			// 1. [value]
			// 2. [value, constraint]
			// 3. [value, constraint, value]
			// 4. [value, result] (after call)
			// 5. [success/failure] (after wrap)

			// Push value (save for later)
			err := c.Compile(node.Left)
			if err != nil {
				return err
			}

			// Push constraint function
			err = c.Compile(cc.Constraint)
			if err != nil {
				return err
			}

			// Push value again (for the call)
			err = c.Compile(node.Left)
			if err != nil {
				return err
			}

			// Call constraint(value)
			c.emit(bytecode.OpCall, 1)

			// Wrap result based on constraint check
			c.emit(bytecode.OpCheckConstraint)
			return nil
		}

		// Check if right side is a call expression: x |> f(y) -> f(x, y)
		// The pipe input becomes the FIRST argument
		if call, ok := node.Right.(*ast.CallExpression); ok {
			// Compile the function
			err := c.Compile(call.Function)
			if err != nil {
				return err
			}

			// Compile the pipe input as first argument
			err = c.Compile(node.Left)
			if err != nil {
				return err
			}

			// Compile the rest of the arguments
			for _, arg := range call.Arguments {
				err = c.Compile(arg)
				if err != nil {
					return err
				}
			}

			// Call with 1 + len(args) arguments
			c.emit(bytecode.OpCall, 1+len(call.Arguments))
			return nil
		}

		// Simple pipe: x |> f -> f(x)
		err := c.Compile(node.Right)
		if err != nil {
			return err
		}

		err = c.Compile(node.Left)
		if err != nil {
			return err
		}

		// Call with 1 argument
		c.emit(bytecode.OpCall, 1)

	case *ast.EffectPipeExpression:
		// expr !> func! is equivalent to func!(expr), returns success/failure
		// Effect functions automatically wrap their return in success
		// Stack order for call: [func, arg1, arg2, ...]

		// Check if right side is NOT an effect function - should use |> instead
		funcExpr := node.Right
		if call, ok := node.Right.(*ast.CallExpression); ok {
			funcExpr = call.Function
		}
		if isEffect, canDetermine := c.isEffectExpression(funcExpr); canDetermine && !isEffect {
			return c.newCompileError(node.Token,
				"cannot use '!>' with pure function; use '|>' instead")
		}

		// Compile the right side (effect function) first
		err := c.Compile(node.Right)
		if err != nil {
			return err
		}

		// Then compile the left side (argument)
		err = c.Compile(node.Left)
		if err != nil {
			return err
		}

		// Just call normally - func! already wraps return in success
		c.emit(bytecode.OpCall, 1)

	case *ast.ErrorPropPipeExpression:
		// expr |>? func: call func(expr), if result is failure return early, otherwise unwrap success
		// Equivalent to:
		//   result = func(expr)
		//   if result is failure: return result  (early return from enclosing function)
		//   else: unwrap result (success value)
		//
		// Also supports |>? func(extra_args) -> func(expr, extra_args)
		if call, ok := node.Right.(*ast.CallExpression); ok {
			// Compile function
			err := c.Compile(call.Function)
			if err != nil {
				return err
			}
			// Compile pipe input as first arg
			err = c.Compile(node.Left)
			if err != nil {
				return err
			}
			// Compile extra args
			for _, arg := range call.Arguments {
				err = c.Compile(arg)
				if err != nil {
					return err
				}
			}
			c.emit(bytecode.OpCall, 1+len(call.Arguments))
		} else {
			// Simple: func(expr)
			err := c.Compile(node.Right)
			if err != nil {
				return err
			}
			err = c.Compile(node.Left)
			if err != nil {
				return err
			}
			c.emit(bytecode.OpCall, 1)
		}

		// Now the result is on the stack. Check if it is a failure.
		// Stack: [..., result]
		c.emit(bytecode.OpDup)       // Stack: [..., result, result]
		c.emit(bytecode.OpIsFailure) // Stack: [..., result, bool]
		// If NOT failure (bool is false), jump past the early return
		skipReturn := c.emit(bytecode.OpJumpIfFalse, 9999) // Stack: [..., result]
		// It's a failure: return it immediately from the enclosing function
		c.emit(bytecode.OpReturn)
		// Patch jump target (after the OpReturn)
		afterReturn := len(c.currentInstructions())
		c.changeOperand(skipReturn, afterReturn)
		// It's a success: unwrap the value
		c.emit(bytecode.OpUnwrap) // Stack: [..., value]

	case *ast.SpreadExpression:
		err := c.Compile(node.Value)
		if err != nil {
			return err
		}
		c.emit(bytecode.OpSpread)

	case *ast.ConstraintCheckExpression:
		// Constraint? creates a function that checks the constraint
		// When used with pipe: value |> Constraint?
		// This compiles to: push constraint, then OpCheckConstraint
		err := c.Compile(node.Constraint)
		if err != nil {
			return err
		}
		c.emit(bytecode.OpCheckConstraint)

	case *ast.MemberExpression:
		err := c.Compile(node.Object)
		if err != nil {
			return err
		}
		nameIndex := c.addConstant(value.String(node.Property.Value))
		c.emit(bytecode.OpGetMember, nameIndex)

	case *ast.MatchExpression:
		err := c.compileMatch(node)
		if err != nil {
			return err
		}

	case *ast.SuccessExpression:
		err := c.Compile(node.Value)
		if err != nil {
			return err
		}
		c.emit(bytecode.OpWrapSuccess)

	case *ast.FailureExpression:
		err := c.Compile(node.Value)
		if err != nil {
			return err
		}
		c.emit(bytecode.OpWrapFailure)

	case *ast.EffectHandleExpression:
		err := c.compileEffectHandle(node)
		if err != nil {
			return err
		}

	case *ast.WildcardExpression:
		// Wildcard matches anything, push true
		c.emit(bytecode.OpTrue)

	case *ast.ReturnExpression:
		err := c.Compile(node.Value)
		if err != nil {
			return err
		}
		c.emit(bytecode.OpReturn)

	case *ast.UseStatement:
		// Build the module path string based on the format
		var modulePath string
		var moduleName string

		if node.Path.IsRemote {
			if node.Path.RawURL != "" {
				// URL format: "github.com/author/repo"
				modulePath = node.Path.RawURL
				// Extract module name from URL (last path component)
				parts := strings.Split(node.Path.RawURL, "/")
				moduleName = parts[len(parts)-1]
				// Remove -calcium suffix if present
				moduleName = strings.TrimSuffix(moduleName, "-calcium")
			} else {
				// author/module format: "ytnobody/json"
				modulePath = node.Path.Author + "/" + node.Path.Name
				moduleName = node.Path.Name
			}
		} else {
			// Standard dotted format: "core.io"
			modulePath = strings.Join(node.Path.Parts, ".")
			moduleName = node.Path.Parts[len(node.Path.Parts)-1]
		}

		// Emit OpLoadModule with the module path
		pathIndex := c.addConstant(value.String(modulePath))
		c.emit(bytecode.OpLoadModule, pathIndex)

		// The module is stored with the extracted module name
		var symbol Symbol
		if node.IsEffect {
			// Effect modules (use xxx!) - members are effect functions
			symbol = c.symbolTable.DefineEffect(moduleName)
		} else {
			symbol = c.symbolTable.Define(moduleName)
		}
		if symbol.Scope == GlobalScope {
			c.emit(bytecode.OpSetGlobal, symbol.Index)
		} else {
			c.emit(bytecode.OpSetLocal, symbol.Index)
		}

	case *ast.NamespaceDeclaration:
		// Namespace declarations don't generate bytecode
		return nil

	case *ast.StayExpression:
		err := c.compileStayExpression(node)
		if err != nil {
			return err
		}

	case *ast.DoExpression:
		// Compile do...end block: execute all intermediate statements (each pops its value),
		// then compile the final expression whose value remains on the stack.
		for _, stmt := range node.Statements {
			err := c.Compile(stmt)
			if err != nil {
				return err
			}
		}
		// Compile the final expression (its value stays on the stack as the block result)
		err := c.Compile(node.FinalExpression)
		if err != nil {
			return err
		}

	default:
		// For unknown node types, we don't have a token, so use a basic error
		return fmt.Errorf("unknown node type: %T", node)
	}

	return nil
}

func (c *Compiler) compileAnd(node *ast.InfixExpression) error {
	// Compile left side
	err := c.Compile(node.Left)
	if err != nil {
		return err
	}

	// Duplicate for short-circuit return value
	c.emit(bytecode.OpDup)

	// Jump to end if false (short-circuit)
	// JumpIfFalse pops the duplicate, leaving original on stack
	jumpPos := c.emit(bytecode.OpJumpIfFalse, 9999)

	// Not short-circuiting: pop the remaining duplicate
	c.emit(bytecode.OpPop)

	// Compile right side (this becomes the result)
	err = c.Compile(node.Right)
	if err != nil {
		return err
	}

	// Patch jump to end
	afterPos := len(c.currentInstructions())
	c.changeOperand(jumpPos, afterPos)

	return nil
}

func (c *Compiler) compileOr(node *ast.InfixExpression) error {
	// Compile left side
	err := c.Compile(node.Left)
	if err != nil {
		return err
	}

	// Duplicate for short-circuit return value
	c.emit(bytecode.OpDup)

	// Jump to end if true (short-circuit)
	// JumpIfTrue pops the duplicate, leaving original on stack
	jumpPos := c.emit(bytecode.OpJumpIfTrue, 9999)

	// Not short-circuiting: pop the remaining duplicate
	c.emit(bytecode.OpPop)

	// Compile right side (this becomes the result)
	err = c.Compile(node.Right)
	if err != nil {
		return err
	}

	// Patch jump to end
	afterPos := len(c.currentInstructions())
	c.changeOperand(jumpPos, afterPos)

	return nil
}

func (c *Compiler) compileMatch(node *ast.MatchExpression) error {
	// Compile subject
	err := c.Compile(node.Subject)
	if err != nil {
		return err
	}

	jumpToEndPositions := []int{}

	for i, mc := range node.Cases {
		isLast := i == len(node.Cases)-1

		if mc.Guard != nil {
			// Guard case: pattern if guard => body
			// The pattern is an identifier that binds the subject value.
			// 1. Bind subject to the pattern variable
			// 2. Evaluate guard condition
			// 3. If guard is false, unbind and jump to next case

			// Duplicate subject for binding
			c.emit(bytecode.OpDup)

			// Bind the pattern identifier as a local variable
			ident, ok := mc.Pattern.(*ast.Identifier)
			if !ok {
				return fmt.Errorf("guard clause requires a variable pattern (e.g., x if x > 0)")
			}
			guardSymbol := c.symbolTable.Define(ident.Value)
			if guardSymbol.Scope == GlobalScope {
				c.emit(bytecode.OpSetGlobal, guardSymbol.Index)
			} else {
				c.emit(bytecode.OpSetLocal, guardSymbol.Index)
			}

			// Evaluate guard condition
			err := c.Compile(mc.Guard)
			if err != nil {
				return err
			}

			var jumpToNextPos int
			if !isLast {
				// Jump to next case if guard fails
				jumpToNextPos = c.emit(bytecode.OpJumpIfFalse, 9999)
			}

			// Guard passed: pop subject, compile body
			c.emit(bytecode.OpPop)

			err = c.Compile(mc.Body)
			if err != nil {
				return err
			}

			if !isLast {
				jumpToEndPositions = append(jumpToEndPositions, c.emit(bytecode.OpJump, 9999))
				// Patch jump to next case
				afterPos := len(c.currentInstructions())
				c.changeOperand(jumpToNextPos, afterPos)
			}
		} else if cp, ok := mc.Pattern.(*ast.ConstructorPattern); ok {
			// Constructor pattern: Some(x) => body
			// Duplicate subject for tag comparison
			c.emit(bytecode.OpDup)

			// Emit OpMatchADT which pops dup, checks tag, pushes true/false
			tagIdx := c.addConstant(value.String(cp.Name))
			c.emit(bytecode.OpMatchADT, tagIdx, len(cp.Fields))

			var jumpToNextPos int
			if !isLast {
				// JumpIfFalse pops the bool
				jumpToNextPos = c.emit(bytecode.OpJumpIfFalse, 9999)
			} else {
				// Last case: pop the bool explicitly
				c.emit(bytecode.OpPop)
			}

			// Stack now: [subject]
			// Bind fields: extract values from the ADT and bind to local variables
			for i, field := range cp.Fields {
				if field.Value != "_" {
					// Duplicate subject again to extract field
					c.emit(bytecode.OpDup)
					// Push index
					idxConst := c.addConstant(value.Int(int64(i)))
					c.emit(bytecode.OpConstant, idxConst)
					c.emit(bytecode.OpIndex)
					// Bind to variable
					sym := c.symbolTable.Define(field.Value)
					if sym.Scope == GlobalScope {
						c.emit(bytecode.OpSetGlobal, sym.Index)
					} else {
						c.emit(bytecode.OpSetLocal, sym.Index)
					}
				}
			}

			// Pop subject
			c.emit(bytecode.OpPop)

			// Compile body
			err = c.Compile(mc.Body)
			if err != nil {
				return err
			}

			if !isLast {
				jumpToEndPositions = append(jumpToEndPositions, c.emit(bytecode.OpJump, 9999))
				afterPos := len(c.currentInstructions())
				c.changeOperand(jumpToNextPos, afterPos)
			}
		} else {
			// Standard case (no guard)

			// Duplicate subject for comparison
			c.emit(bytecode.OpDup)

			if mc.IsDefault {
				// Default case: always matches, pop the duplicate
				c.emit(bytecode.OpPop)
			} else {
				// Compile pattern/condition
				err := c.Compile(mc.Pattern)
				if err != nil {
					return err
				}

				// Compare
				c.emit(bytecode.OpEqual)
			}

			var jumpToNextPos int
			if !isLast && !mc.IsDefault {
				// Jump to next case if no match
				jumpToNextPos = c.emit(bytecode.OpJumpIfFalse, 9999)
			}

			// Pop subject (matched)
			c.emit(bytecode.OpPop)

			// Compile body
			err = c.Compile(mc.Body)
			if err != nil {
				return err
			}

			if !isLast {
				// Jump to end after body
				jumpToEndPositions = append(jumpToEndPositions, c.emit(bytecode.OpJump, 9999))
			}

			if !isLast && !mc.IsDefault {
				// Patch jump to next case
				afterPos := len(c.currentInstructions())
				c.changeOperand(jumpToNextPos, afterPos)
			}
		}
	}

	// Patch all jumps to end
	afterPos := len(c.currentInstructions())
	for _, pos := range jumpToEndPositions {
		c.changeOperand(pos, afterPos)
	}

	return nil
}

func (c *Compiler) compileEffectHandle(node *ast.EffectHandleExpression) error {
	// Compile subject
	err := c.Compile(node.Subject)
	if err != nil {
		return err
	}

	// Check if success, jump to failure handler if not
	c.emit(bytecode.OpDup)
	c.emit(bytecode.OpIsSuccess)
	jumpToFailurePos := c.emit(bytecode.OpJumpIfFalse, 9999)

	// Success case
	c.emit(bytecode.OpUnwrap)

	// Bind to success variable
	if node.SuccessVar != nil {
		symbol := c.symbolTable.Define(node.SuccessVar.Value)
		if symbol.Scope == GlobalScope {
			c.emit(bytecode.OpSetGlobal, symbol.Index)
			c.emit(bytecode.OpGetGlobal, symbol.Index)
		} else {
			c.emit(bytecode.OpSetLocal, symbol.Index)
			c.emit(bytecode.OpGetLocal, symbol.Index)
		}
	}

	// Compile success body
	if node.SuccessBody != nil {
		c.emit(bytecode.OpPop)
		err = c.Compile(node.SuccessBody)
		if err != nil {
			return err
		}
	}

	// Jump to end
	jumpToEndPos := c.emit(bytecode.OpJump, 9999)

	// Failure case
	afterSuccess := len(c.currentInstructions())
	c.changeOperand(jumpToFailurePos, afterSuccess)

	c.emit(bytecode.OpUnwrap)

	// Bind to failure variable
	if node.FailureVar != nil {
		symbol := c.symbolTable.Define(node.FailureVar.Value)
		if symbol.Scope == GlobalScope {
			c.emit(bytecode.OpSetGlobal, symbol.Index)
			c.emit(bytecode.OpGetGlobal, symbol.Index)
		} else {
			c.emit(bytecode.OpSetLocal, symbol.Index)
			c.emit(bytecode.OpGetLocal, symbol.Index)
		}
	}

	// Compile failure body
	if node.FailureBody != nil {
		c.emit(bytecode.OpPop)
		err = c.Compile(node.FailureBody)
		if err != nil {
			return err
		}
	}

	// Patch jump to end
	afterFailure := len(c.currentInstructions())
	c.changeOperand(jumpToEndPos, afterFailure)

	return nil
}

func (c *Compiler) compileStayExpression(node *ast.StayExpression) error {
	// Compile state initialization values and keys
	// Stack will have: [key1, val1, key2, val2, ...]
	// Also collect key constant indices for later symbol definition
	keyIndices := make([]int, len(node.StateInit))
	for i, pair := range node.StateInit {
		// Push key as string
		keyIndex := c.addConstant(value.String(pair.Name.Value))
		keyIndices[i] = keyIndex
		c.emit(bytecode.OpConstant, keyIndex)

		// Push value
		err := c.Compile(pair.Value)
		if err != nil {
			return err
		}
	}

	// Emit OpStayBegin with number of state pairs
	c.emit(bytecode.OpStayBegin, len(node.StateInit))

	// Define state variables with StayStateScope
	// They will be accessed via OpStayGetState using the key constant index
	for i, pair := range node.StateInit {
		c.symbolTable.DefineStayState(pair.Name.Value, keyIndices[i])
	}

	// Compile body statements
	for _, stmt := range node.Body {
		err := c.Compile(stmt)
		if err != nil {
			return err
		}
	}

	// Emit OpStayEnd to start the event loop
	c.emit(bytecode.OpStayEnd)

	return nil
}

func (c *Compiler) loadSymbol(s Symbol) {
	switch s.Scope {
	case GlobalScope:
		c.emit(bytecode.OpGetGlobal, s.Index)
	case LocalScope:
		c.emit(bytecode.OpGetLocal, s.Index)
	case BuiltinScope:
		c.emit(bytecode.OpBuiltin, s.Index)
	case FreeScope:
		c.emit(bytecode.OpGetFree, s.Index)
	case StayStateScope:
		// Index contains the constant index of the key name
		c.emit(bytecode.OpStayGetState, s.Index)
	}
}

func (c *Compiler) addConstant(v value.Value) int {
	c.constants = append(c.constants, v)
	return len(c.constants) - 1
}

func (c *Compiler) emit(op bytecode.OpCode, operands ...int) int {
	ins := bytecode.Make(op, operands...)
	pos := c.addInstruction(ins)

	c.setLastInstruction(op, pos)

	// Record source position in current scope's source map
	if c.currentLine > 0 && c.sourceMapIndex < len(c.sourceMaps) {
		sm := c.sourceMaps[c.sourceMapIndex]
		sm.Add(pos, c.currentLine, c.currentColumn)
	}

	return pos
}

// setPos updates the current source position being compiled
func (c *Compiler) setPos(line, column int) {
	c.currentLine = line
	c.currentColumn = column
}

// setPosFromToken updates the current source position from a token
func (c *Compiler) setPosFromToken(tok token.Token) {
	c.currentLine = tok.Line
	c.currentColumn = tok.Column
}

// SetFileName sets the source file name for error messages and source maps
func (c *Compiler) SetFileName(name string) {
	c.fileName = name
	if c.sourceMapIndex < len(c.sourceMaps) {
		c.sourceMaps[c.sourceMapIndex].File = name
	}
}

// SourceMap returns the source map for the main (global) scope
func (c *Compiler) SourceMap() *bytecode.SourceMap {
	if len(c.sourceMaps) > 0 {
		return c.sourceMaps[0]
	}
	return nil
}

// currentSourceMap returns the source map for the current scope
func (c *Compiler) currentSourceMap() *bytecode.SourceMap {
	if c.sourceMapIndex < len(c.sourceMaps) {
		return c.sourceMaps[c.sourceMapIndex]
	}
	return nil
}

func (c *Compiler) addInstruction(ins []byte) int {
	posNewInstruction := len(c.currentInstructions())
	c.scopes[c.scopeIndex].instructions = append(c.currentInstructions(), ins...)
	return posNewInstruction
}

func (c *Compiler) currentInstructions() bytecode.Instructions {
	return c.scopes[c.scopeIndex].instructions
}

func (c *Compiler) setLastInstruction(op bytecode.OpCode, pos int) {
	c.scopes[c.scopeIndex].previousInstruction = c.scopes[c.scopeIndex].lastInstruction
	c.scopes[c.scopeIndex].lastInstruction = EmittedInstruction{Opcode: op, Position: pos}
}

func (c *Compiler) lastInstructionIs(op bytecode.OpCode) bool {
	if len(c.currentInstructions()) == 0 {
		return false
	}
	return c.scopes[c.scopeIndex].lastInstruction.Opcode == op
}

func (c *Compiler) removeLastPop() {
	last := c.scopes[c.scopeIndex].lastInstruction
	prev := c.scopes[c.scopeIndex].previousInstruction

	c.scopes[c.scopeIndex].instructions = c.currentInstructions()[:last.Position]
	c.scopes[c.scopeIndex].lastInstruction = prev
}

func (c *Compiler) replaceLastPopWithReturn() {
	if c.lastInstructionIs(bytecode.OpPop) {
		c.removeLastPop()
	}
	// Check if the last instruction (before Return) is OpCall
	// If so, replace it with OpTailCall for tail call optimization
	c.replaceLastCallWithTailCall()
	c.emit(bytecode.OpReturn)
}

// replaceLastCallWithTailCall replaces the last OpCall with OpTailCall
// for tail call optimization. This is called when the call is in tail position
// (i.e., the last expression in a function body, right before OpReturn).
func (c *Compiler) replaceLastCallWithTailCall() {
	// Only optimize if we're inside a function scope (not global)
	if c.scopeIndex == 0 {
		return
	}

	last := c.scopes[c.scopeIndex].lastInstruction
	if last.Opcode == bytecode.OpCall {
		// Replace OpCall with OpTailCall in-place
		c.currentInstructions()[last.Position] = byte(bytecode.OpTailCall)
		c.scopes[c.scopeIndex].lastInstruction.Opcode = bytecode.OpTailCall
	}
}

func (c *Compiler) changeOperand(opPos int, operand int) {
	op := bytecode.OpCode(c.currentInstructions()[opPos])
	newInstruction := bytecode.Make(op, operand)

	for i := 0; i < len(newInstruction); i++ {
		c.scopes[c.scopeIndex].instructions[opPos+i] = newInstruction[i]
	}
}

func (c *Compiler) enterScope() {
	c.enterScopeWithName("")
}

func (c *Compiler) enterScopeWithName(funcName string) {
	scope := CompilationScope{
		instructions:        bytecode.Instructions{},
		lastInstruction:     EmittedInstruction{},
		previousInstruction: EmittedInstruction{},
		funcName:            funcName,
	}
	c.scopes = append(c.scopes, scope)
	c.scopeIndex++

	sm := bytecode.NewSourceMap()
	sm.File = c.fileName
	c.sourceMaps = append(c.sourceMaps, sm)
	c.sourceMapIndex++

	c.symbolTable = NewEnclosedSymbolTable(c.symbolTable)
}

func (c *Compiler) leaveScope() bytecode.Instructions {
	instructions := c.currentInstructions()

	c.scopes = c.scopes[:len(c.scopes)-1]
	c.scopeIndex--
	c.symbolTable = c.symbolTable.Outer

	return instructions
}

// leaveScopeWithSourceMap returns both instructions and source map for the scope
func (c *Compiler) leaveScopeWithSourceMap() (bytecode.Instructions, *bytecode.SourceMap) {
	instructions := c.currentInstructions()
	var sm *bytecode.SourceMap
	if c.sourceMapIndex < len(c.sourceMaps) {
		sm = c.sourceMaps[c.sourceMapIndex]
	}

	c.scopes = c.scopes[:len(c.scopes)-1]
	c.scopeIndex--
	c.sourceMaps = c.sourceMaps[:len(c.sourceMaps)-1]
	c.sourceMapIndex--
	c.symbolTable = c.symbolTable.Outer

	return instructions, sm
}

// Bytecode returns the compiled bytecode
func (c *Compiler) Bytecode() *bytecode.Bytecode {
	return &bytecode.Bytecode{
		Instructions: c.currentInstructions(),
		Constants:    make([]interface{}, len(c.constants)),
	}
}

// Constants returns the constants
func (c *Compiler) Constants() []value.Value {
	return c.constants
}

// ResetInstructions clears the instructions for REPL use
// This keeps constants and symbol table but resets the instruction buffer
func (c *Compiler) ResetInstructions() {
	c.scopes[c.scopeIndex].instructions = bytecode.Instructions{}
	c.scopes[c.scopeIndex].lastInstruction = EmittedInstruction{}
	c.scopes[c.scopeIndex].previousInstruction = EmittedInstruction{}
	// Reset source map for current scope
	if c.sourceMapIndex < len(c.sourceMaps) {
		c.sourceMaps[c.sourceMapIndex] = bytecode.NewSourceMap()
		c.sourceMaps[c.sourceMapIndex].File = c.fileName
	}
}

// SymbolTable returns the symbol table
func (c *Compiler) SymbolTable() *SymbolTable {
	return c.symbolTable
}

// ExportedSymbols returns all global scope symbols (for module exports)
func (c *Compiler) ExportedSymbols() map[string]Symbol {
	return c.symbolTable.GlobalSymbols()
}

// GlobalSymbols returns all symbols with GlobalScope
func (s *SymbolTable) GlobalSymbols() map[string]Symbol {
	result := make(map[string]Symbol)
	for name, sym := range s.store {
		if sym.Scope == GlobalScope {
			result[name] = sym
		}
	}
	return result
}

// convertRegexFlags converts Calcium regex flags to Go's (?flags) prefix format
func convertRegexFlags(pattern, flags string) string {
	if flags == "" {
		return pattern
	}

	prefix := ""
	for _, f := range flags {
		switch f {
		case 'i': // case-insensitive
			prefix += "i"
		case 'm': // multiline: ^ and $ match line boundaries
			prefix += "m"
		case 's': // single-line: . matches newline
			prefix += "s"
			// 'g' (global) is not a Go regex flag, handled at match time
		}
	}

	if prefix != "" {
		return "(?" + prefix + ")" + pattern
	}
	return pattern
}
