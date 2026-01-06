package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/Dicklesworthstone/ntm/internal/agentmail"
	"github.com/Dicklesworthstone/ntm/internal/bd"
	"github.com/Dicklesworthstone/ntm/internal/output"
	"github.com/Dicklesworthstone/ntm/internal/tmux"
)

func newMessageCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "message",
		Short: "Unified messaging (Agent Mail + BD)",
	}

	cmd.AddCommand(
		newMessageInboxCmd(),
		newMessageSendCmd(),
	)

	return cmd
}

func newMessageInboxCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "inbox",
		Short: "View unified inbox",
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, _ := os.Getwd()
			session := tmux.GetCurrentSession()
			if session == "" {
				session = filepath.Base(dir)
			}
			agentName := fmt.Sprintf("ntm_%s", session)

			amClient := agentmail.NewClient(agentmail.WithProjectKey(dir))
			bdClient := bd.NewMessageClient(dir, agentName)

			unified := agentmail.NewUnifiedMessenger(amClient, bdClient, dir, agentName)

			msgs, err := unified.Inbox(context.Background())
			if err != nil {
				return err
			}

			if IsJSONOutput() {
				return output.PrintJSON(msgs)
			}

			t := output.NewTable(cmd.OutOrStdout(), "ID", "Channel", "From", "Subject", "Time")
			for _, m := range msgs {
				t.AddRow(m.ID, m.Channel, m.From, m.Subject, m.Timestamp.Format(time.Kitchen))
			}
			t.Render()
			return nil
		},
	}
}

func newMessageSendCmd() *cobra.Command {
	var subject string
	cmd := &cobra.Command{
		Use:   "send <to> <body>",
		Short: "Send message",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			to := args[0]
			body := args[1]

			dir, _ := os.Getwd()
			session := tmux.GetCurrentSession()
			if session == "" {
				session = filepath.Base(dir)
			}
			agentName := fmt.Sprintf("ntm_%s", session)

			amClient := agentmail.NewClient(agentmail.WithProjectKey(dir))
			bdClient := bd.NewMessageClient(dir, agentName)

			unified := agentmail.NewUnifiedMessenger(amClient, bdClient, dir, agentName)

			return unified.Send(context.Background(), to, subject, body)
		},
	}
	cmd.Flags().StringVar(&subject, "subject", "(No Subject)", "Message subject")
	return cmd
}
