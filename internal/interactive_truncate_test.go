package internal

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestTruncateTaskLine(t *testing.T) {
	tests := []struct {
		name           string
		width          int
		title          string
		projects       []string
		additionalInfo string
		expectContains []string
		expectNotFull  string // If title should be truncated, it shouldn't contain the full title
	}{
		{
			name:           "Short title fits completely",
			width:          80,
			title:          "Short task",
			projects:       []string{"work"},
			additionalInfo: " (due tomorrow)",
			expectContains: []string{"Short task", "TODO"},
			expectNotFull:  "",
		},
		{
			name:           "Long title gets truncated",
			width:          50,
			title:          "This is a very long task title that will definitely need to be truncated",
			projects:       []string{"personal"},
			additionalInfo: " (due 01-02)",
			expectContains: []string{"...", "TODO"},
			expectNotFull:  "definitely need to be truncated",
		},
		{
			name:           "Multiple projects preserved",
			width:          60,
			title:          "Task with multiple projects that might get truncated",
			projects:       []string{"work", "urgent", "q4"},
			additionalInfo: "",
			expectContains: []string{"...", "TODO"},
			expectNotFull:  "might get truncated",
		},
		{
			name:           "Very narrow terminal",
			width:          30,
			title:          "Task title",
			projects:       []string{"project"},
			additionalInfo: " (overdue)",
			expectContains: []string{"TODO"},
			expectNotFull:  "", // Title might still fit in 30 chars
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := NewTask(tt.title)
			task.Projects = tt.projects
			task.Status = StatusTODO

			model := NewInteractiveTaskList([]Task{*task})
			model.width = tt.width

			// Simulate the truncation
			result := model.truncateTaskLine("  ", "\x1b[37m", "TODO", "", tt.title, tt.projects, tt.additionalInfo)

			// Strip ANSI codes for easier testing
			cleanResult := model.stripAnsiCodes(result)

			// Check expected contents
			for _, expected := range tt.expectContains {
				if !strings.Contains(cleanResult, expected) {
					t.Errorf("Expected result to contain '%s', got: %s", expected, cleanResult)
				}
			}

			// Check that long titles are truncated
			if tt.expectNotFull != "" && strings.Contains(cleanResult, tt.expectNotFull) {
				t.Errorf("Expected title to be truncated (not contain '%s'), but got: %s", tt.expectNotFull, cleanResult)
			}

			// Verify the result isn't longer than terminal width when rendered
			// (accounting for ANSI codes being stripped in actual display)
			if len(cleanResult) > tt.width+10 { // +10 for some buffer since we're approximating
				t.Errorf("Result too long for width %d: %d chars", tt.width, len(cleanResult))
			}
		})
	}
}

func TestWindowSizeUpdate(t *testing.T) {
	task := NewTask("Test task")
	model := NewInteractiveTaskList([]Task{*task})

	// Initial dimensions should be defaults
	if model.width != 80 || model.height != 24 {
		t.Errorf("Expected default dimensions 80x24, got %dx%d", model.width, model.height)
	}

	// Send window size message
	updatedModel, _ := model.Update(tea.WindowSizeMsg{
		Width:  120,
		Height: 40,
	})

	interactiveModel := updatedModel.(InteractiveTaskList)
	if interactiveModel.width != 120 || interactiveModel.height != 40 {
		t.Errorf("Expected updated dimensions 120x40, got %dx%d", interactiveModel.width, interactiveModel.height)
	}
}

func TestStripAnsiCodes(t *testing.T) {
	model := NewInteractiveTaskList([]Task{})

	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "Plain text",
			expected: "Plain text",
		},
		{
			input:    "\x1b[31mRed text\x1b[0m",
			expected: "Red text",
		},
		{
			input:    "\x1b[1;32mBold green\x1b[0m text \x1b[33myellow\x1b[0m",
			expected: "Bold green text yellow",
		},
		{
			input:    "Mixed \x1b[41mbackground\x1b[0m and \x1b[4munderline\x1b[0m",
			expected: "Mixed background and underline",
		},
	}

	for _, tt := range tests {
		result := model.stripAnsiCodes(tt.input)
		if result != tt.expected {
			t.Errorf("stripAnsiCodes(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}