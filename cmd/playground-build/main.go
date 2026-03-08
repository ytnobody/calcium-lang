// Package main provides a build helper for the Calcium Web Playground.
// It compiles the playground-wasm package to WebAssembly and copies the
// required wasm_exec.js runtime file into the playground/ directory.
//
// Usage:
//
//	go run ./cmd/playground-build/
package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	// Determine the project root (assumes this tool is run from the project root).
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("cannot determine working directory: %v", err)
	}

	playgroundDir := filepath.Join(cwd, "playground")
	wasmOut := filepath.Join(playgroundDir, "calcium.wasm")

	fmt.Println("Building Calcium Web Playground WASM...")

	// Step 1: Build the WASM binary.
	fmt.Printf("  Compiling cmd/playground-wasm -> %s\n", wasmOut)
	cmd := exec.Command("go", "build", "-o", wasmOut, "./cmd/playground-wasm/")
	cmd.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatalf("WASM build failed: %v", err)
	}
	fmt.Println("  WASM build successful.")

	// Step 2: Copy wasm_exec.js from the Go installation.
	gorootCmd := exec.Command("go", "env", "GOROOT")
	gorootOut, err := gorootCmd.Output()
	if err != nil {
		log.Fatalf("cannot determine GOROOT: %v", err)
	}
	goroot := strings.TrimSpace(string(gorootOut))
	wasmExecSrc := filepath.Join(goroot, "misc", "wasm", "wasm_exec.js")
	wasmExecDst := filepath.Join(playgroundDir, "wasm_exec.js")

	fmt.Printf("  Copying wasm_exec.js from %s\n", wasmExecSrc)
	if err := copyFile(wasmExecSrc, wasmExecDst); err != nil {
		log.Fatalf("failed to copy wasm_exec.js: %v", err)
	}
	fmt.Println("  wasm_exec.js copied.")

	fmt.Println()
	fmt.Println("Build complete!")
	fmt.Println()
	fmt.Println("To start the playground server:")
	fmt.Println("  go run ./cmd/playground-server/")
	fmt.Println()
	fmt.Println("Then open: http://localhost:8080")
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source: %w", err)
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("create destination directory: %w", err)
	}

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create destination: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("copy: %w", err)
	}
	return nil
}
