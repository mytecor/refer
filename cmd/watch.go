package cmd

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/meain/refer/internal"
)

func (w Watch) Run(app *App) error {
	return handleWatch(app.Context, app.Database, w.Path, app.Config)
}

func handleWatch(ctx context.Context, database *sql.DB, path string, cfg *internal.Config) error {
	if err := internal.WatchAndIndex(ctx, database, path, cfg); err != nil {
		return fmt.Errorf("watch failed: %w", err)
	}

	return nil
}
