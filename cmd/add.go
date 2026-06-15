package cmd

import (
	"context"
	"database/sql"
	"log"

	"github.com/meain/refer/internal"
)

func (a Add) Run(app *App) error {
	return handleAdd(app.Context, app.Database, app.Config, a.FilePath, a.NoIgnore)
}

func handleAdd(ctx context.Context, database *sql.DB, cfg *internal.Config, filePaths []string, noIgnore bool) error {
	var allPaths []string
	for _, f := range filePaths {
		if internal.IsRemoteURL(f) {
			allPaths = append(allPaths, f)
			continue
		}

		filter, err := internal.NewPathFilter(f, cfg, !noIgnore)
		if err != nil {
			log.Printf("Warning: could not build path filter for %q: %v", f, err)
			filter, _ = internal.NewPathFilter(f, cfg, false)
		}

		paths, err := internal.CollectFiles(f, filter)
		if err != nil {
			log.Printf("Failed to collect files for %q: %v", f, err)
			continue
		}

		allPaths = append(allPaths, paths...)
	}

	if errors := internal.AddDocuments(ctx, database, allPaths, 5); len(errors) > 0 {
		for _, err := range errors {
			log.Printf("Error: %v", err)
		}
	}

	return nil
}
