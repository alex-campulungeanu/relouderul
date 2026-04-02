package helper

import (
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
)

func HomeDir() (string, error) {
	return os.UserHomeDir()
}

func WatchRecursive(watcher *fsnotify.Watcher, root string) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return watcher.Add(path)
		}
		return nil
	})
}
