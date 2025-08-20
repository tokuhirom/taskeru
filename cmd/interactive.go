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
		
		updatedTasks, modified, taskToEdit, deletedTaskIDs, newTaskTitle, shouldReload, showProjectView, err := internal.ShowInteractiveTaskList(tasks)
		if err != nil {
			return fmt.Errorf("failed to show interactive list: %w", err)
		}
		
		// Handle project view
		if showProjectView {
			projectTasks, projectModified, projectEditTask, projectDeletedIDs, projectNewTask, projectReload, err := internal.ShowProjectView(tasks)
			if err != nil {
				fmt.Printf("Failed to show project view: %v\n", err)
			}
			
			// Handle reload request from project view
			if projectReload {
				if projectModified {
					// Save before reloading
					if err := internal.SaveTasks(projectTasks); err != nil {
						fmt.Printf("Failed to save tasks before reload: %v\n", err)
					}
				}
				fmt.Println("Reloading tasks...")
				continue
			}
			
			// Handle new task creation
			if projectNewTask != "" {
				cleanTitle, projects := internal.ExtractProjectsFromTitle(projectNewTask)
				newTask := internal.NewTask(cleanTitle)
				newTask.Projects = projects
				
				if err := internal.AddTask(newTask); err != nil {
					fmt.Printf("Failed to create task: %v\n", err)
				} else {
					fmt.Printf("Task created: %s", cleanTitle)
					if len(projects) > 0 {
						fmt.Printf(" [Projects: %s]", strings.Join(projects, ", "))
					}
					fmt.Println()
				}
				continue
			}
			
			// Handle task deletion
			if len(projectDeletedIDs) > 0 {
				// Collect deleted tasks for trash
				var deletedTasks []internal.Task
				for _, id := range projectDeletedIDs {
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
				
				projectModified = true
			}
			
			// Handle task editing
			if projectEditTask != nil {
				originalUpdated := projectEditTask.Updated
				
				if err := editTaskNoteInteractive(projectEditTask); err != nil {
					fmt.Printf("Editor error: %v\n", err)
					continue
				}
				
				if err := internal.UpdateTaskWithConflictCheck(projectEditTask.ID, originalUpdated, func(t *internal.Task) {
					t.Title = projectEditTask.Title
					t.Projects = projectEditTask.Projects
					t.Note = projectEditTask.Note
				}); err != nil {
					if strings.Contains(err.Error(), "modified by another process") {
						fmt.Println("Conflict: task was modified by another process, please try again")
					} else {
						fmt.Printf("Failed to save task: %v\n", err)
					}
					continue
				}
				
				fmt.Printf("Task updated: %s\n", projectEditTask.Title)
				continue
			}
			
			if projectModified {
				// Save the modified tasks
				if err := internal.SaveTasks(projectTasks); err != nil {
					fmt.Printf("Failed to save tasks: %v\n", err)
				} else {
					if len(projectDeletedIDs) > 0 {
						fmt.Printf("%d task(s) deleted and moved to trash.\n", len(projectDeletedIDs))
					} else {
						fmt.Println("Tasks updated from project view.")
					}
				}
			}
			continue // Go back to the list
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
							 updatedTasks[i].Priority != tasks[j].Priority) {
							updatedTasks[i].Updated = now
							break
						}
					}
				}
				
				if err := internal.SaveTasks(updatedTasks); err != nil {
					fmt.Printf("Failed to save tasks before reload: %v\n", err)
				} else {
					fmt.Println("Saved changes before reloading...")
				}
			}
			fmt.Println("Reloading tasks...")
			continue
		}
		
		// If user quit (pressed q/esc), exit
		if taskToEdit == nil && !modified && len(deletedTaskIDs) == 0 && newTaskTitle == "" {
			break
		}
		
		// Handle new task creation
		if newTaskTitle != "" {
			// Extract projects from title
			cleanTitle, projects := internal.ExtractProjectsFromTitle(newTaskTitle)
			
			newTask := internal.NewTask(cleanTitle)
			newTask.Projects = projects
			
			if err := internal.AddTask(newTask); err != nil {
				fmt.Printf("Failed to create task: %v\n", err)
			} else {
				fmt.Printf("Task created: %s", cleanTitle)
				if len(projects) > 0 {
					fmt.Printf(" [Projects: %s]", strings.Join(projects, ", "))
				}
				fmt.Println()
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
		
		// If tasks were modified (status toggled, priority changed, or deleted), save them
		if modified {
			// Update the Updated timestamp for modified tasks
			now := time.Now()
			for i := range updatedTasks {
				for j := range tasks {
					if updatedTasks[i].ID == tasks[j].ID && 
						(updatedTasks[i].Status != tasks[j].Status || 
						 updatedTasks[i].Priority != tasks[j].Priority) {
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
	
	// Load configuration
	config, _ := internal.LoadConfig()
	
	// Include projects in the title line
	titleWithProjects := task.Title
	for _, project := range task.Projects {
		titleWithProjects += " +" + project
	}
	
	noteContent := task.Note
	
	// Add timestamp if enabled in config
	if config.Editor.AddTimestamp {
		now := time.Now()
		// Format: YYYY-MM-DD(Day) HH:MM
		weekday := now.Format("Mon")
		timestamp := fmt.Sprintf("\n\n## %s(%s) %s\n", now.Format("2006-01-02"), weekday, now.Format("15:04"))
		
		// Append timestamp to existing note or create new note with timestamp
		if noteContent != "" {
			noteContent += timestamp
		} else {
			noteContent = timestamp
		}
	}
	
	content := fmt.Sprintf("# %s\n\n%s", titleWithProjects, noteContent)
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
	
	parsedTitle, parsedProjects, parsedNote := parseEditedContentInteractive(string(editedContent))
	task.Title = parsedTitle
	task.Projects = parsedProjects
	task.Note = parsedNote
	
	return nil
}

func parseEditedContentInteractive(content string) (title string, projects []string, note string) {
	lines := strings.Split(content, "\n")
	
	foundTitle := false
	var noteLines []string
	
	for _, line := range lines {
		if !foundTitle && strings.HasPrefix(line, "# ") {
			titleLine := strings.TrimPrefix(line, "# ")
			// Extract projects from the title line
			title, projects = internal.ExtractProjectsFromTitle(titleLine)
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
	
	return title, projects, note
}