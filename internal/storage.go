package internal

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gofrs/flock"
)

type TaskFile struct {
	Path string
}

func NewTaskFileForTesting(t *testing.T) *TaskFile {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "todo.json")
	return &TaskFile{
		Path: filePath,
	}
}

func NewTaskFileWithPath(path string) *TaskFile {
	return &TaskFile{
		Path: path,
	}
}

func NewTaskFile() *TaskFile {
	filePath := func() string {
		home, err := os.UserHomeDir()
		if err != nil {
			slog.Error("failed to get user home directory",
				slog.Any("error", err))
			// Fallback to the current directory
			return "todo.json"
		}
		return filepath.Join(home, "todo.json")
	}()

	return &TaskFile{
		Path: filePath,
	}
}

func (tf *TaskFile) getTrashFilePath() string {
	// We should not expose this method, right?
	if tf.Path != "" {
		dir := filepath.Dir(tf.Path)
		base := filepath.Base(tf.Path)
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

func (tf *TaskFile) LoadTasks() ([]Task, error) {
	file, err := os.Open(tf.Path)
	if err != nil {
		if os.IsNotExist(err) {
			slog.Info("No task file found, return empty tasks",
				slog.String("Path", tf.Path))
			return []Task{}, nil
		}

		slog.Info("Failed to open task file",
			slog.String("Path", tf.Path),
			slog.Any("error", err))
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
			slog.Error("Failed to unmarshal task",
				slog.Int("line", lineNum),
				slog.String("line_content", line),
				slog.Any("error", err))
			continue
		}

		tasks = append(tasks, task)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read tasks: %w", err)
	}

	return tasks, nil
}

func (tf *TaskFile) saveTasks(tasks []Task) error {
	tempFile, err := os.CreateTemp(filepath.Dir(tf.Path), ".taskeru-*.tmp")
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

	if err := os.Rename(tempPath, tf.Path); err != nil {
		return err
	}

	return nil
}

func (tf *TaskFile) lock() (*flock.Flock, error) {
	lock := flock.New(tf.Path + ".lock")
	if err := lock.Lock(); err != nil {
		return nil, fmt.Errorf("failed to lock task file: %w", err)
	}
	return lock, nil
}

func (tf *TaskFile) AddTask(task *Task) error {
	return tf.AddTasks([]Task{*task})
}

func (tf *TaskFile) AddTasks(newTasks []Task) error {
	lock, err := tf.lock()
	if err != nil {
		return fmt.Errorf("failed to lock task file: %w", err)
	}
	defer func() { _ = lock.Unlock() }()

	tasks, err := tf.LoadTasks()
	if err != nil {
		return fmt.Errorf("failed to load tasks: %w", err)
	}

	tasks = append(tasks, newTasks...)
	return tf.saveTasks(tasks)
}

func (tf *TaskFile) UpdateTaskWithConflictCheck(taskID string, originalUpdated time.Time, updateFunc func(*Task)) error {
	lock, err := tf.lock()
	if err != nil {
		return fmt.Errorf("failed to lock task file: %w", err)
	}
	defer func() { _ = lock.Unlock() }()

	tasks, err := tf.LoadTasks()
	if err != nil {
		return err
	}

	found := false
	for i := range tasks {
		if tasks[i].ID == taskID {
			// Check if the task has been updated since we loaded it
			if !tasks[i].Updated.Equal(originalUpdated) {
				return fmt.Errorf("task has been modified by another process(%v != %v)",
					tasks[i].Updated, originalUpdated)
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

	return tf.saveTasks(tasks)
}

// saveDeletedTasksToTrash saves deleted tasks to trash.json
func (tf *TaskFile) saveDeletedTasksToTrash(deletedTasks []Task) error {
	if len(deletedTasks) == 0 {
		return nil
	}

	trashPath := tf.getTrashFilePath()

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

func (tf *TaskFile) DeleteTask(taskID string) error {
	lock, err := tf.lock()
	if err != nil {
		return fmt.Errorf("failed to lock task file: %w", err)
	}
	defer func() { _ = lock.Unlock() }()

	tasks, err := tf.LoadTasks()
	if err != nil {
		return fmt.Errorf("failed to load tasks: %w", err)
	}

	var (
		remaining []Task
		deleted   []Task
	)
	for _, task := range tasks {
		if task.ID == taskID {
			deleted = append(deleted, task)
		} else {
			remaining = append(remaining, task)
		}
	}

	if len(deleted) == 0 {
		return fmt.Errorf("task with ID %s not found", taskID)
	}

	if err := tf.saveDeletedTasksToTrash(deleted); err != nil {
		return fmt.Errorf("failed to save deleted tasks to trash: %w", err)
	}

	if err := tf.saveTasks(remaining); err != nil {
		return fmt.Errorf("failed to save tasks: %w", err)
	}

	return nil
}
