package runner

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/alex-campulungeanu/relouderul/pkg/config"
)

type Runner struct {
	Service    config.ServiceInfo
	Cmd        *exec.Cmd
	CmdLock    sync.Mutex
	CancelFunc context.CancelFunc
}

func (r *Runner) StartProcess(ctx context.Context) error {
	r.CmdLock.Lock()
	defer r.CmdLock.Unlock()

	slog.Info("▶ Service path", "path", r.Service.Path)
	slog.Info("▶ Starting", "service", r.Service.Name)
	slog.Info("📦 Command:", "command", r.Service.Command)
	slog.Info("📦 Watch path:", "watch_path", r.Service.WatchPath)

	cmd := exec.CommandContext(ctx, r.Service.Command[0], r.Service.Command[1:]...)
	cmd.Dir = r.Service.Path
	if _, err := exec.LookPath(r.Service.Command[0]); err != nil {
		slog.Error("command not found", "command", r.Service.Command[0])
		return nil
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		slog.Error("❌ Failed to start cmd.Start:", "err", err)
		return err
	}

	r.Cmd = cmd

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

func (r *Runner) StopProcess(timeout time.Duration) {
	r.CmdLock.Lock()
	defer r.CmdLock.Unlock()

	if r.Cmd == nil || r.Cmd.Process == nil {
		slog.Error("Failed to get pgid, killing single process")
		return
	}

	slog.Info("Stopping process group")
	pgid, err := syscall.Getpgid(r.Cmd.Process.Pid)
	if err != nil {
		slog.Error("Failed to get pgid, killing single process")
		_ = r.Cmd.Process.Kill()
		return
	}

	_ = syscall.Kill(-pgid, syscall.SIGTERM)

	done := make(chan error, 1)
	go func() {
		done <- r.Cmd.Wait()
	}()

	select {
	case <-done:
		slog.Info("✅ Process stopped")
	case <-time.After(timeout):
		slog.Info("⚠ Force killing process...")
		_ = syscall.Kill(-pgid, syscall.SIGKILL)
	}
}

func (r *Runner) Restart() {
	if r.CancelFunc != nil {
		r.CancelFunc()
	}

	r.StopProcess(3 * time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	r.CancelFunc = cancel

	if err := r.StartProcess(ctx); err != nil {
		slog.Error("❌ Failed to start:", "err", err)
		return
	}
}
