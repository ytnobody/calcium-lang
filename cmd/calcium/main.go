package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/ytnobody/calcium-lang/pkg/bytecode"
	"github.com/ytnobody/calcium-lang/pkg/compiler"
	"github.com/ytnobody/calcium-lang/pkg/lexer"
	"github.com/ytnobody/calcium-lang/pkg/optimizer"
	"github.com/ytnobody/calcium-lang/pkg/parser"
	"github.com/ytnobody/calcium-lang/pkg/value"
	"github.com/ytnobody/calcium-lang/pkg/vm"
)

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorBold   = "\033[1m"
)

// isTerminal checks if stdout is a terminal (for color support)
func isTerminal() bool {
	fileInfo, _ := os.Stdout.Stat()
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// formatError formats an error message with optional colors
func formatError(filename string, errMsg string, useColor bool) string {
	if useColor {
		return fmt.Sprintf("%s%s%s:%s error%s\n%s",
			colorBold, colorCyan, filename, colorRed, colorReset, errMsg)
	}
	return fmt.Sprintf("%s: error\n%s", filename, errMsg)
}

// formatParseErrors formats multiple parse errors for display
func formatParseErrors(filename string, errors []string, useColor bool) string {
	var sb strings.Builder
	for i, e := range errors {
		if i > 0 {
			sb.WriteString("\n\n")
		}
		if useColor {
			sb.WriteString(fmt.Sprintf("%s%s%s:%s error%s\n%s",
				colorBold, colorCyan, filename, colorRed, colorReset, e))
		} else {
			sb.WriteString(fmt.Sprintf("%s: error\n%s", filename, e))
		}
	}
	return sb.String()
}

// formatCompileErrors formats compiler errors (which may be single or multiple) for display
func formatCompileErrors(filename string, err error, useColor bool) string {
	errStr := err.Error()
	// Split on double newlines which separate multiple errors
	parts := strings.Split(errStr, "\n\n")

	var sb strings.Builder
	for i, part := range parts {
		if i > 0 {
			sb.WriteString("\n\n")
		}
		if useColor {
			sb.WriteString(fmt.Sprintf("%s%s%s:%s error%s\n%s",
				colorBold, colorCyan, filename, colorRed, colorReset, part))
		} else {
			sb.WriteString(fmt.Sprintf("%s: error\n%s", filename, part))
		}
	}
	return sb.String()
}

const version = "0.2.0"
const packMagic = "BONEPACK"

func main() {
	// Check if this binary has embedded .bone data
	if tryRunEmbedded() {
		return
	}

	args := os.Args[1:]

	if len(args) == 0 {
		runREPL()
		return
	}

	switch args[0] {
	case "run":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: calcium run [-O0|-O1|-O2] <file.ca|file.bone>")
			os.Exit(1)
		}
		optLevel, fileArg := parseOptLevel(args[1:])
		if fileArg == "" {
			fmt.Fprintln(os.Stderr, "Usage: calcium run [-O0|-O1|-O2] <file.ca|file.bone>")
			os.Exit(1)
		}
		runFileWithOpt(fileArg, optLevel)

	case "compile":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: calcium compile [-O0|-O1|-O2] <file.ca> [-o output.bone]")
			os.Exit(1)
		}
		optLevel, remaining := parseOptLevelArgs(args[1:])
		if len(remaining) == 0 {
			fmt.Fprintln(os.Stderr, "Usage: calcium compile [-O0|-O1|-O2] <file.ca> [-o output.bone]")
			os.Exit(1)
		}
		inputFile := remaining[0]
		outputFile := ""
		for i := 1; i < len(remaining); i++ {
			if remaining[i] == "-o" && i+1 < len(remaining) {
				outputFile = remaining[i+1]
				break
			}
		}
		compileFileWithOpt(inputFile, outputFile, optLevel)

	case "build":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: calcium build [-O0|-O1|-O2] <file.ca|file.bone> [-o output]")
			os.Exit(1)
		}
		optLevel, remaining := parseOptLevelArgs(args[1:])
		if len(remaining) == 0 {
			fmt.Fprintln(os.Stderr, "Usage: calcium build [-O0|-O1|-O2] <file.ca|file.bone> [-o output]")
			os.Exit(1)
		}
		inputFile := remaining[0]
		outputFile := ""
		for i := 1; i < len(remaining); i++ {
			if remaining[i] == "-o" && i+1 < len(remaining) {
				outputFile = remaining[i+1]
				break
			}
		}
		buildExecutableWithOpt(inputFile, outputFile, optLevel)

	case "repl":
		runREPL()

	case "test":
		dir := "."
		if len(args) >= 2 {
			dir = args[1]
		}
		runTests(dir)

	case "version", "--version", "-v":
		fmt.Printf("Calcium %s\n", version)

	case "help", "--help", "-h":
		printHelp()

	default:
		// Check for -O flag followed by a file
		if strings.HasPrefix(args[0], "-O") {
			if len(args) < 2 {
				fmt.Fprintln(os.Stderr, "Usage: calcium [-O0|-O1|-O2] <file.ca|file.bone>")
				os.Exit(1)
			}
			optLevel, fileArg := parseOptLevel(args)
			if fileArg == "" {
				fmt.Fprintln(os.Stderr, "Usage: calcium [-O0|-O1|-O2] <file.ca|file.bone>")
				os.Exit(1)
			}
			runFileWithOpt(fileArg, optLevel)
		} else if strings.HasSuffix(args[0], ".ca") || strings.HasSuffix(args[0], ".bone") {
			// If it looks like a file, try to run it
			runFileWithOpt(args[0], optimizer.O1)
		} else {
			fmt.Fprintf(os.Stderr, "Unknown command: %s\n", args[0])
			printHelp()
			os.Exit(1)
		}
	}
}

