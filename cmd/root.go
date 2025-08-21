package cmd

import (
	"flag"
	"fmt"
	"os"

	"taskeru/internal"
)

func Execute() {
	// Parse global flags first
	var taskFile string
	flag.StringVar(&taskFile, "t", "", "Path to task file (overrides TASKERU_FILE)")

	// Custom usage to handle our command structure
	flag.Usage = func() {
		showHelp()
	}

	// Parse flags only if there are arguments
	if len(os.Args) > 1 {
		// Check if first arg is a flag or command
		if os.Args[1][0] == '-' {
			flag.Parse()
		} else {
			// Parse flags after the command
			if len(os.Args) > 2 {
				flag.CommandLine.Parse(os.Args[2:])
			}
		}
	}

	// Set task file path if specified
	if taskFile != "" {
		internal.SetTaskFilePath(taskFile)
	}

	// Get command after flag parsing
	var command string
	nonFlagArgs := flag.Args()

	if len(os.Args) == 1 || (len(nonFlagArgs) == 0 && taskFile != "") {
		// No command, run interactive mode
		if err := InteractiveCommand(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Determine command
	if os.Args[1][0] != '-' {
		command = os.Args[1]
		// Get remaining args after command and flags
		nonFlagArgs = flag.Args()
	} else {
		// First arg was a flag, command should be in nonFlagArgs
		if len(nonFlagArgs) > 0 {
			command = nonFlagArgs[0]
			nonFlagArgs = nonFlagArgs[1:]
		} else {
			// Only flags, no command - run interactive
			if err := InteractiveCommand(); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			return
		}
	}

	var err error

	switch command {
	case "add", "a":
		err = AddCommand(nonFlagArgs)
	case "ls", "list", "l":
		err = ListCommand()
	case "edit", "e":
		err = EditCommand()
	case "kanban":
		err = KanbanCommand()
	case "init-config":
		err = InitConfigCommand()
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
  taskeru [options] [command] [arguments]

Options:
  -t <file>      Path to task file (default: ~/todo.json)

Commands:
  add <title>    Add a new task (supports +project tags)
  ls, list       List all tasks
  edit, e        Edit a task interactively
  kanban         Show tasks in kanban board view
  init-config    Create default configuration file
  help           Show this help message

Interactive Mode Keys:
  j/k or ↑/↓    Move cursor
  space         Toggle task done/todo
  a             Show all tasks (including old completed)
  c             Create new task
  e             Edit selected task
  d             Delete selected task
  p             Show project view
  r             Reload tasks
  q             Quit

Examples:
  taskeru                           # Interactive mode
  taskeru add "Buy milk +personal"  # Add task with project
  taskeru ls                        # List all tasks
  taskeru edit                      # Select and edit a task
  taskeru -t /tmp/test.json add "Test task"  # Use different file

Environment Variables:
  EDITOR          Editor to use for editing (default: vim)`)
}
