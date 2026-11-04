package main

import (
	"fmt"
	"os"
	"strings"
)

// splitArgs separates options from input files. Every subcommand that takes a list
// of files does this, so it is spelled out once here rather than in each of them.
func splitArgs(args []string) (flags, files []string) {
	for _, a := range args {
		if strings.HasPrefix(a, "-") {
			flags = append(flags, a)
			continue
		}
		files = append(files, a)
	}
	return flags, files
}

// readInput reads a source file, reporting the failure the way every subcommand does
// (`<command>: <error>` on stderr). The bool reports whether it succeeded.
func readInput(command, path string) ([]byte, bool) {
	content, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", command, err)
		return nil, false
	}
	return content, true
}

// unknownFlags reports options a subcommand does not understand. Silently ignoring
// them would make a typo look like a file that produced no output.
func unknownFlags(command string, flags []string, known ...string) bool {
	ok := true
	for _, f := range flags {
		recognised := false
		for _, k := range known {
			if f == k {
				recognised = true
				break
			}
		}
		if !recognised {
			fmt.Fprintf(os.Stderr, "%s: unknown option %s\n", command, f)
			ok = false
		}
	}
	return ok
}
