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

	// Normalize path separators
	relPath = filepath.ToSlash(relPath)

	info, err := os.Stat(path)
	isDir := err == nil && info.IsDir()

	ignored := false
	for _, p := range im.patterns {
		// For directory patterns, check if path is or is under that directory
		if p.isDir {
			if isDir && im.matchPattern(p.pattern, relPath) {
				if p.isNegate {
					ignored = false
				} else {
					ignored = true
				}
			} else if !isDir {
				// Check if file is under a directory that should be ignored
				dir := filepath.ToSlash(filepath.Dir(relPath))
				if im.matchPattern(p.pattern, dir) || strings.HasPrefix(relPath, p.pattern+"/") {
					if p.isNegate {
						ignored = false
					} else {
						ignored = true
					}
				}
			}
		} else {
			// Regular pattern matching
			if im.matchPattern(p.pattern, relPath) {
				if p.isNegate {
					ignored = false
				} else {
					ignored = true
				}
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
	parts := strings.Split(pattern, "**")
	if len(parts) != 2 {
		// Multiple ** not supported, fall back to simple match
		return false
	}

	prefix := strings.TrimSuffix(parts[0], "/")
	suffix := strings.TrimPrefix(parts[1], "/")

	// Check if path starts with prefix
	if prefix != "" && !strings.HasPrefix(path, prefix) {
		return false
	}

	// Check if path ends with suffix
	if suffix != "" {
		// Find suffix in path after prefix
		remaining := strings.TrimPrefix(path, prefix)
		remaining = strings.TrimPrefix(remaining, "/")
		
		// Check all possible positions for suffix
		pathParts := strings.Split(remaining, "/")
		for i := 0; i <= len(pathParts); i++ {
			subPath := strings.Join(pathParts[i:], "/")
			if matched, _ := filepath.Match(suffix, subPath); matched {
				return true
			}
		}
	} else {
		// No suffix, just check prefix
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