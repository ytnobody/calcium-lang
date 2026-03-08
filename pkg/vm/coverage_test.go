package vm

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ytnobody/calcium-lang/pkg/compiler"
	"github.com/ytnobody/calcium-lang/pkg/value"
)

// =============================================================================
// Tests targeting low-coverage functions to improve overall test coverage
// =============================================================================

// --- builtinGet edge cases ---

func TestBuiltinGetArrayAllPaths(t *testing.T) {
	arr := value.Array([]value.Value{value.Int(10), value.Int(20), value.Int(30)})

	// Valid index
	result := builtinGet(arr, value.Int(1))
	if result.AsInt() != 20 {
		t.Errorf("expected 20, got %v", result)
	}

	// Out of bounds with default
	result = builtinGet(arr, value.Int(10), value.Int(99))
	if result.AsInt() != 99 {
		t.Errorf("expected default 99, got %v", result)
	}

	// Out of bounds without default
	result = builtinGet(arr, value.Int(10))
	if result.Type != value.TYPE_NULL {
		t.Errorf("expected null, got %v", result)
	}

	// Negative index
	result = builtinGet(arr, value.Int(-1))
	if result.Type != value.TYPE_NULL {
		t.Errorf("expected null for negative index, got %v", result)
	}

	// Non-integer index for array
	result = builtinGet(arr, value.String("0"))
	if result.Type != value.TYPE_FAILURE {
		t.Errorf("expected failure for non-int index, got %v", result)
	}
}

func TestBuiltinGetHashAllPaths(t *testing.T) {
	h := value.NewHash()
	h.Set(value.String("name"), value.String("Alice"))
	h.Set(value.String("age"), value.Int(30))
	hash := value.HashVal(h)

	// Valid key
	result := builtinGet(hash, value.String("name"))
	if result.AsString() != "Alice" {
		t.Errorf("expected Alice, got %v", result)
	}

	// Missing key without default
	result = builtinGet(hash, value.String("missing"))
	if result.Type != value.TYPE_NULL {
		t.Errorf("expected null, got %v", result)
	}

	// Missing key with default
	result = builtinGet(hash, value.String("missing"), value.String("default"))
	if result.AsString() != "default" {
		t.Errorf("expected default, got %v", result)
	}

	// Non-string key for hash
	result = builtinGet(hash, value.Int(0))
	if result.Type != value.TYPE_FAILURE {
		t.Errorf("expected failure for non-string key, got %v", result)
	}
}

func TestBuiltinGetErrorPaths(t *testing.T) {
	// Too few args
	result := builtinGet(value.Int(1))
	if result.Type != value.TYPE_FAILURE {
		t.Errorf("expected failure for 1 arg, got %v", result)
	}

	// Too many args
	result = builtinGet(value.Int(1), value.Int(2), value.Int(3), value.Int(4))
	if result.Type != value.TYPE_FAILURE {
		t.Errorf("expected failure for 4 args, got %v", result)
	}

	// Unsupported type
	result = builtinGet(value.Int(42), value.Int(0))
	if result.Type != value.TYPE_FAILURE {
		t.Errorf("expected failure for int get, got %v", result)
	}

	// Negative array index with default
	arr := value.Array([]value.Value{value.Int(1)})
	result = builtinGet(arr, value.Int(-1), value.Int(99))
	if result.AsInt() != 99 {
		t.Errorf("expected 99, got %v", result)
	}
}

// --- builtinHashSet and builtinHashMerge ---

func TestBuiltinHashSetAddAndReplace(t *testing.T) {
	h := value.NewHash()
	h.Set(value.String("a"), value.Int(1))
	h.Set(value.String("b"), value.Int(2))
	hash := value.HashVal(h)

	// Add new key
	result := builtinHashSet(hash, value.String("c"), value.Int(3))
	if result.Type != value.TYPE_HASH {
		t.Fatalf("expected hash, got %s", result.Type)
	}
	newHash := result.AsHash()
	val, ok := newHash.Get(value.String("c"))
	if !ok || val.AsInt() != 3 {
		t.Error("new key c not found or wrong value")
	}
	// Original keys preserved
	val, ok = newHash.Get(value.String("a"))
	if !ok || val.AsInt() != 1 {
		t.Error("key a should be preserved")
	}

	// Replace existing key
	result = builtinHashSet(hash, value.String("a"), value.Int(10))
	newHash = result.AsHash()
	val, ok = newHash.Get(value.String("a"))
	if !ok || val.AsInt() != 10 {
		t.Error("key a should be replaced with 10")
	}
	// Other keys preserved
	val, ok = newHash.Get(value.String("b"))
	if !ok || val.AsInt() != 2 {
		t.Error("key b should be preserved")
	}
}

func TestBuiltinHashSetErrors(t *testing.T) {
	h := value.NewHash()
	hash := value.HashVal(h)

	// Wrong arg count
	result := builtinHashSet(hash, value.String("key"))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for 2 args")
	}

	// Non-hash first arg
	result = builtinHashSet(value.Int(42), value.String("key"), value.Int(1))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-hash")
	}

	// Non-string key
	result = builtinHashSet(hash, value.Int(0), value.Int(1))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-string key")
	}
}

