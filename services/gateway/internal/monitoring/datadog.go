package monitoring

import (
	"os"
)

var (
	isEnabled bool
)

// Init initializes monitoring (stub implementation - Datadog disabled due to network restrictions)
func Init(serviceName, serviceVersion string) error {
	ddAPIKey := os.Getenv("DD_API_KEY")
	if ddAPIKey == "" {
		// Datadog is not configured, skip initialization
		isEnabled = false
		return nil
	}

	// Datadog monitoring is disabled in this build
	// To enable, install DataDog dependencies: github.com/DataDog/dd-trace-go and github.com/DataDog/datadog-go/v5
	isEnabled = false
	return nil
}

// Stop stops the monitoring
func Stop() {
	// No-op in stub implementation
}

// IsEnabled returns whether monitoring is enabled
func IsEnabled() bool {
	return isEnabled
}

// Metrics provides helper functions for sending metrics (stub implementation)
type Metrics struct {
}

// NewMetrics creates a new Metrics instance
func NewMetrics() *Metrics {
	return &Metrics{}
}

// Count increments a counter (no-op)
func (m *Metrics) Count(name string, value int64, tags []string) {
	// No-op in stub implementation
}

// Gauge sets a gauge value (no-op)
func (m *Metrics) Gauge(name string, value float64, tags []string) {
	// No-op in stub implementation
}

// Histogram sends a histogram value (no-op)
func (m *Metrics) Histogram(name string, value float64, tags []string) {
	// No-op in stub implementation
}

// Timing sends a timing metric (no-op)
func (m *Metrics) Timing(name string, value int64, tags []string) {
	// No-op in stub implementation
}

// Incr increments a counter by 1 (no-op)
func (m *Metrics) Incr(name string, tags []string) {
	// No-op in stub implementation
}

// Decr decrements a counter by 1 (no-op)
func (m *Metrics) Decr(name string, tags []string) {
	// No-op in stub implementation
}
