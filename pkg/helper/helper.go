package helper

import (
	"encoding/json"
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

func ValidateJson(filePath string, v interface{}) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	dec := json.NewDecoder(file)
	dec.DisallowUnknownFields()

	return dec.Decode(v)
}
