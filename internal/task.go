package internal

import (
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