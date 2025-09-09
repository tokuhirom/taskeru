package internal

import (
	"reflect"
	"testing"
)

func TestExtractProjectsFromTitle(t *testing.T) {
	tests := []struct {
		name             string
		input            string
		expectedTitle    string
		expectedProjects []string
	}{
		{
			name:             "no projects",
			input:            "simple task without projects",
			expectedTitle:    "simple task without projects",
			expectedProjects: []string{},
		},
		{
			name:             "single project at end",
			input:            "task with project +work",
			expectedTitle:    "task with project",
			expectedProjects: []string{"work"},
		},
		{
			name:             "multiple projects at end",
			input:            "urgent task +work +urgent",
			expectedTitle:    "urgent task",
			expectedProjects: []string{"work", "urgent"},
		},
		{
			name:             "project in middle should not be extracted",
			input:            "+work in the middle task",
			expectedTitle:    "+work in the middle task",
			expectedProjects: []string{},
		},
		{
			name:             "ctrl+h pattern should not extract +h",
			input:            "ctrl+hに対応する +prj",
			expectedTitle:    "ctrl+hに対応する",
			expectedProjects: []string{"prj"},
		},
		{
			name:             "mixed: project in middle and at end",
			input:            "途中に+tagがあって最後に +final",
			expectedTitle:    "途中に+tagがあって最後に",
			expectedProjects: []string{"final"},
		},
		{
			name:             "multiple projects with different spacing",
			input:            "task   +proj1  +proj2   ",
			expectedTitle:    "task",
			expectedProjects: []string{"proj1", "proj2"},
		},
		{
			name:             "project with underscore and hyphen",
			input:            "task +my_project +another-project",
			expectedTitle:    "task",
			expectedProjects: []string{"my_project", "another-project"},
		},
		{
			name:             "project with numbers",
			input:            "task +project123 +2025",
			expectedTitle:    "task",
			expectedProjects: []string{"project123", "2025"},
		},
		{
			name:             "only projects",
			input:            "+work +urgent",
			expectedTitle:    "",
			expectedProjects: []string{"work", "urgent"},
		},
		{
			name:             "plus sign without space is not a project",
			input:            "task+notaproject +realproject",
			expectedTitle:    "task+notaproject",
			expectedProjects: []string{"realproject"},
		},
		{
			name:             "Japanese text with projects",
			input:            "バグ修正 +work +緊急",
			expectedTitle:    "バグ修正",
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

func TestCombinedNaturalLanguageDateExtraction(t *testing.T) {
	// Test combining both scheduled and due dates with natural language
	input := "Complex task scheduled:next monday due:next friday +work +urgent"

	// Extract scheduled date first
	cleanTitle, scheduled := ExtractScheduledDateFromTitle(input)
	t.Logf("After scheduled extraction: %q", cleanTitle)
	if scheduled == nil {
		t.Error("Failed to extract scheduled date")
	}

	// Then extract deadline
	cleanTitle, deadline := ExtractDeadlineFromTitle(cleanTitle)
	t.Logf("After deadline extraction: %q", cleanTitle)
	if deadline == nil {
		t.Error("Failed to extract deadline")
	}

	// Then extract projects
	cleanTitle, projects := ExtractProjectsFromTitle(cleanTitle)
	t.Logf("After projects extraction: %q, projects=%v", cleanTitle, projects)

	if cleanTitle != "Complex task" {
		t.Errorf("Expected title 'Complex task', got %q", cleanTitle)
	}

	if len(projects) != 2 || projects[0] != "work" || projects[1] != "urgent" {
		t.Errorf("Expected projects [work, urgent], got %v", projects)
	}
}
