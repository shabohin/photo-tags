package selector

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/shabohin/photo-tags/services/analyzer/internal/api/openrouter"
	"github.com/shabohin/photo-tags/services/analyzer/internal/monitoring"
)

// ModelSelector manages automatic selection and caching of the best free vision model
type ModelSelector struct {
	client         openrouter.OpenRouterClient
	logger         *logrus.Logger
	checkInterval  time.Duration
	currentModel   string
	currentModelMu sync.RWMutex
	fallbackModel  string
	stopChan       chan struct{}
	stoppedChan    chan struct{}
	metrics        *monitoring.Metrics
}

// NewModelSelector creates a new ModelSelector instance
func NewModelSelector(
	client openrouter.OpenRouterClient,
	logger *logrus.Logger,
	checkInterval time.Duration,
	fallbackModel string,
) *ModelSelector {
	return &ModelSelector{
		client:        client,
		logger:        logger,
		checkInterval: checkInterval,
		fallbackModel: fallbackModel,
		stopChan:      make(chan struct{}),
		stoppedChan:   make(chan struct{}),
		metrics:       monitoring.NewMetrics(),
	}
}

// Start begins periodic model checking and selection
func (s *ModelSelector) Start(ctx context.Context) {
	s.logger.WithField("check_interval", s.checkInterval).Info("Starting Model Selector")

	// Perform initial model selection
	s.updateModels(ctx)

	// Start periodic updates
	go s.periodicUpdate(ctx)
}

// periodicUpdate runs the periodic model update loop
func (s *ModelSelector) periodicUpdate(ctx context.Context) {
	ticker := time.NewTicker(s.checkInterval)
	defer ticker.Stop()
	defer close(s.stoppedChan)

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Model Selector stopping due to context cancellation")
			return
		case <-s.stopChan:
			s.logger.Info("Model Selector stopping")
			return
		case <-ticker.C:
			s.updateModels(ctx)
		}
	}
}

// updateModels fetches available models and selects the best one
func (s *ModelSelector) updateModels(ctx context.Context) {
	s.logger.Info("Updating available models")
	s.metrics.Incr("model_selector.update.attempts", []string{})

	models, err := s.client.GetAvailableModels(ctx)
	if err != nil {
		s.metrics.Incr("model_selector.update.errors", []string{"error:fetch_failed"})
		s.logger.WithError(err).Error("Failed to fetch available models")
		// Keep using the current model if update fails
		if s.getCurrentModelUnsafe() == "" && s.fallbackModel != "" {
			s.logger.WithField("model", s.fallbackModel).Warn("Using fallback model due to fetch error")
			s.metrics.Incr("model_selector.fallback_used", []string{"reason:fetch_failed"})
			s.setCurrentModel(s.fallbackModel)
		}
		return
	}

	selected, err := s.client.SelectBestFreeVisionModel(models)
	if err != nil {
		s.metrics.Incr("model_selector.update.errors", []string{"error:selection_failed"})
		s.logger.WithError(err).Error("Failed to select best free vision model")
		// Use fallback model if selection fails
		if s.fallbackModel != "" {
			s.logger.WithField("model", s.fallbackModel).Warn("Using fallback model due to selection error")
			s.metrics.Incr("model_selector.fallback_used", []string{"reason:selection_failed"})
			s.setCurrentModel(s.fallbackModel)
		}
		return
	}

	previousModel := s.getCurrentModelUnsafe()
	s.setCurrentModel(selected.ID)

	// Record successful model selection
	s.metrics.Incr("model_selector.update.success", []string{})
	s.metrics.Gauge("model_selector.context_length", float64(selected.ContextLen), []string{"model:" + selected.ID})

	if previousModel != selected.ID {
		s.metrics.Incr("model_selector.model_changed", []string{})
		s.logger.WithFields(logrus.Fields{
			"previous_model": previousModel,
			"new_model":      selected.ID,
			"model_name":     selected.Name,
			"context_len":    selected.ContextLen,
		}).Info("Selected model changed")
	} else {
		s.logger.WithFields(logrus.Fields{
			"model":       selected.ID,
			"model_name":  selected.Name,
			"context_len": selected.ContextLen,
		}).Info("Model selection confirmed")
	}
}

// GetCurrentModel returns the currently selected model ID
func (s *ModelSelector) GetCurrentModel() (string, error) {
	s.currentModelMu.RLock()
	defer s.currentModelMu.RUnlock()

	if s.currentModel == "" {
		return "", fmt.Errorf("no model selected yet")
	}

	return s.currentModel, nil
}

// getCurrentModelUnsafe returns the current model without locking (for internal use)
func (s *ModelSelector) getCurrentModelUnsafe() string {
	s.currentModelMu.RLock()
	defer s.currentModelMu.RUnlock()
	return s.currentModel
}

// setCurrentModel sets the current model in a thread-safe manner
func (s *ModelSelector) setCurrentModel(modelID string) {
	s.currentModelMu.Lock()
	defer s.currentModelMu.Unlock()
	s.currentModel = modelID
}

// Stop stops the periodic model updates
func (s *ModelSelector) Stop() {
	s.logger.Info("Stopping Model Selector")
	close(s.stopChan)
	<-s.stoppedChan
	s.logger.Info("Model Selector stopped")
}
