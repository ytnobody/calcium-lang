package compiler

import (
	"strings"
	"testing"

	"github.com/ytnobody/calcium-lang/pkg/ast"
	"github.com/ytnobody/calcium-lang/pkg/bytecode"
	"github.com/ytnobody/calcium-lang/pkg/lexer"
	"github.com/ytnobody/calcium-lang/pkg/parser"
	"github.com/ytnobody/calcium-lang/pkg/token"
	"github.com/ytnobody/calcium-lang/pkg/value"
)

// parse helper: parses calcium source code into an AST program
func parse(input string) *ast.Program {
	l := lexer.New(input)
	p := parser.New(l)
	return p.ParseProgram()
}

// compileSource helper: parses and compiles source, returning the compiler and any error
func compileSource(input string) (*Compiler, error) {
	program := parse(input)
	comp := New()
	comp.SetInput(input)
	err := comp.Compile(program)
	return comp, err
}

// mustCompile helper: compiles source and fails test on error
func mustCompile(t *testing.T, input string) *Compiler {
	t.Helper()
	comp, err := compileSource(input)
	if err != nil {
		t.Fatalf("compile error for %q: %s", input, err)
	}
	return comp
}

// ---------------------------------------------------------------------------
// Compiler creation
// ---------------------------------------------------------------------------

func TestNew(t *testing.T) {
	comp := New()
	if comp == nil {
		t.Fatal("New() returned nil")
	}
	if comp.symbolTable == nil {
		t.Fatal("symbolTable is nil")
	}
	if len(comp.constants) != 0 {
		t.Fatalf("expected 0 constants, got %d", len(comp.constants))
	}
	if comp.scopeIndex != 0 {
		t.Fatalf("expected scopeIndex 0, got %d", comp.scopeIndex)
	}
	// Check that builtins are defined
	builtins := []string{"len", "concat", "to_string", "get", "has", "head", "tail", "push", "range", "map", "filter", "reduce", "keys", "values", "hash_set", "hash_merge"}
	for _, name := range builtins {
		if _, ok := comp.symbolTable.Resolve(name); !ok {
			t.Errorf("builtin %q not defined", name)
		}
	}
}

func TestNewWithState(t *testing.T) {
	st := NewSymbolTable()
	st.Define("x")
	consts := []value.Value{value.Int(42)}

	comp := NewWithState(st, consts)
	if comp == nil {
		t.Fatal("NewWithState() returned nil")
	}
	if len(comp.constants) != 1 {
		t.Fatalf("expected 1 constant, got %d", len(comp.constants))
	}
	if comp.symbolTable != st {
		t.Fatal("symbolTable not set correctly")
	}
}

func TestSetInput(t *testing.T) {
	comp := New()
	comp.SetInput("hello\nworld")
	if comp.input != "hello\nworld" {
		t.Fatalf("input not set correctly")
	}
}

func TestSetFileName(t *testing.T) {
	comp := New()
	comp.SetFileName("test.ca")
	if comp.fileName != "test.ca" {
		t.Fatalf("fileName not set correctly")
	}
	sm := comp.SourceMap()
	if sm == nil {
		t.Fatal("SourceMap() returned nil")
	}
	if sm.File != "test.ca" {
		t.Fatalf("expected source map file 'test.ca', got %q", sm.File)
	}
}

// ---------------------------------------------------------------------------
// Literal compilation
// ---------------------------------------------------------------------------

func TestCompileIntegerLiteral(t *testing.T) {
	comp := mustCompile(t, "42;")
	consts := comp.Constants()
	if len(consts) != 1 {
		t.Fatalf("expected 1 constant, got %d", len(consts))
	}
	if consts[0].AsInt() != 42 {
		t.Errorf("expected constant 42, got %d", consts[0].AsInt())
	}
	bc := comp.Bytecode()
	if len(bc.Instructions) == 0 {
		t.Fatal("expected non-empty instructions")
	}
}

func TestCompileFloatLiteral(t *testing.T) {
	comp := mustCompile(t, "3.14;")
	consts := comp.Constants()
	if len(consts) != 1 {
		t.Fatalf("expected 1 constant, got %d", len(consts))
	}
	if consts[0].AsFloat() != 3.14 {
		t.Errorf("expected constant 3.14, got %f", consts[0].AsFloat())
	}
}

func TestCompileStringLiteral(t *testing.T) {
	comp := mustCompile(t, `"hello";`)
	consts := comp.Constants()
	if len(consts) != 1 {
		t.Fatalf("expected 1 constant, got %d", len(consts))
	}
	if consts[0].AsString() != "hello" {
		t.Errorf("expected constant 'hello', got %q", consts[0].AsString())
	}
}

func TestCompileBooleanLiteral(t *testing.T) {
	tests := []struct {
		input string
	}{
		{"true;"},
		{"false;"},
	}
	for _, tt := range tests {
		comp := mustCompile(t, tt.input)
		bc := comp.Bytecode()
		if len(bc.Instructions) == 0 {
			t.Errorf("expected non-empty instructions for %q", tt.input)
		}
	}
}

// ---------------------------------------------------------------------------
// Arithmetic operations
// ---------------------------------------------------------------------------

func TestCompileArithmeticOperations(t *testing.T) {
	tests := []struct {
		input         string
		expectedConst int
	}{
		{"1 + 2;", 2},
		{"3 - 1;", 2},
		{"2 * 3;", 2},
		{"6 / 2;", 2},
		{"7 % 3;", 2},
		{"2 ** 3;", 2},
	}
	for _, tt := range tests {
		comp := mustCompile(t, tt.input)
		consts := comp.Constants()
		if len(consts) != tt.expectedConst {
			t.Errorf("for %q: expected %d constants, got %d", tt.input, tt.expectedConst, len(consts))
		}
	}
}

// ---------------------------------------------------------------------------
// Comparison operations
// ---------------------------------------------------------------------------

func TestCompileComparisonOperations(t *testing.T) {
	tests := []string{
		"1 == 2;",
		"1 != 2;",
		"1 < 2;",
		"1 > 2;",
		"1 <= 2;",
		"1 >= 2;",
	}
	for _, input := range tests {
		comp := mustCompile(t, input)
		if len(comp.Bytecode().Instructions) == 0 {
			t.Errorf("expected non-empty instructions for %q", input)
		}
	}
}

// ---------------------------------------------------------------------------
// Prefix expressions
// ---------------------------------------------------------------------------

