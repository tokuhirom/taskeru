package cmd

import (
	"fmt"
	"os"
)

func Execute() {
	if len(os.Args) == 1 {
		if err := ListCommand(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}
	
	command := os.Args[1]
	args := os.Args[2:]
	
	var err error
	
	switch command {
	case "add", "a":
		err = AddCommand(args)
	case "ls", "list", "l":
		err = ListCommand()
	case "edit", "e":
		err = EditCommand()
	case "help", "-h", "--help":
		showHelp()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		showHelp()
		os.Exit(1)
	}
	
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func showHelp() {
	fmt.Println(`taskeru - Simple CLI task management tool

Usage:
  taskeru [command] [arguments]

Commands:
  add <title>    Add a new task
  ls, list       List all tasks (default when no command given)
  edit, e        Edit a task interactively
  help           Show this help message

Examples:
  taskeru                    # List all tasks
  taskeru add "Buy milk"     # Add a new task
  taskeru edit               # Select and edit a task

Environment Variables:
  TASKERU_FILE    Path to the task file (default: ~/todo.json)
  EDITOR          Editor to use for editing (default: vim)`)
}