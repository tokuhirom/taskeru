package internal

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/require"
)

func TestDateEditMode(t *testing.T) {
	// Create test tasks
	tasks := []Task{
		*NewTask("Task without dates"),
		*NewTask("Task with deadline"),
		*NewTask("Task with scheduled date"),
	}

	// Set existing dates
	deadline := time.Now().AddDate(0, 0, 7) // 7 days from now
	tasks[1].DueDate = &deadline

	scheduled := time.Now().AddDate(0, 0, 3) // 3 days from now
	tasks[2].ScheduledDate = &scheduled

	taskFile := NewTaskFileForTesting(t)
	for _, task := range tasks {
		require.NoError(t, taskFile.AddTask(&task))
	}

	model, err := NewInteractiveTaskListWithFilter(taskFile, "")
	require.NoError(t, err)

	// Test entering deadline edit mode with D
	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'D'}})
	interactiveModel := updatedModel.(InteractiveTaskList)

	if interactiveModel.dateEditMode != "deadline" {
		t.Error("Should be in deadline edit mode after pressing D")
	}

	if interactiveModel.dateEditBuffer != "" {
		t.Error("Date edit buffer should be empty for task without deadline")
	}

	// Test ESC to cancel date edit
	updatedModel, _ = interactiveModel.Update(tea.KeyMsg{Type: tea.KeyEsc})
	interactiveModel = updatedModel.(InteractiveTaskList)

	if interactiveModel.dateEditMode != "" {
		t.Error("Should exit date edit mode after pressing ESC")
	}

	// Test entering scheduled date edit mode with S
	updatedModel, _ = interactiveModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'S'}})
	interactiveModel = updatedModel.(InteractiveTaskList)

	if interactiveModel.dateEditMode != "scheduled" {
		t.Error("Should be in scheduled edit mode after pressing S")
	}

	// Cancel again
	updatedModel, _ = interactiveModel.Update(tea.KeyMsg{Type: tea.KeyEsc})
	interactiveModel = updatedModel.(InteractiveTaskList)
}

func TestDateEditWithExistingDates(t *testing.T) {
	tasks := []Task{
		*NewTask("Task with dates"),
	}

	// Set existing dates
	deadline := time.Date(2025, 12, 31, 23, 59, 59, 0, time.Local)
	tasks[0].DueDate = &deadline

	scheduled := time.Date(2025, 12, 25, 0, 0, 0, 0, time.Local)
	tasks[0].ScheduledDate = &scheduled

	taskFile := NewTaskFileForTesting(t)
	require.NoError(t, taskFile.AddTask(&tasks[0]))

	model, err := NewInteractiveTaskListWithFilter(taskFile, "")
	require.NoError(t, err)

	// Test D key pre-fills existing deadline
	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'D'}})
	interactiveModel := updatedModel.(InteractiveTaskList)

	if interactiveModel.dateEditBuffer != "2025-12-31" {
		t.Errorf("Date edit buffer should contain existing deadline, got %q", interactiveModel.dateEditBuffer)
	}

	if interactiveModel.dateEditCursor != len("2025-12-31") {
		t.Error("Date edit cursor should be at end of buffer")
	}

	// Cancel and test S key pre-fills existing scheduled date
	updatedModel, _ = interactiveModel.Update(tea.KeyMsg{Type: tea.KeyEsc})
	interactiveModel = updatedModel.(InteractiveTaskList)

	updatedModel, _ = interactiveModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'S'}})
	interactiveModel = updatedModel.(InteractiveTaskList)

	if interactiveModel.dateEditBuffer != "2025-12-25" {
		t.Errorf("Date edit buffer should contain existing scheduled date, got %q", interactiveModel.dateEditBuffer)
	}
}

