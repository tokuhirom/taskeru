package internal

import (
	"testing"
)

func TestParseNaturalDate(t *testing.T) {
	// Set a fixed "now" for predictable testing
	// Note: The actual implementation uses time.Now(), so results may vary
	// These tests verify that the parser accepts various formats

	tests := []struct {
		input       string
		shouldParse bool
		description string
	}{
		// Natural language relative dates
		{"next tuesday", true, "Should parse 'next tuesday'"},
		{"next week", true, "Should parse 'next week'"},
		{"in 3 days", true, "Should parse 'in 3 days'"},
		{"in 2 weeks", true, "Should parse 'in 2 weeks'"},
		{"tomorrow at 3pm", true, "Should parse 'tomorrow at 3pm'"},
		{"next friday at 5pm", true, "Should parse 'next friday at 5pm'"},
		{"in 1 month", true, "Should parse 'in 1 month'"},
		{"next month", true, "Should parse 'next month'"},

		// Traditional formats (fallback)
		{"today", true, "Should parse 'today'"},
		{"tomorrow", true, "Should parse 'tomorrow'"},
		{"monday", true, "Should parse 'monday'"},
		{"2024-12-31", true, "Should parse ISO date"},
		{"12/25", true, "Should parse MM/DD"},
		{"12-25", true, "Should parse MM-DD"},

		// Invalid inputs
		{"gibberish", false, "Should not parse gibberish"},
		{"", false, "Should not parse empty string"},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			result, _ := ParseNaturalDate(tt.input)

			if tt.shouldParse && result == nil {
				t.Errorf("Expected to parse '%s', but got nil", tt.input)
			}

			if !tt.shouldParse && result != nil {
				t.Errorf("Expected not to parse '%s', but got %v", tt.input, result)
			}

			// If parsed, verify it's set to end of day
			if result != nil {
				hour := result.Hour()
				if hour != 23 {
					t.Errorf("Expected hour to be 23 (end of day), got %d", hour)
				}
			}
		})
	}
}

func TestExtractDeadlineFromTitleV2WithNaturalLanguage(t *testing.T) {
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
			cleanTitle, deadline := ExtractDeadlineFromTitleV2(tt.input)

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

func TestExtractScheduledDateFromTitleV2WithNaturalLanguage(t *testing.T) {
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
			cleanTitle, scheduled := ExtractScheduledDateFromTitleV2(tt.input)

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

func TestCombinedNaturalLanguageDateExtraction(t *testing.T) {
	// Test combining both scheduled and due dates with natural language
	input := "Complex task scheduled:next monday due:next friday +work +urgent"

	// Extract scheduled date first
	cleanTitle, scheduled := ExtractScheduledDateFromTitleV2(input)
	t.Logf("After scheduled extraction: %q", cleanTitle)
	if scheduled == nil {
		t.Error("Failed to extract scheduled date")
	}

	// Then extract deadline
	cleanTitle, deadline := ExtractDeadlineFromTitleV2(cleanTitle)
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
