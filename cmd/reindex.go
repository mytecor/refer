package cmd

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/meain/refer/internal"
)

func (r Reindex) Run(app *App) error {
	return handleReindex(app.Context, app.Database, app.DatabasePath)
}

func handleReindex(ctx context.Context, database *sql.DB, databasePath string) error {
	sampleEmbedding, err := internal.CreateEmbedding(ctx, "refer")
	if err != nil {
		return fmt.Errorf("failed to create embedding: %w", err)
	}

	embeddingSize := len(sampleEmbedding)
	tempFile := os.TempDir() + "referdb"
	tempDB, _, err := internal.CreateDB(tempFile)
	if err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}
	defer tempDB.Close()

	if err := internal.InitDatabase(tempDB, len(sampleEmbedding)); err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	if err := internal.SaveConfig(
		tempDB,
		map[string]string{
			"embedding_model": internal.Model,
			"embedding_size":  fmt.Sprintf("%d", embeddingSize),
		},
	); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	originalConfig, err := internal.GetConfig(database)
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}

	originalCount := 0
	changedCount := 0

	if originalConfig["embedding_model"] != internal.Model ||
		originalConfig["embedding_size"] != fmt.Sprintf("%d", embeddingSize) {
		docs, err := internal.GetAllFilePaths(database)
		if err != nil {
			return fmt.Errorf("failed to get existing documents: %w", err)
		}

		if errors := internal.AddDocuments(ctx, tempDB, docs, 5); len(errors) > 0 {
			for _, err := range errors {
				log.Printf("Error during reindex: %v", err)
			}
		}

		originalCount = len(docs)
		changedCount = originalCount
	} else {
		docs, err := internal.GetAllDocuments(database)
		if err != nil {
			return fmt.Errorf("failed to get existing documents: %w", err)
		}

		originalCount = len(docs)

		for _, doc := range docs {
			newDoc, err := internal.FetchDocument(doc.Path)
			if err != nil {
				log.Printf("Ignoring missing document: %s", doc.Path)
				continue
			}

			if newDoc.Content != doc.Content {
				emb, err := internal.CreateAndSerializeEmbedding(ctx, newDoc.Content)
				if err != nil {
					return fmt.Errorf("failed to create embedding for %s: %w", doc.Path, err)
				}

				if err := internal.UpdateDocument(tempDB, newDoc, emb); err != nil {
					return fmt.Errorf("failed to update document %s: %w", doc.Path, err)
				}

				changedCount++
			} else {
				emb, err := internal.GetDocumentEmbedding(database, doc.ID)
				if err != nil {
					return fmt.Errorf("failed to get document embedding: %w", err)
				}

				if err := internal.UpdateDocument(tempDB, newDoc, emb); err != nil {
					return fmt.Errorf("failed to update document %s: %w", doc.Path, err)
				}
			}
		}
	}

	if err := os.Rename(tempFile, databasePath); err != nil {
		return fmt.Errorf("failed to update database: %w", err)
	}

	fmt.Println("Successfully reindexed all documents")
	fmt.Printf("Original documents: %d\n", originalCount)
	fmt.Printf("Changed documents: %d\n", changedCount)

	return nil
}
