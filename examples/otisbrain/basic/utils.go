package basic

import (
	"os"
	"strings"
)

// ExtractNamespaceFromPath extracts the namespace from the file path
func ExtractNamespaceFromPath(filePath string) string {
	// Path format: logs/basic/{namespace}/
	parts := strings.Split(filePath, string(os.PathSeparator))
	for i, part := range parts {
		if part == "basic" && i+1 < len(parts) {
			return parts[i+1] // The next part should be the namespace
		}
	}
	return "default" // fallback to default namespace
}

// CreateDirectory creates a directory if it doesn't exist
func CreateDirectory(path string) error {
	return os.MkdirAll(path, 0755)
}

// WriteFile writes content to a file
func WriteFile(filename, content string) error {
	return os.WriteFile(filename, []byte(content), 0644)
}
