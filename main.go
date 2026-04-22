package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/alex-campulungeanu/logarul"
	"github.com/alex-campulungeanu/relouderul/pkg/config"
	"github.com/alex-campulungeanu/relouderul/pkg/helper"
	"github.com/fsnotify/fsnotify"
)

type Runner struct {
	service    config.ServiceInfo
	cmd        *exec.Cmd
	cmdLock    sync.Mutex
	cancelFunc context.CancelFunc
}

func (r *Runner) startProcess(ctx context.Context) error {
	r.cmdLock.Lock()
	defer r.cmdLock.Unlock()

	slog.Info("▶ Service path", "path", r.service.Path)
	slog.Info("▶ Starting", "service", r.service.Name)
	slog.Info("📦 Command:", "command", r.service.Command)
	slog.Info("📦 Watch path:", "watch_path", r.service.WatchPath)

	cmd := exec.CommandContext(ctx, r.service.Command[0], r.service.Command[1:]...)
	cmd.Dir = r.service.Path
	if _, err := exec.LookPath(r.service.Command[0]); err != nil {
		slog.Error("command not found", "command", r.service.Command[0])
		return nil
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		slog.Error("❌ Failed to start cmd.Start:", "err", err)
		return err
	}

	r.cmd = cmd

	go func() {
		err := cmd.Wait()
		if err != nil && !errors.Is(err, context.Canceled) {
			slog.Info("⚠ Process exited with error", "err", err)
		} else {
			slog.Info("ℹ Process exited")
		}
	}()

	return nil
}

func (r *Runner) stopProcess(timeout time.Duration) {
	r.cmdLock.Lock()
	defer r.cmdLock.Unlock()

	if r.cmd == nil || r.cmd.Process == nil {
		slog.Error("Failed to get pgid, killing single process")
		return
	}

	slog.Info("Stopping process group")
	pgid, err := syscall.Getpgid(r.cmd.Process.Pid)
	if err != nil {
		slog.Error("Failed to get pgid, killing single process")
		_ = r.cmd.Process.Kill()
		return
	}

	_ = syscall.Kill(-pgid, syscall.SIGTERM)

	done := make(chan error, 1)
	go func() {
		done <- r.cmd.Wait()
	}()

	select {
	case <-done:
		slog.Info("✅ Process stopped")
	case <-time.After(timeout):
		slog.Info("⚠ Force killing process...")
		_ = syscall.Kill(-pgid, syscall.SIGKILL)
	}
}

func (r *Runner) restart() {
	if r.cancelFunc != nil {
		r.cancelFunc()
	}

	r.stopProcess(3 * time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	r.cancelFunc = cancel

	if err := r.startProcess(ctx); err != nil {
		slog.Error("❌ Failed to start:", "err", err)
		return
	}
}

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

	runner := &Runner{
		service: service,
	}

	// Initial start
	runner.restart()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		slog.Error("Error when watcher", "err", err)
	}
	defer watcher.Close()

	// Watch service + libs recursively
	if err := helper.WatchRecursive(watcher, service.WatchPath); err != nil {
		slog.Error("Error when watching", "err", err)
	}

	if err := helper.WatchRecursive(watcher, filepath.Join(runner.service.Path, "libs")); err != nil {
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
			runner.restart()
		})
	}

	// Handle OS signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	for {
		select {
		case event := <-watcher.Events:
			if strings.HasSuffix(event.Name, ".py") {
				triggerRestart()
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
			runner.stopProcess(3 * time.Second)
			return
		}
	}
}

func main() {
	// logFile, err := rlogger.InitLogger("./data/kubernetes.log", slog.LevelInfo)
	// if err != nil {
	// 	panic(err)
	// }
	// defer logFile.Close()
	cfg := logarul.NewMinimalConfig()
	cfg.Level = slog.LevelInfo
	logger, err := logarul.New(cfg)
	if err != nil {
		panic(err)
	}
	slog.SetDefault(logger)

	serviceName := flag.String("service", "", "Service to run")
	edit := flag.Bool("edit", false, "Edit the config file")
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

	if *serviceName == "" {
		log.Fatal("Usage: --service=<name>")
	}

	run(*serviceName, configService)
}
