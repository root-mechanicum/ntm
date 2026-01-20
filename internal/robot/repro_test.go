package robot

import (
	"testing"
	"time"

	"github.com/Dicklesworthstone/ntm/internal/agentmail"
	"github.com/Dicklesworthstone/ntm/internal/tmux"
	"github.com/stretchr/testify/assert"
)

func TestResolveAgentsForSession_CustomTypes(t *testing.T) {
	// Mock panes
	panes := []tmux.Pane{
		{ID: "%1", Title: "myproj__cursor_1"},
		{ID: "%2", Title: "myproj__windsurf_1"},
	}

	// Mock agents
	agents := []agentmail.Agent{
		{Name: "CursorAgent", Program: "cursor", LastActiveTS: time.Now()},
		{Name: "WindsurfAgent", Program: "windsurf", LastActiveTS: time.Now()},
	}

	mapping := resolveAgentsForSession(panes, agents)

	assert.NotNil(t, mapping)
	assert.Equal(t, "CursorAgent", mapping["cursor_1"])
	assert.Equal(t, "WindsurfAgent", mapping["windsurf_1"])
}
