package main

import (
	"fmt"
	"os"

	"github.com/ytnobody/calcium-lang/pkg/formatter"
	"github.com/ytnobody/calcium-lang/pkg/lexer"
	"github.com/ytnobody/calcium-lang/pkg/token"
)

// runLex prints the token stream of the given source files.
//
// This is a debugging aid for the self-hosted implementation: it makes the
// lexer's output observable on its own, before a parser exists, so the token
// stream can be compared against a reimplementation.
//
// The format is one token per line, tab-separated:
//
//	<line>:<column>\t<kind>\t<literal>
//
// Only backslash, newline, tab and carriage return are escaped in the literal,
// so that a line always represents exactly one token and both implementations
// can emit the format with the same few rules. It is not JSON on purpose: JSON
// key order and escaping rules would make byte comparison unnecessarily
// fragile.
func runLex(args []string) int {
	flags, files := splitArgs(args)
	if !unknownFlags("calcium lex", flags) || len(files) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: calcium lex <file.ca> [...]")
		return 1
	}

	for _, file := range files {
		content, ok := readInput("calcium lex", file)
		if !ok {
			return 1
		}
		lex := lexer.New(string(content))
		count := 0
		for {
			tok := lex.NextToken()
			fmt.Printf("%d:%d\t%s\t%s\n", tok.Line, tok.Column, tok.Type, formatter.EscapeControl(tok.Literal))
			count++
			if tok.Type == token.EOF {
				break
			}
		}
		fmt.Printf("# tokens: %d\n", count)
	}
	return 0
}
