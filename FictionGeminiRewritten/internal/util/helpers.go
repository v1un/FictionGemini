package util

import "strings"

// Min returns the minimum of two integers.
func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ExtractExample is a placeholder for a more sophisticated function to get relevant examples from text.
// For now, it returns a generic string or a snippet.
// TODO: In a future iteration, this could use regex or basic NLP for better snippet extraction.
func ExtractExample(content string, category string) string {
	// This is a very basic placeholder.
	// For now, just return a generic indication or a snippet of the beginning of the content.
	// The category parameter is not used in this basic version but kept for future enhancement.
	trimmedContent := strings.TrimSpace(content)
	if len(trimmedContent) > 30 {
		return trimmedContent[:30] + "..."
	}
	return trimmedContent
}

