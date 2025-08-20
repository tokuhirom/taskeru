package internal

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-runewidth"
)

type KanbanView struct {
	allTasks        []Task
	tasksByStatus   map[string][]Task
	columns         []string // Status names in order
	currentColumn   int
	currentRow      int
	scrollOffset    int  // Vertical scroll offset for current column
	quit            bool
	modified        bool
	showAll         bool
	selectedTask    *Task
	confirmDelete   bool
	deletedTaskIDs  []string
	editTask        *Task
	shouldReload    bool
	viewHeight      int  // Terminal height for viewport
	createMode      bool // Mode for creating new task
	inputBuffer     string // Input buffer for new task title
	inputCursor     int  // Cursor position in input buffer
}

func NewKanbanView(tasks []Task) *KanbanView {
	// Sort tasks first
	SortTasks(tasks)
	
	// Get all statuses
	columns := GetAllStatuses()
	
	// Filter visible tasks by default
	filteredTasks := FilterVisibleTasks(tasks, false)
	
	// Group tasks by status
	tasksByStatus := make(map[string][]Task)
	for _, status := range columns {
		tasksByStatus[status] = []Task{}
	}
	
	for _, task := range filteredTasks {
		// Normalize status to uppercase to handle legacy lowercase statuses
		normalizedStatus := strings.ToUpper(task.Status)
		if normalizedStatus == "TODO" || normalizedStatus == "DOING" || 
		   normalizedStatus == "WAITING" || normalizedStatus == "DONE" || 
		   normalizedStatus == "WONTDO" {
			// Map lowercase "todo" to "TODO" column
			if _, ok := tasksByStatus[normalizedStatus]; ok {
				tasksByStatus[normalizedStatus] = append(tasksByStatus[normalizedStatus], task)
			}
		} else {
			// Unknown status goes to TODO column
			tasksByStatus[StatusTODO] = append(tasksByStatus[StatusTODO], task)
		}
	}
	
	return &KanbanView{
		allTasks:       tasks,
		tasksByStatus:  tasksByStatus,
		columns:        columns,
		currentColumn:  0,
		currentRow:     0,
		scrollOffset:   0,
		quit:           false,
		modified:       false,
		showAll:        false,
		confirmDelete:  false,
		deletedTaskIDs: []string{},
		editTask:       nil,
		shouldReload:   false,
		viewHeight:     30, // Default height, will be updated with WindowSizeMsg
		createMode:     false,
		inputBuffer:    "",
		inputCursor:    0,
	}
}

func (m KanbanView) Init() tea.Cmd {
	return nil
}

