package internal

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestDetailedStatusCycle(t *testing.T) {
	// Create test tasks
	tasks := []Task{
		*NewTask("Task 1"),
		*NewTask("Task 2"),
		*NewTask("Task 3"),
	}

	// All tasks start as TODO
	for i := range tasks {
		tasks[i].Status = StatusTODO
		tasks[i].Priority = "medium"
	}

	model := NewInteractiveTaskList(tasks)

	// Move cursor to Task 2 (index 1)
	model.cursor = 1
	targetTaskID := model.tasks[1].ID
	t.Logf("Initial: cursor at position 1, Task ID: %s, Title: %s", targetTaskID, model.tasks[1].Title)

	// Press 's' to change to DOING
	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	interactiveModel := updatedModel.(InteractiveTaskList)

	t.Logf("\nAfter pressing 's' (should be DOING):")
	t.Logf("  Cursor position: %d", interactiveModel.cursor)

	// Find where Task 2 ended up
	for i, task := range interactiveModel.tasks {
		if task.ID == targetTaskID {
			t.Logf("  Task 2 is now at position %d with status %s", i, task.Status)
			break
		}
	}

	// Check what the cursor is pointing to
	if interactiveModel.cursor < len(interactiveModel.tasks) {
		currentTask := interactiveModel.tasks[interactiveModel.cursor]
		t.Logf("  Cursor is pointing to: %s (ID: %s, Status: %s)",
			currentTask.Title, currentTask.ID, currentTask.Status)
	}

	// Check all tasks
	t.Log("\n  All tasks after change:")
	for i, task := range interactiveModel.allTasks {
		if task.ID == targetTaskID {
			t.Logf("    [allTasks %d] %s - %s <-- Target", i, task.Title, task.Status)
		} else {
			t.Logf("    [allTasks %d] %s - %s", i, task.Title, task.Status)
		}
	}

	t.Log("\n  Visible tasks after change:")
	for i, task := range interactiveModel.tasks {
		if i == interactiveModel.cursor {
			t.Logf("    [%d] %s - %s <-- Cursor", i, task.Title, task.Status)
		} else {
			t.Logf("    [%d] %s - %s", i, task.Title, task.Status)
		}
	}
}
