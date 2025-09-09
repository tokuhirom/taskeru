package internal

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type InteractiveTaskList struct {
	allTasks          []Task
	tasks             []Task
	cursor            int
	modified          bool
	showAll           bool
	quit              bool
	confirmDelete     bool
	deletedTaskIDs    []string
	inputMode         bool
	inputBuffer       string
	inputCursor       int // Cursor position in input buffer
	newTaskTitle      string
	shouldReload      bool
	searchMode        bool
	searchQuery       string
	searchCursor      int
	matchingTasks     map[string]bool // Track which tasks match the search
	dateEditMode      string          // "deadline" or "scheduled"
	dateEditBuffer    string
	dateEditCursor    int
	projectFilter     string // Filter tasks by project
	projectSelectMode bool   // Mode for selecting project filter
	projectCursor     int    // Cursor position in project list
	width             int    // Terminal width
	height            int    // Terminal height
}

func NewInteractiveTaskListWithFilter(tasks []Task, projectFilter string) *InteractiveTaskList {
	// Sort tasks before displaying
	SortTasks(tasks)

	// Apply project filter if specified
	var filteredByProject []Task
	if projectFilter != "" {
		filteredByProject = FilterTasksByProject(tasks, projectFilter)
	} else {
		filteredByProject = tasks
	}

	// Then apply visibility filter
	filteredTasks := FilterVisibleTasks(filteredByProject, false)

	return &InteractiveTaskList{
		allTasks:          tasks,
		tasks:             filteredTasks,
		cursor:            0,
		modified:          false,
		showAll:           false,
		confirmDelete:     false,
		deletedTaskIDs:    []string{},
		inputMode:         false,
		inputBuffer:       "",
		inputCursor:       0,
		newTaskTitle:      "",
		shouldReload:      false,
		searchMode:        false,
		searchQuery:       "",
		searchCursor:      0,
		matchingTasks:     make(map[string]bool),
		dateEditMode:      "",
		dateEditBuffer:    "",
		dateEditCursor:    0,
		projectFilter:     projectFilter,
		projectSelectMode: false,
		projectCursor:     0,
		width:             80, // Default width
		height:            24, // Default height
	}
}

func (m InteractiveTaskList) Init() tea.Cmd {
	return nil
}

// truncateTaskLine truncates the task line to fit within the terminal width
// It prioritizes showing project names by truncating the title if necessary
func (m InteractiveTaskList) truncateTaskLine(cursor string, statusColor string, status string, priority string, title string, projects []string, additionalInfo string) string {
	// Calculate base components length
	baseLen := len(cursor) + len(status) + 1 + len(priority) + 1 // cursor + status + space + priority + space

	// Calculate projects display length (including color codes, which we'll estimate)
	projectsStr := ""
	projectsDisplayLen := 0
	for _, project := range projects {
		projectsStr += fmt.Sprintf(" +%s", project)
		projectsDisplayLen += len(project) + 2 // +project and space
	}

	// Calculate additional info length (dates, etc)
	additionalDisplayLen := 0
	if additionalInfo != "" {
		// Remove ANSI codes for length calculation
		strippedInfo := m.stripAnsiCodes(additionalInfo)
		additionalDisplayLen = len(strippedInfo)
	}

	// Available width for title
	availableWidth := m.width - baseLen - projectsDisplayLen - additionalDisplayLen - 5 // 5 for safety margin

	// Truncate title if necessary
	displayTitle := title
	if availableWidth > 10 && len(title) > availableWidth { // Keep at least 10 chars for title
		displayTitle = title[:availableWidth-3] + "..."
	} else if availableWidth <= 10 && len(title) > 10 {
		// If very limited space, show minimal title
		displayTitle = title[:7] + "..."
	}

	return fmt.Sprintf("%s%s%-7s %s %s", cursor, statusColor, status, priority, displayTitle)
}

