package internal

import (
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Task struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Created     time.Time  `json:"created"`
	Updated     time.Time  `json:"updated"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	DueDate     *time.Time `json:"due_date,omitempty"`
	Priority    string     `json:"priority,omitempty"`
	Status      string     `json:"status"`
	Note        string     `json:"note,omitempty"`
	Projects    []string   `json:"projects,omitempty"`
}

func NewTask(title string) *Task {
	now := time.Now()
	return &Task{
		ID:      uuid.New().String(),
		Title:   title,
		Created: now,
		Updated: now,
		Status:  "todo",
	}
}

func (t *Task) SetPriority(priority string) {
	if priority == "high" || priority == "medium" || priority == "low" {
		t.Priority = priority
	}
}

func (t *Task) SetStatus(status string) {
	if status == "todo" || status == "in_progress" || status == "done" {
		oldStatus := t.Status
		t.Status = status
		now := time.Now()
		t.Updated = now
		
		// Record completion time when marking as done
		if status == "done" && oldStatus != "done" {
			t.CompletedAt = &now
		} else if status != "done" && oldStatus == "done" {
			// Clear completion time when unmarking as done
			t.CompletedAt = nil
		}
	}
}

func (t *Task) SetDueDate(dueDate time.Time) {
	t.DueDate = &dueDate
}

func (t *Task) DisplayStatus() string {
	switch t.Status {
	case "done":
		return "✓"
	case "in_progress":
		return "→"
	default:
		return "○"
	}
}

func (t *Task) DisplayPriority() string {
	switch t.Priority {
	case "high":
		return "!!!"
	case "medium":
		return "!!"
	case "low":
		return "!"
	default:
		return "   "
	}
}

func (t *Task) IsOldCompleted() bool {
	if t.Status != "done" || t.CompletedAt == nil {
		return false
	}
	
	// Check if completed before today (yesterday or earlier)
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	return t.CompletedAt.Before(today)
}

func FilterVisibleTasks(tasks []Task, showAll bool) []Task {
	if showAll {
		return tasks
	}
	
	var visible []Task
	for _, task := range tasks {
		if !task.IsOldCompleted() {
			visible = append(visible, task)
		}
	}
	return visible
}

// ExtractProjectsFromTitle extracts project tags (+project) from the end of title and returns cleaned title and projects
func ExtractProjectsFromTitle(title string) (string, []string) {
	// Extract project tags only from the end of the string
	// Pattern: (whitespace or start) followed by +project at the end
	projectEndRegex := regexp.MustCompile(`(\s+|^)\+(\S+)\s*$`)
	
	var projects []string
	cleanTitle := title
	
	// Keep extracting project tags from the end until no more are found
	for {
		match := projectEndRegex.FindStringSubmatch(cleanTitle)
		if match == nil {
			break
		}
		
		// Add project to the beginning (since we're extracting from the end)
		// match[2] is the project name (match[1] is the whitespace or start)
		projects = append([]string{match[2]}, projects...)
		
		// Remove the matched project tag from the string
		cleanTitle = projectEndRegex.ReplaceAllString(cleanTitle, "")
	}
	
	cleanTitle = strings.TrimSpace(cleanTitle)
	
	return cleanTitle, projects
}

// GetAllProjects returns all unique projects from a list of tasks
func GetAllProjects(tasks []Task) []string {
	projectMap := make(map[string]bool)
	var projects []string
	
	for _, task := range tasks {
		for _, project := range task.Projects {
			if !projectMap[project] {
				projectMap[project] = true
				projects = append(projects, project)
			}
		}
	}
	
	return projects
}

// FilterTasksByProject returns tasks that belong to a specific project
func FilterTasksByProject(tasks []Task, project string) []Task {
	var filtered []Task
	for _, task := range tasks {
		for _, p := range task.Projects {
			if p == project {
				filtered = append(filtered, task)
				break
			}
		}
	}
	return filtered
}