package internal

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type InteractiveTaskList struct {
	allTasks        []Task
	tasks           []Task
	cursor          int
	modified        bool
	showAll         bool
	quit            bool
	confirmDelete   bool
	deletedTaskIDs  []string
	inputMode       bool
	inputBuffer     string
	inputCursor     int // Cursor position in input buffer
	newTaskTitle    string
	shouldReload    bool
	showProjectView bool
	searchMode      bool
	searchQuery     string
	searchCursor    int
	matchingTasks   map[string]bool // Track which tasks match the search
}

func NewInteractiveTaskList(tasks []Task) *InteractiveTaskList {
	// Sort tasks before displaying
	SortTasks(tasks)
	filteredTasks := FilterVisibleTasks(tasks, false)
	return &InteractiveTaskList{
		allTasks:        tasks,
		tasks:           filteredTasks,
		cursor:          0,
		modified:        false,
		showAll:         false,
		confirmDelete:   false,
		deletedTaskIDs:  []string{},
		inputMode:       false,
		inputBuffer:     "",
		inputCursor:     0,
		newTaskTitle:    "",
		shouldReload:    false,
		showProjectView: false,
		searchMode:      false,
		searchQuery:     "",
		searchCursor:    0,
		matchingTasks:   make(map[string]bool),
	}
}

func (m InteractiveTaskList) Init() tea.Cmd {
	return nil
}