func TestDateEditInput(t *testing.T) {
	tasks := []Task{
		*NewTask("Test task"),
	}

	taskFile := NewTaskFileForTesting(t)
	require.NoError(t, taskFile.AddTask(&tasks[0]))

	model, err := NewInteractiveTaskListWithFilter(taskFile, "")
	require.NoError(t, err)

	// Enter deadline edit mode
	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'D'}})
	interactiveModel := updatedModel.(InteractiveTaskList)

	// Type "today"
	for _, ch := range "today" {
		updatedModel, _ = interactiveModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		interactiveModel = updatedModel.(InteractiveTaskList)
	}

	if interactiveModel.dateEditBuffer != "today" {
		t.Errorf("Date edit buffer should be 'today', got %q", interactiveModel.dateEditBuffer)
	}

	// Test backspace
	updatedModel, _ = interactiveModel.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	interactiveModel = updatedModel.(InteractiveTaskList)

	if interactiveModel.dateEditBuffer != "toda" {
		t.Errorf("After backspace, buffer should be 'toda', got %q", interactiveModel.dateEditBuffer)
	}

	// Test cursor movement
	updatedModel, _ = interactiveModel.Update(tea.KeyMsg{Type: tea.KeyCtrlA})
	interactiveModel = updatedModel.(InteractiveTaskList)

	if interactiveModel.dateEditCursor != 0 {
		t.Error("Ctrl+A should move cursor to beginning")
	}

	updatedModel, _ = interactiveModel.Update(tea.KeyMsg{Type: tea.KeyCtrlE})
	interactiveModel = updatedModel.(InteractiveTaskList)

	if interactiveModel.dateEditCursor != len("toda") {
		t.Error("Ctrl+E should move cursor to end")
	}
}

func TestDateEditApply(t *testing.T) {
	tasks := []Task{
		*NewTask("Test task"),
	}

	taskFile := NewTaskFileForTesting(t)
	require.NoError(t, taskFile.AddTask(&tasks[0]))

	model, err := NewInteractiveTaskListWithFilter(taskFile, "")
	require.NoError(t, err)

	// Enter deadline edit mode
	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'D'}})
	interactiveModel := updatedModel.(InteractiveTaskList)

	// Type "2025-12-31"
	for _, ch := range "2025-12-31" {
		updatedModel, _ = interactiveModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		interactiveModel = updatedModel.(InteractiveTaskList)
	}

	// Apply with Enter
	updatedModel, _ = interactiveModel.Update(tea.KeyMsg{Type: tea.KeyEnter})
	interactiveModel = updatedModel.(InteractiveTaskList)

	if interactiveModel.dateEditMode != "" {
		t.Error("Should exit date edit mode after pressing Enter")
	}

	if !interactiveModel.modified {
		t.Error("Model should be marked as modified after applying date")
	}

	// Check that the task was updated
	if interactiveModel.allTasks[0].DueDate == nil {
		t.Error("Task should have deadline set")
	} else {
		// Check that it's end of day
		if interactiveModel.allTasks[0].DueDate.Hour() != 23 ||
			interactiveModel.allTasks[0].DueDate.Minute() != 59 ||
			interactiveModel.allTasks[0].DueDate.Second() != 59 {
			t.Error("Deadline should be set to end of day (23:59:59)")
		}
	}
}

func TestScheduledDateEditApply(t *testing.T) {
	tasks := []Task{
		*NewTask("Test task"),
	}
	taskFile := NewTaskFileForTesting(t)
	require.NoError(t, taskFile.AddTask(&tasks[0]))

	model, err := NewInteractiveTaskListWithFilter(taskFile, "")
	require.NoError(t, err)

	// Enter scheduled date edit mode
	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'S'}})
	interactiveModel := updatedModel.(InteractiveTaskList)

	// Type "tomorrow"
	for _, ch := range "tomorrow" {
		updatedModel, _ = interactiveModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		interactiveModel = updatedModel.(InteractiveTaskList)
	}

	// Apply with Enter
	updatedModel, _ = interactiveModel.Update(tea.KeyMsg{Type: tea.KeyEnter})
	interactiveModel = updatedModel.(InteractiveTaskList)

	if interactiveModel.dateEditMode != "" {
		t.Error("Should exit date edit mode after pressing Enter")
	}

	// Check that the task was updated
	if interactiveModel.allTasks[0].ScheduledDate == nil {
		t.Error("Task should have scheduled date set")
	} else {
		// Check that it's start of day
		if interactiveModel.allTasks[0].ScheduledDate.Hour() != 0 ||
			interactiveModel.allTasks[0].ScheduledDate.Minute() != 0 ||
			interactiveModel.allTasks[0].ScheduledDate.Second() != 0 {
			t.Error("Scheduled date should be set to start of day (00:00:00)")
		}
	}
}

