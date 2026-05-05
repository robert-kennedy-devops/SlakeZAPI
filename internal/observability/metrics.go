package observability

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	registry            *prometheus.Registry
	httpRequestsTotal   *prometheus.CounterVec
	httpDurationSeconds *prometheus.HistogramVec
	httpInflight        prometheus.Gauge
	readiness           *prometheus.GaugeVec
	queueDepth          *prometheus.GaugeVec
}

func NewMetrics() *Metrics {
	registry := prometheus.NewRegistry()
	m := &Metrics{
		registry: registry,
		httpRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "slakezapi",
				Name:      "http_requests_total",
				Help:      "Total HTTP requests handled by the API.",
			},
			[]string{"method", "path", "status"},
		),
		httpDurationSeconds: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "slakezapi",
				Name:      "http_request_duration_seconds",
				Help:      "HTTP request duration in seconds.",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"method", "path", "status"},
		),
		httpInflight: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: "slakezapi",
				Name:      "http_inflight_requests",
				Help:      "Current number of inflight HTTP requests.",
			},
		),
		readiness: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: "slakezapi",
				Name:      "component_readiness",
				Help:      "Readiness state of major components. 1=ready, 0=not ready.",
			},
			[]string{"component"},
		),
		queueDepth: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: "slakezapi",
				Name:      "queue_depth",
				Help:      "Current queue depth by queue type.",
			},
			[]string{"queue"},
		),
	}

	registry.MustRegister(
		m.httpRequestsTotal,
		m.httpDurationSeconds,
		m.httpInflight,
		m.readiness,
		m.queueDepth,
	)
	return m
}

func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}

func (m *Metrics) Instrument(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.httpInflight.Inc()
		defer m.httpInflight.Dec()

		start := time.Now()
		rw := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)

		status := strconv.Itoa(rw.status)
		path := r.Pattern
		if path == "" {
			path = r.URL.Path
		}
		labels := []string{r.Method, path, status}
		m.httpRequestsTotal.WithLabelValues(labels...).Inc()
		m.httpDurationSeconds.WithLabelValues(labels...).Observe(time.Since(start).Seconds())
	})
}

func (m *Metrics) SetReadiness(component string, ready bool) {
	reading := 0.0
	if ready {
		reading = 1
	}
	m.readiness.WithLabelValues(component).Set(reading)
}

func (m *Metrics) SetQueueDepth(queue string, depth int) {
	m.queueDepth.WithLabelValues(queue).Set(float64(depth))
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}