func (m InteractiveTaskList) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle search mode
		if m.searchMode {
			searchRunes := []rune(m.searchQuery)
			
			switch msg.Type {
			case tea.KeyEnter:
				// Exit search input mode but keep highlights
				m.searchMode = false
				// Jump to first match if there are any
				if len(m.matchingTasks) > 0 {
					m.jumpToFirstMatch()
				}
			case tea.KeyEsc:
				// Exit search input mode but keep highlights (same as Enter)
				m.searchMode = false
			case tea.KeyCtrlA:
				m.searchCursor = 0
			case tea.KeyCtrlE:
				m.searchCursor = len(searchRunes)
			case tea.KeyCtrlF, tea.KeyRight:
				if m.searchCursor < len(searchRunes) {
					m.searchCursor++
				}
			case tea.KeyCtrlB, tea.KeyLeft:
				if m.searchCursor > 0 {
					m.searchCursor--
				}
			case tea.KeyCtrlH, tea.KeyBackspace:
				if m.searchCursor > 0 && len(searchRunes) > 0 {
					m.searchQuery = string(append(searchRunes[:m.searchCursor-1], searchRunes[m.searchCursor:]...))
					m.searchCursor--
					// Update matching tasks
					m.updateMatches()
				}
			case tea.KeyDelete:
				if m.searchCursor < len(searchRunes) {
					m.searchQuery = string(append(searchRunes[:m.searchCursor], searchRunes[m.searchCursor+1:]...))
					// Update matching tasks
					m.updateMatches()
				}
			case tea.KeyRunes:
				newRunes := append(searchRunes[:m.searchCursor], append(msg.Runes, searchRunes[m.searchCursor:]...)...)
				m.searchQuery = string(newRunes)
				m.searchCursor += len(msg.Runes)
				// Update matching tasks
				m.updateMatches()
				// Jump to first match if we just started typing
				if len(searchRunes) == 0 {
					m.jumpToFirstMatch()
				}
			case tea.KeySpace:
				newRunes := append(searchRunes[:m.searchCursor], append([]rune{' '}, searchRunes[m.searchCursor:]...)...)
				m.searchQuery = string(newRunes)
				m.searchCursor++
				// Update matching tasks
				m.updateMatches()
			default:
				str := msg.String()
				if len(str) == 1 && str != " " {
					newRunes := append(searchRunes[:m.searchCursor], append([]rune(str), searchRunes[m.searchCursor:]...)...)
					m.searchQuery = string(newRunes)
					m.searchCursor++
					// Update matching tasks
					m.updateMatches()
					// Jump to first match if we just started typing
					if len(searchRunes) == 0 {
						m.jumpToFirstMatch()
					}
				}
			}
			return m, nil
		}
		
		// Handle input mode
		if m.inputMode {
			runes := []rune(m.inputBuffer)

			switch msg.Type {
			case tea.KeyEnter:
				// Create new task
				if m.inputBuffer != "" {
					m.newTaskTitle = m.inputBuffer
					m.inputMode = false
					m.inputBuffer = ""
					m.inputCursor = 0
					return m, tea.Quit // Exit to create the task
				}
			case tea.KeyEsc:
				// Cancel input
				m.inputMode = false
				m.inputBuffer = ""
				m.inputCursor = 0
			case tea.KeyCtrlA:
				// Move to beginning of line
				m.inputCursor = 0
			case tea.KeyCtrlE:
				// Move to end of line
				m.inputCursor = len(runes)
			case tea.KeyCtrlF, tea.KeyRight:
				// Move forward one character
				if m.inputCursor < len(runes) {
					m.inputCursor++
				}
			case tea.KeyCtrlB, tea.KeyLeft:
				// Move backward one character
				if m.inputCursor > 0 {
					m.inputCursor--
				}
			case tea.KeyCtrlD:
				// Delete character at cursor
				if m.inputCursor < len(runes) {
					m.inputBuffer = string(append(runes[:m.inputCursor], runes[m.inputCursor+1:]...))
				}
			case tea.KeyCtrlK:
				// Kill to end of line
				if m.inputCursor < len(runes) {
					m.inputBuffer = string(runes[:m.inputCursor])
				}
			case tea.KeyTab:
				// Tab completion for project names
				if m.inputCursor == len(runes) { // Only complete at end of input
					// Find the last '+' before cursor
					lastPlusIdx := -1
					for i := m.inputCursor - 1; i >= 0; i-- {
						if runes[i] == '+' {
							lastPlusIdx = i
							break
						}
						if runes[i] == ' ' {
							break // Stop if we hit a space
						}
					}

					if lastPlusIdx >= 0 && lastPlusIdx < m.inputCursor-1 {
						// Get the partial project name
						partial := string(runes[lastPlusIdx+1 : m.inputCursor])

						// Get all existing projects
						allProjects := GetAllProjects(m.allTasks)

						// Find matching projects
						var matches []string
						for _, project := range allProjects {
							if strings.HasPrefix(project, partial) {
								matches = append(matches, project)
							}
						}

						// If exactly one match, complete it
						if len(matches) == 1 {
							// Replace the partial with the full project name
							m.inputBuffer = string(runes[:lastPlusIdx+1]) + matches[0]
							m.inputCursor = len([]rune(m.inputBuffer))
						}
					}
				}
			case tea.KeyCtrlH, tea.KeyBackspace:
				// Remove character before cursor (Ctrl+H is traditional backspace)
				if m.inputCursor > 0 && len(runes) > 0 {
					m.inputBuffer = string(append(runes[:m.inputCursor-1], runes[m.inputCursor:]...))
					m.inputCursor--
				}
			case tea.KeyDelete:
				// Remove character at cursor
				if m.inputCursor < len(runes) {
					m.inputBuffer = string(append(runes[:m.inputCursor], runes[m.inputCursor+1:]...))
				}
			case tea.KeyRunes:
				// Insert runes at cursor position
				newRunes := append(runes[:m.inputCursor], append(msg.Runes, runes[m.inputCursor:]...)...)
				m.inputBuffer = string(newRunes)
				m.inputCursor += len(msg.Runes)
			case tea.KeySpace:
				// Insert space at cursor position
				newRunes := append(runes[:m.inputCursor], append([]rune{' '}, runes[m.inputCursor:]...)...)
				m.inputBuffer = string(newRunes)
				m.inputCursor++
			default:
				// Handle other single characters
				str := msg.String()
				if len(str) == 1 && str != " " {
					newRunes := append(runes[:m.inputCursor], append([]rune(str), runes[m.inputCursor:]...)...)
					m.inputBuffer = string(newRunes)
					m.inputCursor++
				}
			}
			return m, nil
		}

		switch msg.String() {
		case "ctrl+c", "q":
			m.quit = true
			return m, tea.Quit
		
		case "esc":
			// Clear search highlights if search is active
			if m.searchQuery != "" {
				m.searchQuery = ""
				m.searchCursor = 0
				m.matchingTasks = make(map[string]bool)
			} else {
				// Otherwise quit
				m.quit = true
				return m, tea.Quit
			}

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.tasks)-1 {
				m.cursor++
			}

		case " ":
			// Quick toggle between TODO and DONE (most common transition)
			if m.cursor < len(m.tasks) {
				// Find the task in allTasks and update it
				taskID := m.tasks[m.cursor].ID
				for i := range m.allTasks {
					if m.allTasks[i].ID == taskID {
						if m.allTasks[i].Status == StatusDONE {
							m.allTasks[i].SetStatus(StatusTODO)
						} else {
							m.allTasks[i].SetStatus(StatusDONE)
						}

						// Re-sort and update filtered view
						SortTasks(m.allTasks)
						m.tasks = FilterVisibleTasks(m.allTasks, m.showAll)

						// Find the task's new position and move cursor there
						for j, task := range m.tasks {
							if task.ID == taskID {
								m.cursor = j
								break
							}
						}

						// Ensure cursor is within bounds
						if m.cursor >= len(m.tasks) && len(m.tasks) > 0 {
							m.cursor = len(m.tasks) - 1
						}

						m.modified = true
						break
					}
				}
			}

		case "a":
			// Toggle show all tasks
			m.showAll = !m.showAll
			oldCursorTaskID := ""
			if m.cursor < len(m.tasks) {
				oldCursorTaskID = m.tasks[m.cursor].ID
			}

			// Re-sort and filter
			SortTasks(m.allTasks)
			m.tasks = FilterVisibleTasks(m.allTasks, m.showAll)

			// Try to maintain cursor position on the same task
			if oldCursorTaskID != "" {
				for i, task := range m.tasks {
					if task.ID == oldCursorTaskID {
						m.cursor = i
						break
					}
				}
			}

			// Ensure cursor is within bounds
			if m.cursor >= len(m.tasks) && len(m.tasks) > 0 {
				m.cursor = len(m.tasks) - 1
			}

		case "e":
			// Edit task (open editor)
			if m.cursor < len(m.tasks) {
				return m, tea.Quit
			}

		case "d":
			// Delete task - first press shows confirmation
			if m.cursor < len(m.tasks) && !m.confirmDelete {
				m.confirmDelete = true
			}

		case "y":
			// Confirm deletion
			if m.confirmDelete && m.cursor < len(m.tasks) {
				// Mark task as deleted
				taskID := m.tasks[m.cursor].ID
				m.deletedTaskIDs = append(m.deletedTaskIDs, taskID)

				// Remove from allTasks
				newAllTasks := []Task{}
				for _, t := range m.allTasks {
					if t.ID != taskID {
						newAllTasks = append(newAllTasks, t)
					}
				}
				m.allTasks = newAllTasks

				// Update filtered view
				m.tasks = FilterVisibleTasks(m.allTasks, m.showAll)

				// Adjust cursor if necessary
				if m.cursor >= len(m.tasks) && len(m.tasks) > 0 {
					m.cursor = len(m.tasks) - 1
				}

				m.modified = true
				m.confirmDelete = false
			}

		case "n":
			// Jump to next match when search is active and not in delete confirmation
			if !m.confirmDelete && m.searchQuery != "" {
				m.jumpToNextMatch()
			} else if m.confirmDelete {
				// Cancel deletion
				m.confirmDelete = false
			}

		case "g":
			// Jump to first task
			if !m.confirmDelete && !m.inputMode {
				m.cursor = 0
			}

		case "G":
			// Jump to last task
			if !m.confirmDelete && !m.inputMode && len(m.tasks) > 0 {
				m.cursor = len(m.tasks) - 1
			}

		case "/":
			// Enter search mode
			if !m.confirmDelete {
				m.searchMode = true
				m.searchQuery = ""
				m.searchCursor = 0
				m.matchingTasks = make(map[string]bool)
			}
		
		case "N":
			// Jump to previous match when search is active
			if !m.confirmDelete && m.searchQuery != "" {
				m.jumpToPrevMatch()
			}

		case "c":
			// Create new task
			if !m.confirmDelete {
				m.inputMode = true
				m.inputBuffer = ""
				m.inputCursor = 0
			}

		case "r":
			// Reload tasks
			if !m.confirmDelete && !m.inputMode {
				m.shouldReload = true
				return m, tea.Quit
			}

		case "p":
			// Show project view
			if !m.confirmDelete && !m.inputMode {
				m.showProjectView = true
				return m, tea.Quit
			}

		case "s":
			// Cycle through statuses
			if !m.confirmDelete && !m.inputMode && m.cursor < len(m.tasks) {
				taskIdx := -1
				for i, t := range m.allTasks {
					if t.ID == m.tasks[m.cursor].ID {
						taskIdx = i
						break
					}
				}

				if taskIdx >= 0 {
					currentStatus := m.allTasks[taskIdx].Status
					allStatuses := GetAllStatuses()

					// Find current status index
					currentIdx := 0
					for i, s := range allStatuses {
						if s == currentStatus {
							currentIdx = i
							break
						}
					}

					// Cycle to next status
					nextIdx := (currentIdx + 1) % len(allStatuses)
					m.allTasks[taskIdx].SetStatus(allStatuses[nextIdx])

					// Re-sort and update filtered view
					SortTasks(m.allTasks)
					m.tasks = FilterVisibleTasks(m.allTasks, m.showAll)

					// Find the task's new position and move cursor there
					currentTaskID := m.allTasks[taskIdx].ID
					for i, task := range m.tasks {
						if task.ID == currentTaskID {
							m.cursor = i
							break
						}
					}

					// Ensure cursor is within bounds
					if m.cursor >= len(m.tasks) && len(m.tasks) > 0 {
						m.cursor = len(m.tasks) - 1
					}

					m.modified = true
				}
			}

		case "+":
			// Increase priority
			if !m.confirmDelete && !m.inputMode && m.cursor < len(m.tasks) {
				taskIdx := -1
				currentTaskID := m.tasks[m.cursor].ID
				for i, t := range m.allTasks {
					if t.ID == currentTaskID {
						taskIdx = i
						break
					}
				}

				if taskIdx >= 0 {
					m.allTasks[taskIdx].IncreasePriority()

					// Re-sort and update filtered view
					SortTasks(m.allTasks)
					m.tasks = FilterVisibleTasks(m.allTasks, m.showAll)

					// Find the task's new position and move cursor there
					for i, task := range m.tasks {
						if task.ID == currentTaskID {
							m.cursor = i
							break
						}
					}

					m.modified = true
				}
			}

		case "-":
			// Decrease priority
			if !m.confirmDelete && !m.inputMode && m.cursor < len(m.tasks) {
				taskIdx := -1
				currentTaskID := m.tasks[m.cursor].ID
				for i, t := range m.allTasks {
					if t.ID == currentTaskID {
						taskIdx = i
						break
					}
				}

				if taskIdx >= 0 {
					m.allTasks[taskIdx].DecreasePriority()

					// Re-sort and update filtered view
					SortTasks(m.allTasks)
					m.tasks = FilterVisibleTasks(m.allTasks, m.showAll)

					// Find the task's new position and move cursor there
					for i, task := range m.tasks {
						if task.ID == currentTaskID {
							m.cursor = i
							break
						}
					}

					m.modified = true
				}
			}
		}
	}

	return m, nil
}