func TestDateEditClearDate(t *testing.T) {
	tasks := []Task{
		*NewTask("Task with deadline"),
	}

	// Set existing deadline
	deadline := time.Now().AddDate(0, 0, 7)
	tasks[0].DueDate = &deadline

	taskFile := NewTaskFileForTesting(t)
	require.NoError(t, taskFile.AddTask(&tasks[0]))

	model, err := NewInteractiveTaskListWithFilter(taskFile, "")
	require.NoError(t, err)

	// Enter deadline edit mode
	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'D'}})
	interactiveModel := updatedModel.(InteractiveTaskList)

	// Clear the buffer
	for interactiveModel.dateEditCursor > 0 {
		updatedModel, _ = interactiveModel.Update(tea.KeyMsg{Type: tea.KeyBackspace})
		interactiveModel = updatedModel.(InteractiveTaskList)
	}

	// Apply empty date with Enter
	updatedModel, _ = interactiveModel.Update(tea.KeyMsg{Type: tea.KeyEnter})
	interactiveModel = updatedModel.(InteractiveTaskList)

	// Check that the deadline was cleared
	if interactiveModel.allTasks[0].DueDate != nil {
		t.Error("Task deadline should be cleared when applying empty date")
	}

	if !interactiveModel.modified {
		t.Error("Model should be marked as modified after clearing date")
	}
}

func TestDateEditModeView(t *testing.T) {
	tasks := []Task{
		*NewTask("Test task"),
	}

	taskFile := NewTaskFileForTesting(t)
	require.NoError(t, taskFile.AddTask(&tasks[0]))

	model, err := NewInteractiveTaskListWithFilter(taskFile, "")
	require.NoError(t, err)
	model.dateEditMode = "deadline"
	model.dateEditBuffer = "2025-12-31"
	model.dateEditCursor = 5 // Position after "2025-"

	view := model.View()

	// Check that date edit UI is shown
	if !strings.Contains(view, "📅 Set deadline:") {
		t.Error("Date edit mode view should show calendar icon and deadline label")
	}

	// Check that cursor is displayed
	if !strings.Contains(view, "│") {
		t.Error("Date edit mode should show cursor")
	}

	// Check help text
	if !strings.Contains(view, "Supported formats:") {
		t.Error("Date edit mode should show format help")
	}
	if !strings.Contains(view, "Natural: next tuesday") {
		t.Error("Date edit mode should show natural language format examples")
	}

	// Test scheduled date mode
	model.dateEditMode = "scheduled"
	view = model.View()

	if !strings.Contains(view, "📅 Set scheduled date:") {
		t.Error("Date edit mode view should show scheduled date label")
	}
}

func TestDateEditDoesNotTriggerDuringDelete(t *testing.T) {
	tasks := []Task{
		*NewTask("Test task"),
	}

	taskFile := NewTaskFileForTesting(t)
	require.NoError(t, taskFile.AddTask(&tasks[0]))

	model, err := NewInteractiveTaskListWithFilter(taskFile, "")
	require.NoError(t, err)

	// Enter delete confirmation mode
	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	interactiveModel := updatedModel.(InteractiveTaskList)

	if !interactiveModel.confirmDelete {
		t.Error("Should be in delete confirmation mode after pressing d")
	}

	// Try to enter date edit mode with D (should not work)
	updatedModel, _ = interactiveModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'D'}})
	interactiveModel = updatedModel.(InteractiveTaskList)

	if interactiveModel.dateEditMode != "" {
		t.Error("Should not enter date edit mode when delete confirmation is active")
	}

	// Cancel delete
	updatedModel, _ = interactiveModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	interactiveModel = updatedModel.(InteractiveTaskList)

	// Now D should work
	updatedModel, _ = interactiveModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'D'}})
	interactiveModel = updatedModel.(InteractiveTaskList)

	if interactiveModel.dateEditMode != "deadline" {
		t.Error("Should enter date edit mode after canceling delete confirmation")
	}
}
