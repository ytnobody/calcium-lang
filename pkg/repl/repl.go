// Package repl provides an interactive Read-Eval-Print Loop for the Calcium language.
// It supports command history persistence, startup script loading, and multi-line input.
package repl

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/peterh/liner"
	"github.com/ytnobody/calcium-lang/pkg/compiler"
	"github.com/ytnobody/calcium-lang/pkg/lexer"
	"github.com/ytnobody/calcium-lang/pkg/optimizer"
	"github.com/ytnobody/calcium-lang/pkg/parser"
	"github.com/ytnobody/calcium-lang/pkg/value"
	"github.com/ytnobody/calcium-lang/pkg/vm"
)

// ANSI color codes for terminal output
const (
	colorReset = "\033[0m"
	colorRed   = "\033[31m"
	colorCyan  = "\033[36m"
	colorBold  = "\033[1m"
)

const (
	// DefaultHistoryFile is the default location for REPL command history.
	DefaultHistoryFile = "~/.calcium_history"
	// DefaultStartupScript is the default location for the REPL startup script.
	DefaultStartupScript = "~/.calciumrc"
	// DefaultMaxHistory is the default maximum number of history entries.
	DefaultMaxHistory = 1000

	primaryPrompt      = "ca> "
	continuationPrompt = "..> "
)

// Config holds configuration for the REPL session.
type Config struct {
	// HistoryFile is the path to the file for persisting command history.
	// Use "~/" prefix for home directory. Set to "" to disable history persistence.
	HistoryFile string
	// StartupScript is the path to a Calcium script to execute on startup.
	// Use "~/" prefix for home directory. Set to "" to disable startup script.
	StartupScript string
	// MaxHistorySize is the maximum number of history entries to keep.
	MaxHistorySize int
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		HistoryFile:    DefaultHistoryFile,
		StartupScript:  DefaultStartupScript,
		MaxHistorySize: DefaultMaxHistory,
	}
}

// Session manages the state of an interactive REPL session.
type Session struct {
	config  Config
	liner   *liner.State
	comp    *compiler.Compiler
	machine *vm.VM
}

// NewSession creates a new REPL session with the given configuration.
func NewSession(cfg Config) *Session {
	comp := compiler.New()
	machine := vm.NewForREPL(nil, make([]value.Value, 65536))
	return &Session{
		config:  cfg,
		comp:    comp,
		machine: machine,
	}
}

