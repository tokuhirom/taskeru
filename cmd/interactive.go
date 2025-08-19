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
	for {
		tasks, err := internal.LoadTasks()
		if err != nil {
			return fmt.Errorf("failed to load tasks: %w", err)
		}
		
		updatedTasks, modified, taskToEdit, deletedTaskIDs, newTaskTitle, shouldReload, err := internal.ShowInteractiveTaskList(tasks)
		if err != nil {
			return fmt.Errorf("failed to show interactive list: %w", err)
		}
		
		// Handle reload
		if shouldReload {
			fmt.Println("Reloading tasks...")
			continue
		}
		
		// If user quit (pressed q/esc), exit
		if taskToEdit == nil && !modified && len(deletedTaskIDs) == 0 && newTaskTitle == "" {
			break
		}
		
		// Handle new task creation
		if newTaskTitle != "" {
			newTask := internal.NewTask(newTaskTitle)
			if err := internal.AddTask(newTask); err != nil {
				fmt.Printf("Failed to create task: %v\n", err)
			} else {
				fmt.Printf("Task created: %s\n", newTaskTitle)
			}
			continue // Go back to the list
		}
		
		// Handle deleted tasks
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
			
			// Mark as modified to save the updated list
			modified = true
		}
		
		// If tasks were modified (status toggled or deleted), save them
		if modified {
			// Update the Updated timestamp for modified tasks
			now := time.Now()
			for i := range updatedTasks {
				for j := range tasks {
					if updatedTasks[i].ID == tasks[j].ID && updatedTasks[i].Status != tasks[j].Status {
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
		
		// If user pressed 'e' on a task, open editor
		if taskToEdit != nil {
			// Remember the original updated timestamp for conflict check
			originalUpdated := taskToEdit.Updated
			
			if err := editTaskNoteInteractive(taskToEdit); err != nil {
				// If editor failed, just continue (back to list)
				fmt.Printf("Editor error: %v\n", err)
				continue
			}
			
			if err := internal.UpdateTaskWithConflictCheck(taskToEdit.ID, originalUpdated, func(t *internal.Task) {
				t.Title = taskToEdit.Title
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
			// Continue to show the list again
			continue
		}
	}
	
	return nil
}

func editTaskNoteInteractive(task *internal.Task) error {
	tempFile, err := os.CreateTemp("", "taskeru-*.md")
	if err != nil {
		return err
	}
	defer os.Remove(tempFile.Name())
	
	content := fmt.Sprintf("# %s\n\n%s", task.Title, task.Note)
	if _, err := tempFile.WriteString(content); err != nil {
		tempFile.Close()
		return err
	}
	tempFile.Close()
	
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}
	
	cmd := exec.Command(editor, tempFile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	if err := cmd.Run(); err != nil {
		return nil
	}
	
	editedContent, err := os.ReadFile(tempFile.Name())
	if err != nil {
		return err
	}
	
	parsedTitle, parsedNote := parseEditedContentInteractive(string(editedContent))
	task.Title = parsedTitle
	task.Note = parsedNote
	
	return nil
}

func parseEditedContentInteractive(content string) (title string, note string) {
	lines := strings.Split(content, "\n")
	
	foundTitle := false
	var noteLines []string
	
	for _, line := range lines {
		if !foundTitle && strings.HasPrefix(line, "# ") {
			title = strings.TrimPrefix(line, "# ")
			foundTitle = true
			continue
		}
		
		if foundTitle {
			noteLines = append(noteLines, line)
		}
	}
	
	// Remove leading empty lines
	for len(noteLines) > 0 && noteLines[0] == "" {
		noteLines = noteLines[1:]
	}
	
	// Remove trailing empty lines
	for len(noteLines) > 0 && noteLines[len(noteLines)-1] == "" {
		noteLines = noteLines[:len(noteLines)-1]
	}
	
	note = strings.Join(noteLines, "\n")
	
	// If no title found, use first line as title
	if title == "" && len(lines) > 0 {
		title = lines[0]
		if len(lines) > 1 {
			note = strings.Join(lines[1:], "\n")
		} else {
			note = ""
		}
	}
	
	return title, note
}