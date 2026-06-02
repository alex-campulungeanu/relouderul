package main

import (
	"flag"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/alex-campulungeanu/logarul"
	"github.com/alex-campulungeanu/relouderul/pkg/config"
	"github.com/alex-campulungeanu/relouderul/pkg/helper"
	"github.com/alex-campulungeanu/relouderul/pkg/runner"
	"github.com/fsnotify/fsnotify"
)

func run(serviceKey string, configService config.Service) {

	config.Init(configService)
	configData, err := configService.Read()
	if err != nil {
		slog.Error("error read config file %v", "err", err)
		return
	}

	service, ok := configData[serviceKey]
	if !ok {
		slog.Error("service not found in config", "service", serviceKey)
		return
	}

	runner := &runner.Runner{
		Service: service,
	}

	// Initial start
	runner.Restart()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		slog.Error("Error when watcher", "err", err)
	}
	defer watcher.Close()

	// Watch service + libs recursively
	if err := helper.WatchRecursive(watcher, service.WatchPath); err != nil {
		slog.Error("Error when watching", "err", err)
	}

	if err := helper.WatchRecursive(watcher, filepath.Join(runner.Service.Path, "libs")); err != nil {
		slog.Error("error with recursive", "err", err)
	}

	slog.Info("👀 Watching for changes...")

	// Debounce timer
	var debounceMu sync.Mutex
	var debounceTimer *time.Timer

	triggerRestart := func() {
		debounceMu.Lock()
		defer debounceMu.Unlock()

		if debounceTimer != nil {
			debounceTimer.Stop()
		}

		debounceTimer = time.AfterFunc(500*time.Millisecond, func() {
			log.Println("🔁 Restarting due to changes...")
			runner.Restart()
		})
	}

	// Handle OS signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	for {
		select {
		case event := <-watcher.Events:
			for _, s := range runner.Service.Extensions {
				if strings.HasSuffix(event.Name, s) {
					triggerRestart()
				}
			}

			// Handle new directories (important!)
			if event.Op&fsnotify.Create != 0 {
				info, err := os.Stat(event.Name)
				if err == nil && info.IsDir() {
					_ = helper.WatchRecursive(watcher, event.Name)
				}
			}

		case err := <-watcher.Errors:
			log.Println("Watcher error:", err)

		case sig := <-sigChan:
			log.Printf("✋ Received signal: %v", sig)
			runner.StopProcess(3 * time.Second)
			return
		}
	}
}

func main() {
	cfg := logarul.NewMinimalConfig()
	cfg.Level = slog.LevelInfo
	logger, err := logarul.New(cfg)
	if err != nil {
		panic(err)
	}
	slog.SetDefault(logger)

	serviceName := flag.String("service", "", "Service to run")
	edit := flag.Bool("edit", false, "Edit the config file")
	list := flag.Bool("list", false, "List services in the config file")
	flag.Parse()

	store := config.FileStore{PathProvider: config.NewOSPathProvider()}
	editor := config.OSEditor{PathProvider: config.NewOSPathProvider(), Runner: config.OSRunner{}}
	configService := config.Service{Store: store, Editor: editor}

	//Edit the config file and exit
	if *edit {
		err := configService.Edit()
		if err != nil {
			slog.Error("error edit config file", "err", err)
		}
		return
	}

	if *list {
		keys, err := configService.ListServices()
		if err != nil {
			slog.Error("error listing services", "err", err)
			return
		}
		slog.Info("Available services:", "services", keys)
		return
	}

	if *serviceName == "" {
		log.Fatal("Usage: --service=<name>")
	}

	run(*serviceName, configService)
}