func TestBuiltinHashMergeFull(t *testing.T) {
	h1 := value.NewHash()
	h1.Set(value.String("a"), value.Int(1))
	h1.Set(value.String("b"), value.Int(2))
	hash1 := value.HashVal(h1)

	h2 := value.NewHash()
	h2.Set(value.String("b"), value.Int(20))
	h2.Set(value.String("c"), value.Int(3))
	hash2 := value.HashVal(h2)

	// Merge with overlapping keys
	result := builtinHashMerge(hash1, hash2)
	if result.Type != value.TYPE_HASH {
		t.Fatalf("expected hash, got %s", result.Type)
	}
	merged := result.AsHash()

	val, ok := merged.Get(value.String("a"))
	if !ok || val.AsInt() != 1 {
		t.Error("key a should be from hash1")
	}
	val, ok = merged.Get(value.String("b"))
	if !ok || val.AsInt() != 20 {
		t.Error("key b should be overwritten by hash2")
	}
	val, ok = merged.Get(value.String("c"))
	if !ok || val.AsInt() != 3 {
		t.Error("key c should be from hash2")
	}
}

func TestBuiltinHashMergeErrors(t *testing.T) {
	h := value.NewHash()
	hash := value.HashVal(h)

	result := builtinHashMerge(hash)
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for 1 arg")
	}

	result = builtinHashMerge(value.Int(42), hash)
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-hash first arg")
	}

	result = builtinHashMerge(hash, value.Int(42))
	if result.Type != value.TYPE_FAILURE {
		t.Error("expected failure for non-hash second arg")
	}
}

// --- StackTop and Globals ---

func TestStackTopEmpty(t *testing.T) {
	vm := New(nil)
	result := vm.StackTop()
	if result.Type != value.TYPE_NULL {
		t.Errorf("expected null for empty stack, got %v", result)
	}
}

func TestStackTopNonEmpty(t *testing.T) {
	vm := New(nil)
	vm.push(value.Int(42))
	result := vm.StackTop()
	if result.AsInt() != 42 {
		t.Errorf("expected 42, got %v", result)
	}
}

func TestGlobalsAccessor(t *testing.T) {
	vm := New(nil)
	globals := vm.Globals()
	if len(globals) != GlobalsSize {
		t.Errorf("expected %d globals, got %d", GlobalsSize, len(globals))
	}
}

// --- callValueSync via integration tests ---

func TestCallValueSyncViaPipeline(t *testing.T) {
	// Pipeline with map/filter/reduce uses callValueSync internally
	tests := []vmTestCase{
		{`[1, 2, 3] |> map(x => x * 2);`, []int{2, 4, 6}},
		{`[1, 2, 3, 4] |> filter(x => x > 2);`, []int{3, 4}},
		{`[1, 2, 3] |> reduce((acc, x) => acc + x, 0);`, 6},
	}
	runVmTests(t, tests)
}

// --- Map/Filter/Reduce error paths ---

func TestMapWithWrongArgCount(t *testing.T) {
	input := `map([1, 2]);`
	program := parse(input)
	comp := compiler.New()
	if err := comp.Compile(program); err != nil {
		t.Fatalf("compiler error: %s", err)
	}
	vm := New(comp.Constants())
	var buf bytes.Buffer
	vm.SetOutput(&buf)
	if err := vm.Run(comp.Bytecode().Instructions); err != nil {
		t.Fatalf("vm error: %s", err)
	}
	result := vm.LastPoppedStackElem()
	if result.Type != value.TYPE_FAILURE {
		t.Errorf("expected failure for map with 1 arg, got %s: %v", result.Type, result)
	}
}

func TestMapWithNonArrayFirstArg(t *testing.T) {
	input := `map(42, x => x);`
	program := parse(input)
	comp := compiler.New()
	if err := comp.Compile(program); err != nil {
		t.Fatalf("compiler error: %s", err)
	}
	vm := New(comp.Constants())
	var buf bytes.Buffer
	vm.SetOutput(&buf)
	if err := vm.Run(comp.Bytecode().Instructions); err != nil {
		t.Fatalf("vm error: %s", err)
	}
	result := vm.LastPoppedStackElem()
	if result.Type != value.TYPE_FAILURE {
		t.Errorf("expected failure for map with non-array, got %s: %v", result.Type, result)
	}
}

func TestMapWithNonFunctionSecondArg(t *testing.T) {
	input := `map([1, 2], 42);`
	program := parse(input)
	comp := compiler.New()
	if err := comp.Compile(program); err != nil {
		t.Fatalf("compiler error: %s", err)
	}
	vm := New(comp.Constants())
	var buf bytes.Buffer
	vm.SetOutput(&buf)
	if err := vm.Run(comp.Bytecode().Instructions); err != nil {
		t.Fatalf("vm error: %s", err)
	}
	result := vm.LastPoppedStackElem()
	if result.Type != value.TYPE_FAILURE {
		t.Errorf("expected failure for map with non-function, got %s: %v", result.Type, result)
	}
}

