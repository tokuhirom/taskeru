package internal

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/require"
)

func TestSearchMode(t *testing.T) {
	// Create test tasks
	tasks := []Task{
		*NewTask("Work on project"),
		*NewTask("Buy groceries"),
		*NewTask("Review code"),
	}

	tasks[0].Projects = []string{"work"}
	tasks[0].Note = "Important project deadline"
	tasks[1].Projects = []string{"personal"}
	tasks[1].Note = "Need milk and bread"
	tasks[2].Projects = []string{"work", "urgent"}
	tasks[2].Note = "Review pull request"

	taskFile := NewTaskFileForTesting(t)
	require.NoError(t, taskFile.AddTasks(tasks))

	model, err := NewInteractiveTaskListWithFilter(taskFile, "")
	require.NoError(t, err)

	// Test entering search mode with /
	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	interactiveModel := updatedModel.(InteractiveTaskList)

	if !interactiveModel.searchMode {
		t.Error("Should be in search mode after pressing /")
	}

	// Test typing search query
	searchQuery := "work"
	for _, ch := range searchQuery {
		updatedModel, _ = interactiveModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		interactiveModel = updatedModel.(InteractiveTaskList)
	}

	if interactiveModel.searchQuery != searchQuery {
		t.Errorf("Search query should be %q, got %q", searchQuery, interactiveModel.searchQuery)
	}

	// Update matches
	interactiveModel.updateMatches()

	// Should match tasks with "work" in title or projects
	if len(interactiveModel.matchingTasks) != 2 {
		t.Errorf("Should find 2 tasks matching 'work', found %d", len(interactiveModel.matchingTasks))
	}

	// Test ESC to exit search mode (but keep query)
	updatedModel, _ = interactiveModel.Update(tea.KeyMsg{Type: tea.KeyEsc})
	interactiveModel = updatedModel.(InteractiveTaskList)

	if interactiveModel.searchMode {
		t.Error("Should exit search mode after pressing ESC")
	}

	if interactiveModel.searchQuery == "" {
		t.Error("Search query should be kept after pressing ESC in search mode")
	}

	// Test ESC again to clear search when not in search mode
	updatedModel, _ = interactiveModel.Update(tea.KeyMsg{Type: tea.KeyEsc})
	interactiveModel = updatedModel.(InteractiveTaskList)

	if interactiveModel.searchQuery != "" {
		t.Error("Search query should be cleared after pressing ESC when not in search mode")
	}
}

func TestSearchHighlighting(t *testing.T) {
	tasks := []Task{
		*NewTask("Important meeting"),
		*NewTask("Buy coffee"),
		*NewTask("Write documentation"),
		*NewTask("Fix bug"),
		*NewTask("Team meeting"),
	}

	tasks[0].Projects = []string{"work", "meetings"}
	tasks[1].Projects = []string{"personal"}
	tasks[2].Projects = []string{"work"}
	tasks[2].Note = "Update API documentation"
	tasks[3].Projects = []string{"work", "urgent"}
	tasks[3].Note = "Critical bug in production"
	tasks[4].Projects = []string{"meetings"}

	taskFile := NewTaskFileForTesting(t)
	require.NoError(t, taskFile.AddTasks(tasks))

	model, err := NewInteractiveTaskListWithFilter(taskFile, "")
	require.NoError(t, err)

	tests := []struct {
		query         string
		expectedCount int
		description   string
	}{
		{"meeting", 2, "Should find tasks with 'meeting' in title or projects"},
		{"work", 3, "Should find tasks with 'work' project"},
		{"bug", 1, "Should find task with 'bug' in title or note"},
		{"documentation", 1, "Should find task with 'documentation' in note"},
		{"urgent", 1, "Should find task with 'urgent' project"},
		{"coffee", 1, "Should find task with 'coffee' in title"},
		{"xyz", 0, "Should find no tasks for non-existent query"},
	}

	for _, tt := range tests {
		model.searchQuery = tt.query
		model.updateMatches()

		if len(model.matchingTasks) != tt.expectedCount {
			t.Errorf("%s: expected %d matching tasks, got %d", tt.description, tt.expectedCount, len(model.matchingTasks))
		}
	}
}

func TestSearchCaseSensitivity(t *testing.T) {
	tasks := []Task{
		*NewTask("IMPORTANT TASK"),
		*NewTask("important task"),
		*NewTask("Something else"),
	}

	taskFile := NewTaskFileForTesting(t)
	require.NoError(t, taskFile.AddTasks(tasks))

	model, err := NewInteractiveTaskListWithFilter(taskFile, "")
	require.NoError(t, err)

	// Search should be case-insensitive
	queries := []string{"important", "IMPORTANT", "Important", "iMpOrTaNt"}

	for _, query := range queries {
		model.searchQuery = query
		model.updateMatches()

		if len(model.matchingTasks) != 2 {
			t.Errorf("Case-insensitive search for %q should find 2 matching tasks, found %d", query, len(model.matchingTasks))
		}
	}
}

func TestSearchModeView(t *testing.T) {
	tasks := []Task{
		*NewTask("Test task"),
	}

	taskFile := NewTaskFileForTesting(t)
	require.NoError(t, taskFile.AddTasks(tasks))

	model, err := NewInteractiveTaskListWithFilter(taskFile, "")
	require.NoError(t, err)

	// Enter search mode and set query and cursor
	model.searchMode = true
	model.searchQuery = "test"
	model.searchCursor = 2

	view := model.View()

	// Check that search UI is shown
	if !strings.Contains(view, "🔍 Search:") {
		t.Error("Search mode view should show search icon and label")
	}

	// Check that cursor is displayed
	if !strings.Contains(view, "│") {
		t.Error("Search mode should show cursor")
	}

	// Check that match count is shown
	model.updateMatches()
	view = model.View()
	if !strings.Contains(view, "(1 matches)") {
		t.Error("Search mode should show match count when query is not empty")
	}
}
