package internal

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type ProjectView struct {
	allTasks        []Task
	projects        []string
	projectTasks    map[string][]Task
	cursor          int
	selectedProject string
	inProjectView   bool
	quit            bool
	modified        bool
	showAll         bool
	confirmDelete   bool
	deletedTaskIDs  []string
	editTask        *Task
	newTaskTitle    string
	inputMode       bool
	inputBuffer     string
	inputCursor     int
	shouldReload    bool
}

func NewProjectView(tasks []Task) *ProjectView {
	// Sort tasks first
	SortTasks(tasks)
	
	// Get all unique projects from ALL tasks (before filtering)
	projects := GetAllProjects(tasks)
	sort.Strings(projects)
	
	// Filter visible tasks by default (same as main interactive mode)
	filteredTasks := FilterVisibleTasks(tasks, false)
	
	// Create "No Project" category for tasks without projects
	projectTasks := make(map[string][]Task)
	var noProjectTasks []Task
	
	for _, task := range filteredTasks {
		if len(task.Projects) == 0 {
			noProjectTasks = append(noProjectTasks, task)
		} else {
			for _, project := range task.Projects {
				projectTasks[project] = append(projectTasks[project], task)
			}
		}
	}
	
	if len(noProjectTasks) > 0 {
		projects = append([]string{"[No Project]"}, projects...)
		projectTasks["[No Project]"] = noProjectTasks
	}
	
	return &ProjectView{
		allTasks:        tasks,
		projects:        projects,
		projectTasks:    projectTasks,
		cursor:          0,
		inProjectView:   false,
		quit:            false,
		modified:        false,
		showAll:         false,
		confirmDelete:   false,
		deletedTaskIDs:  []string{},
		editTask:        nil,
		newTaskTitle:    "",
		inputMode:       false,
		inputBuffer:     "",
		inputCursor:     0,
		shouldReload:    false,
	}
}

func (m ProjectView) Init() tea.Cmd {
	return nil
}

