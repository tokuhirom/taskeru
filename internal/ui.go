package internal

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type TaskSelector struct {
	tasks    []Task
	cursor   int
	selected int
	quit     bool
}

func NewTaskSelector(tasks []Task) *TaskSelector {
	return &TaskSelector{
		tasks:    tasks,
		cursor:   0,
		selected: -1,
	}
}

func (m TaskSelector) Init() tea.Cmd {
	return nil
}

func (m TaskSelector) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.quit = true
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.tasks)-1 {
				m.cursor++
			}

		case "enter", " ":
			m.selected = m.cursor
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m TaskSelector) View() string {
	if m.quit {
		return ""
	}

	if len(m.tasks) == 0 {
		return "No tasks found.\n\nPress q to quit."
	}

	var s strings.Builder
	s.WriteString("Select a task to edit:\n\n")

	for i, task := range m.tasks {
		cursor := "  "
		if m.cursor == i {
			cursor = "> "
		}

		status := task.DisplayStatus()
		priority := task.DisplayPriority()

		line := fmt.Sprintf("%s%s %s %s\n", cursor, status, priority, task.Title)
		s.WriteString(line)
	}

	s.WriteString("\n↑/k: up • ↓/j: down • enter: select • q/esc: quit")

	return s.String()
}

func (m TaskSelector) GetSelected() int {
	return m.selected
}

func SelectTask(tasks []Task) (*Task, error) {
	if len(tasks) == 0 {
		return nil, fmt.Errorf("no tasks available")
	}

	selector := NewTaskSelector(tasks)
	p := tea.NewProgram(selector)

	result, err := p.Run()
	if err != nil {
		return nil, err
	}

	finalModel := result.(TaskSelector)
	if finalModel.selected < 0 || finalModel.selected >= len(tasks) {
		return nil, fmt.Errorf("no task selected")
	}

	return &tasks[finalModel.selected], nil
}
