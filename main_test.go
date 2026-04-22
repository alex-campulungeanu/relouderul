package main

import (
	"context"
	"os/exec"
	"testing"
	"time"

	"github.com/alex-campulungeanu/relouderul/pkg/config"
)

func testRunner() *Runner {
	return &Runner{
		service: config.ServiceInfo{
			Name:      "test-service",
			Path:      "/tmp",
			Command:   []string{"echo", "test"},
			WatchPath: "/tmp",
		},
	}
}

func TestRunnerStartProcessS(t *testing.T) {
	runner := testRunner()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := runner.startProcess(ctx)
	if err != nil {
		t.Fatalf("startProcess failed: %v", err)
	}
	if runner.cmd == nil {
		t.Error("cmd should not be nil after startProcess")
	}

	if runner.cmd.Process == nil {
		t.Error("Process should not be nil after startProcess")
	}
	runner.stopProcess(3 * time.Second)
}

func TestRunnerStopProcess(t *testing.T) {
	runner := testRunner()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := runner.startProcess(ctx)
	if err != nil {
		t.Fatalf("startProcess failed: %v", err)
	}

	// runner.stopProcess(3 * time.Second)
	if runner.cmd != nil && runner.cmd.Process != nil {
		ps := exec.Command("ps", "-p", string(rune(runner.cmd.Process.Pid)))
		_, err := ps.Output()
		if err == nil {
			t.Error("Process should have been killed")
		}
	}
}

func TestRunnerRestart(t *testing.T) {
	runner := testRunner()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := runner.startProcess(ctx)
	if err != nil {
		t.Fatalf("startProcess failed: %v", err)
	}

	firstPID := runner.cmd.Process.Pid
	runner.restart()

	if runner.cmd == nil || runner.cmd.Process == nil {
		t.Error("cmd should not be nil after restart")
	}

	if runner.cmd.Process.Pid == firstPID {
		t.Error("Process PID should be different after restart")
	}

	runner.stopProcess(3 * time.Second)
}

func TestRunnerStartProcessCommandNotFound(t *testing.T) {
	runner := &Runner{
		service: config.ServiceInfo{
			Name:      "test",
			Path:      "/tmp",
			Command:   []string{"command-that-definitely-does-not-exist-12345"},
			WatchPath: "/tmp",
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := runner.startProcess(ctx)
	if err != nil {
		t.Fatalf("startProcess should return nil for not found command: %v", err)
	}
	if runner.cmd != nil {
		t.Error("cmd should be nil when command not found")
	}
}
