package vm

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ytnobody/calcium-lang/pkg/compiler"
	"github.com/ytnobody/calcium-lang/pkg/lexer"
	"github.com/ytnobody/calcium-lang/pkg/parser"
	"github.com/ytnobody/calcium-lang/pkg/value"
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
		_ = name       // name is used for debugging
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
	standardBuiltins := []string{"len", "concat", "to_string", "get", "has", "head", "tail", "push", "range", "map", "filter", "reduce", "keys", "values", "hash_set", "hash_merge"}
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

// LoadExternalModule loads a module from the filesystem or remote sources
// It searches in the following order:
// 1. VM cache (vm.modules)
// 2. Global cache (stdlibCache)
// 3. Stdlib (embedded FS)
// 4. Local filesystem (calcium_modules/, relative paths)
// 5. Remote sources (Boneyard registry, GitHub direct)
// Returns the module if found, or nil if not found
func (vm *VM) LoadExternalModule(moduleName string) (*value.Module, error) {
	// Apply import map if available
	resolvedName := moduleName
	if vm.runOptions != nil {
		resolvedName = vm.runOptions.ResolveModuleName(moduleName)
	}

	// Check cache first (using original name for cache key)
	if mod, ok := stdlibCache[moduleName]; ok {
		return mod, nil
	}
	// Also check resolved name
	if resolvedName != moduleName {
		if mod, ok := stdlibCache[resolvedName]; ok {
			return mod, nil
		}
	}

	// Check if this is a remote module (contains /)
	if strings.Contains(resolvedName, "/") {
		// Check if it's a URL format (github.com/...)
		if strings.HasPrefix(resolvedName, "github.com/") {
			return vm.LoadGitHubURLModule(resolvedName)
		}
		// Otherwise it's author/module format
		parts := strings.SplitN(resolvedName, "/", 2)
		if len(parts) == 2 {
			return vm.LoadRemoteModule(parts[0], parts[1])
		}
	}

	// Get the base directory for searching
	var searchDirs []string

	if vm.sourcePath != "" {
		sourceDir := filepath.Dir(vm.sourcePath)
		// Add source directory and parent directories
		for dir := sourceDir; dir != "/" && dir != "."; dir = filepath.Dir(dir) {
			searchDirs = append(searchDirs, dir)
		}
	}

	// Also search current working directory
	if cwd, err := os.Getwd(); err == nil {
		searchDirs = append(searchDirs, cwd)
	}

	// Convert module name to directory path (e.g., "json" -> "json/mod.ca")
	modulePath := strings.ReplaceAll(moduleName, ".", "/")

	// Try to find the module
	var content []byte
	var foundPath string

	for _, dir := range searchDirs {
		// Try moduleName/mod.ca
		tryPath := filepath.Join(dir, modulePath, "mod.ca")
		if data, err := os.ReadFile(tryPath); err == nil {
			content = data
			foundPath = tryPath
			break
		}

		// Try moduleName.ca
		tryPath = filepath.Join(dir, modulePath+".ca")
		if data, err := os.ReadFile(tryPath); err == nil {
			content = data
			foundPath = tryPath
			break
		}
	}

	if content == nil {
		return nil, nil // Not found
	}

	// Parse the source
	l := lexer.New(string(content))
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		return nil, fmt.Errorf("parse errors in %s:\n  %s", foundPath, strings.Join(p.Errors(), "\n  "))
	}

	// Create a compiler with primitive builtins registered
	comp := compiler.New()

	// Register primitives as builtins in the compiler's symbol table
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
	standardBuiltins := []string{"len", "concat", "to_string", "get", "has", "head", "tail", "push", "range", "map", "filter", "reduce", "keys", "values", "hash_set", "hash_merge"}
	for _, name := range standardBuiltins {
		comp.SymbolTable().DefineBuiltin(idx, name)
		idx++
	}

	if err := comp.Compile(program); err != nil {
		return nil, fmt.Errorf("compilation error in %s: %w", foundPath, err)
	}

	// Create a new VM to execute the module
	moduleVM := New(comp.Constants())
	moduleVM.sourcePath = filepath.Dir(foundPath) // Set source path for nested module loading

	// Register primitives as builtins in the VM (in same sorted order)
	primitiveBuiltins := make([]*value.Builtin, 0, len(primitives))
	for _, name := range primitiveNames {
		primitiveBuiltins = append(primitiveBuiltins, primitives[name])
	}
	// Prepend primitive builtins
	moduleVM.builtins = append(primitiveBuiltins, moduleVM.builtins...)

	// Execute the module
	if err := moduleVM.Run(comp.Bytecode().Instructions); err != nil {
		return nil, fmt.Errorf("runtime error in %s: %w", foundPath, err)
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

// LoadRemoteModule loads a module from remote sources (author/module format)
// It tries:
// 1. In-memory cache (stdlibCache)
// 2. Local project directory (calcium_modules/author/module)
// 3. Global cache directory (~/.calcium/cache/author/module)
// 4. Direct GitHub fetch (saves to global cache) - skipped if --cached-only
func (vm *VM) LoadRemoteModule(author, name string) (*value.Module, error) {
	cacheKey := author + "/" + name

	// 1. Check in-memory cache
	if mod, ok := stdlibCache[cacheKey]; ok {
		return mod, nil
	}

	var content []byte
	var moduleDir string

	// 2. Check local project's calcium_modules/
	localDirs := vm.getLocalModuleDirs()
	for _, dir := range localDirs {
		localPath := filepath.Join(dir, "calcium_modules", author, name)
		if c := readCachedModule(localPath); c != nil {
			content = c
			moduleDir = localPath
			break
		}
	}

	// 3. Check global cache directory
	if content == nil {
		globalDir := getModuleCachePath(author, name)
		if c := readCachedModule(globalDir); c != nil {
			content = c
			moduleDir = globalDir
		}
	}

	// 4. If not found, fetch from GitHub (save to global cache)
	// Skip if --cached-only is set
	if content == nil {
		if vm.runOptions != nil && vm.runOptions.CachedOnly {
			return nil, fmt.Errorf("module not in cache (--cached-only mode): %s/%s", author, name)
		}
		content = fetchFromGitHubDirect(author, name)
		if content == nil {
			return nil, fmt.Errorf("remote module not found: %s/%s", author, name)
		}
		// Save to global cache
		globalDir := getModuleCachePath(author, name)
		saveToCacheDir(globalDir, "mod.ca", content)
		moduleDir = globalDir
	}

	// Compile and cache the module
	return vm.compileAndCacheRemoteModule(cacheKey, content, moduleDir)
}

// getLocalModuleDirs returns directories to search for local modules
func (vm *VM) getLocalModuleDirs() []string {
	var dirs []string

	// Add source file directory and parents
	if vm.sourcePath != "" {
		for dir := filepath.Dir(vm.sourcePath); dir != "/" && dir != "."; dir = filepath.Dir(dir) {
			dirs = append(dirs, dir)
		}
	}

	// Add current working directory
	if cwd, err := os.Getwd(); err == nil {
		dirs = append(dirs, cwd)
	}

	return dirs
}

// LoadGitHubURLModule loads a module from a GitHub URL
// URL format: "github.com/author/repo"
func (vm *VM) LoadGitHubURLModule(url string) (*value.Module, error) {
	cacheKey := url

	// 1. Check global cache
	if mod, ok := stdlibCache[cacheKey]; ok {
		return mod, nil
	}

	// Parse URL to get author and repo
	author, repo := parseGitHubURL(url)
	if author == "" || repo == "" {
		return nil, fmt.Errorf("invalid GitHub URL: %s", url)
	}

	// 2. Check local cache directory
	cacheDir := getModuleCachePath(author, repo)
	content := readCachedModule(cacheDir)

	// 3. If not in cache, fetch from GitHub
	// Skip if --cached-only is set
	if content == nil {
		if vm.runOptions != nil && vm.runOptions.CachedOnly {
			return nil, fmt.Errorf("module not in cache (--cached-only mode): %s", url)
		}
		// Try main branch first, then master
		content = fetchRawGitHub(author, repo, "main", "mod.ca")
		if content == nil {
			content = fetchRawGitHub(author, repo, "master", "mod.ca")
		}
		if content == nil {
			return nil, fmt.Errorf("GitHub module not found: %s", url)
		}
		// Save to cache
		saveToCacheDir(cacheDir, "mod.ca", content)
	}

	// Compile and cache the module
	return vm.compileAndCacheRemoteModule(cacheKey, content, cacheDir)
}

// compileAndCacheRemoteModule compiles a remote module and caches it
func (vm *VM) compileAndCacheRemoteModule(cacheKey string, content []byte, cacheDir string) (*value.Module, error) {
	// Parse the source
	l := lexer.New(string(content))
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		return nil, fmt.Errorf("parse errors in remote module %s:\n  %s", cacheKey, strings.Join(p.Errors(), "\n  "))
	}

	// Create a compiler with primitive builtins registered
	comp := compiler.New()

	// Register primitives as builtins in the compiler's symbol table
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
	standardBuiltins := []string{"len", "concat", "to_string", "get", "has", "head", "tail", "push", "range", "map", "filter", "reduce", "keys", "values", "hash_set", "hash_merge"}
	for _, name := range standardBuiltins {
		comp.SymbolTable().DefineBuiltin(idx, name)
		idx++
	}

	if err := comp.Compile(program); err != nil {
		return nil, fmt.Errorf("compilation error in remote module %s: %w", cacheKey, err)
	}

	// Create a new VM to execute the module
	moduleVM := New(comp.Constants())
	moduleVM.sourcePath = cacheDir // Set source path for nested module loading

	// Register primitives as builtins in the VM (in same sorted order)
	primitiveBuiltins := make([]*value.Builtin, 0, len(primitives))
	for _, name := range primitiveNames {
		primitiveBuiltins = append(primitiveBuiltins, primitives[name])
	}
	// Prepend primitive builtins
	moduleVM.builtins = append(primitiveBuiltins, moduleVM.builtins...)

	// Execute the module
	if err := moduleVM.Run(comp.Bytecode().Instructions); err != nil {
		return nil, fmt.Errorf("runtime error in remote module %s: %w", cacheKey, err)
	}

	// Extract exports from compiled symbols
	module := &value.Module{
		Name:    cacheKey,
		Exports: make(map[string]value.Value),
	}

	// Get the constants, globals, and builtins from the module
	moduleConstants := comp.Constants()
	moduleGlobals := moduleVM.globals
	moduleBuiltins := moduleVM.builtins

	// Set Constants, Globals, and Builtins on ALL closures in moduleGlobals
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
	stdlibCache[cacheKey] = module

	return module, nil
}
