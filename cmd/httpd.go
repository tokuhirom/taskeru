package cmd

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"taskeru/internal"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/russross/blackfriday/v2"
)

//go:embed templates/*
var templatesFS embed.FS

var templates *template.Template

func init() {
	funcMap := template.FuncMap{
		"markdown": func(text string) template.HTML {
			output := blackfriday.Run([]byte(text))
			return template.HTML(output)
		},
		"projectColor": func(project string) string {
			// Extract color code from ANSI
			color := internal.GetProjectColor(project)
			// Convert ANSI to CSS color
			if strings.Contains(color, "38;5;") {
				// Extract the color number
				parts := strings.Split(color, ";")
				if len(parts) >= 3 {
					colorNum := strings.TrimSuffix(parts[2], "m")
					return ansi256ToHex(colorNum)
				}
			}
			return "#36b3d9" // Default cyan
		},
		"formatDate": func(t *time.Time) string {
			if t == nil {
				return ""
			}
			return t.Format("2006-01-02")
		},
		"formatDateTime": func(t *time.Time) string {
			if t == nil {
				return ""
			}
			return t.Format("2006-01-02 15:04")
		},
		"monthName": func(month int) string {
			return time.Month(month).String()
		},
		"lower": func(s string) string {
			return strings.ToLower(s)
		},
	}

	var err error
	templates, err = template.New("").Funcs(funcMap).ParseFS(templatesFS, "templates/*.html")
	if err != nil {
		log.Fatal("Failed to parse templates:", err)
	}
}