// isTerminalOutput checks if stdout is a terminal (for color support).
func isTerminalOutput() bool {
	fileInfo, _ := os.Stdout.Stat()
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// formatREPLParseErrors formats parse errors for REPL output with optional color.
func formatREPLParseErrors(errors []string, useColor bool) string {
	var sb strings.Builder
	for i, e := range errors {
		if i > 0 {
			sb.WriteString("\n\n")
		}
		if useColor {
			sb.WriteString(fmt.Sprintf("%s<repl>:%s error%s\n%s",
				colorBold+colorCyan, colorRed, colorReset, e))
		} else {
			sb.WriteString(fmt.Sprintf("<repl>: error\n%s", e))
		}
	}
	if len(errors) > 1 {
		sb.WriteString("\n\n")
		if useColor {
			sb.WriteString(fmt.Sprintf("%serror: aborting due to %d previous errors%s",
				colorBold+colorRed, len(errors), colorReset))
		} else {
			sb.WriteString(fmt.Sprintf("error: aborting due to %d previous errors", len(errors)))
		}
	}
	return sb.String()
}

// inputAction represents what the REPL should do after processing an input line.
type inputAction int

const (
	actionContinue inputAction = iota // continue the REPL loop
	actionExit                        // exit the REPL
)

// inputState holds the mutable state for multi-line input processing.
type inputState struct {
	multiLineBuffer strings.Builder
	prompt          string
}

// newInputState creates a new inputState with the primary prompt.
func newInputState() *inputState {
	return &inputState{prompt: primaryPrompt}
}

// handlePromptError processes errors from the liner Prompt call.
// Returns the action and whether processing should continue (for non-fatal errors).
func (s *Session) handlePromptError(err error, state *inputState) inputAction {
	if err == liner.ErrPromptAborted {
		if state.multiLineBuffer.Len() > 0 {
			// Cancel multi-line input
			state.multiLineBuffer.Reset()
			state.prompt = primaryPrompt
			fmt.Println("(input cancelled)")
			return actionContinue
		}
		fmt.Println("Goodbye!")
		return actionExit
	}
	// EOF (Ctrl+D)
	if err == io.EOF {
		fmt.Println("\nGoodbye!")
		return actionExit
	}
	fmt.Fprintf(os.Stderr, "Input error: %v\n", err)
	return actionContinue
}

// handleInput processes a single line of REPL input.
// Returns the action the REPL loop should take.
func (s *Session) handleInput(input string, state *inputState) inputAction {
	trimmed := strings.TrimSpace(input)

	// Handle multi-line input continuation
	if state.multiLineBuffer.Len() > 0 {
		if trimmed == "" {
			// Empty line ends multi-line input
			fullInput := state.multiLineBuffer.String()
			state.multiLineBuffer.Reset()
			state.prompt = primaryPrompt

			s.liner.AppendHistory(fullInput)
			s.evalAndPrint(fullInput)
			return actionContinue
		}

		state.multiLineBuffer.WriteString("\n")
		state.multiLineBuffer.WriteString(input)

		if IsIncomplete(state.multiLineBuffer.String()) {
			return actionContinue
		}

		fullInput := state.multiLineBuffer.String()
		state.multiLineBuffer.Reset()
		state.prompt = primaryPrompt

		s.liner.AppendHistory(fullInput)
		s.evalAndPrint(fullInput)
		return actionContinue
	}

	if trimmed == "" {
		return actionContinue
	}

	// Handle special REPL commands
	switch trimmed {
	case "exit", "quit":
		fmt.Println("Goodbye!")
		return actionExit
	case "help":
		printREPLHelp()
		return actionContinue
	case "clear":
		s.comp = compiler.New()
		s.machine = vm.New(nil)
		fmt.Println("State cleared.")
		return actionContinue
	}

	// Check if input needs multi-line continuation
	if IsIncomplete(trimmed) {
		state.multiLineBuffer.WriteString(input)
		state.prompt = continuationPrompt
		return actionContinue
	}

	s.liner.AppendHistory(trimmed)
	s.evalAndPrint(trimmed)
	return actionContinue
}

// Run starts the interactive REPL loop.
func (s *Session) Run(version string) {
	fmt.Printf("Calcium %s - Interactive REPL\n", version)
	fmt.Println("Type 'exit' or Ctrl+D to quit, 'help' for help")
	fmt.Println()

	// Initialize liner for readline support
	s.liner = liner.NewLiner()
	defer s.cleanup()

	s.liner.SetCtrlCAborts(true)

	// Load history from file
	s.loadHistory()

	// Load startup script
	s.loadStartupScript()

	state := newInputState()

	// Main REPL loop
	for {
		input, err := s.liner.Prompt(state.prompt)
		if err != nil {
			if s.handlePromptError(err, state) == actionExit {
				return
			}
			continue
		}

		if s.handleInput(input, state) == actionExit {
			return
		}
	}
}

// evalAndPrint evaluates input and prints the result.
func (s *Session) evalAndPrint(input string) {
	result, err := s.Eval(input)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	if result != "" && result != "null" {
		fmt.Println(result)
	}
}

// cleanup saves history and closes the liner state.
func (s *Session) cleanup() {
	s.saveHistory()
	s.liner.Close()
}

// loadHistory loads REPL history from the configured history file.
func (s *Session) loadHistory() {
	if s.config.HistoryFile == "" {
		return
	}
	histPath := expandPath(s.config.HistoryFile)
	if f, err := os.Open(histPath); err == nil {
		s.liner.ReadHistory(f)
		f.Close()
	}
}

// saveHistory saves REPL history to the configured history file.
func (s *Session) saveHistory() {
	if s.config.HistoryFile == "" {
		return
	}
	histPath := expandPath(s.config.HistoryFile)
	// Ensure the directory exists
	if err := os.MkdirAll(filepath.Dir(histPath), 0755); err != nil {
		return
	}
	if f, err := os.Create(histPath); err == nil {
		s.liner.WriteHistory(f)
		f.Close()
	}
}

// loadStartupScript loads and executes the configured startup script.
// Missing startup scripts are silently ignored.
func (s *Session) loadStartupScript() {
	if s.config.StartupScript == "" {
		return
	}
	scriptPath := expandPath(s.config.StartupScript)
	if err := s.execStartupScript(scriptPath); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: startup script error (%s): %v\n", scriptPath, err)
	}
}

