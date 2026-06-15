package cmd

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/meain/refer/internal"
)

func (s Show) Run(app *App) error {
	if s.ID == nil {
		return handleShowAll(app.Database)
	}

	return handleShowByID(app.Database, *s.ID)
}

func handleShowAll(database *sql.DB) error {
	docs, err := internal.GetAllDocuments(database)
	if err != nil {
		return fmt.Errorf("failed to get documents: %w", err)
	}
	if len(docs) == 0 {
		log.Println("No documents found in database")
		return nil
	}
	for _, doc := range docs {
		fmt.Printf("[%d] %s\n", doc.ID, doc.Title)
	}

	return nil
}

func handleShowByID(database *sql.DB, id int) error {
	doc, err := internal.GetDocumentByID(database, id)
	if err != nil {
		return fmt.Errorf("failed to get document with ID %d: %w", id, err)
	}
	if doc == nil {
		return fmt.Errorf("no document found with ID %d", id)
	}
	fmt.Printf("%s\n%s\n", doc.Path, doc.Content)

	return nil
}