func TestCompilePrefixExpressions(t *testing.T) {
	tests := []string{
		"-5;",
		"!true;",
		"!false;",
	}
	for _, input := range tests {
		comp := mustCompile(t, input)
		if len(comp.Bytecode().Instructions) == 0 {
			t.Errorf("expected non-empty instructions for %q", input)
		}
	}
}

// ---------------------------------------------------------------------------
// Variable binding (assignment)
// ---------------------------------------------------------------------------

func TestCompileAssignment(t *testing.T) {
	comp := mustCompile(t, "x = 42;")
	consts := comp.Constants()
	if len(consts) != 1 {
		t.Fatalf("expected 1 constant, got %d", len(consts))
	}

	// x should be defined as global
	sym, ok := comp.SymbolTable().Resolve("x")
	if !ok {
		t.Fatal("variable 'x' not defined")
	}
	if sym.Scope != GlobalScope {
		t.Errorf("expected GlobalScope, got %s", sym.Scope)
	}
}

func TestCompileMultipleAssignments(t *testing.T) {
	comp := mustCompile(t, "x = 1; y = 2; z = x + y;")
	consts := comp.Constants()
	if len(consts) < 2 {
		t.Fatalf("expected at least 2 constants, got %d", len(consts))
	}
}

func TestCompileReassignmentError(t *testing.T) {
	_, err := compileSource("x = 1; x = 2;")
	if err == nil {
		t.Fatal("expected error for reassignment, got nil")
	}
	if !strings.Contains(err.Error(), "already bound") {
		t.Errorf("expected 'already bound' error, got: %s", err)
	}
}

// ---------------------------------------------------------------------------
// Function declarations
// ---------------------------------------------------------------------------

func TestCompileFunctionDeclaration(t *testing.T) {
	tests := []struct {
		input string
		name  string
	}{
		{"func five() = 5;", "five"},
		{"func double(x) = x * 2;", "double"},
		{"func add(a, b) = a + b;", "add"},
	}
	for _, tt := range tests {
		comp := mustCompile(t, tt.input)
		sym, ok := comp.SymbolTable().Resolve(tt.name)
		if !ok {
			t.Errorf("function %q not defined for input %q", tt.name, tt.input)
		}
		if sym.Scope != GlobalScope {
			t.Errorf("expected GlobalScope for %q, got %s", tt.name, sym.Scope)
		}
	}
}

func TestCompileFunctionCallExpression(t *testing.T) {
	input := "func double(x) = x * 2; double(5);"
	comp := mustCompile(t, input)
	bc := comp.Bytecode()
	if len(bc.Instructions) == 0 {
		t.Fatal("expected non-empty instructions")
	}
}

func TestCompileRecursiveFunction(t *testing.T) {
	input := `
		func factorial(n) =
			match n
				0 => 1
				_ => n * factorial(n - 1);
	`
	comp := mustCompile(t, input)
	if comp == nil {
		t.Fatal("compilation returned nil")
	}
}

func TestCompileMultipleFunctions(t *testing.T) {
	input := `
		func outer() = 1;
		func middle() = outer();
		func inner() = middle();
	`
	comp := mustCompile(t, input)
	for _, name := range []string{"outer", "middle", "inner"} {
		if _, ok := comp.SymbolTable().Resolve(name); !ok {
			t.Errorf("function %q not defined", name)
		}
	}
}

// ---------------------------------------------------------------------------
// Lambda expressions
// ---------------------------------------------------------------------------

func TestCompileLambdaExpression(t *testing.T) {
	tests := []string{
		"f = x => x * 2;",
		"f = (x, y) => x + y;",
		"f = () => 42;",
	}
	for _, input := range tests {
		comp := mustCompile(t, input)
		if comp == nil {
			t.Fatalf("compilation returned nil for %q", input)
		}
	}
}

// ---------------------------------------------------------------------------
// Array literals
// ---------------------------------------------------------------------------

func TestCompileArrayLiteral(t *testing.T) {
	tests := []struct {
		input         string
		expectedConst int
	}{
		{"[1, 2, 3];", 3},
		{"[];", 0},
		{"[1];", 1},
	}
	for _, tt := range tests {
		comp := mustCompile(t, tt.input)
		consts := comp.Constants()
		if len(consts) != tt.expectedConst {
			t.Errorf("for %q: expected %d constants, got %d", tt.input, tt.expectedConst, len(consts))
		}
	}
}

// ---------------------------------------------------------------------------
// Hash literals
// ---------------------------------------------------------------------------

func TestCompileHashLiteral(t *testing.T) {
	tests := []string{
		`{};`,
		`{name: "Alice", age: 30};`,
		`{"my key": 42};`,
	}
	for _, input := range tests {
		comp := mustCompile(t, input)
		if comp == nil {
			t.Fatalf("compilation returned nil for %q", input)
		}
	}
}

// ---------------------------------------------------------------------------
// Tuple literals
// ---------------------------------------------------------------------------

func TestCompileTupleLiteral(t *testing.T) {
	tests := []string{
		`(1, 2, 3);`,
		`(1, "hello", true);`,
	}
	for _, input := range tests {
		comp := mustCompile(t, input)
		if comp == nil {
			t.Fatalf("compilation returned nil for %q", input)
		}
	}
}

// ---------------------------------------------------------------------------
// Index expressions
// ---------------------------------------------------------------------------

func TestCompileIndexExpression(t *testing.T) {
	tests := []string{
		`arr = [1, 2, 3]; arr[0];`,
		`arr = [1, 2, 3]; arr[1];`,
		`h = {name: "Alice"}; h["name"];`,
	}
	for _, input := range tests {
		comp := mustCompile(t, input)
		if comp == nil {
			t.Fatalf("compilation returned nil for %q", input)
		}
	}
}

// ---------------------------------------------------------------------------
// Match expressions
// ---------------------------------------------------------------------------

func TestCompileMatchExpression(t *testing.T) {
	tests := []string{
		"match 1 1 => 10 2 => 20 _ => 0;",
		"x = 5; match x 1 => 10 5 => 50 _ => 0;",
	}
	for _, input := range tests {
		comp := mustCompile(t, input)
		if comp == nil {
			t.Fatalf("compilation returned nil for %q", input)
		}
	}
}

func TestCompileMatchWithGuard(t *testing.T) {
	input := `match 5 x if x > 0 => "positive" _ => "non-positive";`
	comp := mustCompile(t, input)
	if comp == nil {
		t.Fatal("compilation returned nil")
	}
}

// ---------------------------------------------------------------------------
// Logical operators (short-circuit)
// ---------------------------------------------------------------------------

