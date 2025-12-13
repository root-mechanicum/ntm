package config

import (
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/Dicklesworthstone/ntm/internal/watcher"
)

// Watch starts watching the config file for changes.
// It calls onChange with the new config when a change is detected.
// It returns a close function to stop watching.
func Watch(onChange func(*Config)) (func(), error) {
	path := DefaultPath()
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolving config path: %w", err)
	}
	path = absPath

	// Create watcher with debounce to avoid multiple reloads on single save
	w, err := watcher.New(func(events []watcher.Event) {
		// We only care if the config file changed
		for _, e := range events {
			if filepath.Clean(e.Path) == filepath.Clean(path) {
				// Reload config
				cfg, err := Load(path)
				if err != nil {
					log.Printf("Error reloading config: %v", err)
					return
				}
				// Notify callback
				if onChange != nil {
					onChange(cfg)
				}
				return
			}
		}
	}, watcher.WithDebounceDuration(500*time.Millisecond))

	if err != nil {
		return nil, fmt.Errorf("creating config watcher: %w", err)
	}

	// Add config file to watcher
	// If file doesn't exist yet, watch the directory
	if err := w.Add(path); err != nil {
		dir := filepath.Dir(path)
		if err := w.Add(dir); err != nil {
			w.Close()
			return nil, fmt.Errorf("watching config path %s: %w", path, err)
		}
	}

	return func() {
		w.Close()
	}, nil
}
