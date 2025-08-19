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
}

func NewProjectView(tasks []Task) *ProjectView {
	// Sort tasks first
	SortTasks(tasks)
	
	// Get all unique projects
	projects := GetAllProjects(tasks)
	sort.Strings(projects)
	
	// Create "No Project" category for tasks without projects
	projectTasks := make(map[string][]Task)
	var noProjectTasks []Task
	
	for _, task := range tasks {
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
		allTasks:      tasks,
		projects:      projects,
		projectTasks:  projectTasks,
		cursor:        0,
		inProjectView: false,
		quit:          false,
		modified:      false,
	}
}

func (m ProjectView) Init() tea.Cmd {
	return nil
}

func (m ProjectView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			if m.inProjectView {
				// Go back to project list
				m.inProjectView = false
				m.selectedProject = ""
			} else {
				m.quit = true
				return m, tea.Quit
			}
			
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
			
		case "down", "j":
			if m.cursor < m.getMaxCursor() {
				m.cursor++
			}
			
		case "enter":
			if !m.inProjectView && m.cursor < len(m.projects) {
				m.selectedProject = m.projects[m.cursor]
				m.inProjectView = true
				m.cursor = 0
			}
			
		case " ":
			// Toggle task status
			if m.inProjectView && m.cursor < len(m.projectTasks[m.selectedProject]) {
				task := &m.projectTasks[m.selectedProject][m.cursor]
				// Find and update in allTasks
				for i := range m.allTasks {
					if m.allTasks[i].ID == task.ID {
						if m.allTasks[i].Status == StatusDONE {
							m.allTasks[i].SetStatus(StatusTODO)
						} else {
							m.allTasks[i].SetStatus(StatusDONE)
						}
						// Update in project view
						m.projectTasks[m.selectedProject][m.cursor] = m.allTasks[i]
						m.modified = true
						break
					}
				}
			}
			
		case "s":
			// Cycle through statuses
			if m.inProjectView && m.cursor < len(m.projectTasks[m.selectedProject]) {
				task := &m.projectTasks[m.selectedProject][m.cursor]
				for i := range m.allTasks {
					if m.allTasks[i].ID == task.ID {
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
						m.projectTasks[m.selectedProject][m.cursor] = m.allTasks[i]
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
		s.WriteString(fmt.Sprintf("Project: \x1b[36m%s\x1b[0m\n", m.selectedProject))
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
					statusColor = "\x1b[32m" // green
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
		
		s.WriteString("\n↑/k: up • ↓/j: down • g/G: first/last • +/-: priority • s: status • space: toggle done • Esc: back • q: quit")
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
					projectDisplay = fmt.Sprintf("\x1b[36m+%s\x1b[0m", project)
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

func ShowProjectView(tasks []Task) ([]Task, bool, error) {
	model := NewProjectView(tasks)
	p := tea.NewProgram(model)
	
	result, err := p.Run()
	if err != nil {
		return nil, false, err
	}
	
	finalModel := result.(ProjectView)
	return finalModel.allTasks, finalModel.modified, nil
}