package internal

import (
	"strings"
	"testing"
)

func TestGetProjectColor(t *testing.T) {
	tests := []struct {
		name     string
		project  string
		wantPrefix string
	}{
		{
			name:     "work project gets consistent color",
			project:  "work",
			wantPrefix: "\x1b[38;5;",
		},
		{
			name:     "personal project gets consistent color",
			project:  "personal",
			wantPrefix: "\x1b[38;5;",
		},
		{
			name:     "same project returns same color",
			project:  "test",
			wantPrefix: "\x1b[38;5;",
		},
		{
			name:     "empty project still gets color",
			project:  "",
			wantPrefix: "\x1b[38;5;",
		},
		{
			name:     "unicode project gets color",
			project:  "日本語プロジェクト",
			wantPrefix: "\x1b[38;5;",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetProjectColor(tt.project)
			
			// Check if it starts with ANSI color code
			if !strings.HasPrefix(got, tt.wantPrefix) {
				t.Errorf("GetProjectColor(%q) = %q, want prefix %q", tt.project, got, tt.wantPrefix)
			}
			
			// Check if it ends with 'm' (ANSI color code terminator)
			if !strings.HasSuffix(got, "m") {
				t.Errorf("GetProjectColor(%q) = %q, should end with 'm'", tt.project, got)
			}
			
			// Verify consistency - same input should produce same output
			got2 := GetProjectColor(tt.project)
			if got != got2 {
				t.Errorf("GetProjectColor(%q) not consistent: first=%q, second=%q", tt.project, got, got2)
			}
		})
	}
}

func TestGetProjectColorDistribution(t *testing.T) {
	// Test that different projects get distributed across colors
	projects := []string{
		"work", "personal", "home", "study", "urgent",
		"finance", "health", "travel", "coding", "reading",
		"gaming", "fitness", "music", "cooking", "photography",
		"writing", "blog", "research", "meeting", "shopping",
		"exercise", "hobby", "family", "vacation", "learning",
		"design", "marketing", "testing", "deployment", "documentation",
	}
	
	colorMap := make(map[string][]string)
	
	for _, project := range projects {
		color := GetProjectColor(project)
		colorMap[color] = append(colorMap[color], project)
	}
	
	// We have 30 colors, so with 30 projects we should have good distribution
	// Allow some collisions but not too many
	maxCollisions := 5 // Maximum projects that can share the same color
	
	for color, projectList := range colorMap {
		if len(projectList) > maxCollisions {
			t.Errorf("Too many projects (%d) share the same color %s: %v", 
				len(projectList), color, projectList)
		}
	}
	
	// Should use at least 15 different colors for 30 projects (50% utilization)
	minColors := 15
	if len(colorMap) < minColors {
		t.Errorf("Not enough color variety: only %d colors used for %d projects (expected at least %d)",
			len(colorMap), len(projects), minColors)
	}
}

func TestGetProjectColorValidANSI256(t *testing.T) {
	// Test that all possible hash values produce valid ANSI 256 colors
	testProjects := []string{
		"a", "b", "c", "test", "longer_project_name",
		"UPPERCASE", "MixedCase", "with-dash", "with_underscore",
		"123numbers", "!@#$%^&*()", "unicode日本語", "emoji😀",
	}
	
	for _, project := range testProjects {
		color := GetProjectColor(project)
		
		// Extract the color number from the ANSI code
		// Format should be: \x1b[38;5;XXXm where XXX is 0-255
		if !strings.HasPrefix(color, "\x1b[38;5;") || !strings.HasSuffix(color, "m") {
			t.Errorf("GetProjectColor(%q) returned invalid ANSI format: %q", project, color)
		}
		
		// The color code is between the prefix and 'm'
		colorCode := strings.TrimPrefix(color, "\x1b[38;5;")
		colorCode = strings.TrimSuffix(colorCode, "m")
		
		// Check if it's one of our predefined colors
		validColors := []string{
			"33", "208", "162", "34", "141", "214", "39", "202", "165", "46",
			"135", "220", "45", "196", "171", "118", "99", "215", "51", "205",
			"155", "105", "222", "87", "198", "120", "147", "209", "81", "169",
		}
		
		found := false
		for _, valid := range validColors {
			if colorCode == valid {
				found = true
				break
			}
		}
		
		if !found {
			t.Errorf("GetProjectColor(%q) returned unexpected color code: %s", project, colorCode)
		}
	}
}