package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/shabohin/photo-tags/services/dashboard/internal/config"
	"github.com/shabohin/photo-tags/services/dashboard/internal/metrics"
)

type Handler struct {
	config         *config.Config
	metricsService *metrics.Service
}

func NewHandler(cfg *config.Config, metricsService *metrics.Service) *Handler {
	return &Handler{
		config:         cfg,
		metricsService: metricsService,
	}
}

func (h *Handler) SetupRoutes(router *mux.Router) {
	// API endpoints
	router.HandleFunc("/api/health", h.handleHealth).Methods("GET")
	router.HandleFunc("/api/metrics", h.handleMetrics).Methods("GET")
	router.HandleFunc("/api/services/status", h.handleServicesStatus).Methods("GET")
	router.HandleFunc("/api/rabbitmq/queues", h.handleRabbitMQQueues).Methods("GET")
	router.HandleFunc("/api/config", h.handleConfig).Methods("GET")

	// Static files
	router.PathPrefix("/").Handler(http.FileServer(http.Dir("./static")))
}

// handleHealth возвращает статус здоровья dashboard
func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	response := metrics.HealthResponse{
		Status: "ok",
		Time:   time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleMetrics возвращает все метрики
func (h *Handler) handleMetrics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	metricsData, err := h.metricsService.GetMetrics(
		ctx,
		h.config.GatewayURL,
		h.config.AnalyzerURL,
		h.config.ProcessorURL,
	)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Добавляем статус RabbitMQ
	rabbitMQStatus := h.metricsService.GetRabbitMQStatus(ctx)
	metricsData.Services = append(metricsData.Services, rabbitMQStatus)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metricsData)
}

// handleServicesStatus возвращает только статусы сервисов
func (h *Handler) handleServicesStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	services := []metrics.ServiceStatus{
		h.metricsService.CheckServiceHealth(ctx, "gateway", h.config.GatewayURL),
		h.metricsService.CheckServiceHealth(ctx, "analyzer", h.config.AnalyzerURL),
		h.metricsService.CheckServiceHealth(ctx, "processor", h.config.ProcessorURL),
		h.metricsService.GetRabbitMQStatus(ctx),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(services)
}

// handleRabbitMQQueues возвращает информацию об очередях RabbitMQ
func (h *Handler) handleRabbitMQQueues(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	queues, err := h.metricsService.GetRabbitMQQueues(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(queues)
}

// handleConfig возвращает конфигурацию для frontend
func (h *Handler) handleConfig(w http.ResponseWriter, r *http.Request) {
	config := map[string]string{
		"rabbitmq_url": h.config.RabbitMQMgmt,
		"minio_url":    h.config.MinIOURL,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(config)
}