func TestFilterWithNonArray(t *testing.T) {
	input := `filter(42, x => true);`
	program := parse(input)
	comp := compiler.New()
	if err := comp.Compile(program); err != nil {
		t.Fatalf("compiler error: %s", err)
	}
	vm := New(comp.Constants())
	var buf bytes.Buffer
	vm.SetOutput(&buf)
	if err := vm.Run(comp.Bytecode().Instructions); err != nil {
		t.Fatalf("vm error: %s", err)
	}
	result := vm.LastPoppedStackElem()
	if result.Type != value.TYPE_FAILURE {
		t.Errorf("expected failure, got %s", result.Type)
	}
}

func TestFilterWithNonFunction(t *testing.T) {
	input := `filter([1, 2], 42);`
	program := parse(input)
	comp := compiler.New()
	if err := comp.Compile(program); err != nil {
		t.Fatalf("compiler error: %s", err)
	}
	vm := New(comp.Constants())
	var buf bytes.Buffer
	vm.SetOutput(&buf)
	if err := vm.Run(comp.Bytecode().Instructions); err != nil {
		t.Fatalf("vm error: %s", err)
	}
	result := vm.LastPoppedStackElem()
	if result.Type != value.TYPE_FAILURE {
		t.Errorf("expected failure, got %s", result.Type)
	}
}

func TestReduceWithNonArray(t *testing.T) {
	input := `reduce(42, (a, x) => a + x, 0);`
	program := parse(input)
	comp := compiler.New()
	if err := comp.Compile(program); err != nil {
		t.Fatalf("compiler error: %s", err)
	}
	vm := New(comp.Constants())
	var buf bytes.Buffer
	vm.SetOutput(&buf)
	if err := vm.Run(comp.Bytecode().Instructions); err != nil {
		t.Fatalf("vm error: %s", err)
	}
	result := vm.LastPoppedStackElem()
	if result.Type != value.TYPE_FAILURE {
		t.Errorf("expected failure, got %s", result.Type)
	}
}

func TestReduceWithNonFunction(t *testing.T) {
	input := `reduce([1, 2], 42, 0);`
	program := parse(input)
	comp := compiler.New()
	if err := comp.Compile(program); err != nil {
		t.Fatalf("compiler error: %s", err)
	}
	vm := New(comp.Constants())
	var buf bytes.Buffer
	vm.SetOutput(&buf)
	if err := vm.Run(comp.Bytecode().Instructions); err != nil {
		t.Fatalf("vm error: %s", err)
	}
	result := vm.LastPoppedStackElem()
	if result.Type != value.TYPE_FAILURE {
		t.Errorf("expected failure, got %s", result.Type)
	}
}

func TestReduceWithWrongArgCount(t *testing.T) {
	input := `reduce([1, 2], (a, x) => a + x);`
	program := parse(input)
	comp := compiler.New()
	if err := comp.Compile(program); err != nil {
		t.Fatalf("compiler error: %s", err)
	}
	vm := New(comp.Constants())
	var buf bytes.Buffer
	vm.SetOutput(&buf)
	if err := vm.Run(comp.Bytecode().Instructions); err != nil {
		t.Fatalf("vm error: %s", err)
	}
	result := vm.LastPoppedStackElem()
	if result.Type != value.TYPE_FAILURE {
		t.Errorf("expected failure, got %s", result.Type)
	}
}

func TestFilterWithWrongArgCount(t *testing.T) {
	input := `filter([1, 2]);`
	program := parse(input)
	comp := compiler.New()
	if err := comp.Compile(program); err != nil {
		t.Fatalf("compiler error: %s", err)
	}
	vm := New(comp.Constants())
	var buf bytes.Buffer
	vm.SetOutput(&buf)
	if err := vm.Run(comp.Bytecode().Instructions); err != nil {
		t.Fatalf("vm error: %s", err)
	}
	result := vm.LastPoppedStackElem()
	if result.Type != value.TYPE_FAILURE {
		t.Errorf("expected failure, got %s", result.Type)
	}
}

// --- callValueSync with builtin (non-closure) callbacks ---

func TestMapWithBuiltinCallback(t *testing.T) {
	// map with to_string as callback (builtin function)
	input := `map([1, 2, 3], to_string);`
	program := parse(input)
	comp := compiler.New()
	if err := comp.Compile(program); err != nil {
		t.Fatalf("compiler error: %s", err)
	}
	vm := New(comp.Constants())
	if err := vm.Run(comp.Bytecode().Instructions); err != nil {
		t.Fatalf("vm error: %s", err)
	}
	result := vm.LastPoppedStackElem()
	if result.Type != value.TYPE_ARRAY {
		t.Fatalf("expected array, got %s", result.Type)
	}
	arr := result.AsArray()
	if len(arr) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(arr))
	}
	if arr[0].AsString() != "1" {
		t.Errorf("expected '1', got %q", arr[0].AsString())
	}
}

// --- Error message paths ---

func TestCallNonCallable(t *testing.T) {
	input := `x = 42; x(1);`
	program := parse(input)
	comp := compiler.New()
	if err := comp.Compile(program); err != nil {
		t.Fatalf("compiler error: %s", err)
	}
	vm := New(comp.Constants())
	err := vm.Run(comp.Bytecode().Instructions)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "cannot call") {
		t.Errorf("unexpected error: %s", err)
	}
}

