package internal

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func GetTaskFilePath() string {
	if path := os.Getenv("TASKERU_FILE"); path != "" {
		return filepath.Clean(path)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "todo.json"
	}
	return filepath.Join(home, "todo.json")
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
	defer file.Close()

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
		tempFile.Close()
		os.Remove(tempPath)
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
		return err
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