package internal

import (
	"regexp"
	"sort"
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

// ExtractDeadlineFromTitle extracts deadline (due:date) from title and returns cleaned title and deadline
func ExtractDeadlineFromTitle(title string) (string, *time.Time) {
	// Pattern for due:date format
	dueRegex := regexp.MustCompile(`\s+due:(\S+)`)
	
	match := dueRegex.FindStringSubmatch(title)
	if match == nil {
		return title, nil
	}
	
	dateStr := match[1]
	var deadline time.Time
	
	// Parse relative dates
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	
	switch strings.ToLower(dateStr) {
	case "today":
		deadline = today.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
	case "tomorrow":
		deadline = today.AddDate(0, 0, 1).Add(23*time.Hour + 59*time.Minute + 59*time.Second)
	case "monday", "mon":
		deadline = nextWeekday(today, time.Monday).Add(23*time.Hour + 59*time.Minute + 59*time.Second)
	case "tuesday", "tue":
		deadline = nextWeekday(today, time.Tuesday).Add(23*time.Hour + 59*time.Minute + 59*time.Second)
	case "wednesday", "wed":
		deadline = nextWeekday(today, time.Wednesday).Add(23*time.Hour + 59*time.Minute + 59*time.Second)
	case "thursday", "thu":
		deadline = nextWeekday(today, time.Thursday).Add(23*time.Hour + 59*time.Minute + 59*time.Second)
	case "friday", "fri":
		deadline = nextWeekday(today, time.Friday).Add(23*time.Hour + 59*time.Minute + 59*time.Second)
	case "saturday", "sat":
		deadline = nextWeekday(today, time.Saturday).Add(23*time.Hour + 59*time.Minute + 59*time.Second)
	case "sunday", "sun":
		deadline = nextWeekday(today, time.Sunday).Add(23*time.Hour + 59*time.Minute + 59*time.Second)
	default:
		// Try to parse as date
		// Try various formats
		formats := []string{
			"2006-01-02",
			"2006/01/02",
			"01-02",
			"01/02",
			"1/2",
			"1-2",
		}
		
		var err error
		for _, format := range formats {
			deadline, err = time.Parse(format, dateStr)
			if err == nil {
				// If year is not specified (format without year), use current year
				if format == "01-02" || format == "01/02" || format == "1/2" || format == "1-2" {
					deadline = time.Date(now.Year(), deadline.Month(), deadline.Day(), 23, 59, 59, 0, now.Location())
					// If the date has already passed this year, assume next year
					if deadline.Before(now) {
						deadline = deadline.AddDate(1, 0, 0)
					}
				} else {
					// Set time to end of day
					deadline = time.Date(deadline.Year(), deadline.Month(), deadline.Day(), 23, 59, 59, 0, now.Location())
				}
				break
			}
		}
		
		if err != nil {
			// Could not parse date, return original title
			return title, nil
		}
	}
	
	// Remove the due:date part from title
	cleanTitle := dueRegex.ReplaceAllString(title, "")
	cleanTitle = strings.TrimSpace(cleanTitle)
	
	return cleanTitle, &deadline
}

// nextWeekday returns the next occurrence of the given weekday
func nextWeekday(from time.Time, weekday time.Weekday) time.Time {
	days := int(weekday - from.Weekday())
	if days <= 0 {
		days += 7
	}
	return from.AddDate(0, 0, days)
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
