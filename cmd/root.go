package cmd

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"taskeru/internal"
)

func Execute() {
	// Parse global flags first
	var taskFileName string
	var projectFilter string
	var logFile string
	flag.StringVar(&taskFileName, "t", "", "Path to task file")
	flag.StringVar(&projectFilter, "p", "", "Filter tasks by project (for ls command)")
	flag.StringVar(&logFile, "l", "log", "Path to log file")

	// Custom usage to handle our command structure
	flag.Usage = func() {
		showHelp()
	}

	// Parse all flags
	flag.Parse()

	taskFile := func() *internal.TaskFile {
		if taskFileName == "" {
			return internal.NewTaskFile()
		} else {
			return internal.NewTaskFileWithPath(taskFileName)
		}
	}()

	// Get command and remaining args
	args := flag.Args()

	// logFile に書いていく
	if logFile != "" {
		f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Failed to open log file: %v\n", err)
			os.Exit(1)
		}
		defer func() {
			_ = f.Close()
		}()
		slog.SetDefault(slog.New(slog.NewTextHandler(f, &slog.HandlerOptions{})))
	} else {
		slog.SetDefault(slog.New(slog.DiscardHandler))
	}

	if len(args) == 0 {
		// No command, run interactive mode (with project filter if specified)
		if err := InteractiveCommandWithFilter(projectFilter, taskFile); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	command := args[0]
	nonFlagArgs := args[1:]

	var err error

	switch command {
	case "add", "a":
		err = AddCommand(taskFile, nonFlagArgs)
	case "ls", "list", "l":
		err = ListCommand(taskFile, projectFilter)
	case "edit", "e":
		err = EditCommand(taskFile)
	case "httpd":
		addr := ""
		if len(nonFlagArgs) > 0 {
			addr = nonFlagArgs[0]
		}
		err = HttpdCommand(taskFile, addr)
	case "init-config":
		err = InitConfigCommand()
	case "help", "-h", "--help":
		showHelp()
	default:
		_, _ = fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		showHelp()
		os.Exit(1)
	}

	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func showHelp() {
	fmt.Println(`taskeru - Simple CLI task management tool

Usage:
  taskeru [options] [command] [arguments]

Options:
  -t <file>      Path to task file (default: ~/todo.json)
  -p <project>   Filter tasks by project (for ls and interactive mode)

Commands:
  add <title>    Add a new task (supports +project, due:date, scheduled:date)
  ls, list       List all tasks (use -p to filter by project)
  edit, e        Edit a task interactively
  httpd [addr]   Start HTTP server for web UI (default: 127.0.0.1:7676)
  init-config    Create default configuration file
  help           Show this help message

Interactive Mode Keys:
  j/k or ↑/↓    Move cursor
  space         Toggle task done/todo
  D             Set deadline for selected task
  S             Set scheduled date for selected task
  /             Search tasks (title, projects, notes)
  a             Show all tasks (including old completed)
  c             Create new task
  e             Edit selected task
  d             Delete selected task
  r             Reload tasks
  q             Quit

Examples:
  taskeru                           # Interactive mode
  taskeru add "Buy milk +personal"  # Add task with project
  taskeru add "Report due:tomorrow" # Add task with deadline
  taskeru add "Review sched:monday due:friday +work"  # Task with scheduled and due date
  taskeru ls                        # List all tasks
  taskeru -p work ls                # List only tasks with +work project
  taskeru edit                      # Select and edit a task
  taskeru -t /tmp/test.json add "Test task"  # Use different file

Date formats (for due: and scheduled:/sched:):
  today             # Today
  tomorrow          # Tomorrow
  monday            # Next Monday (or any weekday)
  2024-12-31        # Specific date (YYYY-MM-DD)
  12-25             # Month-day (current/next year)
  12/25             # Alternative format
  
Note: 
  - due:date sets deadline (end of day, 23:59:59)
  - scheduled:date or sched:date sets when task becomes active (start of day, 00:00:00)

Environment Variables:
  EDITOR          Editor to use for editing (default: vim)`)
}
