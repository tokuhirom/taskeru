package internal

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Task struct {
	ID            string     `json:"id"`
	Title         string     `json:"title"`
	Created       time.Time  `json:"created"`
	Updated       time.Time  `json:"updated"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
	DueDate       *time.Time `json:"due_date,omitempty"`
	ScheduledDate *time.Time `json:"scheduled_date,omitempty"`
	Priority      string     `json:"priority,omitempty"`
	Status        string     `json:"status"`
	Note          string     `json:"note,omitempty"`
	Projects      []string   `json:"projects,omitempty"`
}

// Available task statuses
const (
	StatusTODO    = "TODO"
	StatusDOING   = "DOING"
	StatusWAITING = "WAITING"
	StatusDONE    = "DONE"
	StatusWONTDO  = "WONTDO"
)

// GetAllStatuses returns all available task statuses
func GetAllStatuses() []string {
	return []string{StatusTODO, StatusDOING, StatusWAITING, StatusDONE, StatusWONTDO}
}

func NewTask(title string) *Task {
	now := time.Now()
	return &Task{
		ID:      uuid.New().String(),
		Title:   title,
		Created: now,
		Updated: now,
		Status:  StatusTODO,
	}
}

func (t *Task) String() string {
	var buf strings.Builder
	buf.WriteString(t.Title)
	if len(t.Projects) > 0 {
		buf.WriteString(fmt.Sprintf(" [Projects: %s]", strings.Join(t.Projects, ", ")))
	}
	if t.ScheduledDate != nil {
		buf.WriteString(fmt.Sprintf(" [Scheduled: %s]", t.ScheduledDate.Format("2006-01-02")))
	}
	if t.DueDate != nil {
		buf.WriteString(fmt.Sprintf(" [Due: %s]", t.DueDate.Format("2006-01-02")))
	}
	return t.Title
}

func (t *Task) SetPriority(priority string) {
	// Accept single letter A-Z or empty string
	if len(priority) == 0 {
		t.Priority = ""
		return
	}
	if len(priority) == 1 {
		r := rune(priority[0])
		if r >= 'A' && r <= 'Z' {
			t.Priority = priority
			t.Updated = time.Now()
		}
	}
}

// IncreasePriority increases priority (A is highest, Z is lowest)
func (t *Task) IncreasePriority() {
	if t.Priority == "" {
		t.Priority = "C" // Start from C when no priority
	} else if len(t.Priority) == 1 {
		r := rune(t.Priority[0])
		if r > 'A' {
			t.Priority = string(r - 1)
		}
	}
	t.Updated = time.Now()
}

// DecreasePriority decreases priority (A is highest, Z is lowest)
func (t *Task) DecreasePriority() {
	if t.Priority == "" {
		t.Priority = "D" // Start from D when no priority
	} else if len(t.Priority) == 1 {
		r := rune(t.Priority[0])
		if r < 'Z' {
			t.Priority = string(r + 1)
		}
	}
	t.Updated = time.Now()
}

func (t *Task) SetStatus(status string) {
	// Validate status
	validStatuses := GetAllStatuses()
	isValid := false
	for _, s := range validStatuses {
		if status == s {
			isValid = true
			break
		}
	}

	if !isValid {
		return
	}

	oldStatus := t.Status
	t.Status = status
	now := time.Now()
	t.Updated = now

	// Record completion time when marking as done or wontdo
	if (status == StatusDONE || status == StatusWONTDO) && oldStatus != StatusDONE && oldStatus != StatusWONTDO {
		t.CompletedAt = &now
	} else if status != StatusDONE && status != StatusWONTDO && (oldStatus == StatusDONE || oldStatus == StatusWONTDO) {
		// Clear completion time when unmarking as done/wontdo
		t.CompletedAt = nil
	}
}

func (t *Task) SetDueDate(dueDate time.Time) {
	t.DueDate = &dueDate
}

func (t *Task) DisplayStatus() string {
	return t.Status
}

func (t *Task) DisplayPriority() string {
	if t.Priority == "" {
		return "   "
	}
	// Display as [A], [B], etc.
	return "[" + t.Priority + "]"
}

func (t *Task) IsOldCompleted() bool {
	if (t.Status != StatusDONE && t.Status != StatusWONTDO) || t.CompletedAt == nil {
		return false
	}

	// Check if completed before today (with 4 AM as day boundary)
	// Tasks completed after 4 AM yesterday are still "today's tasks"
	now := time.Now()

	// Calculate today's 4 AM cutoff
	todayAt4AM := time.Date(now.Year(), now.Month(), now.Day(), 4, 0, 0, 0, now.Location())

	// If current time is before 4 AM, use yesterday's 4 AM as the cutoff
	if now.Before(todayAt4AM) {
		todayAt4AM = todayAt4AM.AddDate(0, 0, -1)
	}

	return t.CompletedAt.Before(todayAt4AM)
}

// IsFutureScheduled returns true if the task is scheduled for a future date
func (t *Task) IsFutureScheduled() bool {
	if t.ScheduledDate == nil {
		return false
	}

	now := time.Now()
	// Tasks scheduled for today or earlier are not "future"
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	return t.ScheduledDate.After(today)
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

// GetPriorityValue returns numeric value for sorting (lower is higher priority)
func GetPriorityValue(priority string) float64 {
	if priority == "" {
		// No priority sorts between C and D
		// C = 2.0, D = 3.0, so we use 2.5
		return 2.5
	}
	if len(priority) == 1 {
		r := rune(priority[0])
		if r >= 'A' && r <= 'Z' {
			return float64(int(r) - int('A'))
		}
	}
	return 100 // Very low priority for invalid values
}

// SortTasks sorts tasks by status (active first, completed last), then priority (A-Z), then update time
func SortTasks(tasks []Task) {
	sort.Slice(tasks, func(i, j int) bool {
		// First, completed tasks (DONE/WONTDO) always go to the bottom
		iCompleted := tasks[i].Status == StatusDONE || tasks[i].Status == StatusWONTDO
		jCompleted := tasks[j].Status == StatusDONE || tasks[j].Status == StatusWONTDO

		if iCompleted != jCompleted {
			return !iCompleted // Active tasks come first
		}

		// Both are active or both are completed, sort by priority
		iPriority := GetPriorityValue(tasks[i].Priority)
		jPriority := GetPriorityValue(tasks[j].Priority)

		if iPriority != jPriority {
			return iPriority < jPriority // Lower value = higher priority
		}

		// Same priority, sort by update time (newest first)
		return tasks[i].Updated.After(tasks[j].Updated)
	})
}

// GetProjectColor returns an ANSI 256 color code for a project name
// Uses a hash of the project name to consistently assign one of 30 colors
func GetProjectColor(project string) string {
	// Define 30 distinct ANSI 256 colors for projects
	// Selected to be visible on both light and dark backgrounds
	colors := []string{
		"\x1b[38;5;33m",  // Blue
		"\x1b[38;5;208m", // Orange
		"\x1b[38;5;162m", // Magenta
		"\x1b[38;5;34m",  // Green
		"\x1b[38;5;141m", // Purple
		"\x1b[38;5;214m", // Gold
		"\x1b[38;5;39m",  // Deep Sky Blue
		"\x1b[38;5;202m", // Red Orange
		"\x1b[38;5;165m", // Magenta Pink
		"\x1b[38;5;46m",  // Bright Green
		"\x1b[38;5;135m", // Medium Purple
		"\x1b[38;5;220m", // Yellow
		"\x1b[38;5;45m",  // Turquoise
		"\x1b[38;5;196m", // Red
		"\x1b[38;5;171m", // Light Purple
		"\x1b[38;5;118m", // Light Green
		"\x1b[38;5;99m",  // Slate Purple
		"\x1b[38;5;215m", // Peach
		"\x1b[38;5;51m",  // Cyan
		"\x1b[38;5;205m", // Hot Pink
		"\x1b[38;5;155m", // Pale Green
		"\x1b[38;5;105m", // Light Slate
		"\x1b[38;5;222m", // Light Orange
		"\x1b[38;5;87m",  // Light Cyan
		"\x1b[38;5;198m", // Deep Pink
		"\x1b[38;5;120m", // Light Yellow Green
		"\x1b[38;5;147m", // Light Blue Purple
		"\x1b[38;5;209m", // Salmon
		"\x1b[38;5;81m",  // Sky Blue
		"\x1b[38;5;169m", // Pink
	}

	// Simple hash: sum of character codes
	hash := 0
	for _, ch := range project {
		hash += int(ch)
	}

	// Select color based on hash
	colorIndex := hash % len(colors)
	return colors[colorIndex]
}