func TestCompileLogicalOperators(t *testing.T) {
	tests := []string{
		"true && false;",
		"true || false;",
		"false && true;",
		"false || true;",
	}
	for _, input := range tests {
		comp := mustCompile(t, input)
		if len(comp.Bytecode().Instructions) == 0 {
			t.Errorf("expected non-empty instructions for %q", input)
		}
	}
}

// ---------------------------------------------------------------------------
// Pipe expression
// ---------------------------------------------------------------------------

func TestCompilePipeExpression(t *testing.T) {
	input := "func double(x) = x * 2; 5 |> double;"
	comp := mustCompile(t, input)
	if comp == nil {
		t.Fatal("compilation returned nil")
	}
}

func TestCompilePipeWithArgs(t *testing.T) {
	input := "func add(a, b) = a + b; 5 |> add(10);"
	comp := mustCompile(t, input)
	if comp == nil {
		t.Fatal("compilation returned nil")
	}
}

// ---------------------------------------------------------------------------
// Closure / free variables
// ---------------------------------------------------------------------------

func TestCompileClosureWithFreeVariables(t *testing.T) {
	input := `
		func makeAdder(x) = y => x + y;
	`
	comp := mustCompile(t, input)
	if comp == nil {
		t.Fatal("compilation returned nil")
	}
}

// ---------------------------------------------------------------------------
// Symbol table tests
// ---------------------------------------------------------------------------

func TestSymbolTableDefine(t *testing.T) {
	st := NewSymbolTable()
	a := st.Define("a")
	if a.Name != "a" {
		t.Errorf("expected name 'a', got %q", a.Name)
	}
	if a.Scope != GlobalScope {
		t.Errorf("expected GlobalScope, got %s", a.Scope)
	}
	if a.Index != 0 {
		t.Errorf("expected index 0, got %d", a.Index)
	}

	b := st.Define("b")
	if b.Index != 1 {
		t.Errorf("expected index 1, got %d", b.Index)
	}
}

func TestSymbolTableDefineEffect(t *testing.T) {
	st := NewSymbolTable()
	s := st.DefineEffect("doSomething")
	if !s.IsEffect {
		t.Error("expected IsEffect to be true")
	}
	if s.Scope != GlobalScope {
		t.Errorf("expected GlobalScope, got %s", s.Scope)
	}
}

func TestSymbolTableDefineBuiltin(t *testing.T) {
	st := NewSymbolTable()
	s := st.DefineBuiltin(0, "len")
	if s.Scope != BuiltinScope {
		t.Errorf("expected BuiltinScope, got %s", s.Scope)
	}
	if s.Index != 0 {
		t.Errorf("expected index 0, got %d", s.Index)
	}
}

func TestSymbolTableDefineStayState(t *testing.T) {
	st := NewSymbolTable()
	s := st.DefineStayState("count", 5)
	if s.Scope != StayStateScope {
		t.Errorf("expected StayStateScope, got %s", s.Scope)
	}
	if s.Index != 5 {
		t.Errorf("expected index 5, got %d", s.Index)
	}
}

func TestSymbolTableResolve(t *testing.T) {
	st := NewSymbolTable()
	st.Define("a")
	st.DefineBuiltin(0, "len")

	sym, ok := st.Resolve("a")
	if !ok {
		t.Fatal("failed to resolve 'a'")
	}
	if sym.Name != "a" {
		t.Errorf("expected name 'a', got %q", sym.Name)
	}

	sym, ok = st.Resolve("len")
	if !ok {
		t.Fatal("failed to resolve 'len'")
	}
	if sym.Scope != BuiltinScope {
		t.Errorf("expected BuiltinScope, got %s", sym.Scope)
	}

	_, ok = st.Resolve("nonexistent")
	if ok {
		t.Error("expected false for nonexistent symbol")
	}
}

func TestSymbolTableNestedScopes(t *testing.T) {
	global := NewSymbolTable()
	global.Define("a")

	local := NewEnclosedSymbolTable(global)
	local.Define("b")

	// Can resolve local
	sym, ok := local.Resolve("b")
	if !ok {
		t.Fatal("failed to resolve 'b' in local scope")
	}
	if sym.Scope != LocalScope {
		t.Errorf("expected LocalScope for 'b', got %s", sym.Scope)
	}

	// Can resolve global from local
	sym, ok = local.Resolve("a")
	if !ok {
		t.Fatal("failed to resolve 'a' from enclosed scope")
	}
	if sym.Scope != GlobalScope {
		t.Errorf("expected GlobalScope for 'a', got %s", sym.Scope)
	}
}

func TestSymbolTableFreeVariables(t *testing.T) {
	global := NewSymbolTable()
	global.Define("a")

	first := NewEnclosedSymbolTable(global)
	first.Define("b")

	second := NewEnclosedSymbolTable(first)
	second.Define("c")

	// Resolve 'b' from second scope - should be captured as free
	sym, ok := second.Resolve("b")
	if !ok {
		t.Fatal("failed to resolve 'b' from nested scope")
	}
	if sym.Scope != FreeScope {
		t.Errorf("expected FreeScope, got %s", sym.Scope)
	}

	if len(second.FreeSymbols) != 1 {
		t.Fatalf("expected 1 free symbol, got %d", len(second.FreeSymbols))
	}
	if second.FreeSymbols[0].Name != "b" {
		t.Errorf("expected free symbol 'b', got %q", second.FreeSymbols[0].Name)
	}
}

func TestSymbolTableIsDefinedInCurrentScope(t *testing.T) {
	st := NewSymbolTable()
	st.Define("x")

	if !st.IsDefinedInCurrentScope("x") {
		t.Error("expected 'x' to be defined in current scope")
	}
	if st.IsDefinedInCurrentScope("y") {
		t.Error("expected 'y' to not be defined in current scope")
	}
}

func TestSymbolTableNumDefinitions(t *testing.T) {
	st := NewSymbolTable()
	if st.NumDefinitions() != 0 {
		t.Errorf("expected 0, got %d", st.NumDefinitions())
	}
	st.Define("a")
	st.Define("b")
	if st.NumDefinitions() != 2 {
		t.Errorf("expected 2, got %d", st.NumDefinitions())
	}
}