func printHelp() {
	fmt.Println(`Calcium - A functional programming language

Usage:
  calcium <file.ca>                          Run a Calcium source file
  calcium <file.bone>                        Run a compiled bytecode file
  calcium run [-O0|-O1|-O2] <file>           Run a program
  calcium compile [-O0|-O1|-O2] <file> [-o out]  Compile to bytecode (.bone)
  calcium build [-O0|-O1|-O2] <file> [-o out]    Build standalone executable
  calcium test [dir]                         Run all .test.ca files in directory
  calcium repl                               Start interactive REPL
  calcium version                            Show version
  calcium help                               Show this help

Optimization Levels:
  -O0    No optimization
  -O1    Basic optimization (constant folding, dead code elimination) [default]
  -O2    Full optimization (O1 + bytecode peephole optimization)

Examples:
  calcium hello.ca                    Run source directly (with -O1)
  calcium -O2 hello.ca                Run with full optimization
  calcium compile -O2 hello.ca        Compile with full optimization
  calcium build hello.ca -o hello     Build standalone executable
  calcium test ./tests                Run all tests in ./tests
  calcium ./hello                     Run standalone (no calcium needed)
  calcium repl                        Start REPL`)
}

// parseOptLevel parses optimization level from args, returns level and remaining file arg
func parseOptLevel(args []string) (optimizer.Level, string) {
	if len(args) == 0 {
		return optimizer.O1, ""
	}

	switch args[0] {
	case "-O0":
		if len(args) > 1 {
			return optimizer.O0, args[1]
		}
		return optimizer.O0, ""
	case "-O1":
		if len(args) > 1 {
			return optimizer.O1, args[1]
		}
		return optimizer.O1, ""
	case "-O2":
		if len(args) > 1 {
			return optimizer.O2, args[1]
		}
		return optimizer.O2, ""
	default:
		return optimizer.O1, args[0]
	}
}

// parseOptLevelArgs parses optimization level and returns remaining args
func parseOptLevelArgs(args []string) (optimizer.Level, []string) {
	if len(args) == 0 {
		return optimizer.O1, nil
	}

	switch args[0] {
	case "-O0":
		return optimizer.O0, args[1:]
	case "-O1":
		return optimizer.O1, args[1:]
	case "-O2":
		return optimizer.O2, args[1:]
	default:
		return optimizer.O1, args
	}
}

func runFile(filename string) {
	runFileWithOpt(filename, optimizer.O1)
}

