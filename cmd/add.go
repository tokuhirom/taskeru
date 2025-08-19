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
	
	// Extract projects from title
	cleanTitle, projects := internal.ExtractProjectsFromTitle(title)
	
	task := internal.NewTask(cleanTitle)
	task.Projects = projects
	
	if err := internal.AddTask(task); err != nil {
		return fmt.Errorf("failed to add task: %w", err)
	}
	
	fmt.Printf("Task added: %s", task.Title)
	if len(task.Projects) > 0 {
		fmt.Printf(" [Projects: %s]", strings.Join(task.Projects, ", "))
	}
	fmt.Println()
	return nil
}