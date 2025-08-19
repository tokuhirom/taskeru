package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"taskeru/internal"
)

func EditCommand() error {
	tasks, err := internal.LoadTasks()
	if err != nil {
		return fmt.Errorf("failed to load tasks: %w", err)
	}
	
	// Show all tasks in edit mode (including old completed ones)
	if len(tasks) == 0 {
		fmt.Println("No tasks to edit.")
		return nil
	}
	
	task, err := internal.SelectTask(tasks)
	if err != nil {
		return nil
	}
	
	// Remember the original updated timestamp for conflict check
	originalUpdated := task.Updated
	
	if err := editTaskNote(task); err != nil {
		return fmt.Errorf("failed to edit task: %w", err)
	}
	
	if err := internal.UpdateTaskWithConflictCheck(task.ID, originalUpdated, func(t *internal.Task) {
		t.Title = task.Title
		t.Note = task.Note
	}); err != nil {
		if strings.Contains(err.Error(), "modified by another process") {
			return fmt.Errorf("conflict: task was modified by another process, please try again")
		}
		return fmt.Errorf("failed to save task: %w", err)
	}
	
	fmt.Printf("Task updated: %s\n", task.Title)
	return nil
}

func editTaskNote(task *internal.Task) error {
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
	
	parsedTitle, parsedNote := parseEditedContent(string(editedContent))
	task.Title = parsedTitle
	task.Note = parsedNote
	
	return nil
}

func parseEditedContent(content string) (title string, note string) {
	scanner := bufio.NewScanner(strings.NewReader(content))
	
	foundTitle := false
	var lines []string
	
	for scanner.Scan() {
		line := scanner.Text()
		
		if !foundTitle && strings.HasPrefix(line, "# ") {
			title = strings.TrimPrefix(line, "# ")
			foundTitle = true
			continue
		}
		
		if foundTitle {
			lines = append(lines, line)
		}
	}
	
	for len(lines) > 0 && lines[0] == "" {
		lines = lines[1:]
	}
	
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	
	note = strings.Join(lines, "\n")
	
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