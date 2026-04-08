package helper

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fsnotify/fsnotify"
)

func TestWatchRecursive(t *testing.T) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer watcher.Close()

	tmpDir := t.TempDir()
	subdir := filepath.Join(tmpDir, "subdir")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}

	if err := WatchRecursive(watcher, tmpDir); err != nil {
		t.Fatalf("WatchRecursive failed: %v", err)
	}
	paths := watcher.WatchList()
	if len(paths) < 2 {
		t.Errorf("Expected at least 2 paths (tmpDir + subdir), got %d", len(paths))
	}
}

func TestWatchRecursiveInvalidPath(t *testing.T) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer watcher.Close()

	err = WatchRecursive(watcher, "/nonexistent/path/12345")
	if err == nil {
		t.Error("Expected error for invalid path, got nil")
	}
}