// stripAnsiCodes removes ANSI escape codes from a string
func (m InteractiveTaskList) stripAnsiCodes(s string) string {
	// Simple implementation - removes common ANSI codes
	result := s
	for {
		start := strings.Index(result, "\x1b[")
		if start == -1 {
			break
		}
		end := strings.Index(result[start:], "m")
		if end == -1 {
			break
		}
		result = result[:start] + result[start+end+1:]
	}
	return result
}

// applyFilters applies project filter and visibility filter to tasks
func (m *InteractiveTaskList) applyFilters() {
	// Apply project filter first
	var filteredByProject []Task
	if m.projectFilter != "" {
		filteredByProject = FilterTasksByProject(m.allTasks, m.projectFilter)
	} else {
		filteredByProject = m.allTasks
	}

	// Then apply visibility filter
	m.tasks = FilterVisibleTasks(filteredByProject, m.showAll)
}

// getAvailableProjects returns sorted list of unique projects from all tasks
func (m *InteractiveTaskList) getAvailableProjects() []string {
	projectMap := make(map[string]bool)
	for _, task := range m.allTasks {
		for _, project := range task.Projects {
			projectMap[project] = true
		}
	}

	var projects []string
	for project := range projectMap {
		projects = append(projects, project)
	}

	// Sort projects alphabetically
	sort.Strings(projects)
	return projects
}

