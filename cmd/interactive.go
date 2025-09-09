package cmd

import (
	"fmt"
	"taskeru/internal"

	tea "github.com/charmbracelet/bubbletea"
)

func InteractiveCommandWithFilter(projectFilter string, taskFile *internal.TaskFile) error {
	model, err := internal.NewInteractiveTaskListWithFilter(taskFile, projectFilter)
	if err != nil {
		return fmt.Errorf("failed to create interactive model: %w", err)
	}

	// Start Bubble Tea program with AltScreen
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err = p.Run(); err != nil {
		return fmt.Errorf("failed to run interactive UI: %w", err)
	}

	return nil
}
