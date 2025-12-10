package resilience

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Dicklesworthstone/ntm/internal/config"
)

func TestRestartAgentUsesBuiltPaneCommandAndSendKeys(t *testing.T) {
	// Save originals to restore later
	origSend := sendKeysFn
	origBuild := buildPaneCmdFn
	origSleep := sleepFn
	// Stub functions
	var mu sync.Mutex
	var capturedCmd string
	sendKeysFn = func(paneID, cmd string, enter bool) error {
		mu.Lock()
		defer mu.Unlock()
		capturedCmd = cmd
		if paneID != "pane-1" {
			t.Fatalf("unexpected pane id: %s", paneID)
		}
		if !enter {
			t.Fatalf("expected enter=true")
		}
		return nil
	}
	buildPaneCmdFn = func(projectDir, agentCmd string) (string, error) {
		if projectDir != "/tmp/project with space" {
			return "", fmt.Errorf("unexpected dir: %s", projectDir)
		}
		return fmt.Sprintf("cd %q && %s", projectDir, agentCmd), nil
	}
	sleepFn = func(d time.Duration) {} // no-op for speed

	// Restore hooks after test
	defer func() {
		sendKeysFn = origSend
		buildPaneCmdFn = origBuild
		sleepFn = origSleep
	}()

	cfg := config.Default()
	cfg.Resilience.AutoRestart = true
	cfg.Resilience.RestartDelaySeconds = 0

	m := NewMonitor("sess", "/tmp/project with space", cfg)
	m.agents["pane-1"] = &AgentState{
		PaneID:    "pane-1",
		PaneIndex: 1,
		AgentType: "cc",
		Command:   "claude --model 'safe-model'",
		Healthy:   false,
	}

	m.restartAgent(m.agents["pane-1"])

	mu.Lock()
	defer mu.Unlock()
	if capturedCmd == "" {
		t.Fatalf("sendKeys was not invoked")
	}
	if capturedCmd != "cd \"/tmp/project with space\" && claude --model 'safe-model'" {
		t.Fatalf("unexpected command sent: %s", capturedCmd)
	}
}
