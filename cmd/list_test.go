package cmd

import (
	"bytes"
	"io"
	"os"
	"testing"

	"taskeru/internal"
)

func TestListCommandWithProjectFilter(t *testing.T) {
	taskFile := internal.NewTaskFileForTesting(t)

	// Create test tasks with different projects
	tasks := []internal.Task{
		*internal.NewTask("Work task 1"),
		*internal.NewTask("Personal task 1"),
		*internal.NewTask("Work task 2"),
		*internal.NewTask("Task without project"),
		*internal.NewTask("Personal task 2"),
	}

	// Set projects
	tasks[0].Projects = []string{"work"}
	tasks[1].Projects = []string{"personal"}
	tasks[2].Projects = []string{"work", "urgent"}
	// tasks[3] has no projects
	tasks[4].Projects = []string{"personal", "home"}

	// Save tasks
	err := taskFile.AddTasks(tasks)
	if err != nil {
		t.Fatalf("Failed to save test tasks: %v", err)
	}

	tests := []struct {
		name          string
		projectFilter string
		expectedCount int
		expectedTasks []string
	}{
		{
			name:          "No filter shows all tasks",
			projectFilter: "",
			expectedCount: 5,
			expectedTasks: []string{"Work task 1", "Personal task 1", "Work task 2", "Task without project", "Personal task 2"},
		},
		{
			name:          "Filter by work project",
			projectFilter: "work",
			expectedCount: 2,
			expectedTasks: []string{"Work task 1", "Work task 2"},
		},
		{
			name:          "Filter by personal project",
			projectFilter: "personal",
			expectedCount: 2,
			expectedTasks: []string{"Personal task 1", "Personal task 2"},
		},
		{
			name:          "Filter by urgent project",
			projectFilter: "urgent",
			expectedCount: 1,
			expectedTasks: []string{"Work task 2"},
		},
		{
			name:          "Filter by non-existent project",
			projectFilter: "nonexistent",
			expectedCount: 0,
			expectedTasks: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Run list command with filter
			err := ListCommand(taskFile, tt.projectFilter)
			if err != nil {
				t.Errorf("ListCommand() error = %v", err)
			}

			// Restore stdout
			_ = w.Close()
			os.Stdout = oldStdout

			// Read captured output
			var buf bytes.Buffer
			_, _ = io.Copy(&buf, r)
			output := buf.String()

			// Check if expected tasks are in output
			for _, expectedTask := range tt.expectedTasks {
				if !contains(output, expectedTask) {
					t.Errorf("Expected task '%s' not found in output", expectedTask)
				}
			}

			// Check that unexpected tasks are not in output
			allTasks := []string{"Work task 1", "Personal task 1", "Work task 2", "Task without project", "Personal task 2"}
			for _, task := range allTasks {
				shouldContain := false
				for _, expected := range tt.expectedTasks {
					if task == expected {
						shouldContain = true
						break
					}
				}
				if !shouldContain && contains(output, task) {
					t.Errorf("Unexpected task '%s' found in output when filtering by '%s'", task, tt.projectFilter)
				}
			}

			// Check for filter message
			if tt.projectFilter != "" {
				if tt.expectedCount == 0 {
					expectedMsg := "No tasks found for project: " + tt.projectFilter
					if !contains(output, expectedMsg) {
						t.Errorf("Expected message '%s' not found in output\nActual output:\n%s", expectedMsg, output)
					}
				} else {
					// Check for the project name with or without color codes
					// The actual output includes ANSI color codes like: Tasks for project: [38;5;208m+work[0m
					expectedMsg := "+" + tt.projectFilter
					if !contains(output, expectedMsg) || !contains(output, "Tasks for project:") {
						t.Errorf("Expected message 'Tasks for project: %s' not found in output\nActual output:\n%s", tt.projectFilter, output)
					}
				}
			}
		})
	}
}

func TestListCommandWithCompletedTasksAndProjectFilter(t *testing.T) {
	taskFile := internal.NewTaskFileForTesting(t)

	// Create test tasks with different projects and statuses
	tasks := []internal.Task{
		*internal.NewTask("Active work task"),
		*internal.NewTask("Completed work task"),
		*internal.NewTask("Active personal task"),
		*internal.NewTask("Old completed work task"),
	}

	// Set projects and statuses
	tasks[0].Projects = []string{"work"}
	tasks[1].Projects = []string{"work"}
	tasks[1].SetStatus(internal.StatusDONE)
	tasks[2].Projects = []string{"personal"}
	tasks[3].Projects = []string{"work"}
	tasks[3].SetStatus(internal.StatusDONE)
	// Make task 3 old by setting completed time to 2 days ago
	oldTime := tasks[3].CompletedAt.AddDate(0, 0, -2)
	tasks[3].CompletedAt = &oldTime

	// Save tasks
	err := taskFile.AddTasks(tasks)
	if err != nil {
		t.Fatalf("Failed to save test tasks: %v", err)
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run list command with work filter
	err = ListCommand(taskFile, "work")
	if err != nil {
		t.Errorf("ListCommand() error = %v", err)
	}

	// Restore stdout
	_ = w.Close()
	os.Stdout = oldStdout

	// Read captured output
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	// Should show active and recently completed work tasks, but not old completed ones
	if !contains(output, "Active work task") {
		t.Error("Expected 'Active work task' in output")
	}
	if !contains(output, "Completed work task") {
		t.Error("Expected 'Completed work task' in output")
	}
	if contains(output, "Old completed work task") {
		t.Error("Should not show 'Old completed work task' in output")
	}
	if contains(output, "Active personal task") {
		t.Error("Should not show 'Active personal task' when filtering by work")
	}

	// Check for hidden tasks message (when filtering, it's shown as ", X hidden" in the title)
	if !contains(output, "hidden") {
		t.Error("Expected hidden tasks message in output")
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && bytes.Contains([]byte(s), []byte(substr))
}
