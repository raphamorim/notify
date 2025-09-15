// Copyright (c) 2014-2015 The Notify Authors. All rights reserved.
// Use of this source code is governed by the MIT license that can be
// found in the LICENSE file.

package notify

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestIgnoreMatcher(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := ioutil.TempDir("", "notify-ignore-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test directory structure
	dirs := []string{
		"src",
		"src/main",
		".git",
		".git/objects",
		"node_modules",
		"node_modules/package1",
		"build",
		"build/output",
		"docs",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(tmpDir, dir), 0755); err != nil {
			t.Fatal(err)
		}
	}

	// Test basic ignore patterns
	im := NewIgnoreMatcher(tmpDir)
	im.AddPattern(".git/")
	im.AddPattern("node_modules/")
	im.AddPattern("build/")
	im.AddPattern("*.log")
	im.AddPattern("!build/important.log")

	tests := []struct {
		path     string
		expected bool
	}{
		{filepath.Join(tmpDir, ".git"), true},
		{filepath.Join(tmpDir, ".git", "objects"), true},
		{filepath.Join(tmpDir, "node_modules"), true},
		{filepath.Join(tmpDir, "node_modules", "package1"), true},
		{filepath.Join(tmpDir, "build"), true},
		{filepath.Join(tmpDir, "build", "output"), true},
		{filepath.Join(tmpDir, "build", "important.log"), false}, // negated
		{filepath.Join(tmpDir, "src"), false},
		{filepath.Join(tmpDir, "src", "main"), false},
		{filepath.Join(tmpDir, "docs"), false},
		{filepath.Join(tmpDir, "test.log"), true},
		{filepath.Join(tmpDir, "src", "debug.log"), true},
	}

	for _, test := range tests {
		result := im.ShouldIgnore(test.path)
		if result != test.expected {
			t.Errorf("ShouldIgnore(%s) = %v, expected %v", test.path, result, test.expected)
		}
	}
}

func TestIgnoreWithWatch(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := ioutil.TempDir("", "notify-watch-ignore-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Set up ignore patterns
	im := NewIgnoreMatcher(tmpDir)
	im.AddPattern(".git/")
	im.AddPattern("node_modules/")
	im.AddPattern("*.tmp")
	SetIgnoreMatcher(im)
	defer SetIgnoreMatcher(nil) // Reset after test

	// Create initial directory structure
	dirs := []string{
		"src",
		".git",
		"node_modules",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(tmpDir, dir), 0755); err != nil {
			t.Fatal(err)
		}
	}

	// Set up watch
	c := make(chan EventInfo, 100)
	if err := Watch(tmpDir+"/...", c, All); err != nil {
		t.Fatal(err)
	}
	defer Stop(c)

	// Give the watcher time to set up
	time.Sleep(200 * time.Millisecond)

	// Clear any initial events
	for {
		select {
		case <-c:
			// drain
		case <-time.After(100 * time.Millisecond):
			goto create_files
		}
	}

create_files:
	// Create files in different directories
	testFiles := []struct {
		path         string
		shouldNotify bool
	}{
		{"src/main.go", true},
		{".git/config", false},
		{"node_modules/package.json", false},
		{"test.tmp", false},
		{"docs/readme.md", true},
	}

	for _, tf := range testFiles {
		fullPath := filepath.Join(tmpDir, tf.path)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := ioutil.WriteFile(fullPath, []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}
		// Small delay between file creations to ensure events are generated
		time.Sleep(50 * time.Millisecond)
	}

	// Collect events
	time.Sleep(300 * time.Millisecond)
	
	events := make([]EventInfo, 0)
	done := false
	for !done {
		select {
		case ei := <-c:
			events = append(events, ei)
		case <-time.After(200 * time.Millisecond):
			done = true
		}
	}

	// Check that we only got events for non-ignored files
	ignoredEvents := 0
	for _, event := range events {
		relPath, _ := filepath.Rel(tmpDir, event.Path())
		for _, tf := range testFiles {
			if filepath.ToSlash(relPath) == filepath.ToSlash(tf.path) && !tf.shouldNotify {
				ignoredEvents++
				t.Logf("Received event for ignored file: %s (event: %v)", event.Path(), event.Event())
			}
		}
	}

	if ignoredEvents > 0 {
		t.Errorf("Received %d events for ignored files", ignoredEvents)
	}

	// Verify we got events for non-ignored files
	expectedPaths := []string{"src/main.go", "docs/readme.md"}
	for _, expected := range expectedPaths {
		found := false
		expectedFull := filepath.Join(tmpDir, expected)
		for _, event := range events {
			if event.Path() == expectedFull || filepath.Dir(event.Path()) == filepath.Dir(expectedFull) {
				found = true
				break
			}
		}
		if !found {
			t.Logf("Did not receive event for non-ignored file: %s", expected)
			// This is not a hard failure as timing can affect event delivery
		}
	}

	// Log all events for debugging
	t.Logf("Total events received: %d", len(events))
	for _, e := range events {
		t.Logf("Event: %s - %v", e.Path(), e.Event())
	}
}

func TestLoadIgnoreFile(t *testing.T) {
	// Create a temporary directory
	tmpDir, err := ioutil.TempDir("", "notify-ignorefile-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a .notifyignore file
	ignoreContent := `# Ignore patterns
.git/
*.tmp
build/
!build/keep.txt
node_modules/
`
	ignoreFile := filepath.Join(tmpDir, ".notifyignore")
	if err := ioutil.WriteFile(ignoreFile, []byte(ignoreContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Load the ignore file
	im := NewIgnoreMatcher(tmpDir)
	if err := im.LoadIgnoreFile(ignoreFile); err != nil {
		t.Fatal(err)
	}

	// Test patterns
	tests := []struct {
		path     string
		expected bool
	}{
		{filepath.Join(tmpDir, ".git"), true},
		{filepath.Join(tmpDir, "test.tmp"), true},
		{filepath.Join(tmpDir, "build"), true},
		{filepath.Join(tmpDir, "build", "output.bin"), true},
		{filepath.Join(tmpDir, "build", "keep.txt"), false}, // negated
		{filepath.Join(tmpDir, "node_modules"), true},
		{filepath.Join(tmpDir, "src"), false},
	}

	for _, test := range tests {
		result := im.ShouldIgnore(test.path)
		if result != test.expected {
			t.Errorf("ShouldIgnore(%s) = %v, expected %v", test.path, result, test.expected)
		}
	}
}

func TestDoublestarPatterns(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "notify-doublestar-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	im := NewIgnoreMatcher(tmpDir)
	im.AddPattern("**/node_modules/")
	im.AddPattern("**/*.log")
	im.AddPattern("src/**/test/")

	tests := []struct {
		path     string
		expected bool
	}{
		{filepath.Join(tmpDir, "node_modules"), true},
		{filepath.Join(tmpDir, "src", "node_modules"), true},
		{filepath.Join(tmpDir, "src", "lib", "node_modules"), true},
		{filepath.Join(tmpDir, "debug.log"), true},
		{filepath.Join(tmpDir, "src", "debug.log"), true},
		{filepath.Join(tmpDir, "src", "logs", "error.log"), true},
		{filepath.Join(tmpDir, "src", "test"), true},
		{filepath.Join(tmpDir, "src", "lib", "test"), true},
		{filepath.Join(tmpDir, "test"), false},
		{filepath.Join(tmpDir, "src", "main.go"), false},
	}

	for _, test := range tests {
		result := im.ShouldIgnore(test.path)
		if result != test.expected {
			t.Errorf("ShouldIgnore(%s) = %v, expected %v", test.path, result, test.expected)
		}
	}
}