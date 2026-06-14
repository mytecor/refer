package internal

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
)

func WatchAndIndex(ctx context.Context, db *sql.DB, root string, cfg *Config) error {
	root, err := filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("resolve root path: %w", err)
	}

	info, err := os.Stat(root)
	if err != nil {
		return fmt.Errorf("stat root path: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("watch path must be a directory: %s", root)
	}

	filter, err := NewPathFilter(root, cfg, false)
	if err != nil {
		return err
	}

	paths, err := CollectFiles(root, filter)
	if err != nil {
		return fmt.Errorf("collect files: %w", err)
	}

	if errors := AddDocuments(ctx, db, paths, 5); len(errors) > 0 {
		for _, err := range errors {
			log.Printf("Error: %v", err)
		}
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("create watcher: %w", err)
	}
	defer watcher.Close()

	watchedDirs := map[string]struct{}{}
	if err := addDirectoryWatchers(watcher, watchedDirs, root, filter); err != nil {
		return err
	}

	log.Printf("Watching %s", root)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-watcher.Errors:
			if err != nil {
				log.Printf("Watcher error: %v", err)
			}
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}

			handleWatchEvent(ctx, db, watcher, watchedDirs, filter, event)
		}
	}
}

func addDirectoryWatchers(watcher *fsnotify.Watcher, watchedDirs map[string]struct{}, root string, filter *PathFilter) error {
	return filepath.WalkDir(root, func(path string, dirEntry os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !dirEntry.IsDir() {
			return nil
		}

		if path != root && filter.ShouldSkip(path, true) {
			return filepath.SkipDir
		}

		if _, ok := watchedDirs[path]; ok {
			return nil
		}

		if err := watcher.Add(path); err != nil {
			return fmt.Errorf("watch directory %s: %w", path, err)
		}

		watchedDirs[path] = struct{}{}
		return nil
	})
}

func handleWatchEvent(
	ctx context.Context,
	db *sql.DB,
	watcher *fsnotify.Watcher,
	watchedDirs map[string]struct{},
	filter *PathFilter,
	event fsnotify.Event,
) {
	path := filepath.Clean(event.Name)

	if event.Op&(fsnotify.Remove|fsnotify.Rename) != 0 {
		if _, ok := watchedDirs[path]; ok {
			delete(watchedDirs, path)
			if err := RemoveDocumentsByPrefix(db, path); err != nil {
				log.Printf("Failed to remove documents under %s: %v", path, err)
			}
			return
		}

		if err := RemoveDocumentByPath(db, path); err != nil {
			log.Printf("Failed to remove document %s: %v", path, err)
		}
		return
	}

	if event.Op&(fsnotify.Create|fsnotify.Write) == 0 {
		return
	}

	info, err := os.Stat(path)
	if err != nil {
		return
	}

	if info.IsDir() {
		if filter.ShouldSkip(path, true) {
			return
		}

		if err := addDirectoryWatchers(watcher, watchedDirs, path, filter); err != nil {
			log.Printf("Failed to watch directory %s: %v", path, err)
			return
		}

		paths, err := CollectFiles(path, filter)
		if err != nil {
			log.Printf("Failed to collect files under %s: %v", path, err)
			return
		}

		if errors := AddDocuments(ctx, db, paths, 5); len(errors) > 0 {
			for _, err := range errors {
				log.Printf("Error: %v", err)
			}
		}
		return
	}

	if !filter.ShouldIndex(path) {
		return
	}

	if err := AddDocument(ctx, db, path); err != nil {
		log.Printf("Failed to index %s: %v", path, err)
	}
}