func (m InteractiveTaskList) View() string {
	if m.quit {
		return ""
	}

	if len(m.tasks) == 0 {
		return "No tasks found.\n\nPress q to quit."
	}

	var s strings.Builder
	s.WriteString("Tasks:\n\n")

	for i, task := range m.tasks {
		cursor := "  "
		if m.cursor == i {
			cursor = "> "
		}

		status := task.DisplayStatus()
		priority := task.DisplayPriority()

		// Check if this task matches the search (highlight even when not in search mode)
		isMatch := m.searchQuery != "" && m.matchingTasks[task.ID]
		
		// Add color based on status or highlight for search match
		var line string
		var statusColor string
		
		if isMatch {
			// Highlight matching tasks with yellow background
			statusColor = "\x1b[43m\x1b[30m" // yellow background, black text
		} else {
			switch task.Status {
			case StatusDONE:
				statusColor = "\x1b[90m" // gray
			case StatusDOING:
				statusColor = "\x1b[33m" // yellow
			case StatusWAITING:
				statusColor = "\x1b[34m" // blue
			case StatusWONTDO:
				statusColor = "\x1b[90m" // gray
			default: // TODO
				statusColor = "\x1b[37m" // white
			}
		}

		line = fmt.Sprintf("%s%s%-7s %s %s", cursor, statusColor, status, priority, task.Title)

		// Add projects with color
		if len(task.Projects) > 0 {
			for _, project := range task.Projects {
				projectColor := GetProjectColor(project)
				line += fmt.Sprintf(" %s+%s\x1b[0m", projectColor, project)
			}
		}

		// Add completion date for done/wontdo tasks (dim gray)
		if task.Status == StatusDONE || task.Status == StatusWONTDO {
			if task.CompletedAt != nil {
				line += fmt.Sprintf(" \x1b[90m(%s)\x1b[0m", task.CompletedAt.Format("2006-01-02"))
			}
		}
		line += "\x1b[0m" // Always close the status color
		line += "\n"
		s.WriteString(line)
	}

	if m.searchMode {
		// Show search input
		searchRunes := []rune(m.searchQuery)
		var displayStr string
		
		// Add cursor display
		if m.searchCursor == 0 {
			displayStr = "│" + m.searchQuery
		} else if m.searchCursor < len(searchRunes) {
			displayStr = string(searchRunes[:m.searchCursor]) + "│" + string(searchRunes[m.searchCursor:])
		} else {
			displayStr = m.searchQuery + "│"
		}
		
		s.WriteString("\n\n🔍 Search: " + displayStr)
		if m.searchQuery != "" {
			s.WriteString(fmt.Sprintf(" (%d matches)", len(m.matchingTasks)))
		}
		s.WriteString("\n\nEnter: exit input mode • Esc: exit input mode • Ctrl+A/E: begin/end • Ctrl+F/B: move • Ctrl+H: backspace")
	} else if m.inputMode {
		// Display input with cursor
		runes := []rune(m.inputBuffer)
		var displayStr string

		if m.inputCursor == 0 {
			displayStr = "_" + m.inputBuffer
		} else if m.inputCursor >= len(runes) {
			displayStr = m.inputBuffer + "_"
		} else {
			displayStr = string(runes[:m.inputCursor]) + "_" + string(runes[m.inputCursor:])
		}

		s.WriteString("\n\n📝 New task title: " + displayStr)
		s.WriteString("\n\nEnter: create • Esc: cancel • Tab: complete project • Ctrl+A/E: begin/end • Ctrl+F/B: move • Ctrl+H: backspace • Ctrl+K: kill • Ctrl+D: delete")
	} else if m.confirmDelete {
		s.WriteString("\n\n⚠️  Delete this task? (y/n)")
	} else {
		s.WriteString("\n↑/k: up • ↓/j: down • g/G: first/last • +/-: priority • s: status • space: toggle done • /: search")
		if m.searchQuery != "" && !m.searchMode {
			s.WriteString(" • n/N: next/prev match • ESC: clear search")
		}
		s.WriteString(" • a: all • c: create • e: edit • d: delete • p: projects • r: reload • q: quit")
		if m.showAll {
			s.WriteString(" [ALL]")
		}
		if m.modified {
			s.WriteString(" • *modified*")
		}
	}

	return s.String()
}

