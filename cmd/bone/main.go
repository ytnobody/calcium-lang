package main

import (
	"fmt"
	"os"

	"github.com/ytnobody/calcium-lang/pkg/bone"
)

const version = "0.1.0"

func main() {
	args := os.Args[1:]

	if len(args) == 0 {
		printHelp()
		os.Exit(0)
	}

	switch args[0] {
	case "init":
		handleInit(args[1:])
	case "add":
		handleAdd(args[1:])
	case "remove", "rm":
		handleRemove(args[1:])
	case "list", "ls":
		handleList(args[1:])
	case "update":
		handleUpdate(args[1:])
	case "version", "-v", "--version":
		fmt.Printf("bone version %s\n", version)
	case "help", "-h", "--help":
		printHelp()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", args[0])
		fmt.Fprintln(os.Stderr, "Run 'bone help' for usage.")
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println(`bone - Calcium package manager

Usage:
  bone <command> [arguments]

Commands:
  init [name]              Initialize a new Calcium module
                           If name is provided, creates a new directory
  add <module>[@version]   Add a module from Boneyard
                           Options:
                             --global, -g  Install to global cache (~/.calcium/cache/)
                           Examples: bone add AUTHOR/module
                                     bone add AUTHOR/module@1.0.0
                                     bone add --global AUTHOR/module
  remove <module>          Remove an installed module
  list                     List installed modules
  update [module]          Update modules to latest versions
  version                  Show bone version
  help                     Show this help message

Examples:
  bone init                Initialize module in current directory
  bone init my-lib         Create new module 'my-lib'
  bone add JOHNDOE/utils   Install latest version of utils (local)
  bone add -g ALICE/http   Install to global cache`)
}

func handleInit(args []string) {
	var name string
	if len(args) > 0 {
		name = args[0]
	}

	if err := bone.Init(name); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func handleAdd(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Error: module name required")
		fmt.Fprintln(os.Stderr, "Usage: bone add [--global|-g] <AUTHOR/module>[@version]")
		os.Exit(1)
	}

	global := false
	modules := []string{}

	for _, arg := range args {
		if arg == "--global" || arg == "-g" {
			global = true
		} else {
			modules = append(modules, arg)
		}
	}

	if len(modules) == 0 {
		fmt.Fprintln(os.Stderr, "Error: module name required")
		fmt.Fprintln(os.Stderr, "Usage: bone add [--global|-g] <AUTHOR/module>[@version]")
		os.Exit(1)
	}

	for _, module := range modules {
		if err := bone.Add(module, global); err != nil {
			fmt.Fprintf(os.Stderr, "Error adding %s: %v\n", module, err)
			os.Exit(1)
		}
	}
}

func handleRemove(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Error: module name required")
		fmt.Fprintln(os.Stderr, "Usage: bone remove <AUTHOR/module>")
		os.Exit(1)
	}

	for _, module := range args {
		if err := bone.Remove(module); err != nil {
			fmt.Fprintf(os.Stderr, "Error removing %s: %v\n", module, err)
			os.Exit(1)
		}
	}
}

func handleList(args []string) {
	if err := bone.List(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func handleUpdate(args []string) {
	var module string
	if len(args) > 0 {
		module = args[0]
	}

	if err := bone.Update(module); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
