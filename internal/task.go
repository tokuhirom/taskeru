package internal

import (
	"time"

	"github.com/google/uuid"
)

type Task struct {
	ID       string     `json:"id"`
	Title    string     `json:"title"`
	Created  time.Time  `json:"created"`
	Updated  time.Time  `json:"updated"`
	DueDate  *time.Time `json:"due_date,omitempty"`
	Priority string     `json:"priority,omitempty"`
	Status   string     `json:"status"`
	Note     string     `json:"note,omitempty"`
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
		t.Status = status
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