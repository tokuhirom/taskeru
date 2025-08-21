package internal

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-runewidth"
)

func TestKanbanViewMultiByte(t *testing.T) {
	// Create test tasks with multi-byte characters
	tasks := []Task{
		{
			ID:       "1",
			Title:    "日本語のタスク",
			Status:   StatusTODO,
			Created:  time.Now(),
			Updated:  time.Now(),
			Projects: []string{"プロジェクト名"},
		},
		{
			ID:       "2",
			Title:    "Very long Japanese title that contains multiple characters 長いタイトルでマルチバイト文字を含む",
			Status:   StatusDOING,
			Priority: "A",
			Created:  time.Now(),
			Updated:  time.Now(),
			Projects: []string{"work"},
		},
		{
			ID:       "3",
			Title:    "絵文字を含むタスク 🎉",
			Status:   StatusTODO,
			Created:  time.Now(),
			Updated:  time.Now(),
			Projects: []string{"emoji"},
		},
		{
			ID:      "4",
			Title:   "韓国語 한글 과 中文混合",
			Status:  StatusWAITING,
			Created: time.Now(),
			Updated: time.Now(),
		},
		{
			ID:          "5",
			Title:       "完了したタスク",
			Status:      StatusDONE,
			Created:     time.Now(),
			Updated:     time.Now(),
			CompletedAt: &time.Time{},
		},
	}

	// Create kanban view
	model := NewKanbanView(tasks)

	// Get the initial view
	view := model.View()

	// Print the view for visual inspection
	fmt.Println("=== Kanban View with Multi-byte Characters ===")
	fmt.Println(view)
	fmt.Println("=== End of View ===")

	// Check that the view contains expected elements
	if !strings.Contains(view, "日本語のタスク") {
		t.Error("Japanese task title not found in view")
	}

	if !strings.Contains(view, "プロジェクト名") {
		t.Error("Japanese project name not found in view")
	}

	if !strings.Contains(view, "🎉") {
		t.Error("Emoji not found in view")
	}

	if !strings.Contains(view, "한글") {
		t.Error("Korean text not found in view")
	}

	// Check card borders are present
	if !strings.Contains(view, "┌") || !strings.Contains(view, "┐") {
		t.Error("Card borders not properly rendered")
	}

	// Simulate navigation to check selection
	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRight})
	modelVal := updatedModel.(KanbanView)
	model = &modelVal
	updatedModel2, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
	modelVal2 := updatedModel2.(KanbanView)
	model = &modelVal2

	view2 := model.View()

	// Selected card should have double borders
	if !strings.Contains(view2, "╔") || !strings.Contains(view2, "╗") {
		t.Error("Selected card borders not properly rendered")
	}

	fmt.Println("\n=== After Navigation (DOING column, second task) ===")
	fmt.Println(view2)
	fmt.Println("=== End of View ===")
}

func TestKanbanViewCardAlignment(t *testing.T) {
	// Test that cards with different content lengths align properly
	tasks := []Task{
		{
			ID:      "1",
			Title:   "Short",
			Status:  StatusTODO,
			Created: time.Now(),
			Updated: time.Now(),
		},
		{
			ID:       "2",
			Title:    "Medium length task with some text",
			Status:   StatusTODO,
			Priority: "B",
			Created:  time.Now(),
			Updated:  time.Now(),
			Projects: []string{"project1", "project2"},
		},
		{
			ID:       "3",
			Title:    "Very very very very long task title that should wrap to multiple lines when displayed in the kanban card",
			Status:   StatusDOING,
			Priority: "A",
			Created:  time.Now(),
			Updated:  time.Now(),
			Projects: []string{"longproject", "another", "third"},
		},
	}

	model := NewKanbanView(tasks)
	view := model.View()

	fmt.Println("\n=== Card Alignment Test ===")
	fmt.Println(view)
	fmt.Println("=== End of View ===")

	// Check that all status headers are present
	for _, status := range GetAllStatuses() {
		if !strings.Contains(view, status) {
			t.Errorf("Status %s not found in view", status)
		}
	}

	// Verify wrapping occurs
	lines := strings.Split(view, "\n")
	cardFound := false
	for _, line := range lines {
		if strings.Contains(line, "Very very very") && !strings.Contains(line, "Very very very very long task title") {
			// The title should be wrapped, not on a single line
			cardFound = true
			break
		}
	}

	if !cardFound {
		t.Error("Long title doesn't appear to be wrapped properly")
	}
}

