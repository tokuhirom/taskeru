package internal

import (
	"testing"
	"time"
)

func TestFilterTasksByProject(t *testing.T) {
	tasks := []Task{
		{ID: "1", Title: "Task 1", Projects: []string{"work", "urgent"}},
		{ID: "2", Title: "Task 2", Projects: []string{"personal"}},
		{ID: "3", Title: "Task 3", Projects: []string{"work"}},
		{ID: "4", Title: "Task 4", Projects: []string{}},
	}

	tests := []struct {
		name        string
		project     string
		expectedIDs []string
	}{
		{
			name:        "filter by work project",
			project:     "work",
			expectedIDs: []string{"1", "3"},
		},
		{
			name:        "filter by personal project",
			project:     "personal",
			expectedIDs: []string{"2"},
		},
		{
			name:        "filter by urgent project",
			project:     "urgent",
			expectedIDs: []string{"1"},
		},
		{
			name:        "filter by non-existent project",
			project:     "nonexistent",
			expectedIDs: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := FilterTasksByProject(tasks, tt.project)

			if len(filtered) != len(tt.expectedIDs) {
				t.Errorf("FilterTasksByProject() returned %d tasks, want %d", len(filtered), len(tt.expectedIDs))
				return
			}

			for i, task := range filtered {
				if task.ID != tt.expectedIDs[i] {
					t.Errorf("FilterTasksByProject() task[%d].ID = %v, want %v", i, task.ID, tt.expectedIDs[i])
				}
			}
		})
	}
}

func TestGetAllProjects(t *testing.T) {
	tasks := []Task{
		{ID: "1", Title: "Task 1", Projects: []string{"work", "urgent"}},
		{ID: "2", Title: "Task 2", Projects: []string{"personal"}},
		{ID: "3", Title: "Task 3", Projects: []string{"work"}},
		{ID: "4", Title: "Task 4", Projects: []string{}},
		{ID: "5", Title: "Task 5", Projects: []string{"personal", "hobby"}},
	}

	projects := GetAllProjects(tasks)

	// Create a map for easier checking
	projectMap := make(map[string]bool)
	for _, p := range projects {
		projectMap[p] = true
	}

	expectedProjects := []string{"work", "urgent", "personal", "hobby"}

	if len(projects) != len(expectedProjects) {
		t.Errorf("GetAllProjects() returned %d projects, want %d", len(projects), len(expectedProjects))
	}

	for _, expected := range expectedProjects {
		if !projectMap[expected] {
			t.Errorf("GetAllProjects() missing project %v", expected)
		}
	}

	// Check for no duplicates
	seen := make(map[string]bool)
	for _, p := range projects {
		if seen[p] {
			t.Errorf("GetAllProjects() contains duplicate project %v", p)
		}
		seen[p] = true
	}
}

func TestTaskStatusMethods(t *testing.T) {
	t.Run("GetAllStatuses", func(t *testing.T) {
		statuses := GetAllStatuses()
		expected := []string{StatusTODO, StatusDOING, StatusWAITING, StatusDONE, StatusWONTDO}

		if len(statuses) != len(expected) {
			t.Errorf("GetAllStatuses() returned %d statuses, want %d", len(statuses), len(expected))
		}

		for i, status := range statuses {
			if status != expected[i] {
				t.Errorf("GetAllStatuses()[%d] = %v, want %v", i, status, expected[i])
			}
		}
	})

	t.Run("SetStatus with valid statuses", func(t *testing.T) {
		task := NewTask("Test task")

		// Test each valid status
		for _, status := range GetAllStatuses() {
			task.SetStatus(status)
			if task.Status != status {
				t.Errorf("SetStatus(%v) failed, got %v", status, task.Status)
			}
		}
	})

	t.Run("SetStatus with invalid status", func(t *testing.T) {
		task := NewTask("Test task")
		initialStatus := task.Status

		task.SetStatus("invalid_status")
		if task.Status != initialStatus {
			t.Errorf("SetStatus with invalid status changed task status to %v", task.Status)
		}
	})

	t.Run("CompletedAt handling", func(t *testing.T) {
		task := NewTask("Test task")

		// Mark as DONE
		task.SetStatus(StatusDONE)
		if task.CompletedAt == nil {
			t.Error("CompletedAt should be set when status becomes DONE")
		}

		// Change to DOING
		task.SetStatus(StatusDOING)
		if task.CompletedAt != nil {
			t.Error("CompletedAt should be cleared when status changes from DONE")
		}

		// Mark as WONTDO
		task.SetStatus(StatusWONTDO)
		if task.CompletedAt == nil {
			t.Error("CompletedAt should be set when status becomes WONTDO")
		}

		// Change back to TODO
		task.SetStatus(StatusTODO)
		if task.CompletedAt != nil {
			t.Error("CompletedAt should be cleared when status changes from WONTDO")
		}
	})

	t.Run("DisplayStatus returns status name", func(t *testing.T) {
		tests := []string{StatusTODO, StatusDOING, StatusWAITING, StatusDONE, StatusWONTDO}

		for _, status := range tests {
			task := NewTask("Test")
			task.Status = status
			display := task.DisplayStatus()
			if display != status {
				t.Errorf("DisplayStatus() for %v = %v, want %v", status, display, status)
			}
		}
	})
}

