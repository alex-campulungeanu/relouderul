package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
)

type ServiceInfo struct {
	Name      string   `json:"name"`
	Command   []string `json:"command"`
	WatchPath string   `json:"watch_path"`
}

type Runner struct {
	service     ServiceInfo
	projectPath string

	cmd        *exec.Cmd
	cmdLock    sync.Mutex
	cancelFunc context.CancelFunc
}

func getProjectPath() string {
	path := os.Getenv("DIAGNOSTIC_PATH")
	if path == "" {
		log.Fatal("DIAGNOSTIC_PATH is not set")
	}
	return path
}

func loadServices() map[string]ServiceInfo {
	data, err := os.ReadFile("services.json")
	if err != nil {
		log.Fatal("File services.json not found")
	}

	raw := map[string]ServiceInfo{}
	if err := json.Unmarshal(data, &raw); err != nil {
		log.Fatal(err)
	}

	projectPath := getProjectPath()
	result := make(map[string]ServiceInfo)

	for k, v := range raw {
		v.WatchPath = filepath.Join(projectPath, v.WatchPath)
		result[k] = v
	}

	return result
}

func (r *Runner) startProcess(ctx context.Context) error {
	r.cmdLock.Lock()
	defer r.cmdLock.Unlock()

	log.Printf("▶ Starting %s", r.service.Name)
	log.Printf("📦 Command: %v", r.service.Command)

	cmd := exec.CommandContext(ctx, r.service.Command[0], r.service.Command[1:]...)
	cmd.Dir = r.projectPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		return err
	}

	r.cmd = cmd

	go func() {
		err := cmd.Wait()
		if err != nil && !errors.Is(err, context.Canceled) {
			log.Printf("⚠ Process exited with error: %v", err)
		} else {
			log.Printf("ℹ Process exited")
		}
	}()

	return nil
}

func (r *Runner) stopProcess(timeout time.Duration) {
	r.cmdLock.Lock()
	defer r.cmdLock.Unlock()

	if r.cmd == nil || r.cmd.Process == nil {
		log.Println("Failed to get pgid, killing single process")
		return
	}

	log.Println("Stopping process group")
	pgid, err := syscall.Getpgid(r.cmd.Process.Pid)
	if err != nil {
		log.Println("Failed to get pgid, killing single process")
		_ = r.cmd.Process.Kill()
		return
	}

	_ = syscall.Kill(-pgid, syscall.SIGTERM)

	// log.Println("⏹ Stopping process...")
	done := make(chan error, 1)
	go func() {
		done <- r.cmd.Wait()
	}()

	select {
	case <-done:
		log.Println("✅ Process stopped")
	case <-time.After(timeout):
		log.Println("⚠ Force killing process...")
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
		log.Printf("❌ Failed to start: %v", err)
	}
}

func watchRecursive(watcher *fsnotify.Watcher, root string) error {
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

func run(serviceKey string) {
	services := loadServices()

	srv, ok := services[serviceKey]
	if !ok {
		log.Fatalf("Service %s not found", serviceKey)
	}

	projectPath := getProjectPath()

	runner := &Runner{
		service:     srv,
		projectPath: projectPath,
	}

	// Initial start
	runner.restart()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	// Watch service + libs recursively
	if err := watchRecursive(watcher, srv.WatchPath); err != nil {
		log.Fatal(err)
	}

	if err := watchRecursive(watcher, filepath.Join(projectPath, "libs")); err != nil {
		log.Fatal(err)
	}

	log.Println("👀 Watching for changes...")

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
					_ = watchRecursive(watcher, event.Name)
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
	service := flag.String("service", "", "Service to run")
	flag.Parse()

	if *service == "" {
		log.Fatal("Usage: --service=<name>")
	}

	run(*service)
}