func runFileWithOpt(filename string, optLevel optimizer.Level) {
	content, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	// Check if it's a compiled bytecode file
	if strings.HasSuffix(filename, ".bone") {
		err = executeBytecode(content)
	} else {
		_, err = executeWithFilenameOpt(string(content), filename, optLevel)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	// File execution: don't print the final result (use io.say for output)
}

func compileFile(inputFile, outputFile string) {
	compileFileWithOpt(inputFile, outputFile, optimizer.O1)
}

func compileFileWithOpt(inputFile, outputFile string, optLevel optimizer.Level) {
	useColor := isTerminal()

	// Read source file
	content, err := os.ReadFile(inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	input := string(content)

	// Parse
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		fmt.Fprintln(os.Stderr, formatParseErrors(inputFile, p.Errors(), useColor))
		os.Exit(1)
	}

	// Optimize AST
	opt := optimizer.New(optLevel)
	program = opt.OptimizeAST(program)

	// Compile
	comp := compiler.New()
	comp.SetInput(input)
	err = comp.Compile(program)
	if err != nil {
		fmt.Fprintln(os.Stderr, formatCompileErrors(inputFile, err, useColor))
		os.Exit(1)
	}

	// Get bytecode
	instructions := comp.Bytecode().Instructions

	// Optimize bytecode (for O2)
	instructions = opt.OptimizeBytecode(instructions)

	// Create CompiledBytecode
	cb := &bytecode.CompiledBytecode{
		Instructions: instructions,
		Constants:    comp.Constants(),
	}

	// Serialize
	data, err := cb.Serialize()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Serialization error: %v\n", err)
		os.Exit(1)
	}

	// Determine output filename
	if outputFile == "" {
		ext := filepath.Ext(inputFile)
		outputFile = strings.TrimSuffix(inputFile, ext) + ".bone"
	}

	// Write output file
	err = os.WriteFile(outputFile, data, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
		os.Exit(1)
	}

	optLevelStr := ""
	switch optLevel {
	case optimizer.O0:
		optLevelStr = " (O0)"
	case optimizer.O1:
		optLevelStr = " (O1)"
	case optimizer.O2:
		optLevelStr = " (O2)"
	}
	fmt.Printf("Compiled %s -> %s (%d bytes)%s\n", inputFile, outputFile, len(data), optLevelStr)
}

func executeBytecode(data []byte) error {
	// Deserialize
	cb, err := bytecode.Deserialize(data)
	if err != nil {
		return fmt.Errorf("failed to load bytecode: %w", err)
	}

	// Run
	machine := vm.New(cb.Constants)
	err = machine.Run(cb.Instructions)
	if err != nil {
		return fmt.Errorf("runtime error: %w", err)
	}

	return nil
}

func runREPL() {
	fmt.Printf("Calcium %s - Interactive REPL\n", version)
	fmt.Println("Type 'exit' or Ctrl+D to quit, 'help' for help")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)

	// Keep state across REPL sessions
	comp := compiler.New()
	machine := vm.NewForREPL(nil, make([]value.Value, 65536)) // Don't capture stdin

	for {
		fmt.Print("ca> ")
		if !scanner.Scan() {
			fmt.Println()
			break
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		switch line {
		case "exit", "quit":
			fmt.Println("Goodbye!")
			return
		case "help":
			printREPLHelp()
			continue
		case "clear":
			// Reset state
			comp = compiler.New()
			machine = vm.New(nil)
			fmt.Println("State cleared.")
			continue
		}

		result, err := executeInREPL(line, comp, machine)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		if result != "" && result != "null" {
			fmt.Println(result)
		}
	}
}

func printREPLHelp() {
	fmt.Println(`REPL Commands:
  exit, quit  Exit the REPL
  clear       Clear all defined variables and functions
  help        Show this help

Examples:
  ca> x = 10;
  ca> x * 2;
  20
  ca> func double(n) = n * 2;
  ca> double(21);
  42`)
}

func execute(input string) (string, error) {
	return executeWithFilename(input, "<stdin>")
}

func executeWithFilename(input, filename string) (string, error) {
	return executeWithFilenameOpt(input, filename, optimizer.O1)
}

func executeWithFilenameOpt(input, filename string, optLevel optimizer.Level) (string, error) {
	useColor := isTerminal()

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		return "", fmt.Errorf("%s", formatParseErrors(filename, p.Errors(), useColor))
	}

	// Optimize AST
	opt := optimizer.New(optLevel)
	program = opt.OptimizeAST(program)

	comp := compiler.New()
	comp.SetInput(input)
	err := comp.Compile(program)
	if err != nil {
		return "", fmt.Errorf("%s", formatCompileErrors(filename, err, useColor))
	}

	// Get bytecode and optimize (for O2)
	instructions := comp.Bytecode().Instructions
	instructions = opt.OptimizeBytecode(instructions)

	machine := vm.New(comp.Constants())
	// Set source path for external module resolution
	if absPath, err := filepath.Abs(filename); err == nil {
		machine.SetSourcePath(absPath)
	}
	err = machine.Run(instructions)
	if err != nil {
		return "", fmt.Errorf("%s", formatError(filename, fmt.Sprintf("runtime error: %v", err), useColor))
	}

	result := machine.LastPoppedStackElem()
	return result.String(), nil
}

