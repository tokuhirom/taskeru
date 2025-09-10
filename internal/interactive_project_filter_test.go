package internal

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInteractiveWithProjectFilter(t *testing.T) {
	// Create test tasks with different projects
	tasks := []Task{
		*NewTask("Work task 1"),
		*NewTask("Personal task"),
		*NewTask("Work task 2"),
		*NewTask("No project task"),
	}

	tasks[0].Projects = []string{"work"}
	tasks[1].Projects = []string{"personal"}
	tasks[2].Projects = []string{"work", "urgent"}
	// tasks[3] has no projects

	taskFile := NewTaskFileForTesting(t)
	for _, task := range tasks {
		require.NoError(t, taskFile.AddTask(&task))
	}

	// Test without filter
	model, err := NewInteractiveTaskListWithFilter(taskFile, "")
	require.NoError(t, err, "NewInteractiveTaskListWithFilter()")
	if len(model.tasks) != 4 {
		t.Errorf("Without filter, should have 4 tasks, got %d", len(model.tasks))
	}

	// Test with work filter
	modelWithFilter, err := NewInteractiveTaskListWithFilter(taskFile, "work")
	require.NoError(t, err, "NewInteractiveTaskListWithFilter() with filter")
	if len(modelWithFilter.tasks) != 2 {
		t.Errorf("With 'work' filter, should have 2 tasks, got %d", len(modelWithFilter.tasks))
	}

	// Verify that only work tasks are visible
	for _, task := range modelWithFilter.tasks {
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

	// Test with personal filter
	modelPersonal, err := NewInteractiveTaskListWithFilter(taskFile, "personal")
	require.NoError(t, err, "NewInteractiveTaskListWithFilter() with personal filter")
	if len(modelPersonal.tasks) != 1 {
		t.Errorf("With 'personal' filter, should have 1 task, got %d", len(modelPersonal.tasks))
	}
	if modelPersonal.tasks[0].Title != "Personal task" {
		t.Errorf("With 'personal' filter, should show 'Personal task', got '%s'", modelPersonal.tasks[0].Title)
	}
}

func TestProjectFilterWithShowAll(t *testing.T) {
	// Create test tasks with different statuses
	tasks := []Task{
		*NewTask("Work task TODO"),
		*NewTask("Work task DONE"),
		*NewTask("Personal task TODO"),
		*NewTask("Personal task DONE"),
	}

	tasks[0].Projects = []string{"work"}
	tasks[0].Status = StatusTODO

	tasks[1].Projects = []string{"work"}
	tasks[1].Status = StatusDONE
	tasks[1].SetStatus(StatusDONE) // This sets CompletedAt

	tasks[2].Projects = []string{"personal"}
	tasks[2].Status = StatusTODO

	tasks[3].Projects = []string{"personal"}
	tasks[3].Status = StatusDONE
	tasks[3].SetStatus(StatusDONE) // This sets CompletedAt

	taskFile := NewTaskFileForTesting(t)
	for _, task := range tasks {
		require.NoError(t, taskFile.AddTask(&task))
	}

	// Test work filter without showAll (should hide completed tasks completed today)
	modelWork, err := NewInteractiveTaskListWithFilter(taskFile, "work")
	require.NoError(t, err, "NewInteractiveTaskListWithFilter() with work filter")
	// Since tasks are completed today, they should still be visible
	if len(modelWork.tasks) != 2 {
		t.Errorf("With 'work' filter, should show 2 tasks (including today's completed), got %d", len(modelWork.tasks))
	}

	// Toggle showAll to ensure all work tasks are visible
	modelWork.showAll = true
	modelWork.applyFilters()
	if len(modelWork.tasks) != 2 {
		t.Errorf("With 'work' filter and showAll, should have 2 tasks, got %d", len(modelWork.tasks))
	}
}

func TestProjectFilterTitle(t *testing.T) {
	tasks := []Task{
		*NewTask("Test task"),
	}
	tasks[0].Projects = []string{"myproject"}

	taskFile := NewTaskFileForTesting(t)
	for _, task := range tasks {
		require.NoError(t, taskFile.AddTask(&task))
	}

	// Test that project filter is shown in title
	model, err := NewInteractiveTaskListWithFilter(taskFile, "myproject")
	require.NoError(t, err, "NewInteractiveTaskListWithFilter() with project filter")
	view := model.View()

	// Check that the view contains the project filter (format: "Tasks for project: +myproject")
	if !containsStr(view, "Tasks for project:") || !containsStr(view, "+myproject") {
		t.Errorf("View should contain 'Tasks for project: +myproject' when project filter is set")
	}

	// Test without filter
	modelNoFilter, err := NewInteractiveTaskListWithFilter(taskFile, "")
	require.NoError(t, err, "NewInteractiveTaskListWithFilter() without filter")
	viewNoFilter := modelNoFilter.View()

	if containsStr(viewNoFilter, "Tasks for project:") {
		t.Error("View should not contain project filter when no filter is set")
	}
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || containsStrHelper(s, substr))
}

func containsStrHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
