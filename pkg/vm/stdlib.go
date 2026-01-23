package vm

import (
	"embed"
	"fmt"
	"sort"
	"strings"

	"github.com/example/calcium/pkg/compiler"
	"github.com/example/calcium/pkg/lexer"
	"github.com/example/calcium/pkg/parser"
	"github.com/example/calcium/pkg/value"
)

//go:embed stdlib/core/*.ca
var stdlibFS embed.FS

// stdlibCache caches compiled stdlib modules
var stdlibCache = make(map[string]*value.Module)

// registerPrimitives registers all primitive functions as builtins
// These are available globally with __ prefix (e.g., __print, __sqrt)
func (vm *VM) registerPrimitives() {
	primitives := GetPrimitives()

	// Add primitives to the builtins list
	startIndex := len(vm.builtins)
	for name, builtin := range primitives {
		vm.builtins = append(vm.builtins, builtin)
		// Also add to symbol table if needed
		_ = name      // name is used for debugging
		_ = startIndex // index tracking
	}
}

// LoadStdlibModule loads a stdlib module by name (e.g., "core.math")
// Returns the module if found, or nil if not a stdlib module
func (vm *VM) LoadStdlibModule(moduleName string) (*value.Module, error) {
	// Check cache first
	if mod, ok := stdlibCache[moduleName]; ok {
		return mod, nil
	}

	// Convert module name to file path (e.g., "core.math" -> "stdlib/core/math.ca")
	filePath := "stdlib/" + strings.ReplaceAll(moduleName, ".", "/") + ".ca"

	// Try to read from embedded FS
	content, err := stdlibFS.ReadFile(filePath)
	if err != nil {
		// Not a stdlib module
		return nil, nil
	}

	// Parse the source
	l := lexer.New(string(content))
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		return nil, fmt.Errorf("parse errors in %s:\n  %s", filePath, strings.Join(p.Errors(), "\n  "))
	}

	// Create a compiler with primitive builtins registered
	comp := compiler.New()

	// Register primitives as builtins in the compiler's symbol table
	// Use sorted order to ensure consistent builtin indices
	primitives := GetPrimitives()
	primitiveNames := make([]string, 0, len(primitives))
	for name := range primitives {
		primitiveNames = append(primitiveNames, name)
	}
	sort.Strings(primitiveNames)

	idx := 0
	for _, name := range primitiveNames {
		comp.SymbolTable().DefineBuiltin(idx, name)
		idx++
	}

	// Also register standard builtins
	standardBuiltins := []string{"len", "concat", "to_string", "get", "has", "head", "tail", "push", "range", "map", "filter", "reduce", "keys", "values"}
	for _, name := range standardBuiltins {
		comp.SymbolTable().DefineBuiltin(idx, name)
		idx++
	}

	if err := comp.Compile(program); err != nil {
		return nil, fmt.Errorf("compilation error in %s: %w", filePath, err)
	}

	// Create a new VM to execute the module
	moduleVM := New(comp.Constants())

	// Register primitives as builtins in the VM (in same sorted order)
	primitiveBuiltins := make([]*value.Builtin, 0, len(primitives))
	for _, name := range primitiveNames {
		primitiveBuiltins = append(primitiveBuiltins, primitives[name])
	}
	// Prepend primitive builtins
	moduleVM.builtins = append(primitiveBuiltins, moduleVM.builtins...)

	// Execute the module
	if err := moduleVM.Run(comp.Bytecode().Instructions); err != nil {
		return nil, fmt.Errorf("runtime error in %s: %w", filePath, err)
	}

	// Extract exports from compiled symbols
	module := &value.Module{
		Name:    moduleName,
		Exports: make(map[string]value.Value),
	}

	// Get the constants, globals, and builtins from the module
	moduleConstants := comp.Constants()
	moduleGlobals := moduleVM.globals
	moduleBuiltins := moduleVM.builtins

	// Set Constants, Globals, and Builtins on ALL closures in moduleGlobals
	// This includes private helper functions that exported functions may call
	for i := range moduleGlobals {
		if moduleGlobals[i].Type == value.TYPE_CLOSURE {
			cl := moduleGlobals[i].AsClosure()
			if len(cl.Fn.Constants) == 0 {
				cl.Fn.Constants = moduleConstants
			}
			if len(cl.Fn.Globals) == 0 {
				cl.Fn.Globals = moduleGlobals
			}
			if len(cl.Fn.Builtins) == 0 {
				cl.Fn.Builtins = moduleBuiltins
			}
		}
	}

	for name, symbol := range comp.ExportedSymbols() {
		// Skip private names (starting with _)
		if strings.HasPrefix(name, "_") {
			continue
		}
		module.Exports[name] = moduleVM.globals[symbol.Index]
	}

	// Cache the module
	stdlibCache[moduleName] = module

	return module, nil
}

// IsStdlibModule checks if a module name corresponds to a stdlib module
func IsStdlibModule(moduleName string) bool {
	filePath := "stdlib/" + strings.ReplaceAll(moduleName, ".", "/") + ".ca"
	_, err := stdlibFS.ReadFile(filePath)
	return err == nil
}
