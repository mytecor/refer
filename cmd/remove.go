package cmd

import (
	"database/sql"
	"fmt"

	"github.com/meain/refer/internal"
)

func (r Remove) Run(app *App) error {
	return handleRemove(app.Database, r.ID)
}

func handleRemove(database *sql.DB, id int) error {
	if err := internal.RemoveDocument(database, id); err != nil {
		return fmt.Errorf("failed to remove document: %w", err)
	}
	fmt.Printf("Document %d removed successfully\n", id)

	return nil
}