func HttpdCommand(addr string) error {
	if addr == "" {
		addr = "127.0.0.1:7676"
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Routes
	r.Get("/", kanbanHandler)
	r.Get("/kanban", kanbanHandler)
	r.Get("/daily", dailyReportHandler)
	r.Get("/daily/{year}/{month}", dailyReportHandler)
	r.Get("/api/tasks", apiTasksHandler)
	r.Get("/static/style.css", styleHandler)

	fmt.Printf("Starting HTTP server on http://%s\n", addr)
	fmt.Println("Press Ctrl+C to stop")

	return http.ListenAndServe(addr, r)
}

func kanbanHandler(w http.ResponseWriter, r *http.Request) {
	tasks, err := internal.LoadTasks()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Sort and group tasks by status
	internal.SortTasks(tasks)
	tasksByStatus := groupTasksByStatus(tasks)

	data := struct {
		Title         string
		TasksByStatus map[string][]internal.Task
		Statuses      []string
		ActiveView    string
	}{
		Title:         "Taskeru - Kanban View",
		TasksByStatus: tasksByStatus,
		Statuses:      []string{"TODO", "DOING", "WAITING", "DONE", "WONTDO"},
		ActiveView:    "kanban",
	}

	if err := templates.ExecuteTemplate(w, "kanban_page.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func dailyReportHandler(w http.ResponseWriter, r *http.Request) {
	year := chi.URLParam(r, "year")
	month := chi.URLParam(r, "month")

	now := time.Now()
	var targetYear, targetMonth int

	if year != "" && month != "" {
		targetYear, _ = strconv.Atoi(year)
		targetMonth, _ = strconv.Atoi(month)
	} else {
		targetYear = now.Year()
		targetMonth = int(now.Month())
	}

	targetDate := time.Date(targetYear, time.Month(targetMonth), 1, 0, 0, 0, 0, time.Local)

	tasks, err := internal.LoadTasks()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Group tasks by date
	tasksByDate := groupTasksByDate(tasks, targetDate)

	// Get available months
	availableMonths := getAvailableMonths(tasks)

	data := struct {
		Title           string
		Year            int
		Month           int
		MonthName       string
		TasksByDate     map[string][]internal.Task
		Dates           []string
		AvailableMonths []YearMonth
		ActiveView      string
		PrevMonth       YearMonth
		NextMonth       YearMonth
	}{
		Title:           fmt.Sprintf("Taskeru - Daily Report %d/%02d", targetYear, targetMonth),
		Year:            targetYear,
		Month:           targetMonth,
		MonthName:       targetDate.Month().String(),
		TasksByDate:     tasksByDate,
		Dates:           getSortedDates(tasksByDate),
		AvailableMonths: availableMonths,
		ActiveView:      "daily",
		PrevMonth:       YearMonth{Year: targetDate.AddDate(0, -1, 0).Year(), Month: int(targetDate.AddDate(0, -1, 0).Month())},
		NextMonth:       YearMonth{Year: targetDate.AddDate(0, 1, 0).Year(), Month: int(targetDate.AddDate(0, 1, 0).Month())},
	}

	if err := templates.ExecuteTemplate(w, "daily_page.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func apiTasksHandler(w http.ResponseWriter, r *http.Request) {
	tasks, err := internal.LoadTasks()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}

func styleHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/css")
	w.Write([]byte(cssStyles))
}

func groupTasksByStatus(tasks []internal.Task) map[string][]internal.Task {
	result := make(map[string][]internal.Task)
	statuses := []string{"TODO", "DOING", "WAITING", "DONE", "WONTDO"}

	for _, status := range statuses {
		result[status] = []internal.Task{}
	}

	for _, task := range tasks {
		status := strings.ToUpper(task.Status)
		if _, ok := result[status]; ok {
			result[status] = append(result[status], task)
		} else {
			result["TODO"] = append(result["TODO"], task)
		}
	}

	return result
}

func groupTasksByDate(tasks []internal.Task, targetMonth time.Time) map[string][]internal.Task {
	result := make(map[string][]internal.Task)
	startOfMonth := time.Date(targetMonth.Year(), targetMonth.Month(), 1, 0, 0, 0, 0, time.Local)
	endOfMonth := startOfMonth.AddDate(0, 1, 0).Add(-time.Second)

	for _, task := range tasks {
		// Include tasks updated in the target month
		if task.Updated.After(startOfMonth) && task.Updated.Before(endOfMonth) {
			dateKey := task.Updated.Format("2006-01-02")
			result[dateKey] = append(result[dateKey], task)
		}

		// Also include completed tasks in the target month
		if task.CompletedAt != nil && task.CompletedAt.After(startOfMonth) && task.CompletedAt.Before(endOfMonth) {
			dateKey := task.CompletedAt.Format("2006-01-02")
			// Avoid duplicates
			found := false
			for _, t := range result[dateKey] {
				if t.ID == task.ID {
					found = true
					break
				}
			}
			if !found {
				result[dateKey] = append(result[dateKey], task)
			}
		}
	}

	return result
}

func getSortedDates(tasksByDate map[string][]internal.Task) []string {
	dates := make([]string, 0, len(tasksByDate))
	for date := range tasksByDate {
		dates = append(dates, date)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(dates)))
	return dates
}

type YearMonth struct {
	Year  int
	Month int
}

func getAvailableMonths(tasks []internal.Task) []YearMonth {
	monthMap := make(map[string]bool)

	for _, task := range tasks {
		monthKey := fmt.Sprintf("%d-%02d", task.Updated.Year(), task.Updated.Month())
		monthMap[monthKey] = true

		if task.CompletedAt != nil {
			monthKey = fmt.Sprintf("%d-%02d", task.CompletedAt.Year(), task.CompletedAt.Month())
			monthMap[monthKey] = true
		}
	}

	var months []YearMonth
	for key := range monthMap {
		var year, month int
		fmt.Sscanf(key, "%d-%d", &year, &month)
		months = append(months, YearMonth{Year: year, Month: month})
	}

	sort.Slice(months, func(i, j int) bool {
		if months[i].Year != months[j].Year {
			return months[i].Year > months[j].Year
		}
		return months[i].Month > months[j].Month
	})

	return months
}

func ansi256ToHex(colorNum string) string {
	// Simplified ANSI 256 to hex color mapping
	// This is a subset of common colors used in the project
	colorMap := map[string]string{
		"33":  "#0087ff",
		"208": "#ff8700",
		"162": "#d70087",
		"34":  "#00af00",
		"141": "#af87ff",
		"214": "#ffaf00",
		"39":  "#00afff",
		"202": "#ff5f00",
		"165": "#d700ff",
		"46":  "#00ff00",
		"135": "#af5fff",
		"220": "#ffd700",
		"45":  "#00d7ff",
		"196": "#ff0000",
		"171": "#d75fff",
		"118": "#87ff00",
		"99":  "#875fff",
		"215": "#ffaf5f",
		"51":  "#00ffff",
		"205": "#ff5faf",
		"155": "#afff5f",
		"105": "#8787ff",
		"222": "#ffaf87",
		"87":  "#5fffff",
		"198": "#ff0087",
		"120": "#87ff87",
		"147": "#afafff",
		"209": "#ff875f",
		"81":  "#5fd7ff",
		"169": "#ff87af",
	}

	if hex, ok := colorMap[colorNum]; ok {
		return hex
	}
	return "#36b3d9" // Default
}

const cssStyles = `
:root {
	--bg-primary: #ffffff;
	--bg-secondary: #f5f5f5;
	--text-primary: #333333;
	--text-secondary: #666666;
	--border-color: #e0e0e0;
	--kanban-todo: #f8f9fa;
	--kanban-doing: #fff3cd;
	--kanban-waiting: #cce5ff;
	--kanban-done: #d4edda;
	--kanban-wontdo: #f8d7da;
}

* {
	margin: 0;
	padding: 0;
	box-sizing: border-box;
}

body {
	font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
	background: var(--bg-secondary);
	color: var(--text-primary);
	line-height: 1.6;
}

/* Global Navigation */
.global-nav {
	background: var(--bg-primary);
	border-bottom: 2px solid var(--border-color);
	padding: 1rem 2rem;
	display: flex;
	justify-content: space-between;
	align-items: center;
}

.nav-title {
	font-size: 1.5rem;
	font-weight: bold;
	color: var(--text-primary);
}

.nav-links {
	display: flex;
	gap: 1rem;
}

.nav-links a {
	padding: 0.5rem 1rem;
	text-decoration: none;
	color: var(--text-secondary);
	border-radius: 4px;
	transition: all 0.3s;
}

.nav-links a:hover {
	background: var(--bg-secondary);
	color: var(--text-primary);
}

.nav-links a.active {
	background: #007bff;
	color: white;
}

/* Container */
.container {
	max-width: 1400px;
	margin: 2rem auto;
	padding: 0 1rem;
}

/* Kanban Board */
.kanban-board {
	display: grid;
	grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
	gap: 1rem;
	margin-top: 2rem;
}

.kanban-column {
	background: var(--bg-primary);
	border-radius: 8px;
	padding: 1rem;
	box-shadow: 0 2px 4px rgba(0,0,0,0.1);
}

.kanban-column.todo { background: var(--kanban-todo); }
.kanban-column.doing { background: var(--kanban-doing); }
.kanban-column.waiting { background: var(--kanban-waiting); }
.kanban-column.done { background: var(--kanban-done); }
.kanban-column.wontdo { background: var(--kanban-wontdo); }

.kanban-header {
	font-weight: bold;
	margin-bottom: 1rem;
	padding: 0.5rem;
	text-align: center;
	background: rgba(0,0,0,0.05);
	border-radius: 4px;
}

.kanban-cards {
	display: flex;
	flex-direction: column;
	gap: 0.5rem;
}

.kanban-card {
	background: white;
	padding: 0.75rem;
	border-radius: 4px;
	box-shadow: 0 1px 3px rgba(0,0,0,0.1);
	transition: transform 0.2s;
}

.kanban-card:hover {
	transform: translateY(-2px);
	box-shadow: 0 2px 6px rgba(0,0,0,0.15);
}

.card-priority {
	display: inline-block;
	padding: 0.2rem 0.4rem;
	background: #ff6b6b;
	color: white;
	border-radius: 3px;
	font-size: 0.75rem;
	font-weight: bold;
	margin-right: 0.5rem;
}

.card-title {
	font-weight: 500;
	margin-bottom: 0.25rem;
}

.card-projects {
	display: flex;
	flex-wrap: wrap;
	gap: 0.25rem;
	margin-top: 0.5rem;
}

.project-tag {
	display: inline-block;
	padding: 0.2rem 0.5rem;
	border-radius: 12px;
	font-size: 0.8rem;
	font-weight: 500;
}

/* Daily Report - Simplified */
.month-navigation {
	display: flex;
	justify-content: space-between;
	align-items: center;
	margin-bottom: 2rem;
	padding: 1rem;
	background: var(--bg-primary);
	border-radius: 8px;
	box-shadow: 0 2px 4px rgba(0,0,0,0.1);
}

.month-selector {
	display: flex;
	gap: 0.5rem;
	flex-wrap: wrap;
}

.month-selector a {
	padding: 0.3rem 0.8rem;
	background: var(--bg-secondary);
	color: var(--text-secondary);
	text-decoration: none;
	border-radius: 4px;
	font-size: 0.9rem;
	transition: all 0.3s;
}

.month-selector a:hover {
	background: #007bff;
	color: white;
}

.month-selector a.current {
	background: #007bff;
	color: white;
}

.daily-entries {
	display: flex;
	flex-direction: column;
	gap: 1.5rem;
}

.daily-entry {
	background: var(--bg-primary);
	border-radius: 8px;
	padding: 1rem 1.5rem;
	box-shadow: 0 1px 3px rgba(0,0,0,0.08);
}

.date-header {
	font-size: 1.1rem;
	font-weight: bold;
	margin-bottom: 0.75rem;
	padding-bottom: 0.5rem;
	border-bottom: 2px solid var(--border-color);
	color: var(--text-primary);
}

.task-note h1, .task-note h2, .task-note h3 {
	margin-top: 1rem;
	margin-bottom: 0.5rem;
}

.task-note p {
	margin-bottom: 0.5rem;
}

.task-note ul, .task-note ol {
	margin-left: 2rem;
	margin-bottom: 0.5rem;
}

.task-note code {
	background: #f4f4f4;
	padding: 0.2rem 0.4rem;
	border-radius: 3px;
	font-family: 'Courier New', monospace;
}

.task-note pre {
	background: #f4f4f4;
	padding: 1rem;
	border-radius: 4px;
	overflow-x: auto;
}

.task-note blockquote {
	border-left: 4px solid #007bff;
	padding-left: 1rem;
	margin: 1rem 0;
	color: var(--text-secondary);
}

/* Responsive */
@media (max-width: 768px) {
	.kanban-board {
		grid-template-columns: 1fr;
	}
	
	.month-navigation {
		flex-direction: column;
		gap: 1rem;
	}
}
`