// execStartupScript loads and evaluates a Calcium script file.
// Returns nil if the file does not exist (startup scripts are optional).
func (s *Session) execStartupScript(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("reading startup script: %w", err)
	}

	input := strings.TrimSpace(string(content))
	if input == "" {
		return nil
	}

	fmt.Printf("Loading startup script: %s\n", path)
	_, err = s.Eval(input)
	if err != nil {
		return fmt.Errorf("executing startup script: %w", err)
	}
	return nil
}

// Eval evaluates a Calcium expression in the current REPL session context.
// State (variables, functions) is preserved between calls.
func (s *Session) Eval(input string) (string, error) {
	useColor := isTerminalOutput()

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		return "", fmt.Errorf("%s", formatREPLParseErrors(p.Errors(), useColor))
	}

	// Reset instructions, keep constants and symbols from previous evaluations
	s.comp.ResetInstructions()
	s.comp.SetInput(input)

	err := s.comp.Compile(program)
	if err != nil {
		if useColor {
			return "", fmt.Errorf("%s<repl>:%s error%s\n%v",
				colorBold+colorCyan, colorRed, colorReset, err)
		}
		return "", fmt.Errorf("<repl>: error\n%v", err)
	}

	// Apply O1 optimization
	opt := optimizer.New(optimizer.O1)
	instructions := opt.OptimizeBytecode(s.comp.Bytecode().Instructions)
	constants := s.comp.Constants()

	// Run in REPL mode (avoids stdin capture)
	newMachine := vm.NewForREPL(constants, s.machine.Globals())
	err = newMachine.Run(instructions)
	if err != nil {
		return "", fmt.Errorf("runtime error: %v", err)
	}

	// Preserve globals for the next iteration
	copy(s.machine.Globals(), newMachine.Globals())

	result := newMachine.LastPoppedStackElem()
	return result.String(), nil
}

// IsIncomplete reports whether the given input appears to be an incomplete expression.
// It detects unclosed brackets/parentheses/braces and trailing operators.
func IsIncomplete(input string) bool {
	depth := 0
	inString := false
	inLineComment := false
	prev := rune(0)

	for _, ch := range input {
		if inLineComment {
			if ch == '\n' {
				inLineComment = false
			}
			prev = ch
			continue
		}

		if ch == '/' && prev == '/' {
			inLineComment = true
			prev = ch
			continue
		}

		if inString {
			if ch == '"' && prev != '\\' {
				inString = false
			}
			prev = ch
			continue
		}

		switch ch {
		case '"':
			inString = true
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			depth--
		}
		prev = ch
	}

	if depth > 0 {
		return true
	}

	// Check for trailing operators that indicate continuation
	trimmed := strings.TrimSpace(input)
	if strings.HasSuffix(trimmed, "|>") ||
		strings.HasSuffix(trimmed, "|>?") ||
		strings.HasSuffix(trimmed, "!>") ||
		strings.HasSuffix(trimmed, "=>") ||
		strings.HasSuffix(trimmed, "=") ||
		strings.HasSuffix(trimmed, ",") {
		return true
	}

	return false
}

// expandPath expands a path with "~/" prefix to the user's home directory.
func expandPath(path string) string {
	if !strings.HasPrefix(path, "~/") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return filepath.Join(home, path[2:])
}

// printREPLHelp prints help information for REPL commands.
func printREPLHelp() {
	fmt.Println(`REPL Commands:
  exit, quit  Exit the REPL
  clear       Clear all defined variables and functions
  help        Show this help

Features:
  - Command history (Up/Down arrows) saved to ~/.calcium_history
  - Startup script loaded from ~/.calciumrc
  - Multi-line input: unclosed brackets/parens auto-continue (..> prompt)
  - Ctrl+C cancels multi-line input, Ctrl+D exits

Examples:
  ca> x = 10;
  ca> x * 2;
  20
  ca> func double(n) = n * 2;
  ca> double(21);
  42

Multi-line example:
  ca> func fib(n) =
  ..>   match n
  ..>     0 => 0
  ..>     1 => 1
  ..>     _ => fib(n-1) + fib(n-2);`)
}
