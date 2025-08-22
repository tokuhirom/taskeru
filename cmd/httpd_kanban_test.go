package cmd

import (
	"taskeru/internal"
	"testing"
	"time"
)

func TestGroupTasksByStatusFiltersOldDoneTasks(t *testing.T) {
	now := time.Now()
	twoDaysAgo := now.AddDate(0, 0, -2)
	fiveHoursAgo := now.Add(-5 * time.Hour)

	tasks := []internal.Task{
		{
			ID:     "1",
			Title:  "Active TODO task",
			Status: "TODO",
		},
		{
			ID:     "2",
			Title:  "Currently doing",
			Status: "DOING",
		},
		{
			ID:          "3",
			Title:       "Recently completed (5 hours ago)",
			Status:      "DONE",
			CompletedAt: &fiveHoursAgo,
		},
		{
			ID:          "4",
			Title:       "Old completed task (2 days ago)",
			Status:      "DONE",
			CompletedAt: &twoDaysAgo,
		},
		{
			ID:          "5",
			Title:       "Old WONTDO task (2 days ago)",
			Status:      "WONTDO",
			CompletedAt: &twoDaysAgo,
		},
		{
			ID:          "6",
			Title:       "Recent WONTDO (5 hours ago)",
			Status:      "WONTDO",
			CompletedAt: &fiveHoursAgo,
		},
		{
			ID:     "7",
			Title:  "Waiting task",
			Status: "WAITING",
		},
	}

	result := groupTasksByStatus(tasks)

	// Check TODO tasks
	if len(result["TODO"]) != 1 {
		t.Errorf("Expected 1 TODO task, got %d", len(result["TODO"]))
	}

	// Check DOING tasks
	if len(result["DOING"]) != 1 {
		t.Errorf("Expected 1 DOING task, got %d", len(result["DOING"]))
	}

	// Check WAITING tasks
	if len(result["WAITING"]) != 1 {
		t.Errorf("Expected 1 WAITING task, got %d", len(result["WAITING"]))
	}

	// Check DONE tasks - should only have recent one
	if len(result["DONE"]) != 1 {
		t.Errorf("Expected 1 DONE task (recent only), got %d", len(result["DONE"]))
		for _, task := range result["DONE"] {
			t.Logf("  - %s (completed: %v)", task.Title, task.CompletedAt)
		}
	}
	if len(result["DONE"]) > 0 && result["DONE"][0].Title != "Recently completed (5 hours ago)" {
		t.Errorf("Expected recent DONE task, got %s", result["DONE"][0].Title)
	}

	// Check WONTDO tasks - should only have recent one
	if len(result["WONTDO"]) != 1 {
		t.Errorf("Expected 1 WONTDO task (recent only), got %d", len(result["WONTDO"]))
		for _, task := range result["WONTDO"] {
			t.Logf("  - %s (completed: %v)", task.Title, task.CompletedAt)
		}
	}
	if len(result["WONTDO"]) > 0 && result["WONTDO"][0].Title != "Recent WONTDO (5 hours ago)" {
		t.Errorf("Expected recent WONTDO task, got %s", result["WONTDO"][0].Title)
	}
}

func TestGroupTasksByStatusHandlesNilCompletedAt(t *testing.T) {
	tasks := []internal.Task{
		{
			ID:          "1",
			Title:       "DONE task without CompletedAt",
			Status:      "DONE",
			CompletedAt: nil, // This shouldn't cause a panic
		},
		{
			ID:          "2",
			Title:       "WONTDO task without CompletedAt",
			Status:      "WONTDO",
			CompletedAt: nil, // This shouldn't cause a panic
		},
	}

	result := groupTasksByStatus(tasks)

	// Tasks without CompletedAt should be included (they're considered recent)
	if len(result["DONE"]) != 1 {
		t.Errorf("Expected 1 DONE task without CompletedAt, got %d", len(result["DONE"]))
	}
	if len(result["WONTDO"]) != 1 {
		t.Errorf("Expected 1 WONTDO task without CompletedAt, got %d", len(result["WONTDO"]))
	}
}