func executeInREPL(input string, comp *compiler.Compiler, machine *vm.VM) (string, error) {
	useColor := isTerminal()

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		return "", fmt.Errorf("%s", formatParseErrors("<repl>", p.Errors(), useColor))
	}

	// Reset instructions before compiling new input (keep constants and symbols)
	comp.ResetInstructions()
	comp.SetInput(input)

	err := comp.Compile(program)
	if err != nil {
		return "", fmt.Errorf("%s", formatCompileErrors("<repl>", err, useColor))
	}

	instructions := comp.Bytecode().Instructions
	constants := comp.Constants()

	// Update VM with new constants (use REPL mode to avoid stdin capture)
	newMachine := vm.NewForREPL(constants, machine.Globals())
	err = newMachine.Run(instructions)
	if err != nil {
		return "", fmt.Errorf("runtime error: %v", err)
	}

	// Preserve globals for next iteration
	copy(machine.Globals(), newMachine.Globals())

	result := newMachine.LastPoppedStackElem()
	return result.String(), nil
}

// tryRunEmbedded checks if this binary has embedded .bone data and runs it
func tryRunEmbedded() bool {
	// Get path to own executable
	exePath, err := os.Executable()
	if err != nil {
		return false
	}

	f, err := os.Open(exePath)
	if err != nil {
		return false
	}
	defer f.Close()

	// Get file size
	stat, err := f.Stat()
	if err != nil {
		return false
	}
	fileSize := stat.Size()

	// Need at least 16 bytes for magic + size
	if fileSize < 16 {
		return false
	}

	// Read last 8 bytes (magic marker)
	magic := make([]byte, 8)
	_, err = f.ReadAt(magic, fileSize-8)
	if err != nil {
		return false
	}

	if string(magic) != packMagic {
		return false
	}

	// Read size (8 bytes before magic)
	sizeBytes := make([]byte, 8)
	_, err = f.ReadAt(sizeBytes, fileSize-16)
	if err != nil {
		return false
	}
	boneSize := int64(binary.LittleEndian.Uint64(sizeBytes))

	// Calculate offset to .bone data
	boneOffset := fileSize - 16 - boneSize

	// Read .bone data
	boneData := make([]byte, boneSize)
	_, err = f.ReadAt(boneData, boneOffset)
	if err != nil {
		return false
	}

	// Execute the embedded bytecode
	err = executeBytecode(boneData)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	return true
}

// buildExecutable creates a standalone executable by combining calcium binary with .bone data
func buildExecutable(inputFile, outputFile string) {
	buildExecutableWithOpt(inputFile, outputFile, optimizer.O1)
}

func buildExecutableWithOpt(inputFile, outputFile string, optLevel optimizer.Level) {
	var boneData []byte
	var err error

	// Get .bone data (compile if necessary)
	if strings.HasSuffix(inputFile, ".bone") {
		boneData, err = os.ReadFile(inputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Compile .ca to .bone
		useColor := isTerminal()
		content, err := os.ReadFile(inputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
			os.Exit(1)
		}

		input := string(content)
		l := lexer.New(input)
		p := parser.New(l)
		program := p.ParseProgram()

		if len(p.Errors()) > 0 {
			fmt.Fprintln(os.Stderr, formatParseErrors(inputFile, p.Errors(), useColor))
			os.Exit(1)
		}

		// Optimize AST
		opt := optimizer.New(optLevel)
		program = opt.OptimizeAST(program)

		comp := compiler.New()
		comp.SetInput(input)
		err = comp.Compile(program)
		if err != nil {
			fmt.Fprintln(os.Stderr, formatCompileErrors(inputFile, err, useColor))
			os.Exit(1)
		}

		// Get bytecode and optimize (for O2)
		instructions := comp.Bytecode().Instructions
		instructions = opt.OptimizeBytecode(instructions)

		cb := &bytecode.CompiledBytecode{
			Instructions: instructions,
			Constants:    comp.Constants(),
		}

		boneData, err = cb.Serialize()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Serialization error: %v\n", err)
			os.Exit(1)
		}
	}

	// Determine output filename
	if outputFile == "" {
		ext := filepath.Ext(inputFile)
		outputFile = strings.TrimSuffix(inputFile, ext)
		if runtime.GOOS == "windows" {
			outputFile += ".exe"
		}
	}

	// Get path to own executable (calcium binary)
	exePath, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error finding calcium binary: %v\n", err)
		os.Exit(1)
	}

	// Read calcium binary
	calciumBinary, err := os.ReadFile(exePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading calcium binary: %v\n", err)
		os.Exit(1)
	}

	// Check if calcium binary already has embedded data (strip it)
	calciumBinary = stripEmbeddedData(calciumBinary)

	// Create output file
	outFile, err := os.Create(outputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
		os.Exit(1)
	}
	defer outFile.Close()

	// Write: calcium binary + .bone data + size (8 bytes) + magic (8 bytes)
	_, err = outFile.Write(calciumBinary)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing binary: %v\n", err)
		os.Exit(1)
	}

	_, err = outFile.Write(boneData)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing bone data: %v\n", err)
		os.Exit(1)
	}

	// Write size as 8 bytes (little-endian)
	sizeBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(sizeBytes, uint64(len(boneData)))
	_, err = outFile.Write(sizeBytes)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing size: %v\n", err)
		os.Exit(1)
	}

	// Write magic marker
	_, err = outFile.WriteString(packMagic)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing magic: %v\n", err)
		os.Exit(1)
	}

	// Make executable
	err = outFile.Chmod(0755)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not set executable permission: %v\n", err)
	}

	totalSize := len(calciumBinary) + len(boneData) + 16
	fmt.Printf("Built %s -> %s (%d bytes, runtime: %d, program: %d)\n",
		inputFile, outputFile, totalSize, len(calciumBinary), len(boneData))
}

