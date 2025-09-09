package internal

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetTrashFilePath(t *testing.T) {
	tests := []struct {
		name         string
		setPath      string
		want         string
		wantContains string
	}{
		{
			name:         "default trash Path when not set",
			setPath:      "",
			wantContains: "trash.json",
		},
		{
			name:    "creates trash file in same directory as task file",
			setPath: "/tmp/tasks.json",
			want:    "/tmp/tasks.trash.json",
		},
		{
			name:    "handles files without extension",
			setPath: "/tmp/todofile",
			want:    "/tmp/todofile.trash",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			taskFile := NewTaskFileWithPath(tt.setPath)
			got := taskFile.getTrashFilePath()

			if tt.want != "" {
				if got != tt.want {
					t.Errorf("getTrashFilePath() = %v, want %v", got, tt.want)
				}
			}

			if tt.wantContains != "" {
				if !contains(got, tt.wantContains) {
					t.Errorf("getTrashFilePath() = %v, should contain %v", got, tt.wantContains)
				}
			}
		})
	}
}

func TestSaveAndLoadTasks(t *testing.T) {
	taskFile := NewTaskFileForTesting(t)

	// Create test tasks
	tasks := []Task{
		*NewTask("Test task 1"),
		*NewTask("Test task 2"),
	}

	tasks[0].Projects = []string{"work", "urgent"}
	tasks[1].Projects = []string{"personal"}
	tasks[1].SetStatus(StatusDONE)

	// Save tasks
	err := taskFile.AddTasks(tasks)
	if err != nil {
		t.Fatalf("SaveTasks() error = %v", err)
	}

	// Check file exists
	if _, err := os.Stat(taskFile.Path); os.IsNotExist(err) {
		t.Fatalf("Task file was not created: %v", taskFile.Path)
	}

	// Load tasks back
	loadedTasks, err := taskFile.LoadTasks()
	require.NoError(t, err, "LoadTasks()")

	// Verify tasks
	if len(loadedTasks) != len(tasks) {
		t.Errorf("LoadTasks() returned %d tasks, want %d", len(loadedTasks), len(tasks))
	}

	for i, task := range loadedTasks {
		if task.Title != tasks[i].Title {
			t.Errorf("Task %d title = %v, want %v", i, task.Title, tasks[i].Title)
		}
		if task.Status != tasks[i].Status {
			t.Errorf("Task %d status = %v, want %v", i, task.Status, tasks[i].Status)
		}
		if len(task.Projects) != len(tasks[i].Projects) {
			t.Errorf("Task %d projects count = %v, want %v", i, len(task.Projects), len(tasks[i].Projects))
		}
	}
}

func TestLoadTasksWithInvalidJSON(t *testing.T) {
	// Use temporary directory for test files
	taskFile := NewTaskFileForTesting(t)

	// Set the test file Path

	// Create file with mixed valid and invalid JSON
	content := `{"id":"1","title":"Valid task","status":"TODO"}
invalid json line
{"id":"2","title":"Another valid task","status":"TODO"}
`
	err := os.WriteFile(taskFile.Path, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Load tasks - should skip invalid lines
	tasks, err := taskFile.LoadTasks()
	require.NoError(t, err, "LoadTasks()")

	// Should load only valid tasks
	require.Equal(t, 2, len(tasks))
}

func TestSaveDeletedTasksToTrash(t *testing.T) {
	taskFile := NewTaskFileForTesting(t)
	t.Logf("Task file: %s", taskFile.Path)

	// Create deleted tasks
	deletedTasks := []Task{
		*NewTask("Deleted task 1"),
		*NewTask("Deleted task 2"),
	}

	// Save to trash
	err := taskFile.SaveDeletedTasksToTrash(deletedTasks)
	if err != nil {
		t.Fatalf("SaveDeletedTasksToTrash() error = %v", err)
	}

	// Check trash file exists
	trashFile := taskFile.getTrashFilePath()
	if _, err := os.Stat(trashFile); os.IsNotExist(err) {
		t.Fatalf("Trash file was not created: %v", trashFile)
	}

	// Save more deleted tasks
	moreTasks := []Task{
		*NewTask("Deleted task 3"),
	}

	err = taskFile.SaveDeletedTasksToTrash(moreTasks)
	require.NoError(t, err, "SaveDeletedTasksToTrash()")

	// Verify a trash file contains all deleted tasks
	// Note: We can't easily verify the contents without exposing a load function for trash,
	// but we can at least verify the file exists and grows
	info, err := os.Stat(trashFile)
	if err != nil {
		t.Fatalf("Failed to stat trash file: %v", err)
	}

	if info.Size() == 0 {
		t.Errorf("Trash file is empty, should contain deleted tasks")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && filepath.Base(s) == substr
}
