package watcher

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/shabohin/photo-tags/pkg/logging"
	"github.com/shabohin/photo-tags/services/filewatcher/internal/config"
	"github.com/shabohin/photo-tags/services/filewatcher/internal/processor"
)

// Watcher monitors the input directory for new files
type Watcher struct {
	cfg       *config.Config
	logger    *logging.Logger
	processor *processor.Processor
}

// NewWatcher creates a new Watcher instance
func NewWatcher(
	cfg *config.Config,
	logger *logging.Logger,
	processor *processor.Processor,
) *Watcher {
	return &Watcher{
		cfg:       cfg,
		logger:    logger,
		processor: processor,
	}
}

// Start starts watching the input directory
func (w *Watcher) Start(ctx context.Context) error {
	// Ensure input directory exists
	if err := os.MkdirAll(w.cfg.InputDir, 0755); err != nil {
		return fmt.Errorf("failed to create input directory: %w", err)
	}

	if w.cfg.UseFsnotify {
		return w.watchWithFsnotify(ctx)
	}
	return w.watchWithPolling(ctx)
}

// watchWithFsnotify uses fsnotify to watch for file system events
func (w *Watcher) watchWithFsnotify(ctx context.Context) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create fsnotify watcher: %w", err)
	}
	defer watcher.Close()

	// Add input directory to watcher
	if err := watcher.Add(w.cfg.InputDir); err != nil {
		return fmt.Errorf("failed to add directory to watcher: %w", err)
	}

	w.logger.Info("Started watching directory with fsnotify", map[string]interface{}{
		"directory": w.cfg.InputDir,
	})

	// Track recently processed files to avoid duplicate processing
	recentlyProcessed := make(map[string]time.Time)
	cleanupTicker := time.NewTicker(1 * time.Minute)
	defer cleanupTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil

		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}

			// Only process Create and Write events
			if event.Op&fsnotify.Create == fsnotify.Create || event.Op&fsnotify.Write == fsnotify.Write {
				// Check if file was recently processed
				if lastProcessed, exists := recentlyProcessed[event.Name]; exists {
					if time.Since(lastProcessed) < 5*time.Second {
						continue
					}
				}

				// Wait a bit to ensure file is completely written
				time.Sleep(100 * time.Millisecond)

				// Process the file
				if err := w.processor.ProcessFile(ctx, event.Name); err != nil {
					w.logger.Error("Failed to process file", err)
				} else {
					recentlyProcessed[event.Name] = time.Now()
				}
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			w.logger.Error("Watcher error", err)

		case <-cleanupTicker.C:
			// Clean up old entries from recentlyProcessed
			cutoff := time.Now().Add(-5 * time.Minute)
			for path, t := range recentlyProcessed {
				if t.Before(cutoff) {
					delete(recentlyProcessed, path)
				}
			}
		}
	}
}

// watchWithPolling uses polling to check for new files
func (w *Watcher) watchWithPolling(ctx context.Context) error {
	w.logger.Info("Started watching directory with polling", map[string]interface{}{
		"directory": w.cfg.InputDir,
		"interval":  w.cfg.ScanInterval,
	})

	ticker := time.NewTicker(w.cfg.ScanInterval)
	defer ticker.Stop()

	// Keep track of known files to avoid reprocessing
	knownFiles := make(map[string]bool)

	// Initial scan
	if err := w.scanDirectory(ctx, knownFiles); err != nil {
		w.logger.Error("Initial scan failed", err)
	}

	for {
		select {
		case <-ctx.Done():
			return nil

		case <-ticker.C:
			if err := w.scanDirectory(ctx, knownFiles); err != nil {
				w.logger.Error("Scan failed", err)
			}
		}
	}
}

// scanDirectory scans the input directory for new files
func (w *Watcher) scanDirectory(ctx context.Context, knownFiles map[string]bool) error {
	files, err := os.ReadDir(w.cfg.InputDir)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filePath := filepath.Join(w.cfg.InputDir, file.Name())

		// Skip if already known
		if knownFiles[filePath] {
			continue
		}

		// Process the file
		if err := w.processor.ProcessFile(ctx, filePath); err != nil {
			w.logger.Error("Failed to process file", err)
			continue
		}

		// Mark as known (but it's already moved to processed dir)
		knownFiles[filePath] = true
	}

	// Clean up knownFiles map periodically
	if len(knownFiles) > 1000 {
		// Keep only the last 500 entries
		count := 0
		for k := range knownFiles {
			if count < len(knownFiles)-500 {
				delete(knownFiles, k)
			}
			count++
		}
	}

	return nil
}

// TriggerManualScan triggers a manual scan of the input directory
func (w *Watcher) TriggerManualScan(ctx context.Context) error {
	w.logger.Info("Manual scan triggered", map[string]interface{}{
		"directory": w.cfg.InputDir,
	})

	return w.processor.ProcessBatch(ctx)
}
