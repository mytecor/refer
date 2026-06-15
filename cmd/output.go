package cmd

import (
	"fmt"

	"github.com/meain/refer/internal"
)

func formatBytes(bytes int) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func printNameResults(docs []internal.Document) {
	for _, doc := range docs {
		fmt.Printf("%d: %s (%.4f)\n", doc.ID, doc.Path, doc.Distance)
	}
}

func printLLMResults(docs []internal.Document) {
	for _, doc := range docs {
		if doc.Title != doc.Path {
			fmt.Printf("File: %s\nTitle: %s\n\n```\n%s\n```\n---\n", doc.Path, doc.Title, doc.Content)
		} else {
			fmt.Printf("File: %s\n\n```\n%s\n```\n---\n", doc.Path, doc.Content)
		}
	}
}