func TestSymbolTableGlobalSymbols(t *testing.T) {
	st := NewSymbolTable()
	st.Define("a")
	st.Define("b")
	st.DefineBuiltin(0, "len")

	globals := st.GlobalSymbols()
	if len(globals) != 2 {
		t.Errorf("expected 2 global symbols, got %d", len(globals))
	}
	if _, ok := globals["a"]; !ok {
		t.Error("expected 'a' in global symbols")
	}
	if _, ok := globals["b"]; !ok {
		t.Error("expected 'b' in global symbols")
	}
}

func TestEnclosedSymbolTableDefine(t *testing.T) {
	global := NewSymbolTable()
	local := NewEnclosedSymbolTable(global)
	s := local.Define("x")
	if s.Scope != LocalScope {
		t.Errorf("expected LocalScope, got %s", s.Scope)
	}
}

// ---------------------------------------------------------------------------
// Error types
// ---------------------------------------------------------------------------

func TestCompileError(t *testing.T) {
	e := &CompileError{
		Line:    1,
		Column:  5,
		Message: "test error",
	}
	result := e.Error()
	if !strings.Contains(result, "line 1") {
		t.Errorf("expected 'line 1' in error, got: %s", result)
	}
	if !strings.Contains(result, "test error") {
		t.Errorf("expected 'test error' in error, got: %s", result)
	}
}

func TestCompileErrorWithSource(t *testing.T) {
	e := &CompileError{
		Line:    1,
		Column:  5,
		Message: "test error",
		Source:  "let x = 1",
	}
	result := e.Error()
	if !strings.Contains(result, "let x = 1") {
		t.Errorf("expected source line in error, got: %s", result)
	}
	if !strings.Contains(result, "^") {
		t.Errorf("expected caret in error, got: %s", result)
	}
}

func TestCompileErrors(t *testing.T) {
	// Single error
	errs := &CompileErrors{
		Errors: []error{
			&CompileError{Line: 1, Column: 1, Message: "first"},
		},
	}
	result := errs.Error()
	if !strings.Contains(result, "first") {
		t.Errorf("expected 'first' in error, got: %s", result)
	}

	// Multiple errors
	errs = &CompileErrors{
		Errors: []error{
			&CompileError{Line: 1, Column: 1, Message: "first"},
			&CompileError{Line: 2, Column: 1, Message: "second"},
		},
	}
	result = errs.Error()
	if !strings.Contains(result, "first") {
		t.Errorf("expected 'first' in error, got: %s", result)
	}
	if !strings.Contains(result, "second") {
		t.Errorf("expected 'second' in error, got: %s", result)
	}
}

// ---------------------------------------------------------------------------
// Undefined variable error with suggestion
// ---------------------------------------------------------------------------

func TestUndefinedVariableError(t *testing.T) {
	_, err := compileSource("xyz;")
	if err == nil {
		t.Fatal("expected error for undefined variable")
	}
	if !strings.Contains(err.Error(), "undefined variable") {
		t.Errorf("expected 'undefined variable' error, got: %s", err)
	}
}

func TestUndefinedVariableWithSuggestion(t *testing.T) {
	_, err := compileSource("x = 42; xz;")
	if err == nil {
		t.Fatal("expected error for undefined variable")
	}
	errStr := err.Error()
	if !strings.Contains(errStr, "undefined variable") {
		t.Errorf("expected 'undefined variable' in error, got: %s", errStr)
	}
}

// ---------------------------------------------------------------------------
// Bytecode / Constants / SymbolTable accessors
// ---------------------------------------------------------------------------

func TestBytecodeAccessor(t *testing.T) {
	comp := mustCompile(t, "1;")
	bc := comp.Bytecode()
	if bc == nil {
		t.Fatal("Bytecode() returned nil")
	}
	if len(bc.Instructions) == 0 {
		t.Fatal("expected non-empty instructions")
	}
}

func TestConstantsAccessor(t *testing.T) {
	comp := mustCompile(t, "42;")
	consts := comp.Constants()
	if len(consts) != 1 {
		t.Fatalf("expected 1 constant, got %d", len(consts))
	}
}

func TestSymbolTableAccessor(t *testing.T) {
	comp := mustCompile(t, "x = 1;")
	st := comp.SymbolTable()
	if st == nil {
		t.Fatal("SymbolTable() returned nil")
	}
}

func TestExportedSymbols(t *testing.T) {
	comp := mustCompile(t, "x = 1; y = 2;")
	exported := comp.ExportedSymbols()
	if _, ok := exported["x"]; !ok {
		t.Error("expected 'x' in exported symbols")
	}
	if _, ok := exported["y"]; !ok {
		t.Error("expected 'y' in exported symbols")
	}
}

// ---------------------------------------------------------------------------
// ResetInstructions
// ---------------------------------------------------------------------------

func TestResetInstructions(t *testing.T) {
	comp := mustCompile(t, "1;")
	if len(comp.Bytecode().Instructions) == 0 {
		t.Fatal("expected non-empty instructions before reset")
	}
	comp.ResetInstructions()
	if len(comp.Bytecode().Instructions) != 0 {
		t.Fatal("expected empty instructions after reset")
	}
}

// ---------------------------------------------------------------------------
// SourceMap
// ---------------------------------------------------------------------------

func TestSourceMap(t *testing.T) {
	comp := mustCompile(t, "1;")
	sm := comp.SourceMap()
	if sm == nil {
		t.Fatal("SourceMap() returned nil")
	}
}

// ---------------------------------------------------------------------------
// Effect function declaration
// ---------------------------------------------------------------------------

func TestCompileEffectFunctionDeclaration(t *testing.T) {
	input := `func! doAction(x) = x;`
	comp := mustCompile(t, input)
	sym, ok := comp.SymbolTable().Resolve("doAction")
	if !ok {
		t.Fatal("effect function 'doAction' not defined")
	}
	if !sym.IsEffect {
		t.Error("expected IsEffect to be true for func!")
	}
}

// ---------------------------------------------------------------------------
// Effect pipe error (|> with effect function)
// ---------------------------------------------------------------------------

func TestCompileEffectPipeWithPureFunction(t *testing.T) {
	// Using !> with a pure function should produce error
	input := `func pure(x) = x; 5 !> pure;`
	_, err := compileSource(input)
	if err == nil {
		t.Fatal("expected error for !> with pure function")
	}
	if !strings.Contains(err.Error(), "pure function") {
		t.Errorf("expected 'pure function' error, got: %s", err)
	}
}

func TestCompilePipeWithEffectFunction(t *testing.T) {
	// Using |> with an effect function should produce error
	input := `func! effect(x) = x; 5 |> effect;`
	_, err := compileSource(input)
	if err == nil {
		t.Fatal("expected error for |> with effect function")
	}
	if !strings.Contains(err.Error(), "effect function") {
		t.Errorf("expected 'effect function' error, got: %s", err)
	}
}

