package internal

import (
	"testing"
	"time"
)

func TestExtractScheduledDateFromTitle(t *testing.T) {
	tests := []struct {
		input         string
		expectedTitle string
		hasScheduled  bool
		description   string
	}{
		{
			input:         "Plan project scheduled:tomorrow",
			expectedTitle: "Plan project",
			hasScheduled:  true,
			description:   "Should extract 'tomorrow' scheduled date",
		},
		{
			input:         "Review code sched:monday",
			expectedTitle: "Review code",
			hasScheduled:  true,
			description:   "Should extract with 'sched:' shorthand",
		},
		{
			input:         "Meeting scheduled:2025-01-15",
			expectedTitle: "Meeting",
			hasScheduled:  true,
			description:   "Should extract specific date",
		},
		{
			input:         "Task scheduled:today",
			expectedTitle: "Task",
			hasScheduled:  true,
			description:   "Should extract 'today' scheduled date",
		},
		{
			input:         "Task without scheduled",
			expectedTitle: "Task without scheduled",
			hasScheduled:  false,
			description:   "Should return unchanged when no scheduled date",
		},
		{
			input:         "Task scheduled:invalid",
			expectedTitle: "Task scheduled:invalid",
			hasScheduled:  false,
			description:   "Should return unchanged for invalid scheduled date",
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
		})
	}
}

func TestExtractBothScheduledAndDueDate(t *testing.T) {
	input := "Task scheduled:tomorrow due:friday +work"

	// Extract scheduled date
	cleanTitle, scheduled := ExtractScheduledDateFromTitle(input)

	// Extract deadline
	cleanTitle, deadline := ExtractDeadlineFromTitle(cleanTitle)

	// Extract projects
	cleanTitle, projects := ExtractProjectsFromTitle(cleanTitle)

	if cleanTitle != "Task" {
		t.Errorf("Expected title 'Task', got %q", cleanTitle)
	}

	if scheduled == nil {
		t.Error("Expected scheduled date to be extracted")
	}

	if deadline == nil {
		t.Error("Expected deadline to be extracted")
	}

	if len(projects) != 1 || projects[0] != "work" {
		t.Errorf("Expected projects [work], got %v", projects)
	}
}

func TestIsFutureScheduled(t *testing.T) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	tomorrow := today.AddDate(0, 0, 1)
	yesterday := today.AddDate(0, 0, -1)

	tests := []struct {
		name     string
		task     Task
		isFuture bool
	}{
		{
			name:     "No scheduled date",
			task:     Task{},
			isFuture: false,
		},
		{
			name:     "Scheduled tomorrow",
			task:     Task{ScheduledDate: &tomorrow},
			isFuture: true,
		},
		{
			name:     "Scheduled today",
			task:     Task{ScheduledDate: &today},
			isFuture: false,
		},
		{
			name:     "Scheduled yesterday",
			task:     Task{ScheduledDate: &yesterday},
			isFuture: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.task.IsFutureScheduled()
			if result != tt.isFuture {
				t.Errorf("IsFutureScheduled() = %v, want %v", result, tt.isFuture)
			}
		})
	}
}

func TestScheduledDateStartOfDay(t *testing.T) {
	// Scheduled dates should be at start of day (00:00:00)
	_, scheduled := ExtractScheduledDateFromTitle("Task scheduled:tomorrow")

	if scheduled == nil {
		t.Fatal("Expected scheduled date to be extracted")
	}

	if scheduled.Hour() != 0 || scheduled.Minute() != 0 || scheduled.Second() != 0 {
		t.Errorf("Scheduled date should be at start of day, got %v", scheduled.Format("15:04:05"))
	}
}

func TestDeadlineEndOfDay(t *testing.T) {
	// Deadlines should be at end of day (23:59:59)
	_, deadline := ExtractDeadlineFromTitle("Task due:tomorrow")

	if deadline == nil {
		t.Fatal("Expected deadline to be extracted")
	}

	if deadline.Hour() != 23 || deadline.Minute() != 59 || deadline.Second() != 59 {
		t.Errorf("Deadline should be at end of day, got %v", deadline.Format("15:04:05"))
	}
}
