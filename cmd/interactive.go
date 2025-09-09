package cmd

import (
	"fmt"
	"time"

	"taskeru/internal"

	tea "github.com/charmbracelet/bubbletea"
)

func InteractiveCommandWithFilter(projectFilter string, taskFile *internal.TaskFile) error {
	model, err := internal.NewInteractiveTaskListWithFilter(taskFile, projectFilter)
	if err != nil {
		return fmt.Errorf("failed to create interactive model: %w", err)
	}

	// Start Bubble Tea program with AltScreen
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err = p.Run(); err != nil {
		return fmt.Errorf("failed to run interactive UI: %w", err)
	}

	for {
		tasks, err := taskFile.LoadTasks()
		if err != nil {
			return fmt.Errorf("failed to load tasks: %w", err)
		}

		//updatedTasks, modified, taskToEdit, deletedTaskIDs, newTaskTitle, shouldReload, err := internal.ShowInteractiveTaskListWithFilter(tasks, projectFilter)
		var updatedTasks []internal.Task
		var modified bool
		var deletedTaskIDs []string

		// Handle task deletion
		if len(deletedTaskIDs) > 0 {
			// Collect deleted tasks for trash
			var deletedTasks []internal.Task
			for _, id := range deletedTaskIDs {
				for _, task := range tasks {
					if task.ID == id {
						deletedTasks = append(deletedTasks, task)
						break
					}
				}
			}

			// Save to trash
			if err := taskFile.SaveDeletedTasksToTrash(deletedTasks); err != nil {
				fmt.Printf("Warning: failed to save to trash: %v\n", err)
			}

			// Mark as modified to trigger save
			modified = true
		}

		// Save modifications and exit
		if modified {
			// Update the Updated timestamp for modified tasks
			now := time.Now()
			for i := range updatedTasks {
				for j := range tasks {
					if updatedTasks[i].ID == tasks[j].ID &&
						(updatedTasks[i].Status != tasks[j].Status ||
							updatedTasks[i].Priority != tasks[j].Priority ||
							updatedTasks[i].DueDate != tasks[j].DueDate ||
							updatedTasks[i].ScheduledDate != tasks[j].ScheduledDate) {
						updatedTasks[i].Updated = now
						break
					}
				}
			}

			if err := taskFile.SaveTasks(updatedTasks); err != nil {
				return fmt.Errorf("failed to save tasks: %w", err)
			}

			if len(deletedTaskIDs) > 0 {
				fmt.Printf("%d task(s) deleted and moved to trash.\n", len(deletedTaskIDs))
			} else {
				fmt.Println("Tasks updated.")
			}
		}

		// Normal exit
		break
	}

	return nil
}