func TestWrongArgCount(t *testing.T) {
	input := `func f(x, y) = x + y; f(1);`
	program := parse(input)
	comp := compiler.New()
	if err := comp.Compile(program); err != nil {
		t.Fatalf("compiler error: %s", err)
	}
	vm := New(comp.Constants())
	err := vm.Run(comp.Bytecode().Instructions)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "argument error") {
		t.Errorf("unexpected error: %s", err)
	}
}

// --- Index operator error paths ---

func TestIndexOnBool(t *testing.T) {
	input := `true[0];`
	program := parse(input)
	comp := compiler.New()
	if err := comp.Compile(program); err != nil {
		t.Fatalf("compiler error: %s", err)
	}
	vm := New(comp.Constants())
	err := vm.Run(comp.Bytecode().Instructions)
	if err == nil {
		t.Fatal("expected error for indexing bool")
	}
	if !strings.Contains(err.Error(), "not supported") {
		t.Errorf("unexpected error: %s", err)
	}
}

func TestTupleStringIndex(t *testing.T) {
	input := `(1, 2, 3)["0"];`
	program := parse(input)
	comp := compiler.New()
	if err := comp.Compile(program); err != nil {
		t.Fatalf("compiler error: %s", err)
	}
	vm := New(comp.Constants())
	err := vm.Run(comp.Bytecode().Instructions)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "tuple index must be an integer") {
		t.Errorf("unexpected error: %s", err)
	}
}

func TestArrayStringIndex(t *testing.T) {
	input := `[1, 2, 3]["0"];`
	program := parse(input)
	comp := compiler.New()
	if err := comp.Compile(program); err != nil {
		t.Fatalf("compiler error: %s", err)
	}
	vm := New(comp.Constants())
	err := vm.Run(comp.Bytecode().Instructions)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "array index must be an integer") {
		t.Errorf("unexpected error: %s", err)
	}
}

// --- Negation error ---

func TestNegateString(t *testing.T) {
	input := `-"hello";`
	program := parse(input)
	comp := compiler.New()
	if err := comp.Compile(program); err != nil {
		t.Fatalf("compiler error: %s", err)
	}
	vm := New(comp.Constants())
	err := vm.Run(comp.Bytecode().Instructions)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "cannot negate") {
		t.Errorf("unexpected error: %s", err)
	}
}

// --- Division by zero ---

func TestDivisionByZeroFloat(t *testing.T) {
	input := `1.0 / 0.0;`
	program := parse(input)
	comp := compiler.New()
	if err := comp.Compile(program); err != nil {
		t.Fatalf("compiler error: %s", err)
	}
	vm := New(comp.Constants())
	err := vm.Run(comp.Bytecode().Instructions)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "division by zero") {
		t.Errorf("unexpected error: %s", err)
	}
}

// --- Comparison error ---

func TestCompareStrings(t *testing.T) {
	input := `"a" < "b";`
	program := parse(input)
	comp := compiler.New()
	if err := comp.Compile(program); err != nil {
		t.Fatalf("compiler error: %s", err)
	}
	vm := New(comp.Constants())
	err := vm.Run(comp.Bytecode().Instructions)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "comparison") {
		t.Errorf("unexpected error: %s", err)
	}
}

// --- Binary op on bools ---

func TestBinaryOpOnBools(t *testing.T) {
	input := `true - false;`
	program := parse(input)
	comp := compiler.New()
	if err := comp.Compile(program); err != nil {
		t.Fatalf("compiler error: %s", err)
	}
	vm := New(comp.Constants())
	err := vm.Run(comp.Bytecode().Instructions)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "cannot apply binary operator") {
		t.Errorf("unexpected error: %s", err)
	}
}

// --- ADT index ---

func TestADTFieldIndex(t *testing.T) {
	input := `type Shape = Circle(r); c = Circle(5); c[0];`
	program := parse(input)
	comp := compiler.New()
	if err := comp.Compile(program); err != nil {
		t.Fatalf("compiler error: %s", err)
	}
	vm := New(comp.Constants())
	if err := vm.Run(comp.Bytecode().Instructions); err != nil {
		t.Fatalf("vm error: %s", err)
	}
	result := vm.LastPoppedStackElem()
	if result.AsInt() != 5 {
		t.Errorf("expected 5, got %v", result)
	}
}

func TestADTFieldOutOfBounds(t *testing.T) {
	input := `type Shape = Circle(r); c = Circle(5); c[5];`
	program := parse(input)
	comp := compiler.New()
	if err := comp.Compile(program); err != nil {
		t.Fatalf("compiler error: %s", err)
	}
	vm := New(comp.Constants())
	if err := vm.Run(comp.Bytecode().Instructions); err != nil {
		t.Fatalf("vm error: %s", err)
	}
	result := vm.LastPoppedStackElem()
	if result.Type != value.TYPE_NULL {
		t.Errorf("expected null for out of bounds ADT index, got %v", result)
	}
}

func TestADTFieldStringIndex(t *testing.T) {
	input := `type Shape = Circle(r); c = Circle(5); c["r"];`
	program := parse(input)
	comp := compiler.New()
	if err := comp.Compile(program); err != nil {
		t.Fatalf("compiler error: %s", err)
	}
	vm := New(comp.Constants())
	err := vm.Run(comp.Bytecode().Instructions)
	if err == nil {
		t.Fatal("expected error for string index on ADT")
	}
	if !strings.Contains(err.Error(), "ADT field index must be an integer") {
		t.Errorf("unexpected error: %s", err)
	}
}

