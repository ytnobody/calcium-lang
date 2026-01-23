package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/example/calcium/pkg/bytecode"
	"github.com/example/calcium/pkg/compiler"
	"github.com/example/calcium/pkg/lexer"
	"github.com/example/calcium/pkg/parser"
	"github.com/example/calcium/pkg/vm"
)

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
			fmt.Fprintln(os.Stderr, "Usage: calcium run <file.ca|file.bone>")
			os.Exit(1)
		}
		runFile(args[1])

	case "compile":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: calcium compile <file.ca> [-o output.bone]")
			os.Exit(1)
		}
		outputFile := ""
		if len(args) >= 4 && args[2] == "-o" {
			outputFile = args[3]
		}
		compileFile(args[1], outputFile)

	case "build":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: calcium build <file.ca|file.bone> [-o output]")
			os.Exit(1)
		}
		outputFile := ""
		if len(args) >= 4 && args[2] == "-o" {
			outputFile = args[3]
		}
		buildExecutable(args[1], outputFile)

	case "repl":
		runREPL()

	case "version", "--version", "-v":
		fmt.Printf("Calcium %s\n", version)

	case "help", "--help", "-h":
		printHelp()

	default:
		// If it looks like a file, try to run it
		if strings.HasSuffix(args[0], ".ca") || strings.HasSuffix(args[0], ".bone") {
			runFile(args[0])
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
  calcium <file.ca>                   Run a Calcium source file
  calcium <file.bone>                 Run a compiled bytecode file
  calcium run <file.ca|file.bone>     Run a program
  calcium compile <file.ca> [-o out]  Compile to bytecode (.bone)
  calcium build <file> [-o out]       Build standalone executable
  calcium repl                        Start interactive REPL
  calcium version                     Show version
  calcium help                        Show this help

Examples:
  calcium hello.ca                    Run source directly
  calcium compile hello.ca            Compile to hello.bone
  calcium build hello.ca -o hello     Build standalone executable
  calcium ./hello                     Run standalone (no calcium needed)
  calcium repl                        Start REPL`)
}

func runFile(filename string) {
	content, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	// Check if it's a compiled bytecode file
	if strings.HasSuffix(filename, ".bone") {
		err = executeBytecode(content)
	} else {
		_, err = execute(string(content))
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	// File execution: don't print the final result (use io.say for output)
}

func compileFile(inputFile, outputFile string) {
	// Read source file
	content, err := os.ReadFile(inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	// Parse
	l := lexer.New(string(content))
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		fmt.Fprintf(os.Stderr, "Parse errors:\n")
		for _, e := range p.Errors() {
			fmt.Fprintf(os.Stderr, "  %s\n", e)
		}
		os.Exit(1)
	}

	// Compile
	comp := compiler.New()
	err = comp.Compile(program)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Compilation error: %v\n", err)
		os.Exit(1)
	}

	// Create CompiledBytecode
	cb := &bytecode.CompiledBytecode{
		Instructions: comp.Bytecode().Instructions,
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

	fmt.Printf("Compiled %s -> %s (%d bytes)\n", inputFile, outputFile, len(data))
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
	machine := vm.New(nil)

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
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		return "", fmt.Errorf("parse errors:\n  %s", strings.Join(p.Errors(), "\n  "))
	}

	comp := compiler.New()
	err := comp.Compile(program)
	if err != nil {
		return "", fmt.Errorf("compilation error: %v", err)
	}

	machine := vm.New(comp.Constants())
	err = machine.Run(comp.Bytecode().Instructions)
	if err != nil {
		return "", fmt.Errorf("runtime error: %v", err)
	}

	result := machine.LastPoppedStackElem()
	return result.String(), nil
}

func executeInREPL(input string, comp *compiler.Compiler, machine *vm.VM) (string, error) {
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		return "", fmt.Errorf("parse errors:\n  %s", strings.Join(p.Errors(), "\n  "))
	}

	err := comp.Compile(program)
	if err != nil {
		return "", fmt.Errorf("compilation error: %v", err)
	}

	// Update VM with new constants
	newMachine := vm.NewWithGlobals(comp.Constants(), machine.Globals())
	err = newMachine.Run(comp.Bytecode().Instructions)
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
		content, err := os.ReadFile(inputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
			os.Exit(1)
		}

		l := lexer.New(string(content))
		p := parser.New(l)
		program := p.ParseProgram()

		if len(p.Errors()) > 0 {
			fmt.Fprintf(os.Stderr, "Parse errors:\n")
			for _, e := range p.Errors() {
				fmt.Fprintf(os.Stderr, "  %s\n", e)
			}
			os.Exit(1)
		}

		comp := compiler.New()
		err = comp.Compile(program)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Compilation error: %v\n", err)
			os.Exit(1)
		}

		cb := &bytecode.CompiledBytecode{
			Instructions: comp.Bytecode().Instructions,
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