func (m InteractiveTaskList) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Update terminal dimensions
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		// Handle project select mode
		if m.projectSelectMode {
			projects := m.getAvailableProjects()

			switch msg.String() {
			case "esc", "q":
				m.projectSelectMode = false
				m.projectCursor = 0
			case "enter":
				if m.projectCursor == 0 {
					// "All tasks" selected
					m.projectFilter = ""
				} else if m.projectCursor <= len(projects) {
					m.projectFilter = projects[m.projectCursor-1]
				}
				m.projectSelectMode = false
				m.projectCursor = 0
				// Re-apply filters
				m.applyFilters()
			case "up", "k":
				if m.projectCursor > 0 {
					m.projectCursor--
				}
			case "down", "j":
				if m.projectCursor < len(projects) {
					m.projectCursor++
				}
			}
			return m, nil
		}

		// Handle date edit mode
		if m.dateEditMode != "" {
			dateRunes := []rune(m.dateEditBuffer)

			switch msg.Type {
			case tea.KeyEnter:
				// Apply the date change
				if m.cursor < len(m.tasks) {
					taskID := m.tasks[m.cursor].ID
					for i := range m.allTasks {
						if m.allTasks[i].ID == taskID {
							// Parse the date
							var parsedDate *time.Time
							if m.dateEditBuffer != "" {
								// Try to parse using ExtractDeadlineFromTitle logic
								_, deadline := ExtractDeadlineFromTitle("dummy due:" + m.dateEditBuffer)
								if m.dateEditMode == "deadline" {
									parsedDate = deadline
								} else if m.dateEditMode == "scheduled" && deadline != nil {
									// Convert to start of day for scheduled date
									t := time.Date(deadline.Year(), deadline.Month(), deadline.Day(), 0, 0, 0, 0, deadline.Location())
									parsedDate = &t
								}
							}

							// Update the task
							switch m.dateEditMode {
							case "deadline":
								m.allTasks[i].DueDate = parsedDate
							case "scheduled":
								m.allTasks[i].ScheduledDate = parsedDate
							}
							m.allTasks[i].Updated = time.Now()

							// Update filtered view
							m.applyFilters()
							m.modified = true
							break
						}
					}
				}
				m.dateEditMode = ""
				m.dateEditBuffer = ""
				m.dateEditCursor = 0
			case tea.KeyEsc:
				// Cancel date edit
				m.dateEditMode = ""
				m.dateEditBuffer = ""
				m.dateEditCursor = 0
			case tea.KeyCtrlA:
				m.dateEditCursor = 0
			case tea.KeyCtrlE:
				m.dateEditCursor = len(dateRunes)
			case tea.KeyCtrlF, tea.KeyRight:
				if m.dateEditCursor < len(dateRunes) {
					m.dateEditCursor++
				}
			case tea.KeyCtrlB, tea.KeyLeft:
				if m.dateEditCursor > 0 {
					m.dateEditCursor--
				}
			case tea.KeyCtrlH, tea.KeyBackspace:
				if m.dateEditCursor > 0 && len(dateRunes) > 0 {
					m.dateEditBuffer = string(append(dateRunes[:m.dateEditCursor-1], dateRunes[m.dateEditCursor:]...))
					m.dateEditCursor--
				}
			case tea.KeyDelete:
				if m.dateEditCursor < len(dateRunes) {
					m.dateEditBuffer = string(append(dateRunes[:m.dateEditCursor], dateRunes[m.dateEditCursor+1:]...))
				}
			case tea.KeyRunes:
				newRunes := append(dateRunes[:m.dateEditCursor], append(msg.Runes, dateRunes[m.dateEditCursor:]...)...)
				m.dateEditBuffer = string(newRunes)
				m.dateEditCursor += len(msg.Runes)
			case tea.KeySpace:
				newRunes := append(dateRunes[:m.dateEditCursor], append([]rune{' '}, dateRunes[m.dateEditCursor:]...)...)
				m.dateEditBuffer = string(newRunes)
				m.dateEditCursor++
			default:
				str := msg.String()
				if len(str) == 1 && str != " " {
					newRunes := append(dateRunes[:m.dateEditCursor], append([]rune(str), dateRunes[m.dateEditCursor:]...)...)
					m.dateEditBuffer = string(newRunes)
					m.dateEditCursor++
				}
			}
			return m, nil
		}

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
						m.applyFilters()

						// Find the task's new position and move cursor there
						foundTask := false
						for j, task := range m.tasks {
							if task.ID == taskID {
								m.cursor = j
								foundTask = true
								break
							}
						}

						// If task is no longer visible (e.g., DONE task hidden), keep cursor at same position
						if !foundTask {
							// Ensure cursor is within bounds
							if m.cursor >= len(m.tasks) && len(m.tasks) > 0 {
								m.cursor = len(m.tasks) - 1
							}
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
			m.applyFilters()

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

		case "D":
			// Set deadline for current task
			if m.cursor < len(m.tasks) && !m.confirmDelete {
				m.dateEditMode = "deadline"
				// Pre-fill with current deadline if exists
				task := m.tasks[m.cursor]
				if task.DueDate != nil {
					m.dateEditBuffer = task.DueDate.Format("2006-01-02")
				} else {
					m.dateEditBuffer = ""
				}
				m.dateEditCursor = len(m.dateEditBuffer)
			}

		case "S":
			// Set scheduled date for current task
			if m.cursor < len(m.tasks) && !m.confirmDelete {
				m.dateEditMode = "scheduled"
				// Pre-fill with current scheduled date if exists
				task := m.tasks[m.cursor]
				if task.ScheduledDate != nil {
					m.dateEditBuffer = task.ScheduledDate.Format("2006-01-02")
				} else {
					m.dateEditBuffer = ""
				}
				m.dateEditCursor = len(m.dateEditBuffer)
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
				m.applyFilters()

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
			// Enter project select mode
			if !m.confirmDelete && !m.inputMode {
				m.projectSelectMode = true
				m.projectCursor = 0
				// Find current project in list
				if m.projectFilter != "" {
					projects := m.getAvailableProjects()
					for i, p := range projects {
						if p == m.projectFilter {
							m.projectCursor = i + 1 // +1 because 0 is "All tasks"
							break
						}
					}
				}
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
					// Save the task ID before any changes
					currentTaskID := m.allTasks[taskIdx].ID
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
					m.applyFilters()

					// Find the task's new position and move cursor there
					foundTask := false
					for i, task := range m.tasks {
						if task.ID == currentTaskID {
							m.cursor = i
							foundTask = true
							break
						}
					}

					// If task is no longer visible (e.g., DONE task hidden), keep cursor at same position
					if !foundTask {
						// Ensure cursor is within bounds
						if m.cursor >= len(m.tasks) && len(m.tasks) > 0 {
							m.cursor = len(m.tasks) - 1
						}
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
					m.applyFilters()

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
					m.applyFilters()

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
	if m.projectFilter != "" {
		// Show project filter with color and count
		projectColor := GetProjectColor(m.projectFilter)
		totalCount := len(m.tasks)
		// Calculate hidden count only for this project
		allProjectTasks := FilterTasksByProject(m.allTasks, m.projectFilter)
		hiddenCount := len(allProjectTasks) - totalCount
		s.WriteString(fmt.Sprintf("Tasks for project: %s+%s\x1b[0m", projectColor, m.projectFilter))
		if totalCount > 0 || hiddenCount > 0 {
			s.WriteString(fmt.Sprintf(" (%d task", totalCount))
			if totalCount != 1 {
				s.WriteString("s")
			}
			if hiddenCount > 0 {
				s.WriteString(fmt.Sprintf(", %d hidden", hiddenCount))
			}
			s.WriteString(")")
		}
		s.WriteString("\n\n")
	} else {
		s.WriteString("Tasks:\n\n")
	}

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

		// Build additional info (dates)
		additionalInfo := ""

		// Add scheduled date if future
		if task.IsFutureScheduled() {
			schedIn := time.Until(*task.ScheduledDate)
			if schedIn < 24*time.Hour {
				// Starts tomorrow - green
				additionalInfo += " \x1b[32m(starts tomorrow)\x1b[0m"
			} else if schedIn < 7*24*time.Hour {
				// Starts this week - dim green
				additionalInfo += fmt.Sprintf(" \x1b[92m(starts %s)\x1b[0m", task.ScheduledDate.Format("Mon"))
			} else {
				// Starts later - dim
				additionalInfo += fmt.Sprintf(" \x1b[90m(starts %s)\x1b[0m", task.ScheduledDate.Format("01-02"))
			}
		}

		// Add completion date for done/wontdo tasks (dim gray)
		if task.Status == StatusDONE || task.Status == StatusWONTDO {
			if task.CompletedAt != nil {
				additionalInfo += fmt.Sprintf(" \x1b[90m(completed %s)\x1b[0m", task.CompletedAt.Format("2006-01-02"))
			}
		} else if task.DueDate != nil {
			// Add due date with color based on urgency
			dueIn := time.Until(*task.DueDate)
			if dueIn < 0 {
				// Overdue - red
				additionalInfo += fmt.Sprintf(" \x1b[91m(overdue %s)\x1b[0m", task.DueDate.Format("01-02"))
			} else if dueIn < 24*time.Hour {
				// Due today - yellow
				additionalInfo += " \x1b[93m(due today)\x1b[0m"
			} else if dueIn < 48*time.Hour {
				// Due tomorrow - light yellow
				additionalInfo += " \x1b[33m(due tomorrow)\x1b[0m"
			} else if dueIn < 7*24*time.Hour {
				// Due this week - cyan
				additionalInfo += fmt.Sprintf(" \x1b[36m(due %s)\x1b[0m", task.DueDate.Format("Mon"))
			} else {
				// Due later - dim
				additionalInfo += fmt.Sprintf(" \x1b[90m(due %s)\x1b[0m", task.DueDate.Format("01-02"))
			}
		}

		// Build the complete line with truncation
		// First build projects string with colors
		projectsStr := ""
		if len(task.Projects) > 0 {
			for _, project := range task.Projects {
				projectColor := GetProjectColor(project)
				projectsStr += fmt.Sprintf(" %s+%s\x1b[0m", projectColor, project)
			}
		}

		// Use truncate function to build the line with all components
		line = m.truncateTaskLine(cursor, statusColor, status, priority, task.Title, task.Projects, additionalInfo)

		// Add projects and additional info (already accounted for in truncation calculation)
		line += projectsStr
		line += additionalInfo
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
	} else if m.dateEditMode != "" {
		// Display date edit input
		dateRunes := []rune(m.dateEditBuffer)
		var displayStr string

		if m.dateEditCursor == 0 {
			displayStr = "│" + m.dateEditBuffer
		} else if m.dateEditCursor < len(dateRunes) {
			displayStr = string(dateRunes[:m.dateEditCursor]) + "│" + string(dateRunes[m.dateEditCursor:])
		} else {
			displayStr = m.dateEditBuffer + "│"
		}

		dateType := "deadline"
		if m.dateEditMode == "scheduled" {
			dateType = "scheduled date"
		}

		s.WriteString(fmt.Sprintf("\n\n📅 Set %s: %s", dateType, displayStr))
		s.WriteString("\n\nEnter: apply • Esc: cancel")
		s.WriteString("\n\nSupported formats:")
		s.WriteString("\n  • Natural: next tuesday, in 3 days, next week, in 2 weeks")
		s.WriteString("\n  • Simple: today, tomorrow, monday")
		s.WriteString("\n  • Dates: 2024-12-31, 12-25, 12/25")
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
	} else if m.projectSelectMode {
		// Show project selection UI
		s.WriteString("\n\n📁 Select project filter:\n\n")

		projects := m.getAvailableProjects()

		// First option is "All tasks" with total count
		cursor := "  "
		if m.projectCursor == 0 {
			cursor = "> "
		}
		allVisibleCount := len(FilterVisibleTasks(m.allTasks, m.showAll))
		s.WriteString(fmt.Sprintf("%s[All tasks] (%d)\n", cursor, allVisibleCount))

		// Show each project with color and count
		for i, project := range projects {
			cursor := "  "
			if i+1 == m.projectCursor {
				cursor = "> "
			}

			// Count tasks for this project
			projectTasks := FilterTasksByProject(m.allTasks, project)
			visibleProjectTasks := FilterVisibleTasks(projectTasks, m.showAll)
			count := len(visibleProjectTasks)

			// Get project color
			color := GetProjectColor(project)
			s.WriteString(fmt.Sprintf("%s%s+%s\x1b[0m (%d)\n", cursor, color, project, count))
		}

		s.WriteString("\n↑/k: up • ↓/j: down • Enter: select • Esc/q: cancel")
	} else if m.confirmDelete {
		s.WriteString("\n\n⚠️  Delete this task? (y/n)")
	} else {
		s.WriteString("\n↑/k: up • ↓/j: down • g/G: first/last • +/-: priority • s: status • D: deadline • S: scheduled • space: toggle done • /: search")
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

func ShowInteractiveTaskListWithFilter(tasks []Task, projectFilter string) ([]Task, bool, *Task, []string, string, bool, error) {
	model := NewInteractiveTaskListWithFilter(tasks, projectFilter)
	p := tea.NewProgram(model)

	result, err := p.Run()
	if err != nil {
		return nil, false, nil, nil, "", false, err
	}

	finalModel := result.(InteractiveTaskList)

	// Check if user wants to edit a task
	if finalModel.ShouldEdit() {
		return finalModel.GetTasks(), finalModel.IsModified(), finalModel.GetSelectedTask(), finalModel.GetDeletedTaskIDs(), finalModel.GetNewTaskTitle(), finalModel.ShouldReload(), nil
	}

	return finalModel.GetTasks(), finalModel.IsModified(), nil, finalModel.GetDeletedTaskIDs(), finalModel.GetNewTaskTitle(), finalModel.ShouldReload(), nil
}
