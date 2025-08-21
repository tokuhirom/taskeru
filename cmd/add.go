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

	// Extract deadline from title
	cleanTitle, deadline := internal.ExtractDeadlineFromTitle(title)
	
	// Extract projects from title
	cleanTitle, projects := internal.ExtractProjectsFromTitle(cleanTitle)

	task := internal.NewTask(cleanTitle)
	task.Projects = projects
	task.DueDate = deadline

	if err := internal.AddTask(task); err != nil {
		return fmt.Errorf("failed to add task: %w", err)
	}

	fmt.Printf("Task added: %s", task.Title)
	if len(task.Projects) > 0 {
		fmt.Printf(" [Projects: %s]", strings.Join(task.Projects, ", "))
	}
	if task.DueDate != nil {
		fmt.Printf(" [Due: %s]", task.DueDate.Format("2006-01-02"))
	}
	fmt.Println()
	return nil
}
