package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all the Prometheus metrics for the ingest service
type Metrics struct {
	EventsTotal         prometheus.Counter
	EventsInvalidTotal  prometheus.Counter
	NatsPublishErrors   prometheus.Counter
	EventsDedupedTotal  prometheus.Counter
}

// NewMetrics creates a new Metrics instance with all counters
func NewMetrics() *Metrics {
	return &Metrics{
		EventsTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "events_total",
			Help: "Total number of events processed",
		}),
		EventsInvalidTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "events_invalid_total",
			Help: "Total number of invalid events rejected",
		}),
		NatsPublishErrors: promauto.NewCounter(prometheus.CounterOpts{
			Name: "nats_publish_errors_total",
			Help: "Total number of NATS publish errors",
		}),
		EventsDedupedTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "events_deduped_total",
			Help: "Total number of duplicate event_id payloads ignored after persistence check",
		}),
	}
}

// IncrementEventsTotal increments the events_total counter
func (m *Metrics) IncrementEventsTotal() {
	m.EventsTotal.Inc()
}

// IncrementEventsInvalid increments the events_invalid_total counter
func (m *Metrics) IncrementEventsInvalid() {
	m.EventsInvalidTotal.Inc()
}

// IncrementNatsPublishErrors increments the nats_publish_errors_total counter
func (m *Metrics) IncrementNatsPublishErrors() {
	m.NatsPublishErrors.Inc()
}

// IncrementEventsDeduped increments events_deduped_total (WO-OPS-004).
func (m *Metrics) IncrementEventsDeduped() {
	m.EventsDedupedTotal.Inc()
}