// ---------------------------------------------------------------------------
// convertRegexFlags
// ---------------------------------------------------------------------------

func TestConvertRegexFlags(t *testing.T) {
	tests := []struct {
		pattern  string
		flags    string
		expected string
	}{
		{"hello", "", "hello"},
		{"hello", "i", "(?i)hello"},
		{"hello", "im", "(?im)hello"},
		{"hello", "ims", "(?ims)hello"},
		{"hello", "g", "hello"}, // g is not a Go flag
	}
	for _, tt := range tests {
		result := convertRegexFlags(tt.pattern, tt.flags)
		if result != tt.expected {
			t.Errorf("convertRegexFlags(%q, %q) = %q, expected %q", tt.pattern, tt.flags, result, tt.expected)
		}
	}
}

// ---------------------------------------------------------------------------
// levenshteinDistance
// ---------------------------------------------------------------------------

func TestLevenshteinDistance(t *testing.T) {
	tests := []struct {
		a, b     string
		expected int
	}{
		{"", "", 0},
		{"a", "", 1},
		{"", "b", 1},
		{"kitten", "sitting", 3},
		{"hello", "hello", 0},
		{"abc", "abd", 1},
	}
	for _, tt := range tests {
		result := levenshteinDistance(tt.a, tt.b)
		if result != tt.expected {
			t.Errorf("levenshteinDistance(%q, %q) = %d, expected %d", tt.a, tt.b, result, tt.expected)
		}
	}
}

// ---------------------------------------------------------------------------
// findSimilarName
// ---------------------------------------------------------------------------

