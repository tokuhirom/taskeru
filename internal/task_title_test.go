package internal

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseTitle(t *testing.T) {
	tests := []struct {
		input            string
		expectedTitle    string
		expectedProjects []string
	}{
		{
			input:            "Finish report +work +urgent",
			expectedTitle:    "Finish report",
			expectedProjects: []string{"work", "urgent"},
		},
		{
			input:            "Buy groceries +personal",
			expectedTitle:    "Buy groceries",
			expectedProjects: []string{"personal"},
		},
		{
			input:            "Simple task without projects",
			expectedTitle:    "Simple task without projects",
			expectedProjects: []string{},
		},
		{
			input:            "Task with project in middle +work and more text",
			expectedTitle:    "Task with project in middle +work and more text",
			expectedProjects: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			parsed := ParseTitle(tt.input)

			require.Equal(t, tt.expectedTitle, parsed.Title)
			require.Equal(t, tt.expectedProjects, parsed.Projects)
		})
	}
}

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

func TestExtractDeadlineFromTitleWithNaturalLanguage(t *testing.T) {
	tests := []struct {
		input         string
		expectedTitle string
		hasDeadline   bool
		description   string
	}{
		// Natural language dates
		{
			input:         "Finish report due:next tuesday",
			expectedTitle: "Finish report",
			hasDeadline:   true,
			description:   "Should extract 'next tuesday' deadline",
		},
		{
			input:         "Review PR due:in 2 days +work",
			expectedTitle: "Review PR +work",
			hasDeadline:   true,
			description:   "Should extract 'in 2 days' with project tag",
		},
		{
			input:         "Meeting prep due:tomorrow at 3pm +urgent",
			expectedTitle: "Meeting prep +urgent",
			hasDeadline:   true,
			description:   "Should extract 'tomorrow at 3pm' with project",
		},
		{
			input:         "Task due:next friday +project1 +project2",
			expectedTitle: "Task +project1 +project2",
			hasDeadline:   true,
			description:   "Should handle natural date with multiple projects",
		},

		// Traditional formats (backward compatibility)
		{
			input:         "Buy groceries due:today",
			expectedTitle: "Buy groceries",
			hasDeadline:   true,
			description:   "Should still work with simple 'today'",
		},
		{
			input:         "Write report due:2024-12-31",
			expectedTitle: "Write report",
			hasDeadline:   true,
			description:   "Should still work with ISO date",
		},
		{
			input:         "Task without deadline",
			expectedTitle: "Task without deadline",
			hasDeadline:   false,
			description:   "Should return unchanged when no deadline",
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			cleanTitle, deadline := ExtractDeadlineFromTitle(tt.input)

			if cleanTitle != tt.expectedTitle {
				t.Errorf("Expected title %q, got %q", tt.expectedTitle, cleanTitle)
			}

			if tt.hasDeadline && deadline == nil {
				t.Error("Expected deadline to be extracted, but got nil")
			}

			if !tt.hasDeadline && deadline != nil {
				t.Errorf("Expected no deadline, but got %v", deadline)
			}
		})
	}
}

func TestExtractScheduledDateFromTitleWithNaturalLanguage(t *testing.T) {
	tests := []struct {
		input         string
		expectedTitle string
		hasScheduled  bool
		description   string
	}{
		// Natural language dates
		{
			input:         "Start project scheduled:next monday",
			expectedTitle: "Start project",
			hasScheduled:  true,
			description:   "Should extract 'next monday' scheduled date",
		},
		{
			input:         "Begin work scheduled:in 1 week +important",
			expectedTitle: "Begin work +important",
			hasScheduled:  true,
			description:   "Should extract 'in 1 week' with project tag",
		},
		{
			input:         "Task scheduled:next month +work +planning",
			expectedTitle: "Task +work +planning",
			hasScheduled:  true,
			description:   "Should handle natural date with multiple projects",
		},

		// Traditional formats
		{
			input:         "Task scheduled:tomorrow",
			expectedTitle: "Task",
			hasScheduled:  true,
			description:   "Should work with simple 'tomorrow'",
		},
		{
			input:         "Task scheduled:2025-01-15",
			expectedTitle: "Task",
			hasScheduled:  true,
			description:   "Should work with ISO date",
		},
		{
			input:         "Task without scheduled date",
			expectedTitle: "Task without scheduled date",
			hasScheduled:  false,
			description:   "Should return unchanged when no scheduled date",
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			cleanTitle, scheduled := ExtractScheduledDateFromTitle(tt.input)

			if cleanTitle != tt.expectedTitle {
				t.Errorf("Expected title %q, got %q", tt.expectedTitle, cleanTitle)
			}

			if tt.hasScheduled && scheduled == nil {
				t.Error("Expected scheduled date to be extracted, but got nil")
			}

			if !tt.hasScheduled && scheduled != nil {
				t.Errorf("Expected no scheduled date, but got %v", scheduled)
			}

			// Verify scheduled dates are set to start of day
			if scheduled != nil {
				hour := scheduled.Hour()
				if hour != 0 {
					t.Errorf("Expected hour to be 0 (start of day) for scheduled date, got %d", hour)
				}
			}
		})
	}
}
