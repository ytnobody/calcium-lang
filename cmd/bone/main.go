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
	case "install":
		handleInstall()
	case "remove", "rm":
		handleRemove(args[1:])
	case "list", "ls":
		handleList(args[1:])
	case "update":
		handleUpdate(args[1:])
	case "config":
		handleConfig(args[1:])
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
  install                  Install all modules from calcium.lock
  add <module>[@version]   Add a module from Boneyard
                           Options:
                             --global, -g  Install to global cache (~/.calcium/cache/)
                           Examples: bone add AUTHOR/module
                                     bone add AUTHOR/module@1.0.0
                                     bone add --global AUTHOR/module
  remove <module>          Remove an installed module
  list                     List installed modules
  update [module]          Update modules to latest versions
  config                   Show all configuration
  config get <key>         Get a configuration value
  config set <key> <value> Set a configuration value
  version                  Show bone version
  help                     Show this help message

Configuration keys:
  registry_url             URL of the module registry

Examples:
  bone init                Initialize module in current directory
  bone init my-lib         Create new module 'my-lib'
  bone add JOHNDOE/utils   Install latest version of utils (local)
  bone add -g ALICE/http   Install to global cache
  bone config set registry_url https://example.com/registry`)
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

func handleInstall() {
	if err := bone.Install(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func handleConfig(args []string) {
	if len(args) == 0 {
		// Show all config
		if err := bone.ShowConfig(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	switch args[0] {
	case "get":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Error: key required")
			fmt.Fprintln(os.Stderr, "Usage: bone config get <key>")
			os.Exit(1)
		}
		value, err := bone.GetConfig(args[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(value)

	case "set":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "Error: key and value required")
			fmt.Fprintln(os.Stderr, "Usage: bone config set <key> <value>")
			os.Exit(1)
		}
		if err := bone.SetConfig(args[1], args[2]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Set %s = %s\n", args[1], args[2])

	default:
		fmt.Fprintf(os.Stderr, "Unknown config command: %s\n", args[0])
		fmt.Fprintln(os.Stderr, "Usage: bone config [get|set] ...")
		os.Exit(1)
	}
}
