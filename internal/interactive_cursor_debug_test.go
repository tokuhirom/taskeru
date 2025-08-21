package internal

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestDebugCursorBehavior(t *testing.T) {
	// Create test tasks
	tasks := []Task{
		*NewTask("Task 1"),
		*NewTask("Task 2"),
		*NewTask("Task 3"),
	}

	// All tasks start as TODO
	for i := range tasks {
		tasks[i].Status = StatusTODO
	}

	model := NewInteractiveTaskList(tasks)

	t.Logf("Initial state: cursor=%d, tasks=%d", model.cursor, len(model.tasks))
	for i, task := range model.tasks {
		t.Logf("  [%d] %s - %s", i, task.Title, task.Status)
	}

	// Move cursor to Task 2 (index 1)
	model.cursor = 1
	t.Logf("\nCursor moved to position 1 (Task 2)")

	// Toggle Task 2 to DONE with space
	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeySpace})
	interactiveModel := updatedModel.(InteractiveTaskList)

	t.Logf("\nAfter toggling to DONE:")
	t.Logf("  cursor=%d, tasks=%d", interactiveModel.cursor, len(interactiveModel.tasks))
	for i, task := range interactiveModel.tasks {
		t.Logf("  [%d] %s - %s", i, task.Title, task.Status)
	}

	// Check what task the cursor is pointing to
	if interactiveModel.cursor < len(interactiveModel.tasks) {
		currentTask := interactiveModel.tasks[interactiveModel.cursor]
		t.Logf("\nCursor is pointing to: %s", currentTask.Title)
	}
}

func TestDebugStatusCycle(t *testing.T) {
	// Create test tasks
	tasks := []Task{
		*NewTask("Task 1"),
		*NewTask("Task 2"),
		*NewTask("Task 3"),
	}

	// All tasks start as TODO
	for i := range tasks {
		tasks[i].Status = StatusTODO
	}

	model := NewInteractiveTaskList(tasks)

	// Move cursor to Task 2 (index 1)
	model.cursor = 1
	t.Logf("Initial: cursor at position 1 (Task 2)")

	// Cycle through statuses with 's'
	statuses := GetAllStatuses()
	for _, expectedStatus := range statuses[1:] { // Skip TODO since we start there
		updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
		interactiveModel := updatedModel.(InteractiveTaskList)
		model = &interactiveModel

		t.Logf("\nAfter changing to %s:", expectedStatus)
		t.Logf("  cursor=%d, tasks=%d", model.cursor, len(model.tasks))

		if model.cursor < len(model.tasks) {
			currentTask := model.tasks[model.cursor]
			t.Logf("  Cursor points to: %s - %s", currentTask.Title, currentTask.Status)
		}
	}
}