func (m KanbanView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Update viewport height when window is resized
		m.viewHeight = msg.Height
		return m, nil
		
	case tea.KeyMsg:
		// Handle input in create mode
		if m.createMode {
			runes := []rune(m.inputBuffer)
			
			switch msg.Type {
			case tea.KeyEnter:
				// Create the task with project extraction
				if strings.TrimSpace(m.inputBuffer) != "" {
					// Extract projects from title
					cleanTitle, projects := ExtractProjectsFromTitle(m.inputBuffer)
					
					newTask := NewTask(cleanTitle)
					// Set status based on current column
					newTask.SetStatus(m.columns[m.currentColumn])
					// Add extracted projects
					newTask.Projects = projects
					m.allTasks = append(m.allTasks, *newTask)
					
					// Re-sort and rebuild
					SortTasks(m.allTasks)
					m.rebuildKanban()
					
					// Find the new task in the current column and select it
					for i, task := range m.tasksByStatus[m.columns[m.currentColumn]] {
						if task.ID == newTask.ID {
							m.currentRow = i
							m.updateScrollForSelection()
							break
						}
					}
					
					m.modified = true
				}
				m.createMode = false
				m.inputBuffer = ""
				m.inputCursor = 0
				
			case tea.KeyEscape:
				// Cancel creation
				m.createMode = false
				m.inputBuffer = ""
				m.inputCursor = 0
				
			case tea.KeyLeft, tea.KeyCtrlB:
				// Move cursor left
				if m.inputCursor > 0 {
					m.inputCursor--
				}
				
			case tea.KeyRight, tea.KeyCtrlF:
				// Move cursor right
				if m.inputCursor < len(runes) {
					m.inputCursor++
				}
				
			case tea.KeyHome, tea.KeyCtrlA:
				// Move to beginning
				m.inputCursor = 0
				
			case tea.KeyEnd, tea.KeyCtrlE:
				// Move to end
				m.inputCursor = len(runes)
				
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
						partial := string(runes[lastPlusIdx+1:m.inputCursor])
						
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
				// Remove character before cursor
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
				// Insert runes at cursor position (handles multi-byte characters properly)
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
			if m.confirmDelete {
				m.confirmDelete = false
			} else {
				m.quit = true
				return m, tea.Quit
			}
			
		case "left", "h":
			if !m.confirmDelete && m.currentColumn > 0 {
				m.currentColumn--
				// Reset scroll offset when changing columns
				m.scrollOffset = 0
				// Adjust row if necessary
				if m.currentRow >= len(m.tasksByStatus[m.columns[m.currentColumn]]) {
					m.currentRow = max(0, len(m.tasksByStatus[m.columns[m.currentColumn]])-1)
				}
				// Update scroll to show selected card
				m.updateScrollForSelection()
			}
			
		case "right", "l":
			if !m.confirmDelete && m.currentColumn < len(m.columns)-1 {
				m.currentColumn++
				// Reset scroll offset when changing columns
				m.scrollOffset = 0
				// Adjust row if necessary
				if m.currentRow >= len(m.tasksByStatus[m.columns[m.currentColumn]]) {
					m.currentRow = max(0, len(m.tasksByStatus[m.columns[m.currentColumn]])-1)
				}
				// Update scroll to show selected card
				m.updateScrollForSelection()
			}
			
		case "up", "k":
			if !m.confirmDelete && m.currentRow > 0 {
				m.currentRow--
				m.updateScrollForSelection()
			}
			
		case "down", "j":
			if !m.confirmDelete && m.currentRow < len(m.tasksByStatus[m.columns[m.currentColumn]])-1 {
				m.currentRow++
				m.updateScrollForSelection()
			}
			
		case "ctrl+u":
			// Page up (half page)
			if !m.confirmDelete {
				pageSize := max(1, (m.viewHeight-10)/2)
				m.currentRow = max(0, m.currentRow-pageSize)
				m.updateScrollForSelection()
			}
			
		case "ctrl+d":
			// Page down (half page)
			if !m.confirmDelete {
				pageSize := max(1, (m.viewHeight-10)/2)
				maxRow := len(m.tasksByStatus[m.columns[m.currentColumn]]) - 1
				m.currentRow = min(maxRow, m.currentRow+pageSize)
				m.updateScrollForSelection()
			}
			
		case " ":
			// Quick toggle between TODO and DONE
			if m.getCurrentTask() != nil {
				taskID := m.getCurrentTask().ID
				for i := range m.allTasks {
					if m.allTasks[i].ID == taskID {
						var newStatus string
						if m.allTasks[i].Status == StatusDONE {
							newStatus = StatusTODO
							m.allTasks[i].SetStatus(StatusTODO)
						} else {
							newStatus = StatusDONE
							m.allTasks[i].SetStatus(StatusDONE)
						}
						
						// Rebuild kanban view
						m.rebuildKanban()
						
						// Follow the task to its new column
						for colIdx, col := range m.columns {
							if col == newStatus {
								m.currentColumn = colIdx
								// Find the task in the new column
								for rowIdx, task := range m.tasksByStatus[col] {
									if task.ID == taskID {
										m.currentRow = rowIdx
										m.updateScrollForSelection()
										break
									}
								}
								break
							}
						}
						
						m.modified = true
						break
					}
				}
			}
			
		case "s":
			// Cycle through statuses
			if m.getCurrentTask() != nil {
				taskID := m.getCurrentTask().ID
				for i := range m.allTasks {
					if m.allTasks[i].ID == taskID {
						currentStatus := m.allTasks[i].Status
						allStatuses := GetAllStatuses()
						
						// Find current status index
						currentIdx := 0
						for j, s := range allStatuses {
							if s == currentStatus {
								currentIdx = j
								break
							}
						}
						
						// Cycle to next status
						nextIdx := (currentIdx + 1) % len(allStatuses)
						newStatus := allStatuses[nextIdx]
						m.allTasks[i].SetStatus(newStatus)
						
						// Rebuild kanban view
						m.rebuildKanban()
						
						// Follow the task to its new column
						for colIdx, col := range m.columns {
							if col == newStatus {
								m.currentColumn = colIdx
								// Find the task in the new column
								for rowIdx, task := range m.tasksByStatus[col] {
									if task.ID == taskID {
										m.currentRow = rowIdx
										m.updateScrollForSelection()
										break
									}
								}
								break
							}
						}
						
						m.modified = true
						break
					}
				}
			}
			
		case "+":
			// Increase priority
			if m.getCurrentTask() != nil {
				taskID := m.getCurrentTask().ID
				for i := range m.allTasks {
					if m.allTasks[i].ID == taskID {
						m.allTasks[i].IncreasePriority()
						
						// Re-sort and rebuild
						SortTasks(m.allTasks)
						m.rebuildKanban()
						m.modified = true
						break
					}
				}
			}
			
		case "-":
			// Decrease priority
			if m.getCurrentTask() != nil {
				taskID := m.getCurrentTask().ID
				for i := range m.allTasks {
					if m.allTasks[i].ID == taskID {
						m.allTasks[i].DecreasePriority()
						
						// Re-sort and rebuild
						SortTasks(m.allTasks)
						m.rebuildKanban()
						m.modified = true
						break
					}
				}
			}
			
		case "a":
			// Toggle show all
			m.showAll = !m.showAll
			m.rebuildKanban()
			
		case "c":
			// Create new task
			if !m.confirmDelete {
				m.createMode = true
				m.inputBuffer = ""
				m.inputCursor = 0
			}
			
		case "e":
			// Edit task
			if m.getCurrentTask() != nil {
				m.editTask = m.getCurrentTask()
				return m, tea.Quit
			}
			
		case "d":
			// Delete task - first press shows confirmation
			if m.getCurrentTask() != nil && !m.confirmDelete {
				m.confirmDelete = true
			}
			
		case "y":
			// Confirm deletion
			if m.confirmDelete && m.getCurrentTask() != nil {
				taskID := m.getCurrentTask().ID
				m.deletedTaskIDs = append(m.deletedTaskIDs, taskID)
				
				// Remove from allTasks
				newAllTasks := []Task{}
				for _, t := range m.allTasks {
					if t.ID != taskID {
						newAllTasks = append(newAllTasks, t)
					}
				}
				m.allTasks = newAllTasks
				
				// Rebuild kanban
				m.rebuildKanban()
				
				// Adjust cursor if necessary
				if m.currentRow >= len(m.tasksByStatus[m.columns[m.currentColumn]]) && m.currentRow > 0 {
					m.currentRow--
				}
				
				m.modified = true
				m.confirmDelete = false
			}
			
		case "n":
			// Cancel deletion
			if m.confirmDelete {
				m.confirmDelete = false
			}
			
		case "r":
			// Reload
			if !m.confirmDelete {
				m.shouldReload = true
				return m, tea.Quit
			}
			
		case "g":
			// Jump to first task in column
			m.currentRow = 0
			m.scrollOffset = 0
			
		case "G":
			// Jump to last task in column
			if len(m.tasksByStatus[m.columns[m.currentColumn]]) > 0 {
				m.currentRow = len(m.tasksByStatus[m.columns[m.currentColumn]]) - 1
				m.updateScrollForSelection()
			}
		}
	}
	
	return m, nil
}