func (m ProjectView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle input mode for creating new tasks
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
			case tea.KeyCtrlH, tea.KeyBackspace:
				if m.inputCursor > 0 && len(runes) > 0 {
					m.inputBuffer = string(append(runes[:m.inputCursor-1], runes[m.inputCursor:]...))
					m.inputCursor--
				}
			case tea.KeyRunes:
				newRunes := append(runes[:m.inputCursor], append(msg.Runes, runes[m.inputCursor:]...)...)
				m.inputBuffer = string(newRunes)
				m.inputCursor += len(msg.Runes)
			case tea.KeySpace:
				newRunes := append(runes[:m.inputCursor], append([]rune{' '}, runes[m.inputCursor:]...)...)
				m.inputBuffer = string(newRunes)
				m.inputCursor++
			}
			return m, nil
		}
		
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			if m.confirmDelete {
				m.confirmDelete = false
			} else if m.inProjectView {
				// Go back to project list
				m.inProjectView = false
				m.selectedProject = ""
			} else {
				m.quit = true
				return m, tea.Quit
			}
			
		case "up", "k":
			if !m.confirmDelete && !m.inputMode && m.cursor > 0 {
				m.cursor--
			}
			
		case "down", "j":
			if !m.confirmDelete && !m.inputMode && m.cursor < m.getMaxCursor() {
				m.cursor++
			}
			
		case "enter":
			if !m.inProjectView && m.cursor < len(m.projects) {
				m.selectedProject = m.projects[m.cursor]
				m.inProjectView = true
				m.cursor = 0
			}
			
		case "a":
			// Toggle show all tasks
			if m.inProjectView && !m.confirmDelete && !m.inputMode {
				m.showAll = !m.showAll
				m.rebuildProjectTasksWithFilter()
			}
			
		case "c":
			// Create new task
			if m.inProjectView && !m.confirmDelete {
				m.inputMode = true
				m.inputBuffer = ""
				if m.selectedProject != "[No Project]" {
					// Pre-populate with project tag
					m.inputBuffer = " +" + m.selectedProject
				}
				m.inputCursor = 0
			}
			
		case "e":
			// Edit task
			if m.inProjectView && m.cursor < len(m.projectTasks[m.selectedProject]) {
				m.editTask = &m.projectTasks[m.selectedProject][m.cursor]
				return m, tea.Quit
			}
			
		case "d":
			// Delete task - first press shows confirmation
			if m.inProjectView && m.cursor < len(m.projectTasks[m.selectedProject]) && !m.confirmDelete {
				m.confirmDelete = true
			}
			
		case "y":
			// Confirm deletion
			if m.confirmDelete && m.inProjectView && m.cursor < len(m.projectTasks[m.selectedProject]) {
				taskID := m.projectTasks[m.selectedProject][m.cursor].ID
				m.deletedTaskIDs = append(m.deletedTaskIDs, taskID)
				
				// Remove from allTasks
				newAllTasks := []Task{}
				for _, t := range m.allTasks {
					if t.ID != taskID {
						newAllTasks = append(newAllTasks, t)
					}
				}
				m.allTasks = newAllTasks
				
				// Rebuild project tasks
				m.rebuildProjectTasksWithFilter()
				
				// Adjust cursor if necessary
				if m.cursor >= len(m.projectTasks[m.selectedProject]) && len(m.projectTasks[m.selectedProject]) > 0 {
					m.cursor = len(m.projectTasks[m.selectedProject]) - 1
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
			// Reload tasks
			if m.inProjectView && !m.confirmDelete && !m.inputMode {
				m.shouldReload = true
				return m, tea.Quit
			}
			
		case " ":
			// Toggle task status
			if m.inProjectView && m.cursor < len(m.projectTasks[m.selectedProject]) {
				currentTaskID := m.projectTasks[m.selectedProject][m.cursor].ID
				// Find and update in allTasks
				for i := range m.allTasks {
					if m.allTasks[i].ID == currentTaskID {
						if m.allTasks[i].Status == StatusDONE {
							m.allTasks[i].SetStatus(StatusTODO)
						} else {
							m.allTasks[i].SetStatus(StatusDONE)
						}
						
						// Re-sort all tasks
						SortTasks(m.allTasks)
						
						// Rebuild project tasks with filter
						m.rebuildProjectTasksWithFilter()
						
						// Find the task's new position and move cursor there
						for j, task := range m.projectTasks[m.selectedProject] {
							if task.ID == currentTaskID {
								m.cursor = j
								break
							}
						}
						
						// Ensure cursor is within bounds
						if m.cursor >= len(m.projectTasks[m.selectedProject]) && len(m.projectTasks[m.selectedProject]) > 0 {
							m.cursor = len(m.projectTasks[m.selectedProject]) - 1
						}
						
						m.modified = true
						break
					}
				}
			}
			
		case "s":
			// Cycle through statuses
			if m.inProjectView && m.cursor < len(m.projectTasks[m.selectedProject]) {
				currentTaskID := m.projectTasks[m.selectedProject][m.cursor].ID
				for i := range m.allTasks {
					if m.allTasks[i].ID == currentTaskID {
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
						m.allTasks[i].SetStatus(allStatuses[nextIdx])
						
						// Re-sort all tasks
						SortTasks(m.allTasks)
						
						// Rebuild project tasks with filter
						m.rebuildProjectTasksWithFilter()
						
						// Find the task's new position and move cursor there
						for j, task := range m.projectTasks[m.selectedProject] {
							if task.ID == currentTaskID {
								m.cursor = j
								break
							}
						}
						
						// Ensure cursor is within bounds
						if m.cursor >= len(m.projectTasks[m.selectedProject]) && len(m.projectTasks[m.selectedProject]) > 0 {
							m.cursor = len(m.projectTasks[m.selectedProject]) - 1
						}
						
						m.modified = true
						break
					}
				}
			}
			
		case "+":
			// Increase priority
			if m.inProjectView && m.cursor < len(m.projectTasks[m.selectedProject]) {
				currentTaskID := m.projectTasks[m.selectedProject][m.cursor].ID
				
				// Update in allTasks
				for i := range m.allTasks {
					if m.allTasks[i].ID == currentTaskID {
						m.allTasks[i].IncreasePriority()
						break
					}
				}
				
				// Re-sort all tasks
				SortTasks(m.allTasks)
				
				// Rebuild project tasks
				m.rebuildProjectTasks()
				
				// Find the task's new position and move cursor there
				for i, task := range m.projectTasks[m.selectedProject] {
					if task.ID == currentTaskID {
						m.cursor = i
						break
					}
				}
				
				m.modified = true
			}
			
		case "-":
			// Decrease priority
			if m.inProjectView && m.cursor < len(m.projectTasks[m.selectedProject]) {
				currentTaskID := m.projectTasks[m.selectedProject][m.cursor].ID
				
				// Update in allTasks
				for i := range m.allTasks {
					if m.allTasks[i].ID == currentTaskID {
						m.allTasks[i].DecreasePriority()
						break
					}
				}
				
				// Re-sort all tasks
				SortTasks(m.allTasks)
				
				// Rebuild project tasks
				m.rebuildProjectTasks()
				
				// Find the task's new position and move cursor there
				for i, task := range m.projectTasks[m.selectedProject] {
					if task.ID == currentTaskID {
						m.cursor = i
						break
					}
				}
				
				m.modified = true
			}
			
		case "g":
			// Jump to first task
			if m.inProjectView {
				m.cursor = 0
			}
			
		case "G":
			// Jump to last task
			if m.inProjectView && len(m.projectTasks[m.selectedProject]) > 0 {
				m.cursor = len(m.projectTasks[m.selectedProject]) - 1
			}
		}
	}
	
	return m, nil
}

