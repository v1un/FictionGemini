package services

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// SanitizeStringForPath cleans a string to be file system friendly.
func SanitizeStringForPath(input string, makeLower bool) string {
	if makeLower {
		input = strings.ToLower(input)
	}
	input = strings.ReplaceAll(input, " ", "_")

	// Allow basic alphanumeric, underscore, hyphen, period. Remove others.
	reg := regexp.MustCompile("[^a-zA-Z0-9_\\-\\.]+")
	input = reg.ReplaceAllString(input, "")

	maxLen := 50 // Max length for a path component
	if len(input) > maxLen {
		input = input[:maxLen]
	}
	if input == "" {
		return "unnamed" // Default if input becomes empty
	}
	return input
}

// GenerateLogIdentifier creates a unique identifier for a generation session.
func GenerateLogIdentifier(seriesName string) string {
	timestamp := time.Now().Format("20060102_150405.000") // Millisecond precision for uniqueness
	return fmt.Sprintf("%s_%s", SanitizeStringForPath(seriesName, true), timestamp)
}

// SaveJSONToFile saves the jsonData to a file within a specific session's logIdentifier directory.
// baseDir is typically "./jsons".
// subDirType is e.g., "lorebook_comprehensive", "narrator_card".
// itemName is the sanitized name of the specific item being saved (e.g., sanitized lorebook name or card name).
func SaveJSONToFile(baseDir, seriesName, subDirType, itemName, logIdentifier string, jsonData []byte) (string, error) {
	sanitizedSeries := SanitizeStringForPath(seriesName, true)
	sanitizedItemName := SanitizeStringForPath(itemName, false) // Don't force lower for item name, might be a title

	// Construct path: <baseDir>/<sanitizedSeries>/<logIdentifier>/<subDirType>_<sanitizedItemName>.json
	dirPath := filepath.Join(baseDir, sanitizedSeries, logIdentifier)

	if err := os.MkdirAll(dirPath, 0755); err != nil {
		log.Printf("Error creating directory %s: %v", dirPath, err)
		return "", fmt.Errorf("failed to create directory %s: %w", dirPath, err)
	}

	fileName := fmt.Sprintf("%s_%s.json", subDirType, sanitizedItemName)
	if subDirType == "" { // For cases where subDirType might be empty, avoid leading underscore
		fileName = fmt.Sprintf("%s.json", sanitizedItemName)
	}
	
	fullPath := filepath.Join(dirPath, fileName)

	log.Printf("Attempting to save JSON to: %s", fullPath)
	err := os.WriteFile(fullPath, jsonData, 0644)
	if err != nil {
		log.Printf("Error writing file %s: %v", fullPath, err)
		return "", fmt.Errorf("failed to write file %s: %w", fullPath, err)
	}

	log.Printf("Successfully saved JSON to: %s", fullPath)
	return fullPath, nil
}