func (m *KanbanView) getCurrentTask() *Task {
	if m.currentColumn >= 0 && m.currentColumn < len(m.columns) {
		status := m.columns[m.currentColumn]
		tasks := m.tasksByStatus[status]
		if m.currentRow >= 0 && m.currentRow < len(tasks) {
			return &tasks[m.currentRow]
		}
	}
	return nil
}

func (m *KanbanView) updateScrollForSelection() {
	// Calculate the visible area
	overhead := 12 // Fixed UI elements (headers, help text, etc.)
	visibleRows := max(1, (m.viewHeight-overhead)/5) // Average 5 lines per card
	
	// If selected row is above visible area, scroll up
	if m.currentRow < m.scrollOffset {
		m.scrollOffset = m.currentRow
	}
	
	// If selected row is below visible area, scroll down
	if m.currentRow >= m.scrollOffset+visibleRows {
		m.scrollOffset = m.currentRow - visibleRows + 1
	}
	
	// Ensure scroll offset is valid
	m.scrollOffset = max(0, m.scrollOffset)
}

func (m *KanbanView) rebuildKanban() {
	// Re-sort tasks
	SortTasks(m.allTasks)
	
	// Filter tasks based on showAll
	filteredTasks := FilterVisibleTasks(m.allTasks, m.showAll)
	
	// Group tasks by status
	m.tasksByStatus = make(map[string][]Task)
	for _, status := range m.columns {
		m.tasksByStatus[status] = []Task{}
	}
	
	for _, task := range filteredTasks {
		// Normalize status to uppercase to handle legacy lowercase statuses
		normalizedStatus := strings.ToUpper(task.Status)
		if normalizedStatus == "TODO" || normalizedStatus == "DOING" || 
		   normalizedStatus == "WAITING" || normalizedStatus == "DONE" || 
		   normalizedStatus == "WONTDO" {
			// Map lowercase "todo" to "TODO" column
			if _, ok := m.tasksByStatus[normalizedStatus]; ok {
				m.tasksByStatus[normalizedStatus] = append(m.tasksByStatus[normalizedStatus], task)
			}
		} else {
			// Unknown status goes to TODO column
			m.tasksByStatus[StatusTODO] = append(m.tasksByStatus[StatusTODO], task)
		}
	}
}

