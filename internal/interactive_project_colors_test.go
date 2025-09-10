package internal

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/require"
)

func TestProjectSelectionColors(t *testing.T) {
	// Create test tasks with different projects
	tasks := []Task{
		*NewTask("Work task 1"),
		*NewTask("Work task 2"),
		*NewTask("Personal task 1"),
		*NewTask("Personal task 2"),
		*NewTask("Home task"),
		*NewTask("Urgent work"),
	}

	tasks[0].Projects = []string{"work"}
	tasks[1].Projects = []string{"work"}
	tasks[2].Projects = []string{"personal"}
	tasks[3].Projects = []string{"personal"}
	tasks[4].Projects = []string{"home"}
	tasks[5].Projects = []string{"work", "urgent"}

	taskFile := NewTaskFileForTesting(t)
	for i := range tasks {
		require.NoError(t, taskFile.AddTask(&tasks[i]))
	}

	// Create interactive model
	model, err := NewInteractiveTaskListWithFilter(taskFile, "")
	require.NoError(t, err)

	// Enter project select mode
	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")})
	m := updatedModel.(*InteractiveTaskList)

	view := m.View()

	// Check that we're in project select mode
	if !strings.Contains(view, "📁 Select project filter:") {
		t.Error("Should show project selection header")
	}

	// Check for "All tasks" with count
	if !strings.Contains(view, "[All tasks] (6)") {
		t.Error("Should show 'All tasks' with count of 6")
	}

	// Check for colored projects with counts
	// work project should have color code and count of 3
	if !strings.Contains(view, "+work") || !strings.Contains(view, "(3)") {
		t.Error("Should show colored 'work' project with count of 3")
	}

	// personal project should have count of 2
	if !strings.Contains(view, "+personal") || !strings.Contains(view, "(2)") {
		t.Error("Should show colored 'personal' project with count of 2")
	}

	// home project should have count of 1
	if !strings.Contains(view, "+home") || !strings.Contains(view, "(1)") {
		t.Error("Should show colored 'home' project with count of 1")
	}

	// urgent project should have count of 1
	if !strings.Contains(view, "+urgent") || !strings.Contains(view, "(1)") {
		t.Error("Should show colored 'urgent' project with count of 1")
	}

	// Check that ANSI color codes are present (indicating colors)
	if !strings.Contains(view, "\x1b[") {
		t.Error("Should contain ANSI color codes for project colors")
	}
}

func TestProjectSelectionWithHiddenTasks(t *testing.T) {
	// Create test tasks with some completed
	tasks := []Task{
		*NewTask("Work task active"),
		*NewTask("Work task done"),
		*NewTask("Personal task active"),
	}

	tasks[0].Projects = []string{"work"}
	tasks[0].Status = StatusTODO

	tasks[1].Projects = []string{"work"}
	tasks[1].Status = StatusDONE
	tasks[1].SetStatus(StatusDONE) // This sets CompletedAt to now

	tasks[2].Projects = []string{"personal"}
	tasks[2].Status = StatusTODO

	taskFile := NewTaskFileForTesting(t)
	for i := range tasks {
		require.NoError(t, taskFile.AddTask(&tasks[i]))
	}

	// Create interactive model
	model, err := NewInteractiveTaskListWithFilter(taskFile, "")
	require.NoError(t, err)

	// Enter project select mode
	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")})
	m := updatedModel.(*InteractiveTaskList)

	view := m.View()

	// Since the completed task was completed today, it should still be visible
	// So work should show 2 tasks
	if !strings.Contains(view, "+work") || !strings.Contains(view, "(2)") {
		t.Error("Should show 'work' project with 2 visible tasks (including today's completed)")
	}

	// Personal should show 1 task
	if !strings.Contains(view, "+personal") || !strings.Contains(view, "(1)") {
		t.Error("Should show 'personal' project with 1 visible task")
	}
}