// stripEmbeddedData removes any existing embedded .bone data from a calcium binary
func stripEmbeddedData(data []byte) []byte {
	if len(data) < 16 {
		return data
	}

	// Check for magic marker at end
	magic := string(data[len(data)-8:])
	if magic != packMagic {
		return data
	}

	// Read size
	sizeBytes := data[len(data)-16 : len(data)-8]
	boneSize := binary.LittleEndian.Uint64(sizeBytes)

	// Strip embedded data
	originalSize := len(data) - 16 - int(boneSize)
	if originalSize > 0 && originalSize < len(data) {
		return data[:originalSize]
	}

	return data
}

// Ensure io package is used
var _ = io.EOF

// runTests runs all .test.ca files in the specified directory
func runTests(dir string) {
	// Find all .test.ca files
	var testFiles []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".test.ca") {
			testFiles = append(testFiles, path)
		}
		return nil
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error scanning directory: %v\n", err)
		os.Exit(1)
	}

	if len(testFiles) == 0 {
		fmt.Printf("No .test.ca files found in %s\n", dir)
		os.Exit(0)
	}

	// Sort for consistent ordering
	sort.Strings(testFiles)

	fmt.Println("╔══════════════════════════════════════╗")
	fmt.Println("║     Calcium Language Test Suite      ║")
	fmt.Println("╚══════════════════════════════════════╝")
	fmt.Println()

	passed := 0
	failed := 0
	var failedTests []string

	for _, testFile := range testFiles {
		relPath, _ := filepath.Rel(dir, testFile)
		if relPath == "" {
			relPath = testFile
		}

		fmt.Printf("━━━ %s ━━━\n", relPath)

		content, err := os.ReadFile(testFile)
		if err != nil {
			fmt.Printf("  Error reading file: %v\n", err)
			failed++
			failedTests = append(failedTests, relPath)
			continue
		}

		// Run the test file
		absPath, _ := filepath.Abs(testFile)
		_, err = executeTestFile(string(content), absPath)
		if err != nil {
			fmt.Printf("  Error: %v\n", err)
			failed++
			failedTests = append(failedTests, relPath)
		} else {
			passed++
		}
		fmt.Println()
	}

	// Summary
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("Results: %d passed, %d failed (out of %d test files)\n", passed, failed, len(testFiles))

	if failed > 0 {
		fmt.Println("\nFailed tests:")
		for _, t := range failedTests {
			fmt.Printf("  - %s\n", t)
		}
		os.Exit(1)
	}
}

func executeTestFile(input, filename string) (string, error) {
	useColor := isTerminal()

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		return "", fmt.Errorf("%s", formatParseErrors(filename, p.Errors(), useColor))
	}

	// Use O1 optimization for tests
	opt := optimizer.New(optimizer.O1)
	program = opt.OptimizeAST(program)

	comp := compiler.New()
	comp.SetInput(input)
	err := comp.Compile(program)
	if err != nil {
		return "", fmt.Errorf("%s", formatCompileErrors(filename, err, useColor))
	}

	instructions := comp.Bytecode().Instructions
	instructions = opt.OptimizeBytecode(instructions)

	machine := vm.New(comp.Constants())
	machine.SetSourcePath(filename)
	err = machine.Run(instructions)
	if err != nil {
		return "", fmt.Errorf("runtime error: %v", err)
	}

	result := machine.LastPoppedStackElem()
	return result.String(), nil
}
