package cmd

import (
	"fmt"
	"strings"
	"time"

	"taskeru/internal"
)

func ListCommand() error {
	tasks, err := internal.LoadTasks()
	if err != nil {
		return fmt.Errorf("failed to load tasks: %w", err)
	}
	
	// Filter out old completed tasks by default
	visibleTasks := internal.FilterVisibleTasks(tasks, false)
	
	if len(visibleTasks) == 0 {
		fmt.Println("No tasks found.")
		hiddenCount := len(tasks) - len(visibleTasks)
		if hiddenCount > 0 {
			fmt.Printf("(%d old completed tasks hidden)\n", hiddenCount)
		}
		return nil
	}
	
	fmt.Println("Tasks:")
	fmt.Println("------")
	
	for i, task := range visibleTasks {
		status := task.DisplayStatus()
		priority := task.DisplayPriority()
		
		// Add color based on status
		var statusColor string
		switch task.Status {
		case internal.StatusDONE:
			statusColor = "\x1b[32m" // green
		case internal.StatusDOING:
			statusColor = "\x1b[33m" // yellow
		case internal.StatusWAITING:
			statusColor = "\x1b[34m" // blue
		case internal.StatusWONTDO:
			statusColor = "\x1b[90m" // gray
		default:
			statusColor = "" // no color for TODO
		}
		
		if statusColor != "" {
			fmt.Printf("%d. %s%-7s %s %s\x1b[0m", i+1, statusColor, status, priority, task.Title)
		} else {
			fmt.Printf("%d. %-7s %s %s", i+1, status, priority, task.Title)
		}
		
		// Display projects with colors
		if len(task.Projects) > 0 {
			var projectStrs []string
			for _, project := range task.Projects {
				// Use cyan color for projects
				projectStrs = append(projectStrs, fmt.Sprintf("\x1b[36m+%s\x1b[0m", project))
			}
			fmt.Printf(" %s", strings.Join(projectStrs, " "))
		}
		
		// Display completion date for done/wontdo tasks (dim gray)
		if (task.Status == internal.StatusDONE || task.Status == internal.StatusWONTDO) && task.CompletedAt != nil {
			// Use dim gray color (ANSI 90) for completed date
			fmt.Printf(" \x1b[90m(completed %s)\x1b[0m", task.CompletedAt.Format("2006-01-02"))
		} else if task.DueDate != nil {
			dueIn := time.Until(*task.DueDate)
			if dueIn < 0 {
				fmt.Printf(" (overdue)")
			} else if dueIn < 24*time.Hour {
				fmt.Printf(" (due today)")
			} else if dueIn < 48*time.Hour {
				fmt.Printf(" (due tomorrow)")
			} else {
				fmt.Printf(" (due %s)", task.DueDate.Format("2006-01-02"))
			}
		}
		
		fmt.Println()
		
		if task.Note != "" {
			lines := getFirstNLines(task.Note, 1)
			if len(lines) > 0 && lines[0] != "" {
				fmt.Printf("   └─ %s\n", lines[0])
			}
		}
	}
	
	hiddenCount := len(tasks) - len(visibleTasks)
	if hiddenCount > 0 {
		fmt.Printf("\n(%d old completed tasks hidden)\n", hiddenCount)
	}
	
	return nil
}

func getFirstNLines(text string, n int) []string {
	lines := []string{}
	current := ""
	
	for _, r := range text {
		if r == '\n' {
			lines = append(lines, current)
			current = ""
			if len(lines) >= n {
				break
			}
		} else {
			current += string(r)
		}
	}
	
	if current != "" && len(lines) < n {
		lines = append(lines, current)
	}
	
	return lines
}