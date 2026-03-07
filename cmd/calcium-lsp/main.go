// calcium-lsp is the Language Server Protocol implementation for the Calcium language.
// It communicates via stdio using the JSON-RPC 2.0 protocol.
//
// Usage:
//
//	calcium-lsp [--log <logfile>]
package main

import (
	"flag"
	"io"
	"log"
	"os"

	"github.com/ytnobody/calcium-lang/pkg/lsp"
)

func main() {
	logFile := flag.String("log", "", "path to log file (default: discard)")
	flag.Parse()

	var logger *log.Logger
	if *logFile != "" {
		f, err := os.OpenFile(*logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatalf("failed to open log file: %v", err)
		}
		defer f.Close()
		logger = log.New(f, "[calcium-lsp] ", log.LstdFlags)
	} else {
		logger = log.New(io.Discard, "", 0)
	}

	logger.Println("calcium-lsp starting")

	server := lsp.NewServer(os.Stdin, os.Stdout, logger)
	server.Run()

	logger.Println("calcium-lsp exiting")
}
