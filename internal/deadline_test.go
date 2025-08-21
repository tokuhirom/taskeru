package internal

import (
	"testing"
	"time"
)

func TestExtractDeadlineFromTitle(t *testing.T) {
	tests := []struct {
		input         string
		expectedTitle string
		hasDeadline   bool
		description   string
	}{
		{
			input:         "Buy groceries due:today",
			expectedTitle: "Buy groceries",
			hasDeadline:   true,
			description:   "Should extract 'today' deadline",
		},
		{
			input:         "Write report due:tomorrow",
			expectedTitle: "Write report",
			hasDeadline:   true,
			description:   "Should extract 'tomorrow' deadline",
		},
		{
			input:         "Meeting due:2024-12-31",
			expectedTitle: "Meeting",
			hasDeadline:   true,
			description:   "Should extract specific date",
		},
		{
			input:         "Task with spaces due:monday",
			expectedTitle: "Task with spaces",
			hasDeadline:   true,
			description:   "Should extract weekday deadline",
		},
		{
			input:         "Task due:12-25",
			expectedTitle: "Task",
			hasDeadline:   true,
			description:   "Should extract MM-DD format",
		},
		{
			input:         "Task due:12/25",
			expectedTitle: "Task",
			hasDeadline:   true,
			description:   "Should extract MM/DD format",
		},
		{
			input:         "Task without deadline",
			expectedTitle: "Task without deadline",
			hasDeadline:   false,
			description:   "Should return unchanged when no deadline",
		},
		{
			input:         "Task due:invalid",
			expectedTitle: "Task due:invalid",
			hasDeadline:   false,
			description:   "Should return unchanged for invalid deadline",
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

func TestExtractDeadlineRelativeDates(t *testing.T) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())
	
	tests := []struct {
		input    string
		expected time.Time
	}{
		{"Task due:today", today},
		{"Task due:tomorrow", today.AddDate(0, 0, 1)},
		{"Task due:mon", nextWeekday(today, time.Monday)},
		{"Task due:tuesday", nextWeekday(today, time.Tuesday)},
		{"Task due:wed", nextWeekday(today, time.Wednesday)},
		{"Task due:thursday", nextWeekday(today, time.Thursday)},
		{"Task due:fri", nextWeekday(today, time.Friday)},
		{"Task due:saturday", nextWeekday(today, time.Saturday)},
		{"Task due:sun", nextWeekday(today, time.Sunday)},
	}
	
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			_, deadline := ExtractDeadlineFromTitle(tt.input)
			
			if deadline == nil {
				t.Fatal("Expected deadline to be extracted")
			}
			
			// Compare dates (ignore time differences within the same day)
			if deadline.Year() != tt.expected.Year() ||
				deadline.Month() != tt.expected.Month() ||
				deadline.Day() != tt.expected.Day() {
				t.Errorf("Expected deadline %v, got %v", tt.expected, *deadline)
			}
		})
	}
}

func TestExtractDeadlineWithProjects(t *testing.T) {
	input := "Task due:tomorrow +work +urgent"
	expectedTitle := "Task"
	
	// First extract deadline
	cleanTitle, deadline := ExtractDeadlineFromTitle(input)
	
	// Then extract projects
	cleanTitle, projects := ExtractProjectsFromTitle(cleanTitle)
	
	if cleanTitle != expectedTitle {
		t.Errorf("Expected title %q, got %q", expectedTitle, cleanTitle)
	}
	
	if deadline == nil {
		t.Error("Expected deadline to be extracted")
	}
	
	if len(projects) != 2 {
		t.Errorf("Expected 2 projects, got %d", len(projects))
	}
	
	if projects[0] != "work" || projects[1] != "urgent" {
		t.Errorf("Expected projects [work, urgent], got %v", projects)
	}
}

func TestNextWeekday(t *testing.T) {
	// Test from a Monday
	monday := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC) // 2024-01-01 is a Monday
	
	tests := []struct {
		from     time.Time
		weekday  time.Weekday
		expected time.Time
	}{
		{monday, time.Tuesday, monday.AddDate(0, 0, 1)},   // Next day
		{monday, time.Wednesday, monday.AddDate(0, 0, 2)}, // 2 days later
		{monday, time.Sunday, monday.AddDate(0, 0, 6)},    // 6 days later
		{monday, time.Monday, monday.AddDate(0, 0, 7)},    // Next Monday (7 days)
	}
	
	for _, tt := range tests {
		result := nextWeekday(tt.from, tt.weekday)
		if !result.Equal(tt.expected) {
			t.Errorf("nextWeekday(%v, %v) = %v, want %v",
				tt.from.Format("2006-01-02 Mon"),
				tt.weekday,
				result.Format("2006-01-02 Mon"),
				tt.expected.Format("2006-01-02 Mon"))
		}
	}
}