func (m InteractiveTaskList) ShouldEdit() bool {
	return !m.quit && m.cursor >= 0 && m.cursor < len(m.tasks)
}

func (m InteractiveTaskList) GetSelectedTask() *Task {
	if m.cursor >= 0 && m.cursor < len(m.tasks) {
		return &m.tasks[m.cursor]
	}
	return nil
}

func (m InteractiveTaskList) GetTasks() []Task {
	return m.allTasks
}

func (m InteractiveTaskList) IsModified() bool {
	return m.modified
}

func (m InteractiveTaskList) GetDeletedTaskIDs() []string {
	return m.deletedTaskIDs
}

func (m InteractiveTaskList) GetNewTaskTitle() string {
	return m.newTaskTitle
}

func (m InteractiveTaskList) ShouldReload() bool {
	return m.shouldReload
}

// updateMatches updates which tasks match the current search query
func (m *InteractiveTaskList) updateMatches() {
	m.matchingTasks = make(map[string]bool)
	
	if m.searchQuery == "" {
		return
	}
	
	query := strings.ToLower(m.searchQuery)
	
	for _, task := range m.tasks {
		// Search in title, projects, and note
		titleMatch := strings.Contains(strings.ToLower(task.Title), query)
		noteMatch := strings.Contains(strings.ToLower(task.Note), query)
		
		// Check projects
		projectMatch := false
		for _, project := range task.Projects {
			if strings.Contains(strings.ToLower(project), query) {
				projectMatch = true
				break
			}
		}
		
		if titleMatch || noteMatch || projectMatch {
			m.matchingTasks[task.ID] = true
		}
	}
}