// --- Spread operator error ---

func TestSpreadNonArray(t *testing.T) {
	input := `func f(a, b) = a + b; x = 42; x... |> f;`
	program := parse(input)
	comp := compiler.New()
	if err := comp.Compile(program); err != nil {
		t.Fatalf("compiler error: %s", err)
	}
	vm := New(comp.Constants())
	err := vm.Run(comp.Bytecode().Instructions)
	if err == nil {
		t.Fatal("expected error for spread on non-array")
	}
	if !strings.Contains(err.Error(), "spread") || !strings.Contains(err.Error(), "array") {
		t.Errorf("unexpected error: %s", err)
	}
}

// --- Member access error paths ---

func TestMemberAccessOnUnsupportedType(t *testing.T) {
	input := `x = 42; x.field;`
	program := parse(input)
	comp := compiler.New()
	if err := comp.Compile(program); err != nil {
		t.Fatalf("compiler error: %s", err)
	}
	vm := New(comp.Constants())
	err := vm.Run(comp.Bytecode().Instructions)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "cannot access property") {
		t.Errorf("unexpected error: %s", err)
	}
}

// --- VerifyCachedModule ---

func TestVerifyCachedModule(t *testing.T) {
	opts := DefaultRunOptions()

	// No lock file entry should pass
	err := opts.VerifyCachedModule("test", "/nonexistent")
	if err != nil {
		t.Errorf("expected nil for module without lock entry, got %v", err)
	}

	// With lock file
	opts.LockFile = &LockFile{
		Modules: map[string]ModuleLock{
			"test": {
				Version: "1.0.0",
				Files: map[string]string{
					"mod.ca": "abc123", // wrong hash
				},
			},
		},
	}

	// Should fail: file doesn't exist
	err = opts.VerifyCachedModule("test", "/nonexistent/path")
	if err == nil {
		t.Error("expected error for missing file")
	}

	// Create temp file and verify hash mismatch
	tmpDir := t.TempDir()
	modFile := filepath.Join(tmpDir, "mod.ca")
	os.WriteFile(modFile, []byte("hello"), 0644)

	opts.LockFile.Modules["test"] = ModuleLock{
		Version: "1.0.0",
		Files: map[string]string{
			"mod.ca": "wronghash",
		},
	}

	err = opts.VerifyCachedModule("test", tmpDir)
	if err == nil {
		t.Error("expected error for hash mismatch")
	}
	if !strings.Contains(err.Error(), "hash mismatch") {
		t.Errorf("unexpected error: %s", err)
	}
}

// --- Cache helpers ---

func TestSaveToCacheDir(t *testing.T) {
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "test_cache", "author", "module")

	err := saveToCacheDir(cacheDir, "mod.ca", []byte("export let x = 42"))
	if err != nil {
		t.Fatalf("saveToCacheDir error: %s", err)
	}

	content := readCachedModule(cacheDir)
	if content == nil {
		t.Fatal("expected cached content")
	}
	if string(content) != "export let x = 42" {
		t.Errorf("expected correct content, got %q", string(content))
	}
}

func TestEnsureCacheDir(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	err := ensureCacheDir("testauthor", "testmodule")
	if err != nil {
		t.Fatalf("ensureCacheDir error: %s", err)
	}
}

func TestReadCachedModuleNotFound(t *testing.T) {
	content := readCachedModule("/nonexistent/path")
	if content != nil {
		t.Error("expected nil for non-existent module")
	}
}

func TestClearCache(t *testing.T) {
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "test_cache")
	os.MkdirAll(cacheDir, 0755)
	os.WriteFile(filepath.Join(cacheDir, "test.txt"), []byte("test"), 0644)

	err := ClearCache()
	_ = err
}

// --- Pipeline operator with higher-order functions ---

func TestPipelineMap(t *testing.T) {
	tests := []vmTestCase{
		{`[1, 2, 3] |> map(x => x * 2);`, []int{2, 4, 6}},
	}
	runVmTests(t, tests)
}

func TestPipelineFilter(t *testing.T) {
	tests := []vmTestCase{
		{`[1, 2, 3, 4, 5] |> filter(x => x > 3);`, []int{4, 5}},
	}
	runVmTests(t, tests)
}

func TestPipelineReduce(t *testing.T) {
	tests := []vmTestCase{
		{`[1, 2, 3, 4] |> reduce((acc, x) => acc + x, 0);`, 10},
	}
	runVmTests(t, tests)
}

// --- Chained pipelines ---

func TestChainedPipelines(t *testing.T) {
	tests := []vmTestCase{
		{`[1, 2, 3, 4, 5] |> filter(x => x > 2) |> map(x => x * 10);`, []int{30, 40, 50}},
	}
	runVmTests(t, tests)
}

// --- Tail call optimization ---

func TestTailCallWithDifferentFunctions(t *testing.T) {
	tests := []vmTestCase{
		{`func factorial(n, acc) = match n 0 => acc _ => factorial(n - 1, n * acc); factorial(10, 1);`, 3628800},
	}
	runVmTests(t, tests)
}

// --- Integer arithmetic edge cases ---

