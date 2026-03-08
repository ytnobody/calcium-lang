// Package main provides a simple HTTP server for the Calcium Web Playground.
// It serves files from the playground/ directory and sets the correct
// Content-Type and COOP/COEP headers required for SharedArrayBuffer (used by
// some Go WASM runtimes).
//
// Usage:
//
//	go run ./cmd/playground-server/
//
// Then open http://localhost:8080 in your browser.
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

func main() {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	dir := flag.String("dir", "playground", "Directory to serve (default: playground/)")
	flag.Parse()

	// Resolve the playground directory relative to the binary location or cwd.
	playgroundDir := *dir
	if !filepath.IsAbs(playgroundDir) {
		cwd, err := os.Getwd()
		if err != nil {
			log.Fatalf("cannot determine working directory: %v", err)
		}
		playgroundDir = filepath.Join(cwd, playgroundDir)
	}

	if _, err := os.Stat(playgroundDir); os.IsNotExist(err) {
		log.Fatalf("playground directory not found: %s", playgroundDir)
	}

	mux := http.NewServeMux()
	mux.Handle("/", addHeaders(http.FileServer(http.Dir(playgroundDir))))

	fmt.Printf("Calcium Playground server listening on http://localhost%s\n", *addr)
	fmt.Printf("Serving files from: %s\n", playgroundDir)
	fmt.Println()
	fmt.Println("To build the WASM runtime first, run:")
	fmt.Println("  GOOS=js GOARCH=wasm go build -o playground/calcium.wasm ./cmd/playground-wasm/")
	fmt.Println()
	fmt.Println("Also copy wasm_exec.js from the Go installation:")
	fmt.Println("  cp $(go env GOROOT)/misc/wasm/wasm_exec.js playground/")
	fmt.Println()

	if err := http.ListenAndServe(*addr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

// addHeaders wraps a handler to add the necessary HTTP headers for WASM.
func addHeaders(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Required for SharedArrayBuffer in modern browsers.
		w.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
		w.Header().Set("Cross-Origin-Embedder-Policy", "require-corp")

		// Set correct MIME type for WASM.
		if filepath.Ext(r.URL.Path) == ".wasm" {
			w.Header().Set("Content-Type", "application/wasm")
		}

		h.ServeHTTP(w, r)
	})
}
