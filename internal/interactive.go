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
}

func NewInteractiveTaskList(tasks []Task) *InteractiveTaskList {
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
	}
}

func (m InteractiveTaskList) Init() tea.Cmd {
	return nil
}

func (m InteractiveTaskList) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
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
						// Update the filtered view
						m.tasks[m.cursor] = m.allTasks[i]
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
			// Cancel deletion
			if m.confirmDelete {
				m.confirmDelete = false
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
					
					// Update filtered view
					m.tasks = FilterVisibleTasks(m.allTasks, m.showAll)
					
					// Adjust cursor if necessary
					if m.cursor >= len(m.tasks) && len(m.tasks) > 0 {
						m.cursor = len(m.tasks) - 1
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
		
		// Add color based on status
		var line string
		var statusColor string
		switch task.Status {
		case StatusDONE:
			statusColor = "\x1b[32m" // green
		case StatusDOING:
			statusColor = "\x1b[33m" // yellow
		case StatusWAITING:
			statusColor = "\x1b[34m" // blue
		case StatusWONTDO:
			statusColor = "\x1b[90m" // gray
		default: // TODO
			statusColor = ""
		}
		
		if statusColor != "" {
			line = fmt.Sprintf("%s%s%-7s %s %s", cursor, statusColor, status, priority, task.Title)
		} else {
			line = fmt.Sprintf("%s%-7s %s %s", cursor, status, priority, task.Title)
		}
		
		// Add projects with cyan color
		if len(task.Projects) > 0 {
			for _, project := range task.Projects {
				line += fmt.Sprintf(" \x1b[36m+%s\x1b[0m", project)
			}
		}
		
		// Add completion date for done/wontdo tasks (dim gray)
		if task.Status == StatusDONE || task.Status == StatusWONTDO {
			if task.CompletedAt != nil {
				line += fmt.Sprintf(" \x1b[90m(%s)\x1b[0m", task.CompletedAt.Format("2006-01-02"))
			}
			if statusColor != "" {
				line += "\x1b[0m" // Close the status color
			}
		}
		line += "\n"
		s.WriteString(line)
	}
	
	if m.inputMode {
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
		s.WriteString("\n\nEnter: create • Esc: cancel • Ctrl+A/E: begin/end • Ctrl+F/B: move • Ctrl+H: backspace • Ctrl+K: kill • Ctrl+D: delete")
	} else if m.confirmDelete {
		s.WriteString("\n\n⚠️  Delete this task? (y/n)")
	} else {
		s.WriteString("\n↑/k: up • ↓/j: down • s: cycle status • space: toggle done • a: show all • c: create • e: edit • d: delete • p: projects • r: reload • q: quit")
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