package cmd

import (
	"database/sql"
	"fmt"

	"github.com/meain/refer/internal"
)

func (s StatsCmd) Run(app *App) error {
	return handleStats(app.Database)
}

func handleStats(database *sql.DB) error {
	stats, err := internal.GetDatabaseStats(database)
	if err != nil {
		return fmt.Errorf("failed to get database stats: %w", err)
	}

	fmt.Printf("Documents: %d\n", stats["documents"])
	fmt.Printf("Total Content Size: %s\n", formatBytes(stats["total_content_bytes"]))

	return nil
}
