package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"taskeru/internal"
)

func KanbanCommand() error {
	for {
		tasks, err := internal.LoadTasks()
		if err != nil {
			return fmt.Errorf("failed to load tasks: %w", err)
		}
		
		kanbanTasks, kanbanModified, kanbanEditTask, kanbanDeletedIDs, kanbanReload, err := internal.ShowKanbanView(tasks)
		if err != nil {
			return fmt.Errorf("failed to show kanban view: %w", err)
		}
		
		// Handle reload request
		if kanbanReload {
			if kanbanModified {
				// Save before reloading
				now := time.Now()
				for i := range kanbanTasks {
					for j := range tasks {
						if kanbanTasks[i].ID == tasks[j].ID && 
							(kanbanTasks[i].Status != tasks[j].Status || 
							 kanbanTasks[i].Priority != tasks[j].Priority) {
							kanbanTasks[i].Updated = now
							break
						}
					}
				}
				
				if err := internal.SaveTasks(kanbanTasks); err != nil {
					fmt.Printf("Failed to save tasks before reload: %v\n", err)
				} else {
					fmt.Println("Saved changes before reloading...")
				}
			}
			fmt.Println("Reloading tasks...")
			continue
		}
		
		// If user quit without modifications, exit
		if !kanbanModified && kanbanEditTask == nil && len(kanbanDeletedIDs) == 0 {
			break
		}
		
		// Handle task deletion
		if len(kanbanDeletedIDs) > 0 {
			// Collect deleted tasks for trash
			var deletedTasks []internal.Task
			for _, id := range kanbanDeletedIDs {
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
			
			kanbanModified = true
		}
		
		// Handle task editing
		if kanbanEditTask != nil {
			originalUpdated := kanbanEditTask.Updated
			
			if err := editTaskNoteKanban(kanbanEditTask); err != nil {
				fmt.Printf("Editor error: %v\n", err)
				continue
			}
			
			if err := internal.UpdateTaskWithConflictCheck(kanbanEditTask.ID, originalUpdated, func(t *internal.Task) {
				t.Title = kanbanEditTask.Title
				t.Note = kanbanEditTask.Note
			}); err != nil {
				if strings.Contains(err.Error(), "modified by another process") {
					fmt.Println("Conflict: task was modified by another process, please try again")
				} else {
					fmt.Printf("Failed to save task: %v\n", err)
				}
				continue
			}
			
			fmt.Printf("Task updated: %s\n", kanbanEditTask.Title)
			continue
		}
		
		// Save modifications
		if kanbanModified {
			// Update timestamps for modified tasks
			now := time.Now()
			for i := range kanbanTasks {
				for j := range tasks {
					if kanbanTasks[i].ID == tasks[j].ID && 
						(kanbanTasks[i].Status != tasks[j].Status || 
						 kanbanTasks[i].Priority != tasks[j].Priority) {
						kanbanTasks[i].Updated = now
						break
					}
				}
			}
			
			if err := internal.SaveTasks(kanbanTasks); err != nil {
				return fmt.Errorf("failed to save tasks: %w", err)
			}
			
			if len(kanbanDeletedIDs) > 0 {
				fmt.Printf("%d task(s) deleted and moved to trash.\n", len(kanbanDeletedIDs))
			} else {
				fmt.Println("Tasks updated.")
			}
		}
	}
	
	return nil
}

func editTaskNoteKanban(task *internal.Task) error {
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
	
	parsedTitle, parsedNote := parseEditedContentKanban(string(editedContent))
	task.Title = parsedTitle
	task.Note = parsedNote
	
	return nil
}

func parseEditedContentKanban(content string) (title string, note string) {
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