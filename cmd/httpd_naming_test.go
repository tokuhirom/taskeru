package cmd

import (
	"strings"
	"testing"
)

func TestNoConfluenceSpecificNaming(t *testing.T) {
	// Check that CSS styles don't contain Confluence-specific naming
	if strings.Contains(cssStyles, "confluence") || strings.Contains(cssStyles, "Confluence") {
		t.Error("CSS should not contain Confluence-specific naming")
	}

	// Check for generic naming
	if !strings.Contains(cssStyles, "daily-entry") {
		t.Error("CSS should contain generic daily-entry class")
	}

	if !strings.Contains(cssStyles, "task-note") {
		t.Error("CSS should contain generic task-note class")
	}
}

func TestGenericNamingInTemplates(t *testing.T) {
	// Note: In a real test, we would load and check the actual templates
	// For now, we verify the CSS doesn't have vendor-specific names

	cssContent := cssStyles

	// Should not contain vendor-specific terms
	vendorTerms := []string{"confluence", "jira", "slack", "teams", "notion"}
	for _, term := range vendorTerms {
		if strings.Contains(strings.ToLower(cssContent), term) {
			t.Errorf("CSS should not contain vendor-specific term: %s", term)
		}
	}

	// Should contain generic terms
	genericTerms := []string{"task", "note", "daily", "kanban", "project"}
	for _, term := range genericTerms {
		if !strings.Contains(strings.ToLower(cssContent), term) {
			t.Errorf("CSS should contain generic term: %s", term)
		}
	}
}
