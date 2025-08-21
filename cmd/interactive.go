package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"taskeru/internal"
)

func InteractiveCommand() error {
	return InteractiveCommandWithFilter("")
}

func InteractiveCommandWithFilter(projectFilter string) error {
	for {
		tasks, err := internal.LoadTasks()
		if err != nil {
			return fmt.Errorf("failed to load tasks: %w", err)
		}

		updatedTasks, modified, taskToEdit, deletedTaskIDs, newTaskTitle, shouldReload, err := internal.ShowInteractiveTaskListWithFilter(tasks, projectFilter)
		if err != nil {
			return fmt.Errorf("failed to show interactive list: %w", err)
		}

		// Handle reload
		if shouldReload {
			// Save modifications before reloading if needed
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

				if err := internal.SaveTasks(updatedTasks); err != nil {
					return fmt.Errorf("failed to save tasks: %w", err)
				}
			}
			fmt.Println("Reloading tasks...")
			continue
		}

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
			if err := internal.SaveDeletedTasksToTrash(deletedTasks); err != nil {
				fmt.Printf("Warning: failed to save to trash: %v\n", err)
			}

			// Mark as modified to trigger save
			modified = true
		}

		// Handle new task creation
		if newTaskTitle != "" {
			// Extract projects and scheduled/due dates from title
			cleanTitle, projects := internal.ExtractProjectsFromTitle(newTaskTitle)
			cleanTitle, scheduledDate := internal.ExtractScheduledDateFromTitle(cleanTitle)
			cleanTitle, dueDate := internal.ExtractDeadlineFromTitle(cleanTitle)

			newTask := internal.NewTask(cleanTitle)
			newTask.Projects = projects
			newTask.ScheduledDate = scheduledDate
			newTask.DueDate = dueDate

			if err := internal.AddTask(newTask); err != nil {
				return fmt.Errorf("failed to create task: %w", err)
			}

			fmt.Printf("Task created: %s", cleanTitle)
			if len(projects) > 0 {
				fmt.Printf(" [Projects: %s]", strings.Join(projects, ", "))
			}
			if scheduledDate != nil {
				fmt.Printf(" [Scheduled: %s]", scheduledDate.Format("2006-01-02"))
			}
			if dueDate != nil {
				fmt.Printf(" [Due: %s]", dueDate.Format("2006-01-02"))
			}
			fmt.Println()
			continue // Go back to the list after creating
		}

		// Handle edit task
		if taskToEdit != nil {
			// Remember the original updated time for conflict detection
			originalUpdated := taskToEdit.Updated

			// Open editor
			if err := editTaskNoteInteractive(taskToEdit); err != nil {
				fmt.Printf("Editor error: %v\n", err)
				continue
			}

			// Update the task with conflict check
			if err := internal.UpdateTaskWithConflictCheck(taskToEdit.ID, originalUpdated, func(t *internal.Task) {
				t.Title = taskToEdit.Title
				t.Projects = taskToEdit.Projects
				t.Note = taskToEdit.Note
			}); err != nil {
				if strings.Contains(err.Error(), "modified by another process") {
					fmt.Println("Conflict: task was modified by another process, please try again")
				} else {
					fmt.Printf("Failed to save task: %v\n", err)
				}
				continue
			}

			fmt.Printf("Task updated: %s\n", taskToEdit.Title)
			continue // Go back to the list after editing
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

			if err := internal.SaveTasks(updatedTasks); err != nil {
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

func editTaskNoteInteractive(task *internal.Task) error {
	// Create temp file with Markdown extension
	tmpfile, err := os.CreateTemp("", "task-*.md")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpfile.Name())

	// Pre-fill with current title and note
	content := fmt.Sprintf("# %s\n\n%s", task.Title, task.Note)
	if _, err := tmpfile.WriteString(content); err != nil {
		return fmt.Errorf("failed to write to temp file: %w", err)
	}
	tmpfile.Close()

	// Open editor
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}

	cmd := exec.Command(editor, tmpfile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("editor failed: %w", err)
	}

	// Read back the edited content
	editedContent, err := os.ReadFile(tmpfile.Name())
	if err != nil {
		return fmt.Errorf("failed to read edited file: %w", err)
	}

	// Parse the content
	lines := strings.Split(string(editedContent), "\n")
	newTitle := task.Title // Default to original title
	noteLines := []string{}
	inNote := false

	for _, line := range lines {
		if !inNote && strings.HasPrefix(line, "# ") {
			// Extract title from first heading
			newTitle = strings.TrimSpace(strings.TrimPrefix(line, "#"))
			inNote = true
		} else if inNote {
			noteLines = append(noteLines, line)
		}
	}

	// Trim leading empty lines from note
	for len(noteLines) > 0 && strings.TrimSpace(noteLines[0]) == "" {
		noteLines = noteLines[1:]
	}

	// Trim trailing empty lines from note
	for len(noteLines) > 0 && strings.TrimSpace(noteLines[len(noteLines)-1]) == "" {
		noteLines = noteLines[:len(noteLines)-1]
	}

	// Extract projects from the new title
	cleanTitle, projects := internal.ExtractProjectsFromTitle(newTitle)

	// Update task
	task.Title = cleanTitle
	task.Projects = projects
	task.Note = strings.Join(noteLines, "\n")

	return nil
}

func editTaskInteractive(task *internal.Task) error {
	// Remember the original updated time for conflict detection
	originalUpdated := task.Updated

	// Open editor for the task
	if err := editTaskNoteInteractive(task); err != nil {
		return fmt.Errorf("editor error: %w", err)
	}

	// Update the task with conflict check
	if err := internal.UpdateTaskWithConflictCheck(task.ID, originalUpdated, func(t *internal.Task) {
		t.Title = task.Title
		t.Projects = task.Projects
		t.Note = task.Note
	}); err != nil {
		return err
	}

	fmt.Printf("Task updated: %s\n", task.Title)
	return nil
}