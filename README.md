# relouderul

A Go-based file watcher and process manager that automatically restarts a monitored service when file changes are detected.

## Overview

**relouderul** watches specified directories for changes (particularly `.py` files) and automatically restarts the managed process. It's designed to simplify development workflows where you need automatic service restarts on code changes.

## Features

- **File Watching**: Monitors directories recursively for changes using `fsnotify`
- **Auto-Restart**: Automatically restarts the managed process when changes are detected
- **Debouncing**: Prevents rapid restarts with a 500ms debounce timer
- **Service Management**: Supports multiple services via JSON configuration
- **Graceful Shutdown**: Handles OS signals (SIGINT, SIGTERM) properly
- **Process Group Management**: Kills entire process groups for clean restarts

## Installation

```bash
make build
```

## Usage

```bash
# Run a service
./dist/rld --service=<service-name>

# Edit config file
./dist/rld --edit
```

## Configuration

The config file is stored at `~/.config/relouderul/config.json`. If it doesn't exist, a template will be created automatically the first time you run the program.

### Editing the Config

Use the `--edit` flag to open the config file in your default editor:

```bash
./dist/rld --edit
```

Set the `EDITOR` environment variable to specify your preferred editor (e.g., `export EDITOR=vim`).

### Config Structure

Each service needs:
- `path`: The working directory
- `name`: Service identifier
- `command`: Command to run (as array)
- `watch_path`: Directory to watch for changes

## Development

### Prerequisites

- Go 1.24+

### Running in Development Mode

```bash
go run main.go 
```

Without flags, this will prompt for a service name. Provide the `--service` flag to specify which service to run:

```bash
go run main.go --service=first
```

### Rebuild on Changes

For automatic rebuilding of the relouderul binary itself, you can use a tool like `air` or run a separate file watcher on the Go source files.

### Debugging

Logs are written to `./data/kubernetes.log` by default. Check this file for runtime information about watched events, process restarts, and any errors.