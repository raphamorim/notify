// Copyright (c) 2014-2015 The Notify Authors. All rights reserved.
// Use of this source code is governed by the MIT license that can be
// found in the LICENSE file.

// +build !windows

package notify_test

import (
	"log"
	"path/filepath"

	"github.com/raphamorim/notify"
)

// ExampleSetIgnorePatterns demonstrates how to use the ignore functionality to filter
// out unwanted file system events.
func ExampleSetIgnorePatterns() {
	// Set up ignore patterns - method 1: using patterns directly
	err := notify.SetIgnorePatterns([]string{
		".git/",
		"node_modules/",
		"*.tmp",
		"*.log",
		"build/",
		"!build/important.txt", // Negation: don't ignore this specific file
	})
	if err != nil {
		log.Fatal(err)
	}

	// Method 2: Load from a .gitignore or .notifyignore file
	// err := notify.LoadIgnoreFile(".notifyignore")
	// if err != nil {
	//     log.Fatal(err)
	// }

	// Method 3: Use default ignore patterns
	// err := notify.EnableDefaultIgnorePatterns()
	// if err != nil {
	//     log.Fatal(err)
	// }

	// Method 4: Create a custom IgnoreMatcher
	// im := notify.NewIgnoreMatcher("/path/to/project")
	// im.AddPattern("vendor/")
	// im.AddPattern("*.bak")
	// notify.SetIgnoreMatcher(im)

	// Create a channel to receive events
	c := make(chan notify.EventInfo, 1)

	// Set up a recursive watch on current directory
	if err := notify.Watch("./...", c, notify.All); err != nil {
		log.Fatal(err)
	}
	defer notify.Stop(c)

	// Events from ignored paths will be automatically filtered out
	for {
		select {
		case ei := <-c:
			log.Println("Event:", ei)
		}
	}
}

// ExampleIgnoreMatcher demonstrates how to use the IgnoreMatcher directly
// for custom ignore logic.
func ExampleIgnoreMatcher() {
	// Create an ignore matcher for a specific directory
	projectDir := "/home/user/myproject"
	im := notify.NewIgnoreMatcher(projectDir)

	// Add gitignore-style patterns
	im.AddPattern(".git/")           // Ignore .git directory
	im.AddPattern("**/*.log")        // Ignore all .log files
	im.AddPattern("**/node_modules/") // Ignore node_modules at any level
	im.AddPattern("/build/")         // Ignore build directory at root only
	im.AddPattern("!important.log")  // Don't ignore important.log

	// Load additional patterns from a file
	if err := im.LoadIgnoreFile(filepath.Join(projectDir, ".notifyignore")); err != nil {
		log.Println("Could not load .notifyignore:", err)
	}

	// Check if paths should be ignored
	paths := []string{
		filepath.Join(projectDir, ".git", "config"),
		filepath.Join(projectDir, "src", "debug.log"),
		filepath.Join(projectDir, "important.log"),
		filepath.Join(projectDir, "src", "main.go"),
	}

	for _, path := range paths {
		if im.ShouldIgnore(path) {
			log.Printf("Ignoring: %s\n", path)
		} else {
			log.Printf("Watching: %s\n", path)
		}
	}

	// Set as the global ignore matcher
	notify.SetIgnoreMatcher(im)
}