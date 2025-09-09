package internal

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/require"
)

func TestProjectSelectionMode(t *testing.T) {
	// Create test tasks with different projects
	tasks := []Task{
		*NewTask("Work task 1"),
		*NewTask("Personal task"),
		*NewTask("Work task 2"),
		*NewTask("Home task"),
		*NewTask("Task without project"),
	}

	tasks[0].Projects = []string{"work"}
	tasks[1].Projects = []string{"personal"}
	tasks[2].Projects = []string{"work", "urgent"}
	tasks[3].Projects = []string{"home"}
	// tasks[4] has no projects

	taskFile := NewTaskFileForTesting(t)
	for _, task := range tasks {
		if err := taskFile.AddTask(&task); err != nil {
			t.Fatalf("Failed to add task: %v", err)
		}
	}

	// Create interactive model
	model, err := NewInteractiveTaskListWithFilter(taskFile, "")
	require.NoError(t, err, "NewInteractiveTaskListWithFilter()")

	// Test entering project select mode with 'p'
	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")})
	m := updatedModel.(InteractiveTaskList)

	// Check that we're in project select mode
	if !m.projectSelectMode {
		t.Error("Should be in project select mode after pressing 'p'")
	}

	view := m.View()
	if !strings.Contains(view, "📁 Select project filter:") {
		t.Error("View should show project selection UI")
	}
	if !strings.Contains(view, "[All tasks]") {
		t.Error("View should show 'All tasks' option")
	}
	if !strings.Contains(view, "home") {
		t.Error("View should list 'home' project")
	}
	if !strings.Contains(view, "personal") {
		t.Error("View should list 'personal' project")
	}
	if !strings.Contains(view, "urgent") {
		t.Error("View should list 'urgent' project")
	}
	if !strings.Contains(view, "work") {
		t.Error("View should list 'work' project")
	}

	// Test navigation - move down to select 'work' (4 down presses: All tasks -> home -> personal -> urgent -> work)
	for i := 0; i < 4; i++ {
		m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = m2.(InteractiveTaskList)
	}

	// Press enter to select 'work'
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = m2.(InteractiveTaskList)

	// Check that we're back to normal mode with filter applied
	if m.projectSelectMode {
		t.Error("Should not be in project select mode after selection")
	}
	if m.projectFilter != "work" {
		t.Errorf("Project filter should be 'work', got '%s'", m.projectFilter)
	}

	view = m.View()
	if !strings.Contains(view, "Tasks for project:") || !strings.Contains(view, "+work") {
		t.Error("View should show project filter in title")
	}

	// Check that only work tasks are visible
	for _, task := range m.tasks {
		hasWork := false
		for _, project := range task.Projects {
			if project == "work" {
				hasWork = true
				break
			}
		}
		if !hasWork {
			t.Errorf("Task '%s' should not be visible with 'work' filter", task.Title)
		}
	}

	// Test clearing filter - press 'p' again
	m2, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")})
	m = m2.(InteractiveTaskList)

	if !m.projectSelectMode {
		t.Error("Should be in project select mode after pressing 'p' again")
	}

	// The cursor should be positioned at the current filter (work)
	// Move cursor to position 0 (All tasks)
	for m.projectCursor > 0 {
		m2, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
		m = m2.(InteractiveTaskList)
	}

	// Select "All tasks" (cursor is now at 0)
	m2, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = m2.(InteractiveTaskList)

	if m.projectFilter != "" {
		t.Errorf("Project filter should be empty after selecting 'All tasks', got '%s'", m.projectFilter)
	}

	view = m.View()
	if strings.Contains(view, "[Project:") {
		t.Error("View should not show project filter when no filter is set")
	}
}

func TestProjectSelectionCancel(t *testing.T) {
	// Create test tasks
	tasks := []Task{
		*ParseTask("Work task +work"),
	}

	taskFile := NewTaskFileForTesting(t)
	for _, task := range tasks {
		if err := taskFile.AddTask(&task); err != nil {
			t.Fatalf("Failed to add task: %v", err)
		}
	}

	// Create interactive model with existing filter
	model, err := NewInteractiveTaskListWithFilter(taskFile, "work")
	require.NoError(t, err, "NewInteractiveTaskListWithFilter()")

	// Enter project select mode
	m2, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")})
	m := m2.(InteractiveTaskList)

	if !m.projectSelectMode {
		t.Error("Should be in project select mode")
	}

	// Cancel with Esc
	m2, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = m2.(InteractiveTaskList)

	if m.projectSelectMode {
		t.Error("Should not be in project select mode after Esc")
	}
	if m.projectFilter != "work" {
		t.Errorf("Project filter should remain 'work' after canceling, got '%s'", m.projectFilter)
	}

	// Test canceling with 'q' as well
	m2, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")})
	m = m2.(InteractiveTaskList)

	m2, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	m = m2.(InteractiveTaskList)

	if m.projectSelectMode {
		t.Error("Should not be in project select mode after 'q'")
	}
	if m.projectFilter != "work" {
		t.Errorf("Project filter should remain 'work' after canceling with 'q', got '%s'", m.projectFilter)
	}
}

func TestProjectSelectionCurrentFilterHighlight(t *testing.T) {
	// Create test tasks
	tasks := []Task{
		*ParseTask("Work task +work"),
		*ParseTask("Personal task +personal"),
	}

	taskFile := NewTaskFileForTesting(t)
	for _, task := range tasks {
		if err := taskFile.AddTask(&task); err != nil {
			t.Fatalf("Failed to add task: %v", err)
		}
	}

	// Create model with 'work' filter
	model, err := NewInteractiveTaskListWithFilter(taskFile, "work")
	require.NoError(t, err, "NewInteractiveTaskListWithFilter()")

	// Enter project select mode
	m2, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")})
	m := m2.(InteractiveTaskList)

	// Check that cursor is positioned at 'work'
	// Projects are sorted alphabetically: personal, work
	// So cursor should be at index 2 (0: All tasks, 1: personal, 2: work)
	if m.projectCursor != 2 {
		t.Errorf("Cursor should be at position 2 for 'work', got %d", m.projectCursor)
	}
}
