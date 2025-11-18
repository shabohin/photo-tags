package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Service struct {
	rabbitMQURL string
}

type ServiceStatus struct {
	Name      string `json:"name"`
	Status    string `json:"status"`
	Healthy   bool   `json:"healthy"`
	URL       string `json:"url,omitempty"`
	CheckedAt string `json:"checked_at"`
}

type QueueInfo struct {
	Name      string `json:"name"`
	Messages  int    `json:"messages"`
	Consumers int    `json:"consumers"`
}

type Metrics struct {
	Services []ServiceStatus `json:"services"`
	Queues   []QueueInfo     `json:"queues"`
	Stats    Stats           `json:"stats"`
}

type Stats struct {
	TotalProcessed int `json:"total_processed"`
	QueuedImages   int `json:"queued_images"`
}

func NewService(rabbitMQURL string) *Service {
	return &Service{
		rabbitMQURL: rabbitMQURL,
	}
}

// CheckServiceHealth проверяет доступность сервиса
func (s *Service) CheckServiceHealth(ctx context.Context, name, url string) ServiceStatus {
	status := ServiceStatus{
		Name:      name,
		CheckedAt: time.Now().Format(time.RFC3339),
		URL:       url,
	}

	if url == "" {
		status.Status = "not_configured"
		status.Healthy = false
		return status
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url+"/health", nil)
	if err != nil {
		status.Status = "error"
		status.Healthy = false
		return status
	}

	resp, err := client.Do(req)
	if err != nil {
		status.Status = "down"
		status.Healthy = false
		return status
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		status.Status = "up"
		status.Healthy = true
	} else {
		status.Status = "unhealthy"
		status.Healthy = false
	}

	return status
}

// GetRabbitMQQueues получает информацию об очередях RabbitMQ
func (s *Service) GetRabbitMQQueues(ctx context.Context) ([]QueueInfo, error) {
	conn, err := amqp.Dial(s.rabbitMQURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}
	defer ch.Close()

	queues := []string{
		"image_uploaded",
		"metadata_generated",
		"image_processed",
	}

	var result []QueueInfo
	for _, queueName := range queues {
		queue, err := ch.QueueInspect(queueName)
		if err != nil {
			// Если очередь не существует, добавляем с нулевыми значениями
			result = append(result, QueueInfo{
				Name:      queueName,
				Messages:  0,
				Consumers: 0,
			})
			continue
		}

		result = append(result, QueueInfo{
			Name:      queue.Name,
			Messages:  queue.Messages,
			Consumers: queue.Consumers,
		})
	}

	return result, nil
}

// GetMetrics собирает все метрики
func (s *Service) GetMetrics(ctx context.Context, gatewayURL, analyzerURL, processorURL string) (*Metrics, error) {
	metrics := &Metrics{
		Services: []ServiceStatus{},
		Queues:   []QueueInfo{},
		Stats: Stats{
			TotalProcessed: 0,
			QueuedImages:   0,
		},
	}

	// Проверяем статус сервисов
	services := []struct {
		name string
		url  string
	}{
		{"gateway", gatewayURL},
		{"analyzer", analyzerURL},
		{"processor", processorURL},
	}

	for _, svc := range services {
		status := s.CheckServiceHealth(ctx, svc.name, svc.url)
		metrics.Services = append(metrics.Services, status)
	}

	// Получаем информацию об очередях
	queues, err := s.GetRabbitMQQueues(ctx)
	if err == nil {
		metrics.Queues = queues

		// Подсчитываем общее количество сообщений в очередях
		for _, queue := range queues {
			metrics.Stats.QueuedImages += queue.Messages
		}
	}

	return metrics, nil
}

// GetRabbitMQStatus проверяет статус RabbitMQ
func (s *Service) GetRabbitMQStatus(ctx context.Context) ServiceStatus {
	status := ServiceStatus{
		Name:      "rabbitmq",
		CheckedAt: time.Now().Format(time.RFC3339),
	}

	conn, err := amqp.Dial(s.rabbitMQURL)
	if err != nil {
		status.Status = "down"
		status.Healthy = false
		return status
	}
	defer conn.Close()

	status.Status = "up"
	status.Healthy = true
	return status
}

// HealthResponse используется для endpoint /health
type HealthResponse struct {
	Status string `json:"status"`
	Time   string `json:"time"`
}

// Marshal преобразует HealthResponse в JSON
func (h *HealthResponse) Marshal() ([]byte, error) {
	return json.Marshal(h)
}