func TestIntegerPow(t *testing.T) {
	tests := []vmTestCase{
		{`2 ** 10;`, 1024},
		{`3 ** 0;`, 1},
	}
	runVmTests(t, tests)
}

func TestIntegerMod(t *testing.T) {
	tests := []vmTestCase{
		{`10 % 3;`, 1},
		{`7 % 2;`, 1},
	}
	runVmTests(t, tests)
}

// --- Float arithmetic edge cases ---

func TestFloatMod(t *testing.T) {
	tests := []vmTestCase{
		{`10.0 % 3.0;`, 1.0},
	}
	runVmTests(t, tests)
}

func TestFloatPow(t *testing.T) {
	tests := []vmTestCase{
		{`2.0 ** 3.0;`, 8.0},
	}
	runVmTests(t, tests)
}

// --- Mixed int/float operations ---

func TestMixedDivision(t *testing.T) {
	tests := []vmTestCase{
		{`10 / 3;`, 3.3333333333333335},
		{`6 / 2;`, 3.0},
	}
	runVmTests(t, tests)
}

// --- has() with unsupported type ---

func TestHasWithUnsupportedType(t *testing.T) {
	result := builtinHas(value.Int(42), value.Int(1))
	if result.Type != value.TYPE_BOOL || result.AsBool() != false {
		t.Errorf("expected false for unsupported type, got %v", result)
	}
}

// --- Equality tests ---

func TestEqualityComplex(t *testing.T) {
	tests := []vmTestCase{
		{`1 == 1;`, true},
		{`1 != 2;`, true},
		{`"hello" == "hello";`, true},
		{`"hello" != "world";`, true},
		{`true == true;`, true},
		{`false == false;`, true},
	}
	runVmTests(t, tests)
}

// --- Array concatenation ---

func TestArrayConcatSpread(t *testing.T) {
	// Use concat builtin instead of spread syntax in array literal
	tests := []vmTestCase{
		{`concat([1, 2], [3, 4, 5]);`, []int{1, 2, 3, 4, 5}},
	}
	runVmTests(t, tests)
}

// --- Closure with free variables ---

func TestClosureWithFreeVars(t *testing.T) {
	tests := []vmTestCase{
		{`func make_adder(x) = y => x + y; add10 = make_adder(10); add10(5);`, 15},
		{`func outer() = do x = 10; inner = () => x + 20; inner() end; outer();`, 30},
	}
	runVmTests(t, tests)
}

// --- Module member access on hash ---

func TestHashDotAccess(t *testing.T) {
	tests := []vmTestCase{
		{`h = {name: "Alice"}; h.name;`, "Alice"},
		{`h = {x: 10}; h.y;`, nil}, // missing key returns null
	}
	runVmTests(t, tests)
}

// --- getFrameSourceMap edge case ---

func TestGetFrameSourceMapNil(t *testing.T) {
	vm := New(nil)
	result := vm.getFrameSourceMap(nil)
	if result != nil {
		t.Error("expected nil for nil frame")
	}

	// Frame with nil closure
	frame := &Frame{cl: nil}
	result = vm.getFrameSourceMap(frame)
	if result != nil {
		t.Error("expected nil for nil closure")
	}

	// Frame with closure but nil fn
	frame = &Frame{cl: &value.Closure{Fn: nil}}
	result = vm.getFrameSourceMap(frame)
	if result != nil {
		t.Error("expected nil for nil fn")
	}
}

// --- stackTrace ---

func TestStackTrace(t *testing.T) {
	vm := New(nil)
	// With only main frame, should return nil
	trace := vm.stackTrace()
	if trace != nil {
		t.Errorf("expected nil for single frame, got %v", trace)
	}
}

// --- Comparisons: all operators ---

func TestAllComparisonOps(t *testing.T) {
	tests := []vmTestCase{
		{`1 < 2;`, true},
		{`2 > 1;`, true},
		{`1 <= 1;`, true},
		{`1 >= 1;`, true},
		{`2 <= 1;`, false},
		{`1 >= 2;`, false},
		{`1.0 < 2.0;`, true},
		{`2.0 > 1.0;`, true},
	}
	runVmTests(t, tests)
}

// --- JumpIfTrue/JumpIfFalse via logical operators ---

func TestLogicalShortCircuit(t *testing.T) {
	tests := []vmTestCase{
		{`true && true;`, true},
		{`true && false;`, false},
		{`false && true;`, false},
		{`true || false;`, true},
		{`false || true;`, true},
		{`false || false;`, false},
	}
	runVmTests(t, tests)
}

// --- Do expression ---

func TestDoExpressionCoverage(t *testing.T) {
	tests := []vmTestCase{
		{`do x = 1; y = 2; x + y end;`, 3},
	}
	runVmTests(t, tests)
}

// --- GetModuleLock nil paths ---

func TestGetModuleLockNilOpts(t *testing.T) {
	var opts *RunOptions
	result := opts.GetModuleLock("test")
	if result != nil {
		t.Error("expected nil from nil opts")
	}
}

func TestGetModuleLockNilLockFile(t *testing.T) {
	opts := DefaultRunOptions()
	result := opts.GetModuleLock("test")
	if result != nil {
		t.Error("expected nil when no lock file")
	}
}