func (m KanbanView) View() string {
	if m.quit {
		return ""
	}
	
	var s strings.Builder
	
	// Calculate column width (terminal width / number of columns)
	// Use a wider terminal assumption for better display
	termWidth := 160
	colWidth := termWidth / len(m.columns)
	cardWidth := max(25, colWidth - 4) // Leave some padding, minimum 25 chars for better multi-byte display
	
	// Header
	s.WriteString("Kanban View\n")
	s.WriteString(strings.Repeat("═", termWidth) + "\n\n")
	
	// Column headers
	for i, status := range m.columns {
		header := fmt.Sprintf(" %s ", status)
		
		// Add color based on status
		var statusColor string
		switch status {
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
		
		// Center the header in column width
		padding := (colWidth - len(header)) / 2
		if i == m.currentColumn {
			s.WriteString(fmt.Sprintf("%s%s▶ %s ◀\x1b[0m", strings.Repeat(" ", max(0, padding-2)), statusColor, status))
		} else {
			s.WriteString(fmt.Sprintf("%s%s%s\x1b[0m", strings.Repeat(" ", padding), statusColor, header))
		}
		
		if i < len(m.columns)-1 {
			s.WriteString(strings.Repeat(" ", colWidth-padding-len(header)))
		}
	}
	s.WriteString("\n")
	
	// Column separator
	for range m.columns {
		s.WriteString(strings.Repeat("─", colWidth))
	}
	s.WriteString("\n\n")
	
	// Calculate visible rows based on terminal height
	// Fixed overhead:
	// - Header + separator + blank = 3 lines
	// - Column headers + separator + blank = 3 lines  
	// - Help text (2 blanks + 2 lines) = 4 lines
	// - Total fixed = 10 lines
	// - Scroll indicators (if shown) = 2 lines each
	// Each card is typically 4-5 lines (borders + title + projects)
	overhead := 12 // Fixed UI elements plus some buffer
	visibleRows := max(1, (m.viewHeight-overhead)/5) // Average 5 lines per card
	
	// Find max tasks in any column for row iteration
	maxRows := 0
	for _, tasks := range m.tasksByStatus {
		if len(tasks) > maxRows {
			maxRows = len(tasks)
		}
	}
	
	// Calculate the range of rows to display
	startRow := m.scrollOffset
	endRow := min(maxRows, startRow+visibleRows)
	
	// Display scroll indicator if there are hidden cards above
	if startRow > 0 {
		for range m.columns {
			s.WriteString(fmt.Sprintf("%s↑ %d more%s", 
				strings.Repeat(" ", colWidth/2-5),
				startRow,
				strings.Repeat(" ", colWidth/2-3)))
		}
		s.WriteString("\n\n")
	}
	
	// Display tasks in columns (only visible rows)
	for row := startRow; row < endRow; row++ {
		// Track the maximum lines needed for this row
		maxLinesInRow := 1
		rowCards := make([][]string, len(m.columns))
		
		// Prepare all cards for this row
		for col, status := range m.columns {
			tasks := m.tasksByStatus[status]
			
			if row < len(tasks) {
				task := tasks[row]
				isSelected := col == m.currentColumn && row == m.currentRow
				
				var cardLines []string
				innerWidth := cardWidth - 6 // Account for borders and padding
				if innerWidth < 10 {
					innerWidth = 10
				}
				
				// Top border
				if isSelected {
					cardLines = append(cardLines, "  ╔" + strings.Repeat("═", innerWidth+2) + "╗")
				} else {
					cardLines = append(cardLines, "  ┌" + strings.Repeat("─", innerWidth+2) + "┐")
				}
				
				// Priority and title
				contentLine := ""
				if task.Priority != "" {
					contentLine = fmt.Sprintf("[%s] ", task.Priority)
				}
				
				// Title (wrap to max 3 lines)
				titleWidth := innerWidth - len(contentLine)
				if titleWidth < 10 {
					titleWidth = 10
				}
				titleLines := wrapText(task.Title, titleWidth)
				if len(titleLines) > 3 {
					titleLines = titleLines[:3]
					if len(titleLines[2]) > titleWidth-3 {
						titleLines[2] = titleLines[2][:titleWidth-3] + "..."
					}
				}
				
				// Add first title line
				if len(titleLines) > 0 {
					contentLine += titleLines[0]
				}
				border := "│"
				if isSelected {
					border = "║"
				}
				cardLines = append(cardLines, fmt.Sprintf("  %s %s%s %s", 
					border, 
					contentLine, 
					strings.Repeat(" ", max(0, innerWidth-displayWidth(contentLine))),
					border))
				
				// Add remaining title lines
				for i := 1; i < len(titleLines); i++ {
					line := titleLines[i]
					cardLines = append(cardLines, fmt.Sprintf("  %s %s%s %s",
						border,
						line,
						strings.Repeat(" ", max(0, innerWidth-displayWidth(line))),
						border))
				}
				
				// Add projects on a new line
				if len(task.Projects) > 0 {
					projectLine := ""
					for i, project := range task.Projects {
						if i > 0 {
							projectLine += " "
						}
						color := GetProjectColor(project)
						projectLine += fmt.Sprintf("%s+%s\x1b[0m", color, project)
					}
					// Truncate if too long
					if displayWidth(projectLine) > innerWidth {
						// Keep only projects that fit
						projectLine = ""
						for i, project := range task.Projects {
							testLine := projectLine
							if i > 0 {
								testLine += " "
							}
							color := GetProjectColor(project)
							testLine += fmt.Sprintf("%s+%s\x1b[0m", color, project)
							if displayWidth(testLine) <= innerWidth-3 {
								projectLine = testLine
							} else {
								projectLine += "..."
								break
							}
						}
					}
					cardLines = append(cardLines, fmt.Sprintf("  %s %s%s %s",
						border,
						projectLine,
						strings.Repeat(" ", max(0, innerWidth-displayWidth(projectLine))),
						border))
				}
				
				// Bottom border
				if isSelected {
					cardLines = append(cardLines, "  ╚" + strings.Repeat("═", innerWidth+2) + "╝")
				} else {
					cardLines = append(cardLines, "  └" + strings.Repeat("─", innerWidth+2) + "┘")
				}
				
				rowCards[col] = cardLines
				if len(cardLines) > maxLinesInRow {
					maxLinesInRow = len(cardLines)
				}
			}
		}
		
		// Print all lines for this row
		for lineIdx := 0; lineIdx < maxLinesInRow; lineIdx++ {
			for col := range m.columns {
				if rowCards[col] != nil && lineIdx < len(rowCards[col]) {
					line := rowCards[col][lineIdx]
					s.WriteString(line)
					// Pad to column width
					lineLen := displayWidth(line)
					if lineLen < colWidth {
						s.WriteString(strings.Repeat(" ", colWidth-lineLen))
					}
				} else {
					// Empty line for this card
					s.WriteString(strings.Repeat(" ", colWidth))
				}
			}
			s.WriteString("\n")
		}
		
		// No extra spacing between cards - they already have borders
	}
	
	// Display scroll indicator if there are hidden cards below
	if endRow < maxRows {
		s.WriteString("\n")
		for i := range m.columns {
			remainingTasks := 0
			if tasks, ok := m.tasksByStatus[m.columns[i]]; ok {
				remainingTasks = max(0, len(tasks)-endRow)
			}
			if remainingTasks > 0 {
				s.WriteString(fmt.Sprintf("%s↓ %d more%s",
					strings.Repeat(" ", colWidth/2-5),
					remainingTasks,
					strings.Repeat(" ", colWidth/2-3)))
			} else {
				s.WriteString(strings.Repeat(" ", colWidth))
			}
		}
		s.WriteString("\n")
	}
	
	// Help text
	if m.createMode {
		// Display input with cursor
		runes := []rune(m.inputBuffer)
		var display strings.Builder
		display.WriteString("\n\n📝 New task: ")
		
		for i := 0; i <= len(runes); i++ {
			if i == m.inputCursor {
				display.WriteString("▏") // cursor
			}
			if i < len(runes) {
				display.WriteRune(runes[i])
			}
		}
		s.WriteString(display.String())
		s.WriteString("\n(Enter to create, Esc to cancel, Tab to complete project)")
	} else if m.confirmDelete {
		s.WriteString("\n\n⚠️  Delete this task? (y/n)")
	} else {
		s.WriteString("\n\n←/h: left • →/l: right • ↑/k: up • ↓/j: down • ctrl+u/d: page up/down")
		s.WriteString("\nc: create • +/-: priority • s: status • space: toggle done • a: show all • e: edit • d: delete • r: reload • q: quit")
		if m.showAll {
			s.WriteString(" [ALL]")
		}
		if m.modified {
			s.WriteString(" • *modified*")
		}
	}
	
	return s.String()
}

// Helper function to wrap text to specified width considering display width
func wrapText(text string, width int) []string {
	if width <= 0 || text == "" {
		return []string{text}
	}
	
	var lines []string
	var currentLine []rune
	currentWidth := 0
	
	runes := []rune(text)
	
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		runeWidth := runewidth.RuneWidth(r)
		
		// If adding this rune would exceed width, wrap
		if currentWidth > 0 && currentWidth+runeWidth > width {
			// Try to find a good break point (space or punctuation)
			breakPoint := len(currentLine)
			foundBreak := false
			
			// Look backwards for a space or certain punctuation marks
			for j := len(currentLine) - 1; j >= 0 && j >= len(currentLine)-10; j-- {
				if currentLine[j] == ' ' || currentLine[j] == '､' || currentLine[j] == '｡' || 
				   currentLine[j] == '、' || currentLine[j] == '。' || currentLine[j] == '）' || 
				   currentLine[j] == ')' || currentLine[j] == '」' || currentLine[j] == '』' {
					breakPoint = j + 1
					foundBreak = true
					break
				}
			}
			
			// If we found a break point, use it
			if foundBreak && breakPoint < len(currentLine) {
				// Add the line up to the break point
				lines = append(lines, string(currentLine[:breakPoint]))
				// Start new line with the rest
				currentLine = currentLine[breakPoint:]
				// Recalculate width
				currentWidth = 0
				for _, cr := range currentLine {
					currentWidth += runewidth.RuneWidth(cr)
				}
			} else {
				// No good break point, just break at the limit
				lines = append(lines, string(currentLine))
				currentLine = []rune{}
				currentWidth = 0
			}
		}
		
		// Add the current rune
		currentLine = append(currentLine, r)
		currentWidth += runeWidth
		
		// Handle explicit line breaks
		if r == '\n' {
			lines = append(lines, strings.TrimRight(string(currentLine), "\n"))
			currentLine = []rune{}
			currentWidth = 0
		}
	}
	
	// Add any remaining text
	if len(currentLine) > 0 {
		lines = append(lines, string(currentLine))
	}
	
	// Trim spaces from beginning of wrapped lines
	for i := range lines {
		lines[i] = strings.TrimLeft(lines[i], " ")
	}
	
	return lines
}

// Helper function to strip ANSI codes and calculate display width
func stripAnsi(str string) string {
	// Simple implementation - just remove common ANSI codes
	result := str
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

// Calculate display width considering multi-byte characters
func displayWidth(str string) int {
	// First strip ANSI codes
	clean := stripAnsi(str)
	return runewidth.StringWidth(clean)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func ShowKanbanView(tasks []Task) ([]Task, bool, *Task, []string, bool, error) {
	model := NewKanbanView(tasks)
	p := tea.NewProgram(model)
	
	result, err := p.Run()
	if err != nil {
		return nil, false, nil, nil, false, err
	}
	
	finalModel := result.(KanbanView)
	return finalModel.allTasks, finalModel.modified, finalModel.editTask,
		finalModel.deletedTaskIDs, finalModel.shouldReload, nil
}