package main

import (
	"context"
	"os/exec"
	"testing"
	"time"

	"github.com/alex-campulungeanu/relouderul/pkg/config"
	"github.com/alex-campulungeanu/relouderul/pkg/runner"
)

func testRunner() *runner.Runner {
	return &runner.Runner{
		Service: config.ServiceInfo{
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

	err := runner.StartProcess(ctx)
	if err != nil {
		t.Fatalf("startProcess failed: %v", err)
	}
	if runner.Cmd == nil {
		t.Error("cmd should not be nil after startProcess")
	}

	if runner.Cmd.Process == nil {
		t.Error("Process should not be nil after startProcess")
	}
	runner.StopProcess(3 * time.Second)
}

func TestRunnerStopProcess(t *testing.T) {
	runner := testRunner()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := runner.StartProcess(ctx)
	if err != nil {
		t.Fatalf("startProcess failed: %v", err)
	}

	// runner.stopProcess(3 * time.Second)
	if runner.Cmd != nil && runner.Cmd.Process != nil {
		ps := exec.Command("ps", "-p", string(rune(runner.Cmd.Process.Pid)))
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

	err := runner.StartProcess(ctx)
	if err != nil {
		t.Fatalf("startProcess failed: %v", err)
	}

	firstPID := runner.Cmd.Process.Pid
	runner.Restart()

	if runner.Cmd == nil || runner.Cmd.Process == nil {
		t.Error("cmd should not be nil after restart")
	}

	if runner.Cmd.Process.Pid == firstPID {
		t.Error("Process PID should be different after restart")
	}

	runner.StopProcess(3 * time.Second)
}

func TestRunnerStartProcessCommandNotFound(t *testing.T) {
	runner := &runner.Runner{
		Service: config.ServiceInfo{
			Name:      "test",
			Path:      "/tmp",
			Command:   []string{"command-that-definitely-does-not-exist-12345"},
			WatchPath: "/tmp",
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := runner.StartProcess(ctx)
	if err != nil {
		t.Fatalf("startProcess should return nil for not found command: %v", err)
	}
	if runner.Cmd != nil {
		t.Error("cmd should be nil when command not found")
	}
}
