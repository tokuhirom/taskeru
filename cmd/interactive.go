package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
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
	_, err = p.Run()
	if err != nil {
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
		var taskToEdit *internal.Task
		var deletedTaskIDs []string
		var newTaskTitle string

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

		// Handle new task creation
		if newTaskTitle != "" {
			// Extract projects and scheduled/due dates from title
			newTask := internal.ParseTask(newTaskTitle)

			if err := taskFile.AddTask(newTask); err != nil {
				return fmt.Errorf("failed to create task: %w", err)
			}

			fmt.Printf("Task created: %s\n", newTask.String())
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
			if err := taskFile.UpdateTaskWithConflictCheck(taskToEdit.ID, originalUpdated, func(t *internal.Task) {
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

func editTaskNoteInteractive(task *internal.Task) error {
	// Create temp file with Markdown extension
	tmpfile, err := os.CreateTemp("", "task-*.md")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()

	// Load configuration
	config, err := internal.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Pre-fill with current title and note
	noteContent := task.Note

	// Add timestamp if enabled in config
	if config.Editor.AddTimestamp {
		now := time.Now()
		// Format: YYYY-MM-DD(Day) HH:MM
		timestamp := now.Format("\n\n## 2006-01-02(Mon) 15:04\n\n")

		// Append timestamp to existing note or create new note with timestamp
		if noteContent != "" {
			noteContent += timestamp
		} else {
			noteContent = timestamp
		}
	}

	content := fmt.Sprintf("# %s\n\n%s", task.Title, noteContent)
	if _, err := tmpfile.WriteString(content); err != nil {
		return fmt.Errorf("failed to write to temp file: %w", err)
	}
	_ = tmpfile.Close()

	// Open editor
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}
	slog.Debug("Opening editor",
		slog.String("editor", editor),
		slog.String("file", tmpfile.Name()))

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
