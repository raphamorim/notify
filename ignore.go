// Copyright (c) 2014-2015 The Notify Authors. All rights reserved.
// Use of this source code is governed by the MIT license that can be
// found in the LICENSE file.

package notify

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/sabhiram/go-gitignore"
)

// IgnoreMatcher provides gitignore-style pattern matching for paths
type IgnoreMatcher struct {
	patterns  []string
	gitignore *ignore.GitIgnore
	root      string
}

// NewIgnoreMatcher creates a new ignore matcher with the given root directory
func NewIgnoreMatcher(root string) *IgnoreMatcher {
	return &IgnoreMatcher{
		root:     root,
		patterns: make([]string, 0),
	}
}

// AddPattern adds a gitignore-style pattern to the matcher
func (im *IgnoreMatcher) AddPattern(pattern string) {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" || strings.HasPrefix(pattern, "#") {
		return
	}

	im.patterns = append(im.patterns, pattern)
	im.recompile()
}

// recompile rebuilds the gitignore matcher with all current patterns
func (im *IgnoreMatcher) recompile() {
	if len(im.patterns) == 0 {
		im.gitignore = nil
		return
	}
	im.gitignore = ignore.CompileIgnoreLines(im.patterns...)
}

// LoadIgnoreFile loads patterns from a .gitignore or .notifyignore file
func (im *IgnoreMatcher) LoadIgnoreFile(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil // Ignore file doesn't exist, which is fine
	}

	// Read the file and add each line as a pattern
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			im.patterns = append(im.patterns, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	im.recompile()
	return nil
}

// ShouldIgnore returns true if the given path should be ignored
func (im *IgnoreMatcher) ShouldIgnore(path string) bool {
	if im == nil || im.gitignore == nil {
		return false
	}

	// Convert to relative path if absolute
	relPath, err := filepath.Rel(im.root, path)
	if err != nil {
		relPath = path
	}

	// Normalize path separators and trim leading ./
	relPath = filepath.ToSlash(relPath)
	relPath = strings.TrimPrefix(relPath, "./")

	// Check if path matches
	if im.gitignore.MatchesPath(relPath) {
		return true
	}

	// Also check with trailing slash for directory patterns
	// This handles the case where .git/ should match .git directory
	if im.gitignore.MatchesPath(relPath + "/") {
		return true
	}

	return false
}

// DefaultIgnorePatterns returns common patterns that should be ignored by default
func DefaultIgnorePatterns() []string {
	return []string{
		".git/",
		".svn/",
		".hg/",
		".bzr/",
		"node_modules/",
		"vendor/",
		"*.swp",
		"*.swo",
		"*~",
		".DS_Store",
		"Thumbs.db",
		"__pycache__/",
		"*.pyc",
		".idea/",
		".vscode/",
		"*.log",
		".notifyignore",
		".gitignore",
	}
}