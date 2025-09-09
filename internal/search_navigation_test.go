package internal

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/require"
)

func TestSearchNavigation(t *testing.T) {
	// Create test tasks
	tasks := []Task{
		*NewTask("Task 1"),
		*NewTask("Work task"),
		*NewTask("Another task"),
		*NewTask("Work on project"),
		*NewTask("Final task"),
	}
	tasks[0].Priority = "A"
	tasks[1].Priority = "B"
	tasks[2].Priority = "C"
	tasks[3].Priority = "D"
	tasks[4].Priority = "E"

	taskFile := NewTaskFileForTesting(t)
	require.NoError(t, taskFile.AddTasks(tasks))

	model, err := NewInteractiveTaskListWithFilter(taskFile, "")
	require.NoError(t, err)

	// Enter search mode
	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	modelVal := updatedModel.(InteractiveTaskList)
	model = &modelVal

	// Type "work"
	for _, ch := range "work" {
		updatedModel, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		modelVal = updatedModel.(InteractiveTaskList)
		model = &modelVal
	}

	// Exit search mode with Enter
	updatedModel, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	modelVal = updatedModel.(InteractiveTaskList)
	model = &modelVal

	if model.searchMode {
		t.Error("Should exit search mode after pressing Enter")
	}

	if model.searchQuery != "work" {
		t.Error("Search query should be preserved after exiting search mode")
	}

	// Should jump to first match (index 1 - "Work task")
	if model.cursor != 1 {
		t.Errorf("Cursor should be at first match (index 1), got %d", model.cursor)
	}

	// Press 'n' to go to next match
	updatedModel, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	modelVal = updatedModel.(InteractiveTaskList)
	model = &modelVal

	if model.cursor != 3 {
		t.Errorf("Cursor should be at second match (index 3), got %d", model.cursor)
	}

	// Press 'n' again - should wrap to first match
	updatedModel, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	modelVal = updatedModel.(InteractiveTaskList)
	model = &modelVal

	if model.cursor != 1 {
		t.Errorf("Cursor should wrap to first match (index 1), got %d", model.cursor)
	}

	// Press 'N' to go to previous match
	updatedModel, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'N'}})
	modelVal = updatedModel.(InteractiveTaskList)
	model = &modelVal

	if model.cursor != 3 {
		t.Errorf("Cursor should be at previous match (index 3), got %d", model.cursor)
	}

	// Clear search with ESC
	updatedModel, _ = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	modelVal = updatedModel.(InteractiveTaskList)
	model = &modelVal

	if model.searchQuery != "" {
		t.Error("Search query should be cleared after pressing ESC when not in search mode")
	}

	if len(model.matchingTasks) != 0 {
		t.Error("Matching tasks should be cleared after pressing ESC")
	}
}

func TestSearchNavigationWithNoMatches(t *testing.T) {
	tasks := []Task{
		*NewTask("Task 1"),
		*NewTask("Task 2"),
		*NewTask("Task 3"),
	}

	taskFile := NewTaskFileForTesting(t)
	require.NoError(t, taskFile.AddTasks(tasks))

	model, err := NewInteractiveTaskListWithFilter(taskFile, "")
	require.NoError(t, err)

	// Enter search mode and search for non-existent term
	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	modelVal := updatedModel.(InteractiveTaskList)
	model = &modelVal

	for _, ch := range "xyz" {
		updatedModel, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		modelVal = updatedModel.(InteractiveTaskList)
		model = &modelVal
	}

	// Exit search mode
	updatedModel, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	modelVal = updatedModel.(InteractiveTaskList)
	model = &modelVal

	originalCursor := model.cursor

	// Press 'n' - cursor shouldn't move when no matches
	updatedModel, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	modelVal = updatedModel.(InteractiveTaskList)
	model = &modelVal

	if model.cursor != originalCursor {
		t.Error("Cursor should not move when there are no matches")
	}

	// Press 'N' - cursor shouldn't move when no matches
	updatedModel, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'N'}})
	modelVal = updatedModel.(InteractiveTaskList)
	model = &modelVal

	if model.cursor != originalCursor {
		t.Error("Cursor should not move when there are no matches")
	}
}
