package main

import (
	"fmt"
	"os"

	"github.com/ytnobody/calcium-lang/pkg/lexer"
	"github.com/ytnobody/calcium-lang/pkg/parser"
	"github.com/ytnobody/calcium-lang/pkg/typechecker"
)

// runCheck prints the gradual type checker's findings for the given files.
//
// Like `lex`, this exists to make one component observable on its own so a
// reimplementation can be compared against it. The format is one finding per
// line, `<line>:<column>\t<rule>`, with a `<file>: <n> finding(s)` trailer. The
// message text is deliberately absent: diagnostics are not part of the
// compatibility contract, positions and rules are.
func runCheck(args []string) int {
	flags, files := splitArgs(args)
	if !unknownFlags("calcium check", flags) || len(files) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: calcium check <file.ca> [...]")
		return 1
	}

	status := 0
	for _, file := range files {
		content, ok := readInput("calcium check", file)
		if !ok {
			status = 1
			continue
		}
		p := parser.New(lexer.New(string(content)))
		program := p.ParseProgram()
		if len(p.Errors()) > 0 {
			// A file that does not parse has no type information to report.
			fmt.Printf("%s: parse errors\n", file)
			status = 1
			continue
		}
		errs := typechecker.Check(program)
		for _, e := range errs {
			fmt.Printf("%d:%d\ttype\n", e.Line, e.Column)
		}
		fmt.Printf("%s: %d finding(s)\n", file, len(errs))
		if len(errs) > 0 {
			status = 1
		}
	}
	return status
}
