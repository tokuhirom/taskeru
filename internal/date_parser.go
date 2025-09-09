package internal

import (
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

// nextWeekday returns the next occurrence of the given weekday
func nextWeekday(from time.Time, weekday time.Weekday) time.Time {
	days := int(weekday - from.Weekday())
	if days <= 0 {
		days += 7
	}
	return from.AddDate(0, 0, days)
}
