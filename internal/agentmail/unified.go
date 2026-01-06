package agentmail

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/Dicklesworthstone/ntm/internal/bd"
)

type UnifiedMessage struct {
	ID        string    `json:"id"`
	Channel   string    `json:"channel"` // "agentmail" or "bd"
	From      string    `json:"from"`
	Subject   string    `json:"subject"`
	Body      string    `json:"body"`
	Timestamp time.Time `json:"timestamp"`
}

type UnifiedMessenger struct {
	amClient   *Client
	bdClient   *bd.MessageClient
	projectKey string
	agentName  string
}

func NewUnifiedMessenger(am *Client, bd *bd.MessageClient, projectKey, agentName string) *UnifiedMessenger {
	return &UnifiedMessenger{
		amClient:   am,
		bdClient:   bd,
		projectKey: projectKey,
		agentName:  agentName,
	}
}

// Inbox fetches messages from both channels and merges them sorted by timestamp descending
func (m *UnifiedMessenger) Inbox(ctx context.Context) ([]UnifiedMessage, error) {
	var unified []UnifiedMessage

	// Fetch from Agent Mail
	if m.amClient != nil && m.amClient.IsAvailable() {
		opts := FetchInboxOptions{
			ProjectKey:    m.projectKey,
			AgentName:     m.agentName,
			Limit:         50,
			IncludeBodies: true,
		}
		inbox, err := m.amClient.FetchInbox(ctx, opts)
		if err == nil {
			for _, msg := range inbox {
				unified = append(unified, UnifiedMessage{
					ID:        fmt.Sprintf("am-%d", msg.ID),
					Channel:   "agentmail",
					From:      msg.From,
					Subject:   msg.Subject,
					Body:      msg.BodyMD,
					Timestamp: msg.CreatedTS,
				})
			}
		}
	}

	// Fetch from BD
	if m.bdClient != nil {
		bdInbox, err := m.bdClient.Inbox(ctx, false, false)
		if err == nil {
			for _, msg := range bdInbox {
				unified = append(unified, UnifiedMessage{
					ID:        fmt.Sprintf("bd-%s", msg.ID),
					Channel:   "bd",
					From:      msg.From,
					Subject:   "(No Subject)",
					Body:      msg.Body,
					Timestamp: msg.Timestamp,
				})
			}
		}
	}

	// Sort by timestamp desc
	sort.Slice(unified, func(i, j int) bool {
		return unified[i].Timestamp.After(unified[j].Timestamp)
	})

	return unified, nil
}

// Send sends a message via the preferred channel (defaulting to Agent Mail if available, else BD)
// For now, it tries Agent Mail first.
func (m *UnifiedMessenger) Send(ctx context.Context, to, subject, body string) error {
	// Try Agent Mail first
	if m.amClient != nil && m.amClient.IsAvailable() {
		_, err := m.amClient.SendMessage(ctx, SendMessageOptions{
			ProjectKey: m.projectKey,
			SenderName: m.agentName,
			To:         []string{to},
			Subject:    subject,
			BodyMD:     body,
		})
		if err == nil {
			return nil
		}
		// If failed, try BD? Or maybe user specifies channel preference?
		// Fallthrough only on error might be confusing.
		// For now, just return error if AM configured but failed.
		// If AM not configured/available, try BD.
	}

	if m.bdClient != nil {
		return m.bdClient.Send(ctx, to, body)
	}

	return fmt.Errorf("no message channels available")
}
