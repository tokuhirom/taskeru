package internal

import (
	"reflect"
	"testing"
)

func TestExtractProjectsFromTitle(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		expectedTitle   string
		expectedProjects []string
	}{
		{
			name:            "no projects",
			input:           "simple task without projects",
			expectedTitle:   "simple task without projects",
			expectedProjects: []string{},
		},
		{
			name:            "single project at end",
			input:           "task with project +work",
			expectedTitle:   "task with project",
			expectedProjects: []string{"work"},
		},
		{
			name:            "multiple projects at end",
			input:           "urgent task +work +urgent",
			expectedTitle:   "urgent task",
			expectedProjects: []string{"work", "urgent"},
		},
		{
			name:            "project in middle should not be extracted",
			input:           "+work in the middle task",
			expectedTitle:   "+work in the middle task",
			expectedProjects: []string{},
		},
		{
			name:            "ctrl+h pattern should not extract +h",
			input:           "ctrl+hに対応する +prj",
			expectedTitle:   "ctrl+hに対応する",
			expectedProjects: []string{"prj"},
		},
		{
			name:            "mixed: project in middle and at end",
			input:           "途中に+tagがあって最後に +final",
			expectedTitle:   "途中に+tagがあって最後に",
			expectedProjects: []string{"final"},
		},
		{
			name:            "multiple projects with different spacing",
			input:           "task   +proj1  +proj2   ",
			expectedTitle:   "task",
			expectedProjects: []string{"proj1", "proj2"},
		},
		{
			name:            "project with underscore and hyphen",
			input:           "task +my_project +another-project",
			expectedTitle:   "task",
			expectedProjects: []string{"my_project", "another-project"},
		},
		{
			name:            "project with numbers",
			input:           "task +project123 +2025",
			expectedTitle:   "task",
			expectedProjects: []string{"project123", "2025"},
		},
		{
			name:            "only projects",
			input:           "+work +urgent",
			expectedTitle:   "",
			expectedProjects: []string{"work", "urgent"},
		},
		{
			name:            "plus sign without space is not a project",
			input:           "task+notaproject +realproject",
			expectedTitle:   "task+notaproject",
			expectedProjects: []string{"realproject"},
		},
		{
			name:            "Japanese text with projects",
			input:           "バグ修正 +work +緊急",
			expectedTitle:   "バグ修正",
			expectedProjects: []string{"work", "緊急"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTitle, gotProjects := ExtractProjectsFromTitle(tt.input)
			
			if gotTitle != tt.expectedTitle {
				t.Errorf("ExtractProjectsFromTitle() title = %v, want %v", gotTitle, tt.expectedTitle)
			}
			
			// Handle nil vs empty slice comparison
			if len(gotProjects) == 0 && len(tt.expectedProjects) == 0 {
				// Both are empty, that's fine
			} else if !reflect.DeepEqual(gotProjects, tt.expectedProjects) {
				t.Errorf("ExtractProjectsFromTitle() projects = %v, want %v", gotProjects, tt.expectedProjects)
			}
		})
	}
}

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