func TestGetModuleLockMissingModule(t *testing.T) {
	opts := DefaultRunOptions()
	opts.LockFile = &LockFile{Modules: map[string]ModuleLock{}}
	result := opts.GetModuleLock("missing")
	if result != nil {
		t.Error("expected nil for missing module")
	}
}

func TestGetModuleLockFound(t *testing.T) {
	opts := DefaultRunOptions()
	opts.LockFile = &LockFile{
		Modules: map[string]ModuleLock{
			"test": {Version: "1.0.0", URL: "https://example.com"},
		},
	}
	result := opts.GetModuleLock("test")
	if result == nil {
		t.Fatal("expected non-nil lock")
	}
	if result.Version != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", result.Version)
	}
}

// --- String concatenation via + operator ---

func TestStringConcatWithNonString(t *testing.T) {
	tests := []vmTestCase{
		{`"value: " + 42;`, "value: 42"},
		{`42 + " items";`, "42 items"},
		{`"bool: " + true;`, "bool: true"},
	}
	runVmTests(t, tests)
}

// --- builtinConcat with array types ---

func TestBuiltinConcatArrayMixed(t *testing.T) {
	arr := value.Array([]value.Value{value.Int(1)})
	single := value.Int(2)
	result := builtinConcat(arr, single)
	if result.Type != value.TYPE_ARRAY {
		t.Fatalf("expected array, got %s", result.Type)
	}
	resultArr := result.AsArray()
	if len(resultArr) != 2 {
		t.Errorf("expected 2 elements, got %d", len(resultArr))
	}
}

// --- Len on tuple ---

func TestBuiltinLenTuple(t *testing.T) {
	tup := value.Tuple([]value.Value{value.Int(1), value.Int(2), value.Int(3)})
	result := builtinLen(tup)
	if result.AsInt() != 3 {
		t.Errorf("expected 3, got %d", result.AsInt())
	}
}

// --- Len on hash ---

func TestBuiltinLenHash(t *testing.T) {
	h := value.NewHash()
	h.Set(value.String("a"), value.Int(1))
	result := builtinLen(value.HashVal(h))
	if result.AsInt() != 1 {
		t.Errorf("expected 1, got %d", result.AsInt())
	}
}

// --- Additional coverage tests ---

// Test division by zero with integers
func TestDivisionByZeroInt(t *testing.T) {
	input := `1 / 0;`
	program := parse(input)
	comp := compiler.New()
	if err := comp.Compile(program); err != nil {
		t.Fatalf("compiler error: %s", err)
	}
	vm := New(comp.Constants())
	err := vm.Run(comp.Bytecode().Instructions)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "division by zero") {
		t.Errorf("unexpected error: %s", err)
	}
}

// Test match with ADT patterns
func TestMatchADTPatterns(t *testing.T) {
	tests := []vmTestCase{
		// ADT with match
		{`type Maybe = Some(v) | None; x = Some(42); match x Some(v) => v _ => 0;`, int64(42)},
		{`type Maybe = Some(v) | None; x = None; match x Some(v) => v _ => 0;`, int64(0)},
	}
	runVmTests(t, tests)
}

// Test nested function calls
func TestNestedFunctionCalls(t *testing.T) {
	tests := []vmTestCase{
		{`func add(a, b) = a + b; func double(x) = x * 2; double(add(3, 4));`, 14},
		{`func f(x) = x + 1; f(f(f(0)));`, 3},
	}
	runVmTests(t, tests)
}

// Test lambda expressions
func TestLambdaCoverage(t *testing.T) {
	tests := []vmTestCase{
		{`f = x => x * 2; f(5);`, 10},
		{`f = (x, y) => x + y; f(3, 4);`, 7},
		{`f = () => 42; f();`, 42},
	}
	runVmTests(t, tests)
}

// Test match wildcard
func TestMatchWildcard(t *testing.T) {
	tests := []vmTestCase{
		{`match 99 1 => 10 2 => 20 _ => 0;`, 0},
		{`match "hello" "world" => 1 _ => 2;`, 2},
	}
	runVmTests(t, tests)
}

// Test if-like expressions via match
func TestConditionalMatch(t *testing.T) {
	tests := []vmTestCase{
		{`match true true => 1 false => 0;`, 1},
		{`match false true => 1 false => 0;`, 0},
	}
	runVmTests(t, tests)
}

// Test recursive pattern with match guard
func TestMatchGuardCoverage(t *testing.T) {
	tests := []vmTestCase{
		{`match 5 x if x > 0 => "positive" _ => "non-positive";`, "positive"},
		{`match -3 x if x > 0 => "positive" _ => "non-positive";`, "non-positive"},
	}
	runVmTests(t, tests)
}

// Test effect functions
func TestEffectFunctionsCoverage(t *testing.T) {
	input := `func! getData() = 100; getData();`
	program := parse(input)
	comp := compiler.New()
	if err := comp.Compile(program); err != nil {
		t.Fatalf("compiler error: %s", err)
	}
	vm := New(comp.Constants())
	if err := vm.Run(comp.Bytecode().Instructions); err != nil {
		t.Fatalf("vm error: %s", err)
	}
	result := vm.LastPoppedStackElem()
	if !result.IsSuccess() {
		t.Errorf("expected success, got %s", result.Type)
	}
	inner := result.AsSuccess()
	if inner.AsInt() != 100 {
		t.Errorf("expected 100, got %d", inner.AsInt())
	}
}

