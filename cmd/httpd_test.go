package cmd

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"taskeru/internal"
)

func TestGroupTasksByStatus(t *testing.T) {
	tasks := []internal.Task{
		{ID: "1", Title: "Task 1", Status: "TODO"},
		{ID: "2", Title: "Task 2", Status: "DOING"},
		{ID: "3", Title: "Task 3", Status: "DONE"},
		{ID: "4", Title: "Task 4", Status: "todo"}, // lowercase should be handled
	}

	grouped := groupTasksByStatus(tasks)

	if len(grouped["TODO"]) != 2 {
		t.Errorf("Expected 2 TODO tasks, got %d", len(grouped["TODO"]))
	}
	if len(grouped["DOING"]) != 1 {
		t.Errorf("Expected 1 DOING task, got %d", len(grouped["DOING"]))
	}
	if len(grouped["DONE"]) != 1 {
		t.Errorf("Expected 1 DONE task, got %d", len(grouped["DONE"]))
	}
}

func TestGroupTasksByDate(t *testing.T) {
	now := time.Now()
	targetMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)

	lastMonth := now.AddDate(0, -1, 0)
	thisMonth := now

	tasks := []internal.Task{
		{ID: "1", Title: "Last month task", Updated: lastMonth},
		{ID: "2", Title: "This month task", Updated: thisMonth},
		{ID: "3", Title: "Completed this month", Updated: lastMonth, CompletedAt: &thisMonth},
	}

	grouped := groupTasksByDate(tasks, targetMonth)

	// Should have tasks from this month
	todayKey := thisMonth.Format("2006-01-02")
	if len(grouped[todayKey]) < 1 {
		t.Errorf("Expected at least 1 task for today, got %d", len(grouped[todayKey]))
	}

	// Should not have last month's task (unless it was completed this month)
	lastMonthKey := lastMonth.Format("2006-01-02")
	if _, exists := grouped[lastMonthKey]; exists {
		t.Errorf("Should not include last month's tasks in this month's view")
	}
}

func TestAnsi256ToHex(t *testing.T) {
	tests := []struct {
		colorNum string
		expected string
	}{
		{"33", "#0087ff"},
		{"208", "#ff8700"},
		{"999", "#36b3d9"}, // Invalid should return default
	}

	for _, tt := range tests {
		got := ansi256ToHex(tt.colorNum)
		if got != tt.expected {
			t.Errorf("ansi256ToHex(%s) = %s, want %s", tt.colorNum, got, tt.expected)
		}
	}
}

func TestStyleHandler(t *testing.T) {
	req := httptest.NewRequest("GET", "/static/style.css", nil)
	w := httptest.NewRecorder()

	NewController(internal.NewTaskFile()).styleHandler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "text/css" {
		t.Errorf("Expected Content-Type text/css, got %s", contentType)
	}

	body := w.Body.String()
	if !strings.Contains(body, ":root") {
		t.Error("CSS should contain :root variables")
	}
	if !strings.Contains(body, ".kanban-board") {
		t.Error("CSS should contain kanban styles")
	}
	if !strings.Contains(body, ".daily-entry") {
		t.Error("CSS should contain daily report styles")
	}
}

func TestGetAvailableMonths(t *testing.T) {
	now := time.Now()
	lastMonth := now.AddDate(0, -1, 0)
	twoMonthsAgo := now.AddDate(0, -2, 0)

	tasks := []internal.Task{
		{ID: "1", Title: "Task 1", Updated: now},
		{ID: "2", Title: "Task 2", Updated: lastMonth},
		{ID: "3", Title: "Task 3", Updated: twoMonthsAgo},
		{ID: "4", Title: "Task 4", Updated: now, CompletedAt: &lastMonth},
	}

	months := getAvailableMonths(tasks)

	// Should have at least 3 months
	if len(months) < 3 {
		t.Errorf("Expected at least 3 months, got %d", len(months))
	}

	// Should be sorted in descending order (newest first)
	for i := 0; i < len(months)-1; i++ {
		if months[i].Year < months[i+1].Year {
			t.Error("Months should be sorted by year descending")
		}
		if months[i].Year == months[i+1].Year && months[i].Month < months[i+1].Month {
			t.Error("Months should be sorted by month descending within same year")
		}
	}
}

func TestGetSortedDates(t *testing.T) {
	tasksByDate := map[string][]internal.Task{
		"2024-01-03": {},
		"2024-01-01": {},
		"2024-01-02": {},
	}

	dates := getSortedDates(tasksByDate)

	// Should be sorted in descending order
	expected := []string{"2024-01-03", "2024-01-02", "2024-01-01"}
	for i, date := range dates {
		if date != expected[i] {
			t.Errorf("Expected date %s at position %d, got %s", expected[i], i, date)
		}
	}
}