func TestKanbanViewLongTitleWrapping(t *testing.T) {
	// Test case: very long title that would break the layout
	longTitle := "project 内のタスク一覧からも､通常のタスク一覧と同様に､完了処理や編集処理に入れるべき"

	tasks := []Task{
		{
			ID:       "1",
			Title:    longTitle,
			Status:   StatusTODO,
			Priority: "C",
			Created:  time.Now(),
			Updated:  time.Now(),
		},
	}

	model := NewKanbanView(tasks)
	view := model.View()

	fmt.Println("\n=== Long Title Wrapping Test ===")
	fmt.Println(view)
	fmt.Println("=== End of View ===")

	lines := strings.Split(view, "\n")

	// Check that card borders are intact
	hasBrokenBorder := false
	for _, line := range lines {
		// Check if a line has card content but missing closing border
		if strings.Contains(line, "│") {
			// Count the number of border characters
			borderCount := strings.Count(line, "│") + strings.Count(line, "║")
			// Each card should have exactly 2 borders (left and right)
			// With 5 columns, we might have 0, 2, 4, 6, 8, or 10 borders per line
			if borderCount > 0 && borderCount%2 != 0 {
				fmt.Printf("Broken border found: %s\n", line)
				hasBrokenBorder = true
			}
		}
	}

	if hasBrokenBorder {
		t.Error("Card borders are broken due to long title")
	}

	// Check that the long title is properly wrapped within card width
	cardWidth := 25             // Based on current implementation
	innerWidth := cardWidth - 6 // Account for borders and padding

	for _, line := range lines {
		if strings.Contains(line, "│") && strings.Contains(line, longTitle[:10]) {
			// Extract content between borders
			start := strings.Index(line, "│") + 1
			end := strings.LastIndex(line, "│")
			if end > start {
				content := line[start:end]
				// Strip ANSI codes and check display width
				cleanContent := stripAnsi(content)
				displayLen := runewidth.StringWidth(strings.TrimSpace(cleanContent))

				if displayLen > innerWidth+2 { // +2 for some margin
					t.Errorf("Content exceeds card width: %d > %d", displayLen, innerWidth)
					t.Errorf("Line: %s", line)
				}
			}
		}
	}
}

func TestWrapTextFunction(t *testing.T) {
	// Unit test for the wrapText function
	testCases := []struct {
		name     string
		text     string
		width    int
		expected []string
	}{
		{
			name:     "Short text",
			text:     "Hello",
			width:    10,
			expected: []string{"Hello"},
		},
		{
			name:     "Text exactly at width",
			text:     "1234567890",
			width:    10,
			expected: []string{"1234567890"},
		},
		{
			name:     "Text needs wrapping",
			text:     "This is a long text that needs wrapping",
			width:    10,
			expected: []string{"This is a", "long text", "that needs", "wrapping"},
		},
		{
			name:     "Japanese text",
			text:     "日本語のテキストも正しく折り返す必要があります",
			width:    20,
			expected: []string{"日本語のテキストも", "正しく折り返す必要が", "あります"},
		},
		{
			name:     "Mixed English and Japanese",
			text:     "project 内のタスク一覧からも通常のタスク一覧と同様に完了処理や編集処理に入れるべき",
			width:    20,
			expected: []string{"project", "内のタスク一覧から", "も通常のタスク一覧と", "同様に完了処理や編集", "処理に入れるべき"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := wrapText(tc.text, tc.width)

			// Check each line doesn't exceed the width
			for i, line := range result {
				actualWidth := runewidth.StringWidth(line)
				if actualWidth > tc.width {
					t.Errorf("Line %d exceeds width: %q (width=%d, max=%d)",
						i, line, actualWidth, tc.width)
				}
			}

			// For Japanese text, we just verify all characters are present
			// since we may break in the middle of "words"
			joined := strings.Join(result, "")
			original := strings.ReplaceAll(tc.text, " ", "")
			joinedNoSpace := strings.ReplaceAll(joined, " ", "")

			if original != joinedNoSpace {
				t.Errorf("Characters were lost during wrapping:\nOriginal: %q\nWrapped:  %q",
					tc.text, joined)
			}
		})
	}
}
