package internal

import (
	"regexp"
	"strings"
	"time"
)

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
	// Use the new enhanced parser with natural language support
	return ExtractDeadlineFromTitleV2(title)
}

// ExtractScheduledDateFromTitle extracts scheduled date (scheduled:date or sched:date) from title
func ExtractScheduledDateFromTitle(title string) (string, *time.Time) {
	// Use the new enhanced parser with natural language support
	return ExtractScheduledDateFromTitleV2(title)
}

// ExtractDeadlineFromTitleV2 extracts deadline with enhanced natural language support
func ExtractDeadlineFromTitleV2(title string) (string, *time.Time) {
	// First try the enhanced multi-word pattern for natural language dates
	// Match everything after due: until we hit scheduled: or a project tag or end of string
	naturalRegex := regexp.MustCompile(`\s+due:([^+]+?)(\s+(?:due:|scheduled:|\+)|$)`)
	match := naturalRegex.FindStringSubmatch(title)

	if match != nil {
		dateStr := strings.TrimSpace(match[1])
		deadline, _ := ParseNaturalDate(dateStr)

		if deadline != nil {
			// Remove the matched part from title, but preserve project tags
			// If match[2] contains project tag, we need to preserve it
			replacement := ""
			if strings.TrimSpace(match[2]) != "" {
				replacement = match[2] // Keep the project tag part
			}
			cleanTitle := naturalRegex.ReplaceAllString(title, replacement)
			cleanTitle = strings.TrimSpace(cleanTitle)
			// Clean up any double spaces
			cleanTitle = regexp.MustCompile(`\s+`).ReplaceAllString(cleanTitle, " ")
			return cleanTitle, deadline
		}
	}

	// Fallback to single-word pattern for backward compatibility
	simpleRegex := regexp.MustCompile(`\s+due:(\S+)`)
	match = simpleRegex.FindStringSubmatch(title)
	if match == nil {
		return title, nil
	}

	dateStr := match[1]
	deadline, _ := ParseNaturalDate(dateStr)

	if deadline == nil {
		return title, nil
	}

	// Remove the due:date part from title
	cleanTitle := simpleRegex.ReplaceAllString(title, "")
	cleanTitle = strings.TrimSpace(cleanTitle)

	return cleanTitle, deadline
}

// ExtractScheduledDateFromTitleV2 extracts scheduled date with enhanced natural language support
func ExtractScheduledDateFromTitleV2(title string) (string, *time.Time) {
	// First try the enhanced multi-word pattern for natural language dates
	// Match everything after scheduled: or sched: until we hit due: or a project tag or end of string
	naturalRegex := regexp.MustCompile(`\s+(scheduled|sched):([^+]+?)(\s+(?:due:|scheduled:|sched:|\+)|$)`)
	match := naturalRegex.FindStringSubmatch(title)

	if match != nil {
		dateStr := strings.TrimSpace(match[2]) // match[1] is "scheduled" or "sched", match[2] is the date
		scheduled, _ := ParseNaturalDate(dateStr)

		if scheduled != nil {
			// For scheduled dates, set to start of day
			startOfDay := time.Date(scheduled.Year(), scheduled.Month(), scheduled.Day(), 0, 0, 0, 0, scheduled.Location())

			// Remove the matched part from title, but preserve project tags
			// If match[3] contains project tag, we need to preserve it
			replacement := ""
			if strings.TrimSpace(match[3]) != "" {
				replacement = match[3] // Keep the project tag part
			}
			cleanTitle := naturalRegex.ReplaceAllString(title, replacement)
			cleanTitle = strings.TrimSpace(cleanTitle)
			// Clean up any double spaces
			cleanTitle = regexp.MustCompile(`\s+`).ReplaceAllString(cleanTitle, " ")
			return cleanTitle, &startOfDay
		}
	}

	// Fallback to single-word pattern for backward compatibility
	simpleRegex := regexp.MustCompile(`\s+(scheduled|sched):(\S+)`)
	match = simpleRegex.FindStringSubmatch(title)
	if match == nil {
		return title, nil
	}

	dateStr := match[2] // match[1] is "scheduled" or "sched", match[2] is the date
	scheduled, _ := ParseNaturalDate(dateStr)

	if scheduled == nil {
		return title, nil
	}

	// For scheduled dates, set to start of day
	startOfDay := time.Date(scheduled.Year(), scheduled.Month(), scheduled.Day(), 0, 0, 0, 0, scheduled.Location())

	// Remove the scheduled:date part from title
	cleanTitle := simpleRegex.ReplaceAllString(title, "")
	cleanTitle = strings.TrimSpace(cleanTitle)

	return cleanTitle, &startOfDay
}
