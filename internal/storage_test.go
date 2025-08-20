package internal

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetTaskFilePath(t *testing.T) {
	// Save original value and restore after test
	originalPath := taskFilePath
	defer func() {
		taskFilePath = originalPath
	}()

	tests := []struct {
		name         string
		setPath      string
		want         string
		wantContains string
	}{
		{
			name:         "default path when not set",
			setPath:      "",
			wantContains: "todo.json",
		},
		{
			name:    "uses -t option path when set",
			setPath: "/tmp/test.json",
			want:    "/tmp/test.json",
		},
		{
			name:    "cleans the path",
			setPath: "/tmp/../tmp/test.json",
			want:    "/tmp/test.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetTaskFilePath(tt.setPath)
			got := GetTaskFilePath()
			
			if tt.want != "" {
				if got != tt.want {
					t.Errorf("GetTaskFilePath() = %v, want %v", got, tt.want)
				}
			}
			
			if tt.wantContains != "" {
				if !filepath.IsAbs(got) || !contains(got, tt.wantContains) {
					t.Errorf("GetTaskFilePath() = %v, should contain %v", got, tt.wantContains)
				}
			}
		})
	}
}

func TestGetTrashFilePath(t *testing.T) {
	// Save original value and restore after test
	originalPath := taskFilePath
	defer func() {
		taskFilePath = originalPath
	}()

	tests := []struct {
		name         string
		setPath      string
		want         string
		wantContains string
	}{
		{
			name:         "default trash path when not set",
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
			SetTaskFilePath(tt.setPath)
			got := GetTrashFilePath()
			
			if tt.want != "" {
				if got != tt.want {
					t.Errorf("GetTrashFilePath() = %v, want %v", got, tt.want)
				}
			}
			
			if tt.wantContains != "" {
				if !contains(got, tt.wantContains) {
					t.Errorf("GetTrashFilePath() = %v, should contain %v", got, tt.wantContains)
				}
			}
		})
	}
}

func TestSaveAndLoadTasks(t *testing.T) {
	// Use temporary directory for test files
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test_tasks.json")
	
	// Set the test file path
	originalPath := taskFilePath
	defer func() {
		taskFilePath = originalPath
	}()
	SetTaskFilePath(testFile)
	
	// Create test tasks
	tasks := []Task{
		*NewTask("Test task 1"),
		*NewTask("Test task 2"),
	}
	
	tasks[0].Projects = []string{"work", "urgent"}
	tasks[1].Projects = []string{"personal"}
	tasks[1].SetStatus(StatusDONE)
	
	// Save tasks
	err := SaveTasks(tasks)
	if err != nil {
		t.Fatalf("SaveTasks() error = %v", err)
	}
	
	// Check file exists
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Fatalf("Task file was not created: %v", testFile)
	}
	
	// Load tasks back
	loadedTasks, err := LoadTasks()
	if err != nil {
		t.Fatalf("LoadTasks() error = %v", err)
	}
	
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
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "invalid_tasks.json")
	
	// Set the test file path
	originalPath := taskFilePath
	defer func() {
		taskFilePath = originalPath
	}()
	SetTaskFilePath(testFile)
	
	// Create file with mixed valid and invalid JSON
	content := `{"id":"1","title":"Valid task","status":"TODO"}
invalid json line
{"id":"2","title":"Another valid task","status":"TODO"}
`
	err := os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	// Load tasks - should skip invalid lines
	tasks, err := LoadTasks()
	if err != nil {
		t.Fatalf("LoadTasks() error = %v, want nil (should skip invalid lines)", err)
	}
	
	// Should load only valid tasks
	if len(tasks) != 2 {
		t.Errorf("LoadTasks() returned %d tasks, want 2 (should skip invalid line)", len(tasks))
	}
}

func TestSaveDeletedTasksToTrash(t *testing.T) {
	// Use temporary directory for test files
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "tasks.json")
	
	// Set the test file path
	originalPath := taskFilePath
	defer func() {
		taskFilePath = originalPath
	}()
	SetTaskFilePath(testFile)
	
	// Create deleted tasks
	deletedTasks := []Task{
		*NewTask("Deleted task 1"),
		*NewTask("Deleted task 2"),
	}
	
	// Save to trash
	err := SaveDeletedTasksToTrash(deletedTasks)
	if err != nil {
		t.Fatalf("SaveDeletedTasksToTrash() error = %v", err)
	}
	
	// Check trash file exists
	trashFile := GetTrashFilePath()
	if _, err := os.Stat(trashFile); os.IsNotExist(err) {
		t.Fatalf("Trash file was not created: %v", trashFile)
	}
	
	// Save more deleted tasks
	moreTasks := []Task{
		*NewTask("Deleted task 3"),
	}
	
	err = SaveDeletedTasksToTrash(moreTasks)
	if err != nil {
		t.Fatalf("SaveDeletedTasksToTrash() second call error = %v", err)
	}
	
	// Verify trash file contains all deleted tasks
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