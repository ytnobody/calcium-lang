package main

import (
	"fmt"
	"os"
)

// splitArgs separates the options a subcommand knows from its input files. An
// argument is an option only if it is one of `known`; anything else is a file name,
// including one that happens to start with `-` (`calcium lint ./-weird.ca`).
func splitArgs(args []string, known ...string) (flags, files []string) {
	for _, a := range args {
		isFlag := false
		for _, k := range known {
			if a == k {
				isFlag = true
				break
			}
		}
		if isFlag {
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
