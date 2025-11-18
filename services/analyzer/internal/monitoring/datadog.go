package monitoring

import (
	"fmt"
	"os"
	"sync"

	"github.com/DataDog/datadog-go/v5/statsd"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

var (
	statsdClient *statsd.Client
	once         sync.Once
	isEnabled    bool
)

// Init initializes Datadog monitoring (tracing and metrics)
func Init(serviceName, serviceVersion string) error {
	ddAPIKey := os.Getenv("DD_API_KEY")
	if ddAPIKey == "" {
		// Datadog is not configured, skip initialization
		isEnabled = false
		return nil
	}

	isEnabled = true

	// Initialize Datadog tracer
	ddAgentHost := os.Getenv("DD_AGENT_HOST")
	if ddAgentHost == "" {
		ddAgentHost = "datadog"
	}

	ddEnv := os.Getenv("DD_ENV")
	if ddEnv == "" {
		ddEnv = "development"
	}

	tracer.Start(
		tracer.WithService(serviceName),
		tracer.WithServiceVersion(serviceVersion),
		tracer.WithEnv(ddEnv),
		tracer.WithAgentAddr(fmt.Sprintf("%s:8126", ddAgentHost)),
	)

	// Initialize DogStatsD client
	var err error
	once.Do(func() {
		statsdClient, err = statsd.New(
			fmt.Sprintf("%s:8125", ddAgentHost),
			statsd.WithNamespace("photo_tags."),
			statsd.WithTags([]string{
				fmt.Sprintf("service:%s", serviceName),
				fmt.Sprintf("version:%s", serviceVersion),
				fmt.Sprintf("env:%s", ddEnv),
			}),
		)
	})

	if err != nil {
		return fmt.Errorf("failed to initialize DogStatsD client: %w", err)
	}

	return nil
}

// Stop stops the Datadog tracer
func Stop() {
	if !isEnabled {
		return
	}

	tracer.Stop()

	if statsdClient != nil {
		_ = statsdClient.Close()
	}
}

// IsEnabled returns whether Datadog monitoring is enabled
func IsEnabled() bool {
	return isEnabled
}

// GetStatsD returns the DogStatsD client
func GetStatsD() *statsd.Client {
	return statsdClient
}

// Metrics provides helper functions for sending metrics
type Metrics struct {
	client *statsd.Client
}

// NewMetrics creates a new Metrics instance
func NewMetrics() *Metrics {
	return &Metrics{
		client: statsdClient,
	}
}

// Count increments a counter
func (m *Metrics) Count(name string, value int64, tags []string) {
	if m.client != nil && isEnabled {
		_ = m.client.Count(name, value, tags, 1.0)
	}
}

// Gauge sets a gauge value
func (m *Metrics) Gauge(name string, value float64, tags []string) {
	if m.client != nil && isEnabled {
		_ = m.client.Gauge(name, value, tags, 1.0)
	}
}

// Histogram sends a histogram value
func (m *Metrics) Histogram(name string, value float64, tags []string) {
	if m.client != nil && isEnabled {
		_ = m.client.Histogram(name, value, tags, 1.0)
	}
}

// Timing sends a timing metric
func (m *Metrics) Timing(name string, value int64, tags []string) {
	if m.client != nil && isEnabled {
		_ = m.client.TimeInMilliseconds(name, float64(value), tags, 1.0)
	}
}

// Incr increments a counter by 1
func (m *Metrics) Incr(name string, tags []string) {
	m.Count(name, 1, tags)
}

// Decr decrements a counter by 1
func (m *Metrics) Decr(name string, tags []string) {
	m.Count(name, -1, tags)
}
