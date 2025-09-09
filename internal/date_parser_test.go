package internal

import (
	"testing"
)

func TestParseNaturalDate(t *testing.T) {
	// Set a fixed "now" for predictable testing
	// Note: The actual implementation uses time.Now(), so results may vary
	// These tests verify that the parser accepts various formats

	tests := []struct {
		input       string
		shouldParse bool
		description string
	}{
		// Natural language relative dates
		{"next tuesday", true, "Should parse 'next tuesday'"},
		{"next week", true, "Should parse 'next week'"},
		{"in 3 days", true, "Should parse 'in 3 days'"},
		{"in 2 weeks", true, "Should parse 'in 2 weeks'"},
		{"tomorrow at 3pm", true, "Should parse 'tomorrow at 3pm'"},
		{"next friday at 5pm", true, "Should parse 'next friday at 5pm'"},
		{"in 1 month", true, "Should parse 'in 1 month'"},
		{"next month", true, "Should parse 'next month'"},

		// Traditional formats (fallback)
		{"today", true, "Should parse 'today'"},
		{"tomorrow", true, "Should parse 'tomorrow'"},
		{"monday", true, "Should parse 'monday'"},
		{"2024-12-31", true, "Should parse ISO date"},
		{"12/25", true, "Should parse MM/DD"},
		{"12-25", true, "Should parse MM-DD"},

		// Invalid inputs
		{"gibberish", false, "Should not parse gibberish"},
		{"", false, "Should not parse empty string"},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			result, _ := ParseNaturalDate(tt.input)

			if tt.shouldParse && result == nil {
				t.Errorf("Expected to parse '%s', but got nil", tt.input)
			}

			if !tt.shouldParse && result != nil {
				t.Errorf("Expected not to parse '%s', but got %v", tt.input, result)
			}

			// If parsed, verify it's set to end of day
			if result != nil {
				hour := result.Hour()
				if hour != 23 {
					t.Errorf("Expected hour to be 23 (end of day), got %d", hour)
				}
			}
		})
	}
}