func TestSortTasks(t *testing.T) {
	now := time.Now()
	testCases := []struct {
		name      string
		tasks     []Task
		expectID0 string
		expectID1 string
	}{
		{
			name: "Active tasks come before completed tasks",
			tasks: []Task{
				{ID: "1", Status: StatusTODO, Priority: "A", Updated: now},
				{ID: "2", Status: StatusDONE, Priority: "A", Updated: now},
			},
			expectID0: "1", expectID1: "2",
		},
		{
			name: "Priority A comes before Priority B (active)",
			tasks: []Task{
				{ID: "1", Status: StatusTODO, Priority: "B", Updated: now},
				{ID: "2", Status: StatusTODO, Priority: "A", Updated: now},
			},
			expectID0: "2", expectID1: "1",
		},
		{
			name: "No priority sorts between C and D (active)",
			tasks: []Task{
				{ID: "1", Status: StatusTODO, Priority: "", Updated: now},
				{ID: "2", Status: StatusTODO, Priority: "C", Updated: now},
			},
			expectID0: "2", expectID1: "1",
		},
		{
			name: "Updated: newer comes first if priority and status are same (active)",
			tasks: []Task{
				{ID: "a", Status: StatusTODO, Priority: "A", Updated: now.Add(-1 * time.Hour)},
				{ID: "b", Status: StatusTODO, Priority: "A", Updated: now.Add(-2 * time.Hour)},
			},
			expectID0: "a", expectID1: "b",
		},
		{
			name: "ID: larger ID comes first if all else is equal (active)",
			tasks: []Task{
				{ID: "z", Status: StatusTODO, Priority: "A", Updated: now},
				{ID: "y", Status: StatusTODO, Priority: "A", Updated: now},
			},
			expectID0: "z", expectID1: "y",
		},
		{
			name: "Completed: larger ID comes first",
			tasks: []Task{
				{ID: "z", Status: StatusDONE, Priority: "A", Updated: now},
				{ID: "y", Status: StatusDONE, Priority: "A", Updated: now},
			},
			expectID0: "z", expectID1: "y",
		},
		{
			name: "Active: priority, updated, then ID (all different)",
			tasks: []Task{
				{ID: "1", Status: StatusTODO, Priority: "B", Updated: now.Add(-2 * time.Hour)},
				{ID: "2", Status: StatusTODO, Priority: "A", Updated: now.Add(-1 * time.Hour)},
			},
			expectID0: "2", expectID1: "1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tasks := make([]Task, len(tc.tasks))
			copy(tasks, tc.tasks)
			SortTasks(tasks)
			if tasks[0].ID != tc.expectID0 || tasks[1].ID != tc.expectID1 {
				t.Errorf("SortTasks() order = [%v, %v], want [%v, %v]", tasks[0].ID, tasks[1].ID, tc.expectID0, tc.expectID1)
			}
		})
	}
}
