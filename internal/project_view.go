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
}

func NewProjectView(tasks []Task) *ProjectView {
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
		}
	}
	
	return m, nil
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
				
				var line string
				if task.Status == "done" {
					line = fmt.Sprintf("%s\x1b[32m✅ %s %s\x1b[0m\n", cursor, priority, task.Title)
				} else {
					line = fmt.Sprintf("%s%s %s %s\n", cursor, status, priority, task.Title)
				}
				s.WriteString(line)
			}
		}
		
		s.WriteString("\n↑/k: up • ↓/j: down • Esc: back to projects • q: quit")
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
					if task.Status != "done" {
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

func ShowProjectView(tasks []Task) error {
	model := NewProjectView(tasks)
	p := tea.NewProgram(model)
	
	_, err := p.Run()
	return err
}