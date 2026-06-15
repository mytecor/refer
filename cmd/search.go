package cmd

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"slices"

	"github.com/meain/refer/internal"
)

func (s Search) Run(app *App) error {
	queries := s.Query
	if len(queries) == 0 {
		var err error
		queries, err = handleSearchFromStdin()
		if err != nil {
			return err
		}
	}

	return handleSearch(app.Context, app.Database, queries, s.Format, s.Limit, s.Threshold, s.Rerank)
}

func handleSearchFromStdin() ([]string, error) {
	input, err := readAll(os.Stdin)
	if err != nil {
		return nil, fmt.Errorf("failed to read from stdin: %w", err)
	}

	if len(input) == 0 {
		return nil, fmt.Errorf("no input provided")
	}

	return []string{string(input)}, nil
}

func handleSearch(
	ctx context.Context,
	database *sql.DB,
	queries []string,
	format string,
	limit int,
	threshold *float64,
	rerank bool,
) error {
	docs := []internal.Document{}
	for _, query := range queries {
		queryEmbedding, err := internal.CreateEmbedding(ctx, query)
		if err != nil {
			return fmt.Errorf("failed to create query embedding: %w", err)
		}

		sdocs, err := internal.SearchDocuments(
			database,
			queryEmbedding,
			limit,
			threshold,
		)
		if err != nil {
			return fmt.Errorf("search failed: %w", err)
		}

		docs = append(docs, sdocs...)
	}

	docs = dedupeDocuments(docs)

	if rerank {
		var err error
		docs, err = internal.RerankDocuments(queries[0], docs, limit)
		if err != nil {
			return fmt.Errorf("failed to rerank documents: %w", err)
		}
	}

	slices.SortFunc(docs, func(i, j internal.Document) int {
		return int((i.Distance - j.Distance) * 1000)
	})

	switch format {
	case "names":
		printNameResults(docs)
	case "llm":
		printLLMResults(docs)
	default:
		return fmt.Errorf("unknown format: %s", format)
	}

	return nil
}

func dedupeDocuments(docs []internal.Document) []internal.Document {
	distances := map[string]float64{}
	uniqueDocs := []internal.Document{}
	for _, doc := range docs {
		distance, ok := distances[doc.Path]
		if !ok {
			distances[doc.Path] = doc.Distance
			uniqueDocs = append(uniqueDocs, doc)
			continue
		}

		if doc.Distance < distance {
			distances[doc.Path] = doc.Distance
		}
	}

	return uniqueDocs
}
