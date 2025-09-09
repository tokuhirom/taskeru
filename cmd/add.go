package cmd

import (
	"fmt"
	"strings"

	"taskeru/internal"
)

func AddCommand(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("task title is required")
	}

	title := strings.Join(args, " ")

	// Extract scheduled date from title
	task := internal.ParseTask(title)

	taskFile := internal.NewTaskFile()
	if err := taskFile.AddTask(task); err != nil {
		return fmt.Errorf("failed to add task: %w", err)
	}

	fmt.Printf("Task added: %s\n", task)
	return nil
}