func TestFindSimilarName(t *testing.T) {
	candidates := []string{"hello", "world", "foo", "bar"}

	result := findSimilarName("helo", candidates, 2)
	if result != "hello" {
		t.Errorf("expected 'hello', got %q", result)
	}

	result = findSimilarName("xyz", candidates, 2)
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

// ---------------------------------------------------------------------------
// min function
// ---------------------------------------------------------------------------

func TestMin(t *testing.T) {
	if min(1, 2, 3) != 1 {
		t.Error("min(1,2,3) should be 1")
	}
	if min(3, 2, 1) != 1 {
		t.Error("min(3,2,1) should be 1")
	}
	if min(2, 1, 3) != 1 {
		t.Error("min(2,1,3) should be 1")
	}
	if min(3, 1, 2) != 1 {
		t.Error("min(3,1,2) should be 1")
	}
	if min(1, 3, 2) != 1 {
		t.Error("min(1,3,2) should be 1")
	}
	if min(2, 3, 1) != 1 {
		t.Error("min(2,3,1) should be 1")
	}
}

// ---------------------------------------------------------------------------
// getSourceLine
// ---------------------------------------------------------------------------

func TestGetSourceLine(t *testing.T) {
	comp := New()
	comp.SetInput("line1\nline2\nline3")

	if comp.getSourceLine(1) != "line1" {
		t.Errorf("expected 'line1', got %q", comp.getSourceLine(1))
	}
	if comp.getSourceLine(2) != "line2" {
		t.Errorf("expected 'line2', got %q", comp.getSourceLine(2))
	}
	if comp.getSourceLine(3) != "line3" {
		t.Errorf("expected 'line3', got %q", comp.getSourceLine(3))
	}
	if comp.getSourceLine(0) != "" {
		t.Errorf("expected empty for line 0, got %q", comp.getSourceLine(0))
	}
	if comp.getSourceLine(4) != "" {
		t.Errorf("expected empty for line 4, got %q", comp.getSourceLine(4))
	}

	// Empty input
	comp2 := New()
	if comp2.getSourceLine(1) != "" {
		t.Errorf("expected empty for no input, got %q", comp2.getSourceLine(1))
	}
}

// ---------------------------------------------------------------------------
// Scope management
// ---------------------------------------------------------------------------

func TestEnterAndLeaveScope(t *testing.T) {
	comp := New()

	if comp.scopeIndex != 0 {
		t.Fatalf("expected scopeIndex 0, got %d", comp.scopeIndex)
	}

	comp.enterScope()
	if comp.scopeIndex != 1 {
		t.Fatalf("expected scopeIndex 1, got %d", comp.scopeIndex)
	}

	comp.leaveScope()
	if comp.scopeIndex != 0 {
		t.Fatalf("expected scopeIndex 0 after leaving, got %d", comp.scopeIndex)
	}
}

func TestEnterScopeWithName(t *testing.T) {
	comp := New()
	comp.enterScopeWithName("myFunc")
	if comp.scopes[comp.scopeIndex].funcName != "myFunc" {
		t.Errorf("expected funcName 'myFunc', got %q", comp.scopes[comp.scopeIndex].funcName)
	}
	comp.leaveScope()
}

// ---------------------------------------------------------------------------
// Emit and instruction tracking
// ---------------------------------------------------------------------------

func TestEmitAndInstructions(t *testing.T) {
	comp := New()
	pos := comp.emit(bytecode.OpConstant, 0)
	if pos != 0 {
		t.Errorf("expected position 0, got %d", pos)
	}

	pos2 := comp.emit(bytecode.OpPop)
	if pos2 <= pos {
		t.Errorf("expected position > %d, got %d", pos, pos2)
	}
}

func TestLastInstructionIs(t *testing.T) {
	comp := New()
	if comp.lastInstructionIs(bytecode.OpPop) {
		t.Error("expected false for empty instructions")
	}

	comp.emit(bytecode.OpPop)
	if !comp.lastInstructionIs(bytecode.OpPop) {
		t.Error("expected true after emitting OpPop")
	}
}

// ---------------------------------------------------------------------------
// addConstant
// ---------------------------------------------------------------------------

func TestAddConstant(t *testing.T) {
	comp := New()
	idx := comp.addConstant(value.Int(42))
	if idx != 0 {
		t.Errorf("expected index 0, got %d", idx)
	}
	idx = comp.addConstant(value.String("hello"))
	if idx != 1 {
		t.Errorf("expected index 1, got %d", idx)
	}
}

// ---------------------------------------------------------------------------
// Error collection
// ---------------------------------------------------------------------------

func TestErrorCollection(t *testing.T) {
	comp := New()
	if comp.hasErrors() {
		t.Error("expected no errors initially")
	}
	if comp.getErrors() != nil {
		t.Error("expected nil from getErrors initially")
	}

	comp.addError(&CompileError{Line: 1, Column: 1, Message: "err1"})
	if !comp.hasErrors() {
		t.Error("expected hasErrors to be true")
	}
	err := comp.getErrors()
	if err == nil {
		t.Fatal("expected non-nil error")
	}
	if !strings.Contains(err.Error(), "err1") {
		t.Errorf("expected 'err1' in error, got: %s", err)
	}

	// Multiple errors
	comp.addError(&CompileError{Line: 2, Column: 1, Message: "err2"})
	err = comp.getErrors()
	compErrors, ok := err.(*CompileErrors)
	if !ok {
		t.Fatalf("expected *CompileErrors, got %T", err)
	}
	if len(compErrors.Errors) != 2 {
		t.Fatalf("expected 2 errors, got %d", len(compErrors.Errors))
	}
}

// ---------------------------------------------------------------------------
// newCompileError and newCompileErrorAt
// ---------------------------------------------------------------------------

func TestNewCompileError(t *testing.T) {
	comp := New()
	comp.SetInput("hello world")
	tok := makeToken(1, 6)
	err := comp.newCompileError(tok, "something wrong")
	if err.Line != 1 {
		t.Errorf("expected line 1, got %d", err.Line)
	}
	if err.Column != 6 {
		t.Errorf("expected column 6, got %d", err.Column)
	}
	if err.Message != "something wrong" {
		t.Errorf("unexpected message: %s", err.Message)
	}
	if err.Source != "hello world" {
		t.Errorf("expected source 'hello world', got %q", err.Source)
	}
}

func TestNewCompileErrorAt(t *testing.T) {
	comp := New()
	comp.SetInput("first\nsecond")
	err := comp.newCompileErrorAt(2, 3, "error at line 2")
	if err.Line != 2 {
		t.Errorf("expected line 2, got %d", err.Line)
	}
	if err.Source != "second" {
		t.Errorf("expected source 'second', got %q", err.Source)
	}
}

// makeToken is a helper to create a minimal token for testing
func makeToken(line, column int) token.Token {
	return token.Token{Line: line, Column: column}
}

// ---------------------------------------------------------------------------
// getAllDefinedNames
// ---------------------------------------------------------------------------

func TestGetAllDefinedNames(t *testing.T) {
	comp := New()
	// Compile a program that defines variables
	program := parse("x = 1; y = 2;")
	comp.Compile(program)

	names := comp.getAllDefinedNames()
	found := make(map[string]bool)
	for _, n := range names {
		found[n] = true
	}
	if !found["x"] {
		t.Error("expected 'x' in defined names")
	}
	if !found["y"] {
		t.Error("expected 'y' in defined names")
	}
}

// ---------------------------------------------------------------------------
// Modulus and power operators
// ---------------------------------------------------------------------------

func TestCompileModulusAndPower(t *testing.T) {
	tests := []string{
		"7 % 3;",
		"2 ** 8;",
	}
	for _, input := range tests {
		comp := mustCompile(t, input)
		if len(comp.Bytecode().Instructions) == 0 {
			t.Errorf("expected non-empty instructions for %q", input)
		}
	}
}

// ---------------------------------------------------------------------------
// Complex programs
// ---------------------------------------------------------------------------

func TestCompileComplexProgram(t *testing.T) {
	input := `
		func add(a, b) = a + b;
		func mul(a, b) = a * b;
		x = add(1, 2);
		y = mul(3, 4);
		z = x + y;
	`
	comp := mustCompile(t, input)
	for _, name := range []string{"add", "mul", "x", "y", "z"} {
		if _, ok := comp.SymbolTable().Resolve(name); !ok {
			t.Errorf("expected %q to be defined", name)
		}
	}
}

// ---------------------------------------------------------------------------
// setPos / setPosFromToken
// ---------------------------------------------------------------------------

func TestSetPos(t *testing.T) {
	comp := New()
	comp.setPos(10, 5)
	if comp.currentLine != 10 {
		t.Errorf("expected currentLine 10, got %d", comp.currentLine)
	}
	if comp.currentColumn != 5 {
		t.Errorf("expected currentColumn 5, got %d", comp.currentColumn)
	}
}

func TestSetPosFromToken(t *testing.T) {
	comp := New()
	tok := makeToken(3, 7)
	comp.setPosFromToken(tok)
	if comp.currentLine != 3 {
		t.Errorf("expected currentLine 3, got %d", comp.currentLine)
	}
	if comp.currentColumn != 7 {
		t.Errorf("expected currentColumn 7, got %d", comp.currentColumn)
	}
}

// ---------------------------------------------------------------------------
// Regex literal compilation
// ---------------------------------------------------------------------------

func TestCompileRegexLiteral(t *testing.T) {
	input := `/hello/;`
	comp := mustCompile(t, input)
	if len(comp.Constants()) == 0 {
		t.Fatal("expected constants for regex")
	}
}

func TestCompileRegexLiteralWithFlags(t *testing.T) {
	input := `/hello/i;`
	comp := mustCompile(t, input)
	if len(comp.Constants()) == 0 {
		t.Fatal("expected constants for regex with flags")
	}
}

// ---------------------------------------------------------------------------
// Array destructuring
// ---------------------------------------------------------------------------

func TestCompileArrayDestructuring(t *testing.T) {
	input := `arr = [1, 2, 3]; [a, b, c] = arr;`
	comp := mustCompile(t, input)
	for _, name := range []string{"a", "b", "c"} {
		if _, ok := comp.SymbolTable().Resolve(name); !ok {
			t.Errorf("expected %q to be defined after destructuring", name)
		}
	}
}

func TestCompileArrayDestructuringReassignmentError(t *testing.T) {
	input := `a = 1; arr = [1, 2]; [a, b] = arr;`
	_, err := compileSource(input)
	if err == nil {
		t.Fatal("expected error for destructuring reassignment")
	}
	if !strings.Contains(err.Error(), "already bound") {
		t.Errorf("expected 'already bound' error, got: %s", err)
	}
}

// ---------------------------------------------------------------------------
// Head/Tail destructuring
// ---------------------------------------------------------------------------

func TestCompileHeadTailDestructuring(t *testing.T) {
	input := `arr = [1, 2, 3]; [h | t] = arr;`
	comp := mustCompile(t, input)
	if _, ok := comp.SymbolTable().Resolve("h"); !ok {
		t.Error("expected 'h' to be defined")
	}
	if _, ok := comp.SymbolTable().Resolve("t"); !ok {
		t.Error("expected 't' to be defined")
	}
}

// ---------------------------------------------------------------------------
// Do expression
// ---------------------------------------------------------------------------

func TestCompileDoExpression(t *testing.T) {
	input := `
		x = do
			a = 1;
			b = 2;
			a + b
		end;
	`
	comp := mustCompile(t, input)
	if _, ok := comp.SymbolTable().Resolve("x"); !ok {
		t.Error("expected 'x' to be defined")
	}
}

// ---------------------------------------------------------------------------
// Success/Failure expressions
// ---------------------------------------------------------------------------

func TestCompileSuccessExpression(t *testing.T) {
	input := `func! f(x) = x; f(42);`
	comp := mustCompile(t, input)
	if comp == nil {
		t.Fatal("compilation returned nil")
	}
}

// ---------------------------------------------------------------------------
// Member expression
// ---------------------------------------------------------------------------

func TestCompileMemberExpression(t *testing.T) {
	input := `h = {name: "Alice"}; h.name;`
	comp := mustCompile(t, input)
	if comp == nil {
		t.Fatal("compilation returned nil")
	}
}

// ---------------------------------------------------------------------------
// replaceLastPopWithReturn / replaceLastCallWithTailCall
// ---------------------------------------------------------------------------

func TestReplaceLastPopWithReturn(t *testing.T) {
	comp := New()
	comp.enterScopeWithName("test")
	comp.emit(bytecode.OpConstant, 0)
	comp.emit(bytecode.OpPop)

	comp.replaceLastPopWithReturn()
	// After replacing, the last instruction should be OpReturn
	if !comp.lastInstructionIs(bytecode.OpReturn) {
		t.Error("expected last instruction to be OpReturn after replaceLastPopWithReturn")
	}
	comp.leaveScope()
}

// ---------------------------------------------------------------------------
// changeOperand
// ---------------------------------------------------------------------------

func TestChangeOperand(t *testing.T) {
	comp := New()
	pos := comp.emit(bytecode.OpJump, 0)
	comp.changeOperand(pos, 100)

	// The instruction should now jump to position 100
	ins := comp.currentInstructions()
	if len(ins) == 0 {
		t.Fatal("expected non-empty instructions")
	}
}

// ---------------------------------------------------------------------------
// loadSymbol for different scopes
// ---------------------------------------------------------------------------

func TestLoadSymbolAllScopes(t *testing.T) {
	comp := New()

	// Global
	comp.loadSymbol(Symbol{Name: "g", Scope: GlobalScope, Index: 0})
	// Local
	comp.loadSymbol(Symbol{Name: "l", Scope: LocalScope, Index: 0})
	// Builtin
	comp.loadSymbol(Symbol{Name: "b", Scope: BuiltinScope, Index: 0})
	// Free
	comp.loadSymbol(Symbol{Name: "f", Scope: FreeScope, Index: 0})
	// StayState
	comp.loadSymbol(Symbol{Name: "s", Scope: StayStateScope, Index: 0})

	if len(comp.currentInstructions()) == 0 {
		t.Fatal("expected instructions after loading symbols")
	}
}

// ---------------------------------------------------------------------------
// Constraint declaration
// ---------------------------------------------------------------------------

func TestCompileConstraintDeclaration(t *testing.T) {
	input := `
		constraint Positive(n) = n > 0;
	`
	comp := mustCompile(t, input)
	if _, ok := comp.SymbolTable().Resolve("Positive"); !ok {
		t.Error("expected 'Positive' constraint to be defined")
	}
}

// ---------------------------------------------------------------------------
// Function with constraints
// ---------------------------------------------------------------------------

func TestCompileFunctionWithConstraints(t *testing.T) {
	input := `
		constraint Positive(n) = n > 0;
		func abs(x: Positive) = x;
	`
	comp := mustCompile(t, input)
	if _, ok := comp.SymbolTable().Resolve("abs"); !ok {
		t.Error("expected 'abs' to be defined")
	}
}

// ---------------------------------------------------------------------------
// Type declarations (ADT)
// ---------------------------------------------------------------------------

func TestCompileTypeDeclaration(t *testing.T) {
	input := `
		type Option = Some(value) | None;
	`
	comp := mustCompile(t, input)
	if _, ok := comp.SymbolTable().Resolve("Some"); !ok {
		t.Error("expected 'Some' constructor to be defined")
	}
	if _, ok := comp.SymbolTable().Resolve("None"); !ok {
		t.Error("expected 'None' constructor to be defined")
	}
}

// ---------------------------------------------------------------------------
// Match with constructor pattern
// ---------------------------------------------------------------------------

func TestCompileMatchWithConstructorPattern(t *testing.T) {
	input := `
		type Option = Some(value) | None;
		x = Some(42);
		match x
			Some(v) => v
			None => 0;
	`
	comp := mustCompile(t, input)
	if comp == nil {
		t.Fatal("compilation returned nil")
	}
}

// ---------------------------------------------------------------------------
// Wildcard expression
// ---------------------------------------------------------------------------

func TestCompileWildcardExpression(t *testing.T) {
	// Wildcard is used in match default cases
	input := `match 1 _ => 42;`
	comp := mustCompile(t, input)
	if comp == nil {
		t.Fatal("compilation returned nil")
	}
}

// ---------------------------------------------------------------------------
// Effect handle expression
// ---------------------------------------------------------------------------

func TestCompileEffectHandleExpression(t *testing.T) {
	input := `success(42) !? { success(r) => r failure(e) => 0 };`
	comp := mustCompile(t, input)
	if comp == nil {
		t.Fatal("compilation returned nil")
	}
}

func TestCompileEffectHandleWithEffectFunc(t *testing.T) {
	input := `
		func! getData() = 100;
		getData() !? {
			success(r) => r * 2
			failure(e) => 0
		};
	`
	comp := mustCompile(t, input)
	if comp == nil {
		t.Fatal("compilation returned nil")
	}
}

// ---------------------------------------------------------------------------
// Scope-related source map tests
// ---------------------------------------------------------------------------

func TestLeaveScopeWithSourceMap(t *testing.T) {
	comp := New()
	comp.SetFileName("test.ca")
	comp.enterScopeWithName("myFunc")

	// Emit something in the scope
	comp.emit(bytecode.OpConstant, 0)
	comp.emit(bytecode.OpReturn)

	ins, sm := comp.leaveScopeWithSourceMap()
	if len(ins) == 0 {
		t.Fatal("expected non-empty instructions from scope")
	}
	if sm == nil {
		t.Fatal("expected non-nil source map")
	}
	if sm.File != "test.ca" {
		t.Errorf("expected source map file 'test.ca', got %q", sm.File)
	}
}

// ---------------------------------------------------------------------------
// currentSourceMap
// ---------------------------------------------------------------------------

func TestCurrentSourceMap(t *testing.T) {
	comp := New()
	sm := comp.currentSourceMap()
	if sm == nil {
		t.Fatal("expected non-nil current source map")
	}
}

// ---------------------------------------------------------------------------
// ResetInstructions with fileName preserved
// ---------------------------------------------------------------------------

func TestResetInstructionsPreservesFileName(t *testing.T) {
	comp := New()
	comp.SetFileName("test.ca")
	comp.emit(bytecode.OpConstant, 0)
	comp.ResetInstructions()

	sm := comp.SourceMap()
	if sm.File != "test.ca" {
		t.Errorf("expected fileName 'test.ca' preserved after reset, got %q", sm.File)
	}
}

// ---------------------------------------------------------------------------
// DefineEffect in enclosed scope
// ---------------------------------------------------------------------------

func TestDefineEffectInEnclosedScope(t *testing.T) {
	global := NewSymbolTable()
	local := NewEnclosedSymbolTable(global)
	s := local.DefineEffect("action")
	if s.Scope != LocalScope {
		t.Errorf("expected LocalScope for enclosed effect, got %s", s.Scope)
	}
	if !s.IsEffect {
		t.Error("expected IsEffect to be true")
	}
}

// ---------------------------------------------------------------------------
// Resolve builtin from enclosed scope
// ---------------------------------------------------------------------------

func TestResolveBuiltinFromEnclosedScope(t *testing.T) {
	global := NewSymbolTable()
	global.DefineBuiltin(0, "len")

	local := NewEnclosedSymbolTable(global)

	sym, ok := local.Resolve("len")
	if !ok {
		t.Fatal("failed to resolve 'len' from enclosed scope")
	}
	if sym.Scope != BuiltinScope {
		t.Errorf("expected BuiltinScope, got %s", sym.Scope)
	}
}

// ---------------------------------------------------------------------------
// Resolve StayState from enclosed scope (should NOT be captured as free)
// ---------------------------------------------------------------------------

func TestResolveStayStateFromEnclosedScope(t *testing.T) {
	global := NewSymbolTable()
	global.DefineStayState("count", 0)

	local := NewEnclosedSymbolTable(global)

	sym, ok := local.Resolve("count")
	if !ok {
		t.Fatal("failed to resolve 'count' from enclosed scope")
	}
	if sym.Scope != StayStateScope {
		t.Errorf("expected StayStateScope, got %s", sym.Scope)
	}
	if len(local.FreeSymbols) != 0 {
		t.Errorf("StayState should not be captured as free, got %d free symbols", len(local.FreeSymbols))
	}
}

// ---------------------------------------------------------------------------
// Multiple errors collected from separate statements
// ---------------------------------------------------------------------------

func TestMultipleErrorsCollected(t *testing.T) {
	// Two statements referencing undefined variables should collect 2 errors
	input := "abc; def;"
	_, err := compileSource(input)
	if err == nil {
		t.Fatal("expected errors for undefined variables")
	}
	// The error should contain both variable names
	errStr := err.Error()
	if !strings.Contains(errStr, "abc") {
		t.Errorf("expected 'abc' in error, got: %s", errStr)
	}
	if !strings.Contains(errStr, "def") {
		t.Errorf("expected 'def' in error, got: %s", errStr)
	}
}

// ---------------------------------------------------------------------------
// Return expression
// ---------------------------------------------------------------------------

func TestCompileReturnExpression(t *testing.T) {
	input := `
		func f(x) = return x;
	`
	comp := mustCompile(t, input)
	if comp == nil {
		t.Fatal("compilation returned nil")
	}
}

// ---------------------------------------------------------------------------
// Nested function scopes
// ---------------------------------------------------------------------------

func TestCompileNestedFunctionScopes(t *testing.T) {
	// A function that returns a lambda capturing a free variable
	input := `
		func makeAdder(x) = y => x + y;
		add5 = makeAdder(5);
		add5(10);
	`
	comp, err := compileSource(input)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if comp == nil {
		t.Fatal("compilation returned nil")
	}
}

// ---------------------------------------------------------------------------
// Block body lambda
// ---------------------------------------------------------------------------

func TestCompileBlockBodyLambda(t *testing.T) {
	input := `
		f = (x) => do
			y = x * 2;
			y + 1
		end;
	`
	comp := mustCompile(t, input)
	if comp == nil {
		t.Fatal("compilation returned nil")
	}
}

// ---------------------------------------------------------------------------
// Negative test: unknown node type in Compile
// ---------------------------------------------------------------------------

func TestCompileUnknownNodeType(t *testing.T) {
	comp := New()
	// Pass nil - should return an error
	err := comp.Compile(nil)
	if err == nil {
		t.Fatal("expected error for nil node")
	}
}

// ---------------------------------------------------------------------------
// String operations
// ---------------------------------------------------------------------------

func TestCompileStringConcatenation(t *testing.T) {
	input := `"hello" + " " + "world";`
	comp := mustCompile(t, input)
	if len(comp.Constants()) != 3 {
		t.Errorf("expected 3 constants, got %d", len(comp.Constants()))
	}
}

// ---------------------------------------------------------------------------
// Nested match expressions
// ---------------------------------------------------------------------------

func TestCompileNestedMatch(t *testing.T) {
	input := `
		x = 5;
		match x
			1 => match 1
				1 => 10
				_ => 0
			_ => 99;
	`
	comp := mustCompile(t, input)
	if comp == nil {
		t.Fatal("compilation returned nil")
	}
}

// ---------------------------------------------------------------------------
// Multiple match cases
// ---------------------------------------------------------------------------

func TestCompileMatchMultipleCases(t *testing.T) {
	input := `match 3 1 => 10 2 => 20 3 => 30 _ => 0;`
	comp := mustCompile(t, input)
	if comp == nil {
		t.Fatal("compilation returned nil")
	}
}

// ---------------------------------------------------------------------------
// Empty program
// ---------------------------------------------------------------------------

func TestCompileEmptyProgram(t *testing.T) {
	comp := mustCompile(t, "")
	bc := comp.Bytecode()
	if len(bc.Instructions) != 0 {
		t.Errorf("expected empty instructions for empty program, got %d bytes", len(bc.Instructions))
	}
}