func (m *ProjectView) rebuildProjectTasks() {
	// Clear and rebuild project tasks after sorting
	m.projectTasks = make(map[string][]Task)
	var noProjectTasks []Task
	
	for _, task := range m.allTasks {
		if len(task.Projects) == 0 {
			noProjectTasks = append(noProjectTasks, task)
		} else {
			for _, project := range task.Projects {
				m.projectTasks[project] = append(m.projectTasks[project], task)
			}
		}
	}
	
	if len(noProjectTasks) > 0 {
		m.projectTasks["[No Project]"] = noProjectTasks
	}
}

func (m *ProjectView) rebuildProjectTasksWithFilter() {
	// Clear and rebuild project tasks after sorting
	m.projectTasks = make(map[string][]Task)
	var noProjectTasks []Task
	
	// Filter tasks based on showAll
	filteredTasks := FilterVisibleTasks(m.allTasks, m.showAll)
	
	for _, task := range filteredTasks {
		if len(task.Projects) == 0 {
			noProjectTasks = append(noProjectTasks, task)
		} else {
			for _, project := range task.Projects {
				m.projectTasks[project] = append(m.projectTasks[project], task)
			}
		}
	}
	
	if len(noProjectTasks) > 0 {
		m.projectTasks["[No Project]"] = noProjectTasks
	}
}

func (m ProjectView) getMaxCursor() int {
	if m.inProjectView {
		tasks := m.projectTasks[m.selectedProject]
		if len(tasks) > 0 {
			return len(tasks) - 1
		}
		return 0
	}
	if len(m.projects) > 0 {
		return len(m.projects) - 1
	}
	return 0
}

func (m ProjectView) View() string {
	if m.quit {
		return ""
	}
	
	var s strings.Builder
	
	if m.inProjectView {
		// Show tasks in selected project
		projectDisplay := m.selectedProject
		if m.selectedProject != "[No Project]" {
			color := GetProjectColor(m.selectedProject)
			projectDisplay = fmt.Sprintf("%s%s\x1b[0m", color, m.selectedProject)
		}
		s.WriteString(fmt.Sprintf("Project: %s\n", projectDisplay))
		s.WriteString(strings.Repeat("-", 40) + "\n\n")
		
		tasks := m.projectTasks[m.selectedProject]
		if len(tasks) == 0 {
			s.WriteString("No tasks in this project.\n")
		} else {
			for i, task := range tasks {
				cursor := "  "
				if m.cursor == i {
					cursor = "> "
				}
				
				status := task.DisplayStatus()
				priority := task.DisplayPriority()
				
				// Add color based on status
				var statusColor string
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
				
				line := fmt.Sprintf("%s%s%-7s %s %s\x1b[0m\n", cursor, statusColor, status, priority, task.Title)
				s.WriteString(line)
			}
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
			s.WriteString("\n\nEnter: create • Esc: cancel")
		} else if m.confirmDelete {
			s.WriteString("\n\n⚠️  Delete this task? (y/n)")
		} else {
			s.WriteString("\n↑/k: up • ↓/j: down • g/G: first/last • +/-: priority • s: status • space: toggle done • a: show all • c: create • e: edit • d: delete • r: reload • Esc: back • q: quit")
			if m.showAll {
				s.WriteString(" [ALL]")
			}
			if m.modified {
				s.WriteString(" • *modified*")
			}
		}
	} else {
		// Show project list
		s.WriteString("Projects:\n")
		s.WriteString(strings.Repeat("-", 40) + "\n\n")
		
		if len(m.projects) == 0 {
			s.WriteString("No projects found.\n")
		} else {
			for i, project := range m.projects {
				cursor := "  "
				if m.cursor == i {
					cursor = "> "
				}
				
				count := len(m.projectTasks[project])
				activeCount := 0
				for _, task := range m.projectTasks[project] {
					if task.Status != StatusDONE && task.Status != StatusWONTDO {
						activeCount++
					}
				}
				
				projectDisplay := project
				if project != "[No Project]" {
					color := GetProjectColor(project)
					projectDisplay = fmt.Sprintf("%s+%s\x1b[0m", color, project)
				}
				
				line := fmt.Sprintf("%s%s (%d tasks, %d active)\n", cursor, projectDisplay, count, activeCount)
				s.WriteString(line)
			}
		}
		
		s.WriteString("\n↑/k: up • ↓/j: down • Enter: view project • q: quit")
	}
	
	return s.String()
}

func (m ProjectView) GetSelectedProject() string {
	return m.selectedProject
}

func ShowProjectView(tasks []Task) ([]Task, bool, *Task, []string, string, bool, error) {
	model := NewProjectView(tasks)
	p := tea.NewProgram(model)
	
	result, err := p.Run()
	if err != nil {
		return nil, false, nil, nil, "", false, err
	}
	
	finalModel := result.(ProjectView)
	return finalModel.allTasks, finalModel.modified, finalModel.editTask, 
		finalModel.deletedTaskIDs, finalModel.newTaskTitle, finalModel.shouldReload, nil
}