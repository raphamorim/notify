// Copyright (c) 2014-2015 The Notify Authors. All rights reserved.
// Use of this source code is governed by the MIT license that can be
// found in the LICENSE file.

package notify

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// IgnoreMatcher provides gitignore-style pattern matching for paths
type IgnoreMatcher struct {
	patterns []ignorePattern
	root     string
}

type ignorePattern struct {
	pattern  string
	isNegate bool
	isDir    bool
}

// NewIgnoreMatcher creates a new ignore matcher with the given root directory
func NewIgnoreMatcher(root string) *IgnoreMatcher {
	return &IgnoreMatcher{
		root:     root,
		patterns: make([]ignorePattern, 0),
	}
}

// AddPattern adds a gitignore-style pattern to the matcher
func (im *IgnoreMatcher) AddPattern(pattern string) {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" || strings.HasPrefix(pattern, "#") {
		return
	}

	p := ignorePattern{pattern: pattern}

	// Handle negation
	if strings.HasPrefix(pattern, "!") {
		p.isNegate = true
		p.pattern = pattern[1:]
	}

	// Handle directory-only patterns
	if strings.HasSuffix(p.pattern, "/") {
		p.isDir = true
		p.pattern = strings.TrimSuffix(p.pattern, "/")
	}

	im.patterns = append(im.patterns, p)
}

// LoadIgnoreFile loads patterns from a .gitignore or .notifyignore file
func (im *IgnoreMatcher) LoadIgnoreFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Ignore file doesn't exist, which is fine
		}
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		im.AddPattern(scanner.Text())
	}

	return scanner.Err()
}

// ShouldIgnore returns true if the given path should be ignored
func (im *IgnoreMatcher) ShouldIgnore(path string) bool {
	if im == nil || len(im.patterns) == 0 {
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

	// Determine if path is a directory syntactically to avoid FS stat flakiness
	isDir := strings.HasSuffix(relPath, "/")
	if !isDir {
		if fi, err := os.Stat(path); err == nil && fi.IsDir() {
			isDir = true
		}
	}

	ignored := false
	for _, p := range im.patterns {
		pat := strings.TrimPrefix(p.pattern, "./")

		// Directory patterns should match the dir itself or anything under it
		if p.isDir {
			// Exact dir match
			if im.matchPattern(pat, relPath) || strings.HasPrefix(relPath+"/", pat+"/") {
				if p.isNegate {
					ignored = false
				} else {
					ignored = true
				}
				continue
			}
		}

		// Regular pattern matching (files or generic globs)
		if im.matchPattern(pat, relPath) {
			if p.isNegate {
				ignored = false
			} else {
				ignored = true
			}
		}
	}

	return ignored
}

// matchPattern implements gitignore-style pattern matching
func (im *IgnoreMatcher) matchPattern(pattern, path string) bool {
	// Handle patterns starting with /
	if strings.HasPrefix(pattern, "/") {
		pattern = pattern[1:]
		return im.matchGlob(pattern, path)
	}

	// Check exact match first
	if im.matchGlob(pattern, path) {
		return true
	}

	// Pattern can match at any level
	parts := strings.Split(path, "/")
	for i := range parts {
		subPath := strings.Join(parts[i:], "/")
		if im.matchGlob(pattern, subPath) {
			return true
		}
		// Also check individual directory names
		if im.matchGlob(pattern, parts[i]) {
			return true
		}
	}

	return false
}

// matchGlob implements basic glob matching
func (im *IgnoreMatcher) matchGlob(pattern, path string) bool {
	// Handle ** for recursive matching
	if strings.Contains(pattern, "**") {
		return im.matchDoublestar(pattern, path)
	}

	// Simple glob matching
	matched, _ := filepath.Match(pattern, path)
	if matched {
		return true
	}

	// Check if pattern matches any parent directory
	parts := strings.Split(path, "/")
	for i := range parts {
		if matched, _ := filepath.Match(pattern, parts[i]); matched {
			return true
		}
	}

	return false
}

// matchDoublestar handles ** patterns
func (im *IgnoreMatcher) matchDoublestar(pattern, path string) bool {
	// Normalize
	pattern = filepath.ToSlash(pattern)
	path = filepath.ToSlash(path)

	// Edge case: pattern is just **
	if strings.Trim(pattern, "/") == "**" {
		return true
	}

	// Split pattern by ** and ensure order
	parts := strings.Split(pattern, "**")
	// Allow multiple ** by scanning sequentially
	cur := path
	leading := strings.TrimSuffix(parts[0], "/")
	if leading != "" {
		if !strings.HasPrefix(cur, strings.TrimPrefix(leading, "/")) {
			return false
		}
		cur = strings.TrimPrefix(cur, strings.TrimPrefix(leading, "/"))
		cur = strings.TrimPrefix(cur, "/")
	}
	// For each remaining segment after **, ensure it appears in order
	for i := 1; i < len(parts); i++ {
		seg := strings.Trim(parts[i], "/")
		if seg == "" {
			continue
		}
		// Find seg anywhere in cur boundaries
		idx := indexGlob(cur, seg)
		if idx == -1 {
			return false
		}
		cur = cur[idx+len(seg):]
	}
	return true
}

// indexGlob finds the first index where glob seg matches in s (prefix match)
func indexGlob(s, seg string) int {
	// Fast path: literal substring
	if !strings.ContainsAny(seg, "*?[") {
		return strings.Index(s, seg)
	}
	// Try all cut points
	for i := 0; i <= len(s); i++ {
		if ok, _ := filepath.Match(seg, s[i:]); ok {
			return i
		}
	}
	return -1
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