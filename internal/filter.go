package internal

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
)

type PathFilter struct {
	root           string
	includeMatcher gitignore.Matcher
	ignoreMatcher  gitignore.Matcher
	gitRoot        string
	gitMatcher     gitignore.Matcher
}

func NewPathFilter(root string, cfg *Config, useGitIgnore bool) (*PathFilter, error) {
	filter := &PathFilter{root: filepath.Clean(root)}

	if cfg != nil {
		if len(cfg.Include) > 0 {
			filter.includeMatcher = gitignore.NewMatcher(parsePatterns(cfg.Include))
		}
		if len(cfg.Ignore) > 0 {
			filter.ignoreMatcher = gitignore.NewMatcher(parsePatterns(cfg.Ignore))
		}
	}

	if useGitIgnore {
		gitRoot, err := FindGitDir(filter.root)
		if err == nil {
			patterns, err := LoadGitignorePatterns(gitRoot)
			if err != nil {
				return nil, fmt.Errorf("load gitignore patterns: %w", err)
			}

			filter.gitRoot = gitRoot
			filter.gitMatcher = gitignore.NewMatcher(patterns)
		}
	}

	return filter, nil
}

func CollectFiles(root string, filter *PathFilter) ([]string, error) {
	var paths []string

	err := filepath.WalkDir(root, func(path string, dirEntry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if path != root && filter.ShouldSkip(path, dirEntry.IsDir()) {
			if dirEntry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if !dirEntry.IsDir() && filter.ShouldIndex(path) {
			paths = append(paths, path)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return paths, nil
}

func (f *PathFilter) ShouldSkip(path string, isDir bool) bool {
	if f.matchRoot(f.ignoreMatcher, path, isDir) {
		return true
	}

	if f.matchGit(path, isDir) {
		return true
	}

	return false
}

func (f *PathFilter) ShouldIndex(path string) bool {
	if f.ShouldSkip(path, false) {
		return false
	}

	if f.includeMatcher == nil {
		return true
	}

	return f.matchRoot(f.includeMatcher, path, false)
}

func parsePatterns(patterns []string) []gitignore.Pattern {
	parsed := make([]gitignore.Pattern, 0, len(patterns))
	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}
		parsed = append(parsed, gitignore.ParsePattern(filepath.ToSlash(pattern), nil))
	}
	return parsed
}

func (f *PathFilter) matchRoot(matcher gitignore.Matcher, path string, isDir bool) bool {
	if matcher == nil {
		return false
	}

	relPath, err := filepath.Rel(f.root, path)
	if err != nil {
		return false
	}
	if relPath == "." {
		relPath = filepath.Base(path)
	}

	return matcher.Match(splitPath(relPath), isDir)
}

func (f *PathFilter) matchGit(path string, isDir bool) bool {
	if f.gitMatcher == nil || f.gitRoot == "" {
		return false
	}

	relPath, err := filepath.Rel(f.gitRoot, path)
	if err != nil || relPath == "." {
		return false
	}

	return f.gitMatcher.Match(splitPath(relPath), isDir)
}

func splitPath(path string) []string {
	return strings.Split(filepath.ToSlash(path), "/")
}