// Test error handling with !?
func TestErrorHandlingCoverage(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{`success(42) !? { success(r) => r failure(e) => 0 };`, 42},
		{`failure("error") !? { success(r) => r failure(e) => 0 };`, 0},
	}

	for _, tt := range tests {
		program := parse(tt.input)
		comp := compiler.New()
		if err := comp.Compile(program); err != nil {
			t.Fatalf("compiler error for %q: %s", tt.input, err)
		}
		vm := New(comp.Constants())
		if err := vm.Run(comp.Bytecode().Instructions); err != nil {
			t.Fatalf("vm error for %q: %s", tt.input, err)
		}
		result := vm.LastPoppedStackElem()
		if result.Type != value.TYPE_INT {
			t.Errorf("expected int for %q, got %s", tt.input, result.Type)
			continue
		}
		if result.AsInt() != tt.expected {
			t.Errorf("expected %d for %q, got %d", tt.expected, tt.input, result.AsInt())
		}
	}
}

// Test constraint
func TestConstraintCoverage(t *testing.T) {
	input := `constraint Positive(n) = n > 0; 5 |> Positive?;`
	program := parse(input)
	comp := compiler.New()
	if err := comp.Compile(program); err != nil {
		t.Fatalf("compiler error: %s", err)
	}
	vm := New(comp.Constants())
	if err := vm.Run(comp.Bytecode().Instructions); err != nil {
		t.Fatalf("vm error: %s", err)
	}
	result := vm.LastPoppedStackElem()
	if !result.IsSuccess() {
		t.Errorf("expected success for positive constraint, got %s", result.Type)
	}
}

// Test hash operations
func TestHashOperationsCoverage(t *testing.T) {
	tests := []vmTestCase{
		// keys and values
		{`len(keys({a: 1, b: 2}));`, int64(2)},
		{`len(values({a: 1, b: 2}));`, int64(2)},
		// Nested hash
		{`h = {user: {name: "Alice"}}; h.user.name;`, "Alice"},
	}
	runVmTests(t, tests)
}

// Test array destructuring
func TestArrayDestructuringCoverage(t *testing.T) {
	tests := []vmTestCase{
		{`[a, b, c] = [1, 2, 3]; a + b + c;`, int64(6)},
		{`[h | t] = [1, 2, 3]; h;`, int64(1)},
	}
	runVmTests(t, tests)
}

// Test string operations via modules
func TestStringModuleCoverage(t *testing.T) {
	tests := []vmTestCase{
		{`use core.string; string.upper("hello");`, "HELLO"},
		{`use core.string; string.lower("HELLO");`, "hello"},
		{`use core.string; string.trim("  hi  ");`, "hi"},
		{`use core.string; string.starts_with("hello", "he");`, true},
		{`use core.string; string.ends_with("hello", "lo");`, true},
	}
	runVmTests(t, tests)
}

// Test math module coverage
func TestMathModuleCoverage(t *testing.T) {
	tests := []vmTestCase{
		{`use core.math; math.abs(-5);`, int64(5)},
		{`use core.math; math.min(3, 7);`, int64(3)},
		{`use core.math; math.max(3, 7);`, int64(7)},
		{`use core.math; math.floor(3.7);`, int64(3)},
		{`use core.math; math.ceil(3.2);`, int64(4)},
	}
	runVmTests(t, tests)
}

// Test not operator
func TestNotOperator(t *testing.T) {
	tests := []vmTestCase{
		{`!true;`, false},
		{`!false;`, true},
		{`!(1 == 2);`, true},
	}
	runVmTests(t, tests)
}

// Test float negation
func TestFloatNegation(t *testing.T) {
	tests := []vmTestCase{
		{`-3.14;`, -3.14},
		{`-(-2.5);`, 2.5},
	}
	runVmTests(t, tests)
}

// Test integer negation
func TestIntegerNegation(t *testing.T) {
	tests := []vmTestCase{
		{`-42;`, -42},
		{`-(-10);`, 10},
	}
	runVmTests(t, tests)
}

// Test pipe with lambda
func TestPipeWithLambda(t *testing.T) {
	tests := []vmTestCase{
		{`5 |> (x => x * 2);`, 10},
		{`10 |> (x => x + 5) |> (x => x * 2);`, 30},
	}
	runVmTests(t, tests)
}

// Test error propagation pipe
func TestErrorPropPipeCoverage(t *testing.T) {
	input := `
		func wrap_success(x) = success(x * 2);
		result = 5 |>? wrap_success;
		result;
	`
	program := parse(input)
	comp := compiler.New()
	if err := comp.Compile(program); err != nil {
		t.Fatalf("compiler error: %s", err)
	}
	vm := New(comp.Constants())
	if err := vm.Run(comp.Bytecode().Instructions); err != nil {
		t.Fatalf("vm error: %s", err)
	}
	result := vm.LastPoppedStackElem()
	if result.AsInt() != 10 {
		t.Errorf("expected 10, got %v (type %s)", result, result.Type)
	}
}
