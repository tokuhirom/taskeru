package internal

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/require"
)

func TestCursorStaysInPlaceWhenTaskBecomesHidden(t *testing.T) {
	// Create test tasks with different timestamps to ensure consistent ordering
	now := time.Now()
	tasks := []Task{
		*ParseTask("Task 1"),
		*ParseTask("Task 2"),
		*ParseTask("Task 3"),
	}

	// Set different updated times to ensure predictable sort order
	// Newer tasks come first in sorting, so to get Task 1 first, it needs the newest time
	tasks[0].Updated = now.Add(-1 * time.Hour) // Task 1 - newest
	tasks[1].Updated = now.Add(-2 * time.Hour) // Task 2 - middle
	tasks[2].Updated = now.Add(-3 * time.Hour) // Task 3 - oldest

	// All tasks start as TODO
	for i := range tasks {
		tasks[i].Status = StatusTODO
	}

	// Make Task 2 an old completed task so it gets hidden
	oldTime := time.Now().AddDate(0, 0, -2) // 2 days ago
	tasks[1].Status = StatusDONE
	tasks[1].CompletedAt = &oldTime

	taskFile := NewTaskFileForTesting(t)
	require.NoError(t, taskFile.AddTask(&tasks[0]))
	require.NoError(t, taskFile.AddTask(&tasks[1]))
	require.NoError(t, taskFile.AddTask(&tasks[2]))

	model, err := NewInteractiveTaskListWithFilter(taskFile, "")
	require.NoError(t, err)

	// Should only see 2 tasks (Task 1 and Task 3)
	if len(model.tasks) != 2 {
		t.Errorf("Should have 2 visible tasks initially, got %d", len(model.tasks))
	}

	// Move cursor to second visible task (Task 3)
	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	interactiveModel := updatedModel.(InteractiveTaskList)

	if interactiveModel.cursor != 1 {
		t.Errorf("Cursor should be at position 1, got %d", interactiveModel.cursor)
	}

	// Verify we're pointing to Task 3
	if interactiveModel.cursor < len(interactiveModel.tasks) {
		currentTask := interactiveModel.tasks[interactiveModel.cursor]
		if currentTask.Title != "Task 3" {
			t.Errorf("After moving cursor down, should point to 'Task 3', but points to '%s'", currentTask.Title)
			for i, task := range interactiveModel.tasks {
				t.Logf("  [%d] %s - %s", i, task.Title, task.Status)
			}
		}
	}

	// Now change Task 3 to DONE (it will stay visible as it's completed today)
	updatedModel, _ = interactiveModel.Update(tea.KeyMsg{Type: tea.KeySpace})
	interactiveModel = updatedModel.(InteractiveTaskList)

	// Task 3 should still be visible (completed today)
	// So we still have 2 visible tasks
	if len(interactiveModel.tasks) != 2 {
		t.Errorf("Should still have 2 visible tasks, got %d", len(interactiveModel.tasks))
		for i, task := range interactiveModel.tasks {
			t.Logf("  [%d] %s - %s", i, task.Title, task.Status)
		}
	}

	// Cursor should follow Task 3 to its new position
	t.Logf("Cursor position: %d", interactiveModel.cursor)
	if interactiveModel.cursor < len(interactiveModel.tasks) {
		currentTask := interactiveModel.tasks[interactiveModel.cursor]
		if currentTask.Title != "Task 3" {
			t.Logf("Tasks after space:")
			for i, task := range interactiveModel.tasks {
				t.Logf("  [%d] %s - %s", i, task.Title, task.Status)
			}
			t.Errorf("Cursor should still point to 'Task 3', but points to '%s'", currentTask.Title)
		}
	}
}

