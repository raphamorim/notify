notify [![GoDoc](https://godoc.org/github.com/raphamorim/notify?status.svg)](https://godoc.org/github.com/raphamorim/notify)
======

Filesystem event notification library on steroids.

## Features

- **Cross-platform** - Windows, Linux, macOS, BSD
- **Recursive watching** - Watch directories recursively
- **Flexible filtering** - Filter events by type (Create, Write, Remove, Rename)
- **Ignore system** - Exclude unwanted paths using gitignore-style patterns
- **High performance** - Platform-specific implementations for efficiency
- **Robust and tested** - Production ready, used by many projects

## Documentation

[godoc.org/github.com/raphamorim/notify](https://godoc.org/github.com/raphamorim/notify)

## Installation

```bash
go get -u github.com/raphamorim/notify
```

## Quick Start

```go
package main

import (
    "log"
    "github.com/raphamorim/notify"
)

func main() {
    // Make the channel buffered to ensure no event is dropped. Notify will drop
    // an event if the receiver is not able to keep up the sending pace.
    c := make(chan notify.EventInfo, 1)

    // Set up a watchpoint listening for events within a directory tree rooted
    // at current working directory. Dispatch remove events to c.
    if err := notify.Watch("./...", c, notify.Create, notify.Write); err != nil {
        log.Fatal(err)
    }
    defer notify.Stop(c)

    // Block until an event is received.
    for {
        select {
        case ei := <-c:
            log.Println("Got event:", ei)
        }
    }
}
```

## Ignore System

The notify library includes a powerful ignore system that filters out unwanted file system events using gitignore-style patterns.

### Basic Usage

```go
// Method 1: Set ignore patterns directly
err := notify.SetIgnorePatterns([]string{
    ".git/",
    "node_modules/",
    "*.tmp",
    "*.log",
    "!important.log",  // Don't ignore important.log
})

// Method 2: Load from a .notifyignore file
err := notify.LoadIgnoreFile(".notifyignore")

// Method 3: Use default ignore patterns
err := notify.EnableDefaultIgnorePatterns()
```

### Advanced Usage

```go
// Create a custom IgnoreMatcher
im := notify.NewIgnoreMatcher("/path/to/project")
im.AddPattern("vendor/")
im.AddPattern("**/*.bak")
im.AddPattern("build/")
im.AddPattern("!build/important/")  // Don't ignore this subdirectory

// Load additional patterns from a file
im.LoadIgnoreFile(".gitignore")

// Set as the global ignore matcher
notify.SetIgnoreMatcher(im)
```

### Pattern Syntax

| Pattern | Description |
|---------|-------------|
| `*.log` | Match all .log files |
| `build/` | Match directories named "build" |
| `/src` | Match "src" at the root only |
| `**/test/` | Match "test" directories at any level |
| `!important.txt` | Negate a pattern (don't ignore) |
| `*.{jpg,png}` | Match multiple extensions |
| `temp*` | Match files/dirs starting with "temp" |

### Example .notifyignore file

```gitignore
# Version control
.git/
.svn/
.hg/

# Dependencies
node_modules/
vendor/
bower_components/

# Build outputs
build/
dist/
*.exe
*.dll
*.so
*.dylib

# Temporary files
*.tmp
*.temp
*.log
*.swp
*.swo
*~

# OS files
.DS_Store
Thumbs.db

# IDE files
.idea/
.vscode/
*.iml

# But keep important files
!important.log
!build/keep/
```

## Event Types

| Event | Description |
|-------|-------------|
| `notify.Create` | File or directory created |
| `notify.Write` | File written to |
| `notify.Remove` | File or directory deleted |
| `notify.Rename` | File or directory renamed |
| `notify.All` | All events |

## Platform Support

| Platform | Adapter | Status |
|----------|---------|--------|
| Linux | inotify | Supported |
| macOS | FSEvents | Supported |
| Windows | ReadDirectoryChangesW | Supported |
| BSD | kqueue | Supported |
| illumos | FEN | Supported |

## Projects using notify

- [github.com/raphamorim/cmd/notify](https://godoc.org/github.com/raphamorim/cmd/notify)
- [github.com/cortesi/devd](https://github.com/cortesi/devd)
- [github.com/cortesi/modd](https://github.com/cortesi/modd)
- [github.com/syncthing/syncthing](https://github.com/syncthing/syncthing)
- [github.com/OrlovEvgeny/TinyJPG](https://github.com/OrlovEvgeny/TinyJPG)
- [github.com/mitranim/gow](https://github.com/mitranim/gow)

## Contributing

We welcome contributions! Please feel free to submit a Pull Request.

## License

The MIT License (MIT). See LICENSE file for more details.