package internal

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

// Global variable to store the task file path from -t option
var taskFilePath string

// SetTaskFilePath sets the global task file path (from -t option)
func SetTaskFilePath(path string) {
	taskFilePath = path
}

func GetTaskFilePath() string {
	// Priority: -t option > default
	if taskFilePath != "" {
		return filepath.Clean(taskFilePath)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		slog.Error("failed to get user home directory",
			slog.Any("error", err))
		// Fallback to the current directory
		return "todo.json"
	}
	return filepath.Join(home, "todo.json")
}

func GetTrashFilePath() string {
	// If -t option is used, put trash file in the same directory
	if taskFilePath != "" {
		dir := filepath.Dir(taskFilePath)
		base := filepath.Base(taskFilePath)
		ext := filepath.Ext(base)
		name := base[:len(base)-len(ext)]
		return filepath.Join(dir, name+".trash"+ext)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "trash.json"
	}
	return filepath.Join(home, "trash.json")
}

func LoadTasks() ([]Task, error) {
	filePath := GetTaskFilePath()
	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []Task{}, nil
		}
		return nil, err
	}
	defer func() { _ = file.Close() }()

	var tasks []Task
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if line == "" {
			continue
		}

		var task Task
		if err := json.Unmarshal([]byte(line), &task); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Skipping invalid JSON at line %d: %v\n", lineNum, err)
			continue
		}

		// Migrate old data without Updated field
		if task.Updated.IsZero() {
			task.Updated = task.Created
		}

		tasks = append(tasks, task)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return tasks, nil
}

func SaveTasks(tasks []Task) error {
	filePath := GetTaskFilePath()

	tempFile, err := os.CreateTemp(filepath.Dir(filePath), ".taskeru-*.tmp")
	if err != nil {
		return err
	}
	tempPath := tempFile.Name()

	defer func() {
		_ = tempFile.Close()
		_ = os.Remove(tempPath)
	}()

	writer := bufio.NewWriter(tempFile)
	for _, task := range tasks {
		jsonData, err := json.Marshal(task)
		if err != nil {
			return err
		}
		if _, err := writer.Write(jsonData); err != nil {
			return err
		}
		if _, err := writer.WriteString("\n"); err != nil {
			return err
		}
	}

	if err := writer.Flush(); err != nil {
		return err
	}

	if err := tempFile.Sync(); err != nil {
		return err
	}

	if err := tempFile.Close(); err != nil {
		return err
	}

	if err := os.Rename(tempPath, filePath); err != nil {
		return err
	}

	return nil
}

func AddTask(task *Task) error {
	tasks, err := LoadTasks()
	if err != nil {
		return fmt.Errorf("failed to load tasks: %w", err)
	}

	tasks = append(tasks, *task)
	return SaveTasks(tasks)
}

func UpdateTask(taskID string, updateFunc func(*Task)) error {
	tasks, err := LoadTasks()
	if err != nil {
		return err
	}

	found := false
	for i := range tasks {
		if tasks[i].ID == taskID {
			updateFunc(&tasks[i])
			tasks[i].Updated = time.Now()
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("task with ID %s not found", taskID)
	}

	return SaveTasks(tasks)
}

func UpdateTaskWithConflictCheck(taskID string, originalUpdated time.Time, updateFunc func(*Task)) error {
	tasks, err := LoadTasks()
	if err != nil {
		return err
	}

	found := false
	for i := range tasks {
		if tasks[i].ID == taskID {
			// Check if the task has been updated since we loaded it
			if !tasks[i].Updated.Equal(originalUpdated) {
				return fmt.Errorf("task has been modified by another process")
			}
			updateFunc(&tasks[i])
			tasks[i].Updated = time.Now()
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("task with ID %s not found", taskID)
	}

	return SaveTasks(tasks)
}

// SaveDeletedTasksToTrash saves deleted tasks to trash.json
func SaveDeletedTasksToTrash(deletedTasks []Task) error {
	if len(deletedTasks) == 0 {
		return nil
	}

	trashPath := GetTrashFilePath()

	// Load existing trash tasks
	existingTrash := []Task{}
	file, err := os.Open(trashPath)
	if err == nil {
		defer func() { _ = file.Close() }()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}
			var task Task
			if err := json.Unmarshal([]byte(line), &task); err == nil {
				existingTrash = append(existingTrash, task)
			}
		}
	}

	// Mark deleted tasks with deletion time
	now := time.Now()
	for i := range deletedTasks {
		// Store deletion time in Updated field
		deletedTasks[i].Updated = now
	}

	// Append deleted tasks to existing trash
	allTrash := append(existingTrash, deletedTasks...)

	// Write all trash tasks
	tempFile, err := os.CreateTemp(filepath.Dir(trashPath), ".trash-*.tmp")
	if err != nil {
		return err
	}
	tempPath := tempFile.Name()

	defer func() {
		_ = tempFile.Close()
		_ = os.Remove(tempPath)
	}()

	writer := bufio.NewWriter(tempFile)
	for _, task := range allTrash {
		jsonData, err := json.Marshal(task)
		if err != nil {
			return err
		}
		if _, err := writer.Write(jsonData); err != nil {
			return err
		}
		if _, err := writer.WriteString("\n"); err != nil {
			return err
		}
	}

	if err := writer.Flush(); err != nil {
		return err
	}

	if err := tempFile.Sync(); err != nil {
		return err
	}

	if err := tempFile.Close(); err != nil {
		return err
	}

	if err := os.Rename(tempPath, trashPath); err != nil {
		return err
	}

	return nil
}
