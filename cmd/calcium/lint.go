package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/ytnobody/calcium-lang/pkg/linter"
)

type lintFileResult struct {
	File     string           `json:"file"`
	Findings []linter.Finding `json:"findings"`
}

// runLint lints the given source files and reports findings. It returns a
// process exit code (0 = no findings, 1 = findings or read error).
func runLint(args []string) int {
	jsonOut := false
	var files []string

	for _, a := range args {
		switch a {
		case "--json":
			jsonOut = true
		default:
			files = append(files, a)
		}
	}

	if len(files) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: calcium lint [--json] <file.ca> [...]")
		return 1
	}

	results := make([]lintFileResult, 0, len(files))
	hadFindings := false
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "calcium lint: %v\n", err)
			hadFindings = true
			continue
		}
		res := linter.Lint(string(content))
		findings := res.Findings
		if findings == nil {
			findings = []linter.Finding{}
		}
		results = append(results, lintFileResult{File: file, Findings: findings})
		if len(res.Findings) > 0 {
			hadFindings = true
		}
	}

	if jsonOut {
		printLintJSON(os.Stdout, results)
	} else {
		printLintHuman(os.Stdout, results)
	}

	if hadFindings {
		return 1
	}
	return 0
}

func printLintJSON(w io.Writer, results []lintFileResult) {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(results); err != nil {
		fmt.Fprintf(os.Stderr, "calcium lint: failed to encode JSON output: %v\n", err)
	}
}

func printLintHuman(w io.Writer, results []lintFileResult) {
	for _, r := range results {
		if len(r.Findings) == 0 {
			fmt.Fprintf(w, "%s: OK\n", r.File)
			continue
		}
		for _, f := range r.Findings {
			pos := ""
			if f.Line > 0 {
				if f.Column > 0 {
					pos = fmt.Sprintf(":%d:%d", f.Line, f.Column)
				} else {
					pos = fmt.Sprintf(":%d", f.Line)
				}
			}
			fmt.Fprintf(w, "%s%s [%s/%s] %s\n", r.File, pos, f.Rule, f.Severity, f.Message)
		}
	}
}