func TestCursorAtEndWhenLastTaskBecomesHidden(t *testing.T) {
	// Create test tasks where some will be visible and some hidden
	now := time.Now()
	tasks := []Task{
		*NewTask("Task 1"),
		*NewTask("Task 2"),
		*NewTask("Old completed task"),
		*NewTask("Task 3"),
	}

	// Set different updated times to ensure predictable sort order
	// Newer tasks come first in sorting
	tasks[0].Updated = now.Add(-1 * time.Hour) // Task 1 - newest
	tasks[1].Updated = now.Add(-2 * time.Hour) // Task 2
	tasks[2].Updated = now.Add(-3 * time.Hour) // Old completed
	tasks[3].Updated = now.Add(-4 * time.Hour) // Task 3 - oldest

	// Task 1 and 2 are TODO
	tasks[0].Status = StatusTODO
	tasks[1].Status = StatusTODO
	tasks[3].Status = StatusTODO

	// Make one task old completed (hidden by default)
	oldTime := time.Now().AddDate(0, 0, -2) // 2 days ago
	tasks[2].Status = StatusDONE
	tasks[2].CompletedAt = &oldTime

	taskFile := NewTaskFileForTesting(t)
	for _, task := range tasks {
		require.NoError(t, taskFile.AddTask(&task))
	}

	model, err := NewInteractiveTaskListWithFilter(taskFile, "")
	require.NoError(t, err)

	// Should see 3 visible tasks initially
	if len(model.tasks) != 3 {
		t.Errorf("Should have 3 visible tasks initially, got %d", len(model.tasks))
	}

	// Move cursor to last visible task (index 2, which is Task 3)
	model.cursor = 2

	// Toggle Task 3 to DONE with space (will stay visible as completed today)
	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeySpace})
	interactiveModel := updatedModel.(InteractiveTaskList)

	// Task 3 should still be visible (completed today)
	if len(interactiveModel.tasks) != 3 {
		t.Errorf("Should still have 3 visible tasks, got %d", len(interactiveModel.tasks))
	}

	// Cursor should follow the task
	if interactiveModel.cursor < len(interactiveModel.tasks) {
		currentTask := interactiveModel.tasks[interactiveModel.cursor]
		if currentTask.Title != "Task 3" {
			t.Errorf("Cursor should still point to 'Task 3', but points to '%s'", currentTask.Title)
		}
	}
}

func TestCursorFollowsTaskWhenStatusChangesButStaysVisible(t *testing.T) {
	// Create test tasks with different priorities to ensure sorting changes
	tasks := []Task{
		*NewTask("High priority task"),
		*NewTask("Normal task"),
		*NewTask("Low priority task"),
	}

	tasks[0].Priority = "high"
	tasks[0].Status = StatusTODO
	tasks[1].Priority = "medium"
	tasks[1].Status = StatusTODO
	tasks[2].Priority = "low"
	tasks[2].Status = StatusTODO

	taskFile := NewTaskFileForTesting(t)
	for _, task := range tasks {
		require.NoError(t, taskFile.AddTask(&task))
	}

	model, err := NewInteractiveTaskListWithFilter(taskFile, "")
	require.NoError(t, err)
	model.showAll = true // Ensure all tasks stay visible

	// Find the "Normal task" position
	normalTaskIdx := -1
	for i, task := range model.tasks {
		if task.Title == "Normal task" {
			normalTaskIdx = i
			break
		}
	}

	// Move cursor to "Normal task"
	model.cursor = normalTaskIdx

	// Change status to DOING (which might change sort order)
	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	interactiveModel := updatedModel.(InteractiveTaskList)

	// Find where "Normal task" ended up after sorting
	normalTaskNewIdx := -1
	for i, task := range interactiveModel.tasks {
		if task.Title == "Normal task" {
			normalTaskNewIdx = i
			break
		}
	}

	if normalTaskNewIdx == -1 {
		t.Error("Normal task should still be visible")
	} else if interactiveModel.cursor != normalTaskNewIdx {
		t.Errorf("Cursor should follow the task to position %d, but is at %d", normalTaskNewIdx, interactiveModel.cursor)
	}
}

func TestMultipleStatusChangesKeepCursorStable(t *testing.T) {
	// Create test tasks
	tasks := []Task{
		*NewTask("Task A"),
		*NewTask("Task B"),
		*NewTask("Task C"),
		*NewTask("Task D"),
	}

	// All start as TODO
	for i := range tasks {
		tasks[i].Status = StatusTODO
	}

	taskFile := NewTaskFileForTesting(t)
	require.NoError(t, taskFile.AddTasks(tasks))

	model, err := NewInteractiveTaskListWithFilter(taskFile, "")
	require.NoError(t, err, "NewInteractiveTaskListWithFilter()")

	// Move to Task B (index 1)
	model.cursor = 1

	// Press 's' multiple times to cycle through all statuses
	statuses := GetAllStatuses()
	for i := 0; i < len(statuses); i++ {
		updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
		interactiveModel := updatedModel.(InteractiveTaskList)
		model = &interactiveModel

		// Cursor should either:
		// 1. Follow the task if it's still visible
		// 2. Stay at the same index if task became hidden
		if model.cursor >= len(model.tasks) && len(model.tasks) > 0 {
			t.Errorf("Cursor %d is out of bounds (tasks count: %d) after %d status changes",
				model.cursor, len(model.tasks), i+1)
		}
	}
}
