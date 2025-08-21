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
	var projectFilter string
	flag.StringVar(&taskFile, "t", "", "Path to task file")
	flag.StringVar(&projectFilter, "p", "", "Filter tasks by project (for ls command)")

	// Custom usage to handle our command structure
	flag.Usage = func() {
		showHelp()
	}

	// Parse all flags
	flag.Parse()

	// Set task file path if specified
	if taskFile != "" {
		internal.SetTaskFilePath(taskFile)
	}

	// Get command and remaining args
	args := flag.Args()
	
	if len(args) == 0 {
		// No command, run interactive mode
		if err := InteractiveCommand(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	command := args[0]
	nonFlagArgs := args[1:]

	var err error

	switch command {
	case "add", "a":
		err = AddCommand(nonFlagArgs)
	case "ls", "list", "l":
		err = ListCommand(projectFilter)
	case "edit", "e":
		err = EditCommand()
	case "kanban":
		err = KanbanCommand()
	case "httpd":
		addr := ""
		if len(nonFlagArgs) > 0 {
			addr = nonFlagArgs[0]
		}
		err = HttpdCommand(addr)
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
  -p <project>   Filter tasks by project (for ls command)

Commands:
  add <title>    Add a new task (supports +project tags)
  ls, list       List all tasks (use -p to filter by project)
  edit, e        Edit a task interactively
  kanban         Show tasks in kanban board view
  httpd [addr]   Start HTTP server (default: 127.0.0.1:7676)
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
  taskeru -p work ls                # List only tasks with +work project
  taskeru edit                      # Select and edit a task
  taskeru -t /tmp/test.json add "Test task"  # Use different file

Environment Variables:
  EDITOR          Editor to use for editing (default: vim)`)
}
