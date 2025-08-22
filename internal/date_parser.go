package internal

import (
	"regexp"
	"strings"
	"time"

	"github.com/tj/go-naturaldate"
)

// ParseNaturalDate parses a natural language date string
// It supports formats like:
// - "next tuesday"
// - "tomorrow at 3pm"
// - "in 2 weeks"
// - "last monday"
// - "2024-12-31" (fallback to traditional parsing)
func ParseNaturalDate(input string) (*time.Time, error) {
	// Empty input returns nil
	if strings.TrimSpace(input) == "" {
		return nil, nil
	}

	// First try traditional date parsing for exact formats and simple keywords
	traditionalResult, _ := parseTraditionalDate(input)
	if traditionalResult != nil {
		return traditionalResult, nil
	}

	// Only use natural language parsing for phrases that likely contain valid date expressions
	// The library is too lenient and parses random words as "now"
	lowerInput := strings.ToLower(strings.TrimSpace(input))

	// Check if input contains natural language date indicators
	naturalPhrases := []string{
		"next ", "last ", "in ", "ago", "from now", "tomorrow at", "yesterday at",
		"this ", "coming ", "following ",
	}

	isNaturalPhrase := false
	for _, phrase := range naturalPhrases {
		if strings.Contains(lowerInput, phrase) {
			isNaturalPhrase = true
			break
		}
	}

	if !isNaturalPhrase {
		// Not a natural language phrase, don't try to parse
		return nil, nil
	}

	// Try natural language parsing
	result, err := naturaldate.Parse(input, time.Now(), naturaldate.WithDirection(naturaldate.Future))
	if err != nil {
		return nil, nil
	}

	// Set to end of day for deadlines
	endOfDay := time.Date(result.Year(), result.Month(), result.Day(), 23, 59, 59, 0, result.Location())
	return &endOfDay, nil
}

// parseTraditionalDate handles traditional date formats
func parseTraditionalDate(dateStr string) (*time.Time, error) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// Handle simple keywords
	switch strings.ToLower(dateStr) {
	case "today":
		deadline := today.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
		return &deadline, nil
	case "tomorrow":
		deadline := today.AddDate(0, 0, 1).Add(23*time.Hour + 59*time.Minute + 59*time.Second)
		return &deadline, nil
	case "monday", "mon":
		deadline := nextWeekday(today, time.Monday).Add(23*time.Hour + 59*time.Minute + 59*time.Second)
		return &deadline, nil
	case "tuesday", "tue":
		deadline := nextWeekday(today, time.Tuesday).Add(23*time.Hour + 59*time.Minute + 59*time.Second)
		return &deadline, nil
	case "wednesday", "wed":
		deadline := nextWeekday(today, time.Wednesday).Add(23*time.Hour + 59*time.Minute + 59*time.Second)
		return &deadline, nil
	case "thursday", "thu":
		deadline := nextWeekday(today, time.Thursday).Add(23*time.Hour + 59*time.Minute + 59*time.Second)
		return &deadline, nil
	case "friday", "fri":
		deadline := nextWeekday(today, time.Friday).Add(23*time.Hour + 59*time.Minute + 59*time.Second)
		return &deadline, nil
	case "saturday", "sat":
		deadline := nextWeekday(today, time.Saturday).Add(23*time.Hour + 59*time.Minute + 59*time.Second)
		return &deadline, nil
	case "sunday", "sun":
		deadline := nextWeekday(today, time.Sunday).Add(23*time.Hour + 59*time.Minute + 59*time.Second)
		return &deadline, nil
	}

	// Try standard date formats
	formats := []string{
		"2006-01-02",
		"2006/01/02",
		"01-02",
		"01/02",
		"1/2",
		"1-2",
	}

	for _, format := range formats {
		parsed, err := time.Parse(format, dateStr)
		if err == nil {
			// If year is not specified, use current year
			if format == "01-02" || format == "01/02" || format == "1/2" || format == "1-2" {
				deadline := time.Date(now.Year(), parsed.Month(), parsed.Day(), 23, 59, 59, 0, now.Location())
				// If the date has already passed this year, assume next year
				if deadline.Before(now) {
					deadline = deadline.AddDate(1, 0, 0)
				}
				return &deadline, nil
			}
			// Set time to end of day
			deadline := time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 23, 59, 59, 0, now.Location())
			return &deadline, nil
		}
	}

	return nil, nil
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