// jumpToFirstMatch moves cursor to the first matching task
func (m *InteractiveTaskList) jumpToFirstMatch() {
	for i, task := range m.tasks {
		if m.matchingTasks[task.ID] {
			m.cursor = i
			break
		}
	}
}

// jumpToNextMatch moves cursor to the next matching task
func (m *InteractiveTaskList) jumpToNextMatch() {
	if len(m.matchingTasks) == 0 {
		return
	}
	
	// Start searching from the next position
	startPos := m.cursor + 1
	
	// Search from current position to end
	for i := startPos; i < len(m.tasks); i++ {
		if m.matchingTasks[m.tasks[i].ID] {
			m.cursor = i
			return
		}
	}
	
	// Wrap around to beginning
	for i := 0; i < startPos && i < len(m.tasks); i++ {
		if m.matchingTasks[m.tasks[i].ID] {
			m.cursor = i
			return
		}
	}
}

// jumpToPrevMatch moves cursor to the previous matching task
func (m *InteractiveTaskList) jumpToPrevMatch() {
	if len(m.matchingTasks) == 0 {
		return
	}
	
	// Start searching from the previous position
	startPos := m.cursor - 1
	
	// Search from current position to beginning
	for i := startPos; i >= 0; i-- {
		if m.matchingTasks[m.tasks[i].ID] {
			m.cursor = i
			return
		}
	}
	
	// Wrap around to end
	for i := len(m.tasks) - 1; i > startPos && i >= 0; i-- {
		if m.matchingTasks[m.tasks[i].ID] {
			m.cursor = i
			return
		}
	}
}

func (m InteractiveTaskList) ShouldShowProjectView() bool {
	return m.showProjectView
}

func ShowInteractiveTaskList(tasks []Task) ([]Task, bool, *Task, []string, string, bool, bool, error) {
	model := NewInteractiveTaskList(tasks)
	p := tea.NewProgram(model)

	result, err := p.Run()
	if err != nil {
		return nil, false, nil, nil, "", false, false, err
	}

	finalModel := result.(InteractiveTaskList)

	// Check if user wants to edit a task
	if finalModel.ShouldEdit() {
		return finalModel.GetTasks(), finalModel.IsModified(), finalModel.GetSelectedTask(), finalModel.GetDeletedTaskIDs(), finalModel.GetNewTaskTitle(), finalModel.ShouldReload(), finalModel.ShouldShowProjectView(), nil
	}

	return finalModel.GetTasks(), finalModel.IsModified(), nil, finalModel.GetDeletedTaskIDs(), finalModel.GetNewTaskTitle(), finalModel.ShouldReload(), finalModel.ShouldShowProjectView(), nil
}
