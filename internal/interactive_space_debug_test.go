package internal

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestSpaceKeyBehavior(t *testing.T) {
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

	// Make Task 2 an old completed task so it gets hidden
	oldTime := time.Now().AddDate(0, 0, -2) // 2 days ago
	tasks[1].Status = StatusDONE
	tasks[1].CompletedAt = &oldTime

	model := NewInteractiveTaskList(tasks)

	t.Logf("Initial state:")
	t.Logf("  All tasks count: %d", len(model.allTasks))
	t.Logf("  Visible tasks count: %d", len(model.tasks))
	t.Logf("  Cursor position: %d", model.cursor)

	for i, task := range model.tasks {
		t.Logf("    [%d] %s - %s", i, task.Title, task.Status)
	}

	// Move cursor to second visible task (Task 3)
	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	interactiveModel := updatedModel.(InteractiveTaskList)

	t.Logf("\nAfter moving cursor to position 1:")
	t.Logf("  Cursor position: %d", interactiveModel.cursor)
	if interactiveModel.cursor < len(interactiveModel.tasks) {
		t.Logf("  Cursor points to: %s", interactiveModel.tasks[interactiveModel.cursor].Title)
	}

	// Toggle with space
	task3ID := interactiveModel.tasks[interactiveModel.cursor].ID
	updatedModel, _ = interactiveModel.Update(tea.KeyMsg{Type: tea.KeySpace})
	interactiveModel = updatedModel.(InteractiveTaskList)

	t.Logf("\nAfter pressing space:")
	t.Logf("  Visible tasks count: %d", len(interactiveModel.tasks))
	t.Logf("  Cursor position: %d", interactiveModel.cursor)

	// Print all visible tasks
	t.Log("  Visible tasks:")
	for i, task := range interactiveModel.tasks {
		if i == interactiveModel.cursor {
			t.Logf("    [%d] %s - %s <-- Cursor", i, task.Title, task.Status)
		} else {
			t.Logf("    [%d] %s - %s", i, task.Title, task.Status)
		}
	}

	// Find Task 3
	for i, task := range interactiveModel.tasks {
		if task.ID == task3ID {
			t.Logf("  Task 3 is at position %d with status %s", i, task.Status)
			break
		}
	}
}
