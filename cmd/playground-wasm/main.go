//go:build js && wasm

// Package main is the WebAssembly entry point for the Calcium Web Playground.
// It exposes a `runCalcium(code)` JavaScript function that evaluates Calcium
// source code and returns an object with `output` and `error` fields.
package main

import (
	"syscall/js"

	"github.com/ytnobody/calcium-lang/pkg/eval"
)

func runCalcium(_ js.Value, args []js.Value) any {
	if len(args) == 0 {
		return resultObj("", "runCalcium: no code provided")
	}
	source := args[0].String()
	result := eval.Eval(source)

	errMsg := ""
	if result.Error != nil {
		errMsg = result.Error.Error()
	}
	return resultObj(result.Output, errMsg)
}

// resultObj creates a plain JavaScript object with output and error fields.
func resultObj(output, errMsg string) map[string]any {
	return map[string]any{
		"output": output,
		"error":  errMsg,
	}
}

func main() {
	// Register the runCalcium function as a global JavaScript function.
	js.Global().Set("runCalcium", js.FuncOf(runCalcium))

	// Block forever to keep the WASM module alive.
	select {}
}
