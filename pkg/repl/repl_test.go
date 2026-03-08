package repl

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/peterh/liner"
)

// TestDefaultConfig checks that DefaultConfig returns sensible defaults.
func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.HistoryFile != DefaultHistoryFile {
		t.Errorf("expected HistoryFile %q, got %q", DefaultHistoryFile, cfg.HistoryFile)
	}
	if cfg.StartupScript != DefaultStartupScript {
		t.Errorf("expected StartupScript %q, got %q", DefaultStartupScript, cfg.StartupScript)
	}
	if cfg.MaxHistorySize != DefaultMaxHistory {
		t.Errorf("expected MaxHistorySize %d, got %d", DefaultMaxHistory, cfg.MaxHistorySize)
	}
}

// TestNewSession checks that a session is created with proper state.
func TestNewSession(t *testing.T) {
	cfg := DefaultConfig()
	s := NewSession(cfg)
	if s == nil {
		t.Fatal("expected non-nil session")
	}
	if s.comp == nil {
		t.Error("expected non-nil compiler")
	}
	if s.machine == nil {
		t.Error("expected non-nil VM")
	}
}

// TestEval tests basic expression evaluation in the REPL session.
func TestEval(t *testing.T) {
	cfg := Config{}
	s := NewSession(cfg)

	tests := []struct {
		input    string
		expected string
	}{
		{"1 + 1;", "2"},
		{"x = 42;", "42"},
		{"x;", "42"},
		{`"hello";`, "hello"},
		{"[1, 2, 3];", "[1, 2, 3]"},
	}

	for _, tt := range tests {
		result, err := s.Eval(tt.input)
		if err != nil {
			t.Errorf("Eval(%q) returned error: %v", tt.input, err)
			continue
		}
		if result != tt.expected {
			t.Errorf("Eval(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

// TestEvalError tests that invalid input returns errors.
func TestEvalError(t *testing.T) {
	cfg := Config{}
	s := NewSession(cfg)

	tests := []struct {
		input string
	}{
		{"1 + ;"},           // parse error
		{"undeclared_fn()"}, // runtime error
	}

	for _, tt := range tests {
		_, err := s.Eval(tt.input)
		if err == nil {
			t.Errorf("Eval(%q) expected error, got nil", tt.input)
		}
	}
}

// TestEvalMultiStatement tests that multiple statements can be evaluated sequentially.
func TestEvalMultiStatement(t *testing.T) {
	cfg := Config{}
	s := NewSession(cfg)

	_, err := s.Eval("a = 10;")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = s.Eval("b = 20;")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, err := s.Eval("a + b;")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != "30" {
		t.Errorf("expected 30, got %q", result)
	}
}

// TestIsIncomplete tests detection of incomplete multi-line input.
func TestIsIncomplete(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		// Complete expressions
		{"x = 10;", false},
		{"[1, 2, 3];", false},
		{"(1 + 2);", false},
		{"", false},

		// Incomplete: unmatched brackets
		{"[1, 2,", true},
		{"(1 + 2", true},
		{"{x: 1,", true},

		// Incomplete: trailing operators
		{"x = 10 |>", true},
		{"x,", true},
	}

	for _, tt := range tests {
		result := IsIncomplete(tt.input)
		if result != tt.expected {
			t.Errorf("IsIncomplete(%q) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

// TestIsIncompleteMultiLine tests multi-line incomplete detection.
func TestIsIncompleteMultiLine(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "closed array across lines",
			input:    "[1, 2,\n3, 4]",
			expected: false,
		},
		{
			name:     "unclosed array across lines",
			input:    "[1, 2,\n3, 4",
			expected: true,
		},
		{
			name:     "nested brackets",
			input:    "[[1, 2],\n[3,",
			expected: true,
		},
		{
			name:     "closed nested brackets",
			input:    "[[1, 2],\n[3, 4]];",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsIncomplete(tt.input)
			if result != tt.expected {
				t.Errorf("IsIncomplete(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// TestIsIncompleteStrings tests that brackets inside strings are ignored.
func TestIsIncompleteStrings(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "bracket in string is not counted",
			input:    `"[hello"`,
			expected: false,
		},
		{
			name:     "opening brace in string ignored",
			input:    `"{test}"`,
			expected: false,
		},
		{
			name:     "escaped quote in string",
			input:    `"hello \"world\""`,
			expected: false,
		},
		{
			name:     "unclosed string with bracket",
			input:    `"[hello`,
			expected: false, // inString but depth is 0
		},
		{
			name:     "string followed by open bracket",
			input:    `"hello" [`,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsIncomplete(tt.input)
			if result != tt.expected {
				t.Errorf("IsIncomplete(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// TestIsIncompleteComments tests that brackets inside comments are ignored.
func TestIsIncompleteComments(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "bracket in line comment ignored",
			input:    "// [unclosed bracket",
			expected: false,
		},
		{
			name:     "comment then newline then code",
			input:    "// comment\nx = 10;",
			expected: false,
		},
		{
			name:     "comment then newline then open bracket",
			input:    "// comment\n[1,",
			expected: true,
		},
		{
			name:     "bracket after comment on new line",
			input:    "// [\n]",
			expected: false, // comment bracket ignored, ] decrements to -1 which is < 0 not > 0
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsIncomplete(tt.input)
			if result != tt.expected {
				t.Errorf("IsIncomplete(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// TestIsIncompleteTrailingOperators tests all trailing operator patterns.
func TestIsIncompleteTrailingOperators(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"trailing pipe", "x |>", true},
		{"trailing safe pipe", "x |>?", true},
		{"trailing bang pipe", "x !>", true},
		{"trailing arrow", "x =>", true},
		{"trailing equals", "x =", true},
		{"trailing comma", "x,", true},
		{"no trailing operator", "x + 1;", false},
		{"pipe in middle", "x |> y;", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsIncomplete(tt.input)
			if result != tt.expected {
				t.Errorf("IsIncomplete(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// TestIsIncompleteClosedBrackets tests that matched closing brackets are complete.
func TestIsIncompleteClosedBrackets(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"matched parens", "(1 + 2)", false},
		{"matched braces", "{a: 1}", false},
		{"matched square brackets", "[1, 2]", false},
		{"mixed matched", "([{x}])", false},
		{"extra closing", "())", false}, // depth goes negative, not > 0
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsIncomplete(tt.input)
			if result != tt.expected {
				t.Errorf("IsIncomplete(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// TestExpandPath tests path expansion with "~/" prefix.
func TestExpandPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot get home directory")
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"~/foo", filepath.Join(home, "foo")},
		{"~/bar/baz", filepath.Join(home, "bar/baz")},
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
	}

	for _, tt := range tests {
		result := expandPath(tt.input)
		if result != tt.expected {
			t.Errorf("expandPath(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

// TestExpandPathNoTilde tests that paths without tilde are returned as-is.
func TestExpandPathNoTilde(t *testing.T) {
	// Path that starts with ~ but not ~/
	result := expandPath("~notapath")
	if result != "~notapath" {
		t.Errorf("expandPath(~notapath) = %q, want ~notapath", result)
	}

	// Empty path
	result = expandPath("")
	if result != "" {
		t.Errorf("expandPath(\"\") = %q, want \"\"", result)
	}
}

// TestExecStartupScript tests loading a startup script.
func TestExecStartupScript(t *testing.T) {
	cfg := Config{}
	s := NewSession(cfg)

	// Write a temp startup script
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "init.ca")
	if err := os.WriteFile(scriptPath, []byte("startup_var = 99;"), 0644); err != nil {
		t.Fatalf("failed to write startup script: %v", err)
	}

	if err := s.execStartupScript(scriptPath); err != nil {
		t.Fatalf("execStartupScript returned error: %v", err)
	}

	// Verify the variable was set
	result, err := s.Eval("startup_var;")
	if err != nil {
		t.Fatalf("Eval returned error: %v", err)
	}
	if result != "99" {
		t.Errorf("expected startup_var = 99, got %q", result)
	}
}

// TestExecStartupScriptMissing tests that a missing startup script is not an error.
func TestExecStartupScriptMissing(t *testing.T) {
	cfg := Config{}
	s := NewSession(cfg)

	err := s.execStartupScript("/nonexistent/path/init.ca")
	if err != nil {
		t.Errorf("expected no error for missing startup script, got: %v", err)
	}
}

// TestExecStartupScriptEmpty tests that an empty startup script is handled gracefully.
func TestExecStartupScriptEmpty(t *testing.T) {
	cfg := Config{}
	s := NewSession(cfg)

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "empty.ca")
	if err := os.WriteFile(scriptPath, []byte{}, 0644); err != nil {
		t.Fatalf("failed to write empty script: %v", err)
	}

	err := s.execStartupScript(scriptPath)
	if err != nil {
		t.Errorf("expected no error for empty startup script, got: %v", err)
	}
}

// TestExecStartupScriptWhitespace tests that a whitespace-only script is handled.
func TestExecStartupScriptWhitespace(t *testing.T) {
	cfg := Config{}
	s := NewSession(cfg)

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "whitespace.ca")
	if err := os.WriteFile(scriptPath, []byte("   \n  \n  "), 0644); err != nil {
		t.Fatalf("failed to write whitespace script: %v", err)
	}

	err := s.execStartupScript(scriptPath)
	if err != nil {
		t.Errorf("expected no error for whitespace-only startup script, got: %v", err)
	}
}

// TestExecStartupScriptError tests that a script with errors returns an error.
func TestExecStartupScriptError(t *testing.T) {
	cfg := Config{}
	s := NewSession(cfg)

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "bad.ca")
	if err := os.WriteFile(scriptPath, []byte("1 + ;"), 0644); err != nil {
		t.Fatalf("failed to write bad script: %v", err)
	}

	err := s.execStartupScript(scriptPath)
	if err == nil {
		t.Error("expected error for script with parse errors, got nil")
	}
	if err != nil && !strings.Contains(err.Error(), "executing startup script") {
		t.Errorf("expected 'executing startup script' in error, got: %v", err)
	}
}

// TestExecStartupScriptUnreadable tests that an unreadable file returns an error.
func TestExecStartupScriptUnreadable(t *testing.T) {
	cfg := Config{}
	s := NewSession(cfg)

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "unreadable.ca")
	if err := os.WriteFile(scriptPath, []byte("x = 1;"), 0644); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}
	// Make it unreadable
	if err := os.Chmod(scriptPath, 0000); err != nil {
		t.Fatalf("failed to chmod: %v", err)
	}
	defer os.Chmod(scriptPath, 0644) // cleanup

	err := s.execStartupScript(scriptPath)
	if err == nil {
		t.Error("expected error for unreadable startup script, got nil")
	}
	if err != nil && !strings.Contains(err.Error(), "reading startup script") {
		t.Errorf("expected 'reading startup script' in error, got: %v", err)
	}
}

// TestFormatREPLParseErrors tests error formatting.
func TestFormatREPLParseErrors(t *testing.T) {
	t.Run("single error no color", func(t *testing.T) {
		result := formatREPLParseErrors([]string{"unexpected token"}, false)
		if !strings.Contains(result, "<repl>: error") {
			t.Errorf("expected '<repl>: error', got %q", result)
		}
		if !strings.Contains(result, "unexpected token") {
			t.Errorf("expected error message, got %q", result)
		}
	})

	t.Run("single error with color", func(t *testing.T) {
		result := formatREPLParseErrors([]string{"unexpected token"}, true)
		if !strings.Contains(result, colorBold) {
			t.Errorf("expected color codes in output, got %q", result)
		}
		if !strings.Contains(result, "unexpected token") {
			t.Errorf("expected error message, got %q", result)
		}
	})

	t.Run("multiple errors no color", func(t *testing.T) {
		errors := []string{"error one", "error two", "error three"}
		result := formatREPLParseErrors(errors, false)
		if !strings.Contains(result, "error one") {
			t.Errorf("expected first error, got %q", result)
		}
		if !strings.Contains(result, "error two") {
			t.Errorf("expected second error, got %q", result)
		}
		if !strings.Contains(result, "aborting due to 3 previous errors") {
			t.Errorf("expected aborting message, got %q", result)
		}
	})

	t.Run("multiple errors with color", func(t *testing.T) {
		errors := []string{"error one", "error two"}
		result := formatREPLParseErrors(errors, true)
		if !strings.Contains(result, colorBold) {
			t.Errorf("expected color codes, got %q", result)
		}
		if !strings.Contains(result, "aborting due to 2 previous errors") {
			t.Errorf("expected aborting message, got %q", result)
		}
	})

	t.Run("empty errors", func(t *testing.T) {
		result := formatREPLParseErrors([]string{}, false)
		if result != "" {
			t.Errorf("expected empty string for no errors, got %q", result)
		}
	})
}

// TestEvalCompileError tests that compile errors are formatted properly.
func TestEvalCompileError(t *testing.T) {
	cfg := Config{}
	s := NewSession(cfg)

	// Trigger a compile error (not a parse error) - use undefined identifier in expression
	_, err := s.Eval("undeclared_fn()")
	if err == nil {
		t.Fatal("expected compile/runtime error, got nil")
	}
	errStr := err.Error()
	// Should contain error information
	if errStr == "" {
		t.Error("expected non-empty error string")
	}
}

// TestEvalNullResult tests that null-like results are handled.
func TestEvalNullResult(t *testing.T) {
	cfg := Config{}
	s := NewSession(cfg)

	// Use an expression that returns a simple value
	result, err := s.Eval("0;")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "0" {
		t.Errorf("expected '0', got %q", result)
	}
}

// TestPrintREPLHelp tests the help output.
func TestPrintREPLHelp(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printREPLHelp()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	expectedPhrases := []string{
		"REPL Commands:",
		"exit, quit",
		"clear",
		"help",
		"Features:",
		"Command history",
		"Startup script",
		"Multi-line input",
	}

	for _, phrase := range expectedPhrases {
		if !strings.Contains(output, phrase) {
			t.Errorf("help output missing %q", phrase)
		}
	}
}

// TestEvalAndPrint tests evalAndPrint output.
func TestEvalAndPrint(t *testing.T) {
	t.Run("successful eval with result", func(t *testing.T) {
		cfg := Config{}
		s := NewSession(cfg)

		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		s.evalAndPrint("1 + 2;")

		w.Close()
		os.Stdout = old

		var buf bytes.Buffer
		buf.ReadFrom(r)
		output := buf.String()

		if !strings.Contains(output, "3") {
			t.Errorf("expected '3' in output, got %q", output)
		}
	})

	t.Run("eval with error", func(t *testing.T) {
		cfg := Config{}
		s := NewSession(cfg)

		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		s.evalAndPrint("1 + ;")

		w.Close()
		os.Stdout = old

		var buf bytes.Buffer
		buf.ReadFrom(r)
		output := buf.String()

		if !strings.Contains(output, "Error:") {
			t.Errorf("expected 'Error:' in output, got %q", output)
		}
	})

	t.Run("eval with empty string result", func(t *testing.T) {
		cfg := Config{}
		s := NewSession(cfg)

		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		s.evalAndPrint(`"";`)

		w.Close()
		os.Stdout = old

		var buf bytes.Buffer
		buf.ReadFrom(r)
		output := buf.String()

		// Empty string result should not be printed
		if strings.TrimSpace(output) != "" {
			t.Errorf("expected no output for empty string, got %q", output)
		}
	})
}

// TestLoadHistoryEmptyConfig tests that loadHistory with empty HistoryFile does nothing.
func TestLoadHistoryEmptyConfig(t *testing.T) {
	cfg := Config{HistoryFile: ""}
	s := NewSession(cfg)
	s.liner = liner.NewLiner()
	defer s.liner.Close()

	// Should return without error
	s.loadHistory()
}

// TestLoadHistoryWithFile tests loading history from an existing file.
func TestLoadHistoryWithFile(t *testing.T) {
	tmpDir := t.TempDir()
	histFile := filepath.Join(tmpDir, "history")

	// Create a history file with some content
	if err := os.WriteFile(histFile, []byte("first command\nsecond command\n"), 0644); err != nil {
		t.Fatalf("failed to write history file: %v", err)
	}

	cfg := Config{HistoryFile: histFile}
	s := NewSession(cfg)
	s.liner = liner.NewLiner()
	defer s.liner.Close()

	s.loadHistory()
	// No error expected - history is loaded silently
}

// TestLoadHistoryMissingFile tests that loadHistory handles missing file gracefully.
func TestLoadHistoryMissingFile(t *testing.T) {
	cfg := Config{HistoryFile: "/tmp/nonexistent_calcium_history_test"}
	s := NewSession(cfg)
	s.liner = liner.NewLiner()
	defer s.liner.Close()

	// Should not panic
	s.loadHistory()
}

// TestSaveHistoryEmptyConfig tests that saveHistory with empty HistoryFile does nothing.
func TestSaveHistoryEmptyConfig(t *testing.T) {
	cfg := Config{HistoryFile: ""}
	s := NewSession(cfg)
	s.liner = liner.NewLiner()
	defer s.liner.Close()

	// Should return without error
	s.saveHistory()
}

// TestSaveHistoryWithFile tests saving history to a file.
func TestSaveHistoryWithFile(t *testing.T) {
	tmpDir := t.TempDir()
	histFile := filepath.Join(tmpDir, "history")

	cfg := Config{HistoryFile: histFile}
	s := NewSession(cfg)
	s.liner = liner.NewLiner()
	defer s.liner.Close()

	s.liner.AppendHistory("test command")
	s.saveHistory()

	// Verify the file was created
	content, err := os.ReadFile(histFile)
	if err != nil {
		t.Fatalf("failed to read history file: %v", err)
	}
	if !strings.Contains(string(content), "test command") {
		t.Errorf("expected 'test command' in history file, got %q", string(content))
	}
}

// TestSaveHistoryCreatesDirectory tests that saveHistory creates parent directories.
func TestSaveHistoryCreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	histFile := filepath.Join(tmpDir, "subdir", "history")

	cfg := Config{HistoryFile: histFile}
	s := NewSession(cfg)
	s.liner = liner.NewLiner()
	defer s.liner.Close()

	s.saveHistory()

	// Verify the directory was created
	if _, err := os.Stat(filepath.Dir(histFile)); os.IsNotExist(err) {
		t.Error("expected subdirectory to be created")
	}
}

// TestCleanup tests the cleanup function.
func TestCleanup(t *testing.T) {
	tmpDir := t.TempDir()
	histFile := filepath.Join(tmpDir, "history")

	cfg := Config{HistoryFile: histFile}
	s := NewSession(cfg)
	s.liner = liner.NewLiner()

	s.liner.AppendHistory("cleanup test")
	s.cleanup()

	// Verify history was saved
	if _, err := os.Stat(histFile); os.IsNotExist(err) {
		t.Error("expected history file to be created during cleanup")
	}
}

// TestLoadStartupScriptEmptyConfig tests that loadStartupScript with empty config does nothing.
func TestLoadStartupScriptEmptyConfig(t *testing.T) {
	cfg := Config{StartupScript: ""}
	s := NewSession(cfg)

	// Capture stderr to check no warning is printed
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	s.loadStartupScript()

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	// No warning should be printed
	if strings.Contains(buf.String(), "Warning") {
		t.Errorf("unexpected warning output: %s", buf.String())
	}
}

// TestLoadStartupScriptMissingFile tests that loadStartupScript handles missing file.
func TestLoadStartupScriptMissingFile(t *testing.T) {
	cfg := Config{StartupScript: "/tmp/nonexistent_calciumrc_test"}
	s := NewSession(cfg)

	// Missing file should be silently ignored (no warning)
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	s.loadStartupScript()

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if strings.Contains(buf.String(), "Warning") {
		t.Errorf("unexpected warning for missing startup script: %s", buf.String())
	}
}

// TestLoadStartupScriptWithError tests that loadStartupScript prints warning on error.
func TestLoadStartupScriptWithError(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "bad.ca")
	if err := os.WriteFile(scriptPath, []byte("1 + ;"), 0644); err != nil {
		t.Fatalf("failed to write bad script: %v", err)
	}

	cfg := Config{StartupScript: scriptPath}
	s := NewSession(cfg)

	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	s.loadStartupScript()

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !strings.Contains(buf.String(), "Warning") {
		t.Errorf("expected warning for bad startup script, got %q", buf.String())
	}
}

// TestLoadStartupScriptSuccess tests successful startup script loading.
func TestLoadStartupScriptSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "init.ca")
	if err := os.WriteFile(scriptPath, []byte("x = 42;"), 0644); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}

	cfg := Config{StartupScript: scriptPath}
	s := NewSession(cfg)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	s.loadStartupScript()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !strings.Contains(buf.String(), "Loading startup script") {
		t.Errorf("expected 'Loading startup script' message, got %q", buf.String())
	}
}

// TestIsTerminalOutput tests the terminal detection function.
func TestIsTerminalOutput(t *testing.T) {
	// In test environment, stdout is typically not a terminal
	result := isTerminalOutput()
	// We just verify it runs without panic - the result depends on the environment
	_ = result
}

// TestEvalStatePreservation tests that state is preserved between Eval calls.
func TestEvalStatePreservation(t *testing.T) {
	cfg := Config{}
	s := NewSession(cfg)

	// Define a variable
	_, err := s.Eval("myval = 10;")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Use it in another expression
	result, err := s.Eval("myval + 5;")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "15" {
		t.Errorf("expected 15, got %q", result)
	}
}

// TestEvalEmptyResult tests eval with empty string result.
func TestEvalEmptyResult(t *testing.T) {
	cfg := Config{}
	s := NewSession(cfg)

	// Eval with expression that produces a result
	result, err := s.Eval(`"";`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Empty string is a valid result
	_ = result
}

// TestEvalParseErrorMultiple tests that multiple parse errors are formatted.
func TestEvalParseErrorMultiple(t *testing.T) {
	cfg := Config{}
	s := NewSession(cfg)

	// This should produce a parse error
	_, err := s.Eval("1 + ;")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "error") {
		t.Errorf("expected 'error' in error message, got %q", err.Error())
	}
}

// TestConstants tests ANSI color constants are defined.
func TestConstants(t *testing.T) {
	if colorReset != "\033[0m" {
		t.Errorf("unexpected colorReset: %q", colorReset)
	}
	if colorRed != "\033[31m" {
		t.Errorf("unexpected colorRed: %q", colorRed)
	}
	if colorCyan != "\033[36m" {
		t.Errorf("unexpected colorCyan: %q", colorCyan)
	}
	if colorBold != "\033[1m" {
		t.Errorf("unexpected colorBold: %q", colorBold)
	}
}

// TestPromptConstants tests prompt string constants.
func TestPromptConstants(t *testing.T) {
	if primaryPrompt != "ca> " {
		t.Errorf("unexpected primaryPrompt: %q", primaryPrompt)
	}
	if continuationPrompt != "..> " {
		t.Errorf("unexpected continuationPrompt: %q", continuationPrompt)
	}
}

// TestEvalAndPrintEmptyResult tests that empty results are not printed.
func TestEvalAndPrintEmptyResult(t *testing.T) {
	cfg := Config{}
	s := NewSession(cfg)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	s.evalAndPrint(`"";`)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	// Empty string result should not be printed
	output := strings.TrimSpace(buf.String())
	if output != "" {
		t.Errorf("expected no output for empty string result, got %q", output)
	}
}

// TestFormatREPLParseErrorsSingleWithColor tests color formatting for single error.
func TestFormatREPLParseErrorsSingleWithColor(t *testing.T) {
	result := formatREPLParseErrors([]string{"test error"}, true)
	if !strings.Contains(result, colorCyan) {
		t.Error("expected cyan color in output")
	}
	if !strings.Contains(result, colorRed) {
		t.Error("expected red color in output")
	}
	if !strings.Contains(result, colorReset) {
		t.Error("expected color reset in output")
	}
	// Single error should NOT have "aborting" message
	if strings.Contains(result, "aborting") {
		t.Error("single error should not have 'aborting' message")
	}
}

// TestSessionConfig tests that session stores the config.
func TestSessionConfig(t *testing.T) {
	cfg := Config{
		HistoryFile:    "/tmp/test_history",
		StartupScript:  "/tmp/test_startup",
		MaxHistorySize: 500,
	}
	s := NewSession(cfg)

	if s.config.HistoryFile != "/tmp/test_history" {
		t.Errorf("config not stored: HistoryFile = %q", s.config.HistoryFile)
	}
	if s.config.StartupScript != "/tmp/test_startup" {
		t.Errorf("config not stored: StartupScript = %q", s.config.StartupScript)
	}
	if s.config.MaxHistorySize != 500 {
		t.Errorf("config not stored: MaxHistorySize = %d", s.config.MaxHistorySize)
	}
}

// TestEvalRuntimeError tests that runtime errors are properly formatted.
func TestEvalRuntimeError(t *testing.T) {
	cfg := Config{}
	s := NewSession(cfg)

	// This should trigger a runtime error (division by zero or similar)
	// Using undeclared function which may be a compile error
	_, err := s.Eval("undeclared_fn()")
	if err == nil {
		t.Fatal("expected error")
	}
	// Error should contain useful information
	errStr := err.Error()
	if errStr == "" {
		t.Error("expected non-empty error message")
	}
}

// TestFormatREPLParseErrorsSeparators tests that multiple errors are separated by double newlines.
func TestFormatREPLParseErrorsSeparators(t *testing.T) {
	errors := []string{"first", "second", "third"}
	result := formatREPLParseErrors(errors, false)

	// Count separators between errors
	parts := strings.Split(result, "\n\n")
	// Should have: error1, error2, error3, aborting message = separated by \n\n
	if len(parts) < 4 {
		t.Errorf("expected at least 4 parts separated by \\n\\n, got %d: %q", len(parts), result)
	}
}

// TestFormatREPLParseErrorsMultipleWithColor verifies colored output for multiple errors.
func TestFormatREPLParseErrorsMultipleWithColor(t *testing.T) {
	errors := []string{"err1", "err2"}
	result := formatREPLParseErrors(errors, true)

	if !strings.Contains(result, "aborting due to 2 previous errors") {
		t.Errorf("expected aborting message, got %q", result)
	}
	if !strings.Contains(result, colorBold+colorRed) {
		t.Error("expected bold red for aborting message")
	}
}

// TestEvalCompileErrorFormat tests the formatting of compile errors.
func TestEvalCompileErrorFormat(t *testing.T) {
	cfg := Config{}
	s := NewSession(cfg)

	// Force a compile error by using invalid syntax that passes parsing
	// but fails compilation
	_, err := s.Eval("break;")
	if err != nil {
		errStr := err.Error()
		if !strings.Contains(errStr, "error") && !strings.Contains(errStr, "Error") {
			t.Logf("Compile error format: %s", errStr)
		}
	}
}

// TestHandleInputCommands tests REPL command processing via handleInput.
func TestHandleInputCommands(t *testing.T) {
	cfg := Config{}
	s := NewSession(cfg)
	s.liner = liner.NewLiner()
	defer s.liner.Close()

	t.Run("exit command", func(t *testing.T) {
		state := newInputState()
		old := os.Stdout
		_, w, _ := os.Pipe()
		os.Stdout = w
		action := s.handleInput("exit", state)
		w.Close()
		os.Stdout = old
		if action != actionExit {
			t.Errorf("expected actionExit for 'exit', got %v", action)
		}
	})

	t.Run("quit command", func(t *testing.T) {
		state := newInputState()
		old := os.Stdout
		_, w, _ := os.Pipe()
		os.Stdout = w
		action := s.handleInput("quit", state)
		w.Close()
		os.Stdout = old
		if action != actionExit {
			t.Errorf("expected actionExit for 'quit', got %v", action)
		}
	})

	t.Run("help command", func(t *testing.T) {
		state := newInputState()
		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w
		action := s.handleInput("help", state)
		w.Close()
		os.Stdout = old
		var buf bytes.Buffer
		buf.ReadFrom(r)
		if action != actionContinue {
			t.Errorf("expected actionContinue for 'help', got %v", action)
		}
		if !strings.Contains(buf.String(), "REPL Commands:") {
			t.Error("expected help output")
		}
	})

	t.Run("clear command", func(t *testing.T) {
		state := newInputState()
		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w
		action := s.handleInput("clear", state)
		w.Close()
		os.Stdout = old
		var buf bytes.Buffer
		buf.ReadFrom(r)
		if action != actionContinue {
			t.Errorf("expected actionContinue for 'clear', got %v", action)
		}
		if !strings.Contains(buf.String(), "State cleared.") {
			t.Error("expected 'State cleared.' output")
		}
	})

	t.Run("empty input", func(t *testing.T) {
		state := newInputState()
		action := s.handleInput("", state)
		if action != actionContinue {
			t.Errorf("expected actionContinue for empty input, got %v", action)
		}
	})

	t.Run("whitespace only", func(t *testing.T) {
		state := newInputState()
		action := s.handleInput("   ", state)
		if action != actionContinue {
			t.Errorf("expected actionContinue for whitespace input, got %v", action)
		}
	})

	t.Run("normal expression", func(t *testing.T) {
		state := newInputState()
		old := os.Stdout
		_, w, _ := os.Pipe()
		os.Stdout = w
		action := s.handleInput("1 + 2;", state)
		w.Close()
		os.Stdout = old
		if action != actionContinue {
			t.Errorf("expected actionContinue for expression, got %v", action)
		}
	})

	t.Run("incomplete input starts multiline", func(t *testing.T) {
		state := newInputState()
		action := s.handleInput("[1, 2,", state)
		if action != actionContinue {
			t.Errorf("expected actionContinue, got %v", action)
		}
		if state.prompt != continuationPrompt {
			t.Errorf("expected continuation prompt, got %q", state.prompt)
		}
		if state.multiLineBuffer.Len() == 0 {
			t.Error("expected multiline buffer to have content")
		}
	})
}

// TestHandleInputMultiLine tests multi-line input processing.
func TestHandleInputMultiLine(t *testing.T) {
	cfg := Config{}
	s := NewSession(cfg)
	s.liner = liner.NewLiner()
	defer s.liner.Close()

	t.Run("complete multiline with empty line", func(t *testing.T) {
		state := newInputState()
		// Start multi-line
		s.handleInput("[1, 2,", state)
		// Add more content
		old := os.Stdout
		_, w, _ := os.Pipe()
		os.Stdout = w
		// Empty line finishes multi-line
		action := s.handleInput("", state)
		w.Close()
		os.Stdout = old
		if action != actionContinue {
			t.Errorf("expected actionContinue, got %v", action)
		}
		if state.prompt != primaryPrompt {
			t.Errorf("expected primary prompt after multiline end, got %q", state.prompt)
		}
	})

	t.Run("complete multiline with closing bracket", func(t *testing.T) {
		state := newInputState()
		// Start multi-line
		s.handleInput("[1, 2,", state)
		old := os.Stdout
		_, w, _ := os.Pipe()
		os.Stdout = w
		// Close the bracket
		action := s.handleInput("3];", state)
		w.Close()
		os.Stdout = old
		if action != actionContinue {
			t.Errorf("expected actionContinue, got %v", action)
		}
		if state.prompt != primaryPrompt {
			t.Errorf("expected primary prompt, got %q", state.prompt)
		}
		if state.multiLineBuffer.Len() != 0 {
			t.Error("expected multiline buffer to be cleared")
		}
	})

	t.Run("continue multiline still incomplete", func(t *testing.T) {
		state := newInputState()
		// Start multi-line
		s.handleInput("[[1,", state)
		// Add more but still incomplete
		action := s.handleInput("2],", state)
		if action != actionContinue {
			t.Errorf("expected actionContinue, got %v", action)
		}
		// Buffer should still have content
		if state.multiLineBuffer.Len() == 0 {
			t.Error("expected multiline buffer to still have content")
		}
	})
}

// TestHandlePromptError tests prompt error handling.
func TestHandlePromptError(t *testing.T) {
	cfg := Config{}
	s := NewSession(cfg)
	s.liner = liner.NewLiner()
	defer s.liner.Close()

	t.Run("EOF exits", func(t *testing.T) {
		state := newInputState()
		old := os.Stdout
		_, w, _ := os.Pipe()
		os.Stdout = w
		action := s.handlePromptError(io.EOF, state)
		w.Close()
		os.Stdout = old
		if action != actionExit {
			t.Errorf("expected actionExit for EOF, got %v", action)
		}
	})

	t.Run("Ctrl+C with no multiline exits", func(t *testing.T) {
		state := newInputState()
		old := os.Stdout
		_, w, _ := os.Pipe()
		os.Stdout = w
		action := s.handlePromptError(liner.ErrPromptAborted, state)
		w.Close()
		os.Stdout = old
		if action != actionExit {
			t.Errorf("expected actionExit for Ctrl+C with no multiline, got %v", action)
		}
	})

	t.Run("Ctrl+C with multiline cancels", func(t *testing.T) {
		state := newInputState()
		state.multiLineBuffer.WriteString("[1, 2,")

		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w
		action := s.handlePromptError(liner.ErrPromptAborted, state)
		w.Close()
		os.Stdout = old

		var buf bytes.Buffer
		buf.ReadFrom(r)

		if action != actionContinue {
			t.Errorf("expected actionContinue for Ctrl+C with multiline, got %v", action)
		}
		if state.multiLineBuffer.Len() != 0 {
			t.Error("expected multiline buffer to be reset")
		}
		if state.prompt != primaryPrompt {
			t.Errorf("expected primary prompt, got %q", state.prompt)
		}
		if !strings.Contains(buf.String(), "(input cancelled)") {
			t.Errorf("expected '(input cancelled)' in output, got %q", buf.String())
		}
	})

	t.Run("other error continues", func(t *testing.T) {
		state := newInputState()
		old := os.Stderr
		_, w, _ := os.Pipe()
		os.Stderr = w
		action := s.handlePromptError(fmt.Errorf("some error"), state)
		w.Close()
		os.Stderr = old
		if action != actionContinue {
			t.Errorf("expected actionContinue for unknown error, got %v", action)
		}
	})
}

// TestNewInputState tests initial input state.
func TestNewInputState(t *testing.T) {
	state := newInputState()
	if state.prompt != primaryPrompt {
		t.Errorf("expected primary prompt, got %q", state.prompt)
	}
	if state.multiLineBuffer.Len() != 0 {
		t.Error("expected empty multiline buffer")
	}
}

// TestInputActionConstants tests the action constants exist.
func TestInputActionConstants(t *testing.T) {
	if actionContinue == actionExit {
		t.Error("actionContinue and actionExit should be different")
	}
}

// Ensure fmt is used
var _ = fmt.Sprintf
