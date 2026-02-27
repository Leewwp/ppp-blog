package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	httpRequestsTotal  *prometheus.CounterVec
	filterHitsTotal    prometheus.Counter
	requestDurationSec *prometheus.HistogramVec
	gatherer           prometheus.Gatherer
}

func NewMetrics(registry *prometheus.Registry) (*Metrics, error) {
	m := &Metrics{
		httpRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total HTTP requests.",
			},
			[]string{"method", "path", "status"},
		),
		filterHitsTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "filter_hits_total",
				Help: "Total number of filter requests with at least one hit.",
			},
		),
		requestDurationSec: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "request_duration_seconds",
				Help:    "HTTP request duration in seconds.",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "path"},
		),
		gatherer: registry,
	}

	if err := registry.Register(m.httpRequestsTotal); err != nil {
		return nil, err
	}
	if err := registry.Register(m.filterHitsTotal); err != nil {
		return nil, err
	}
	if err := registry.Register(m.requestDurationSec); err != nil {
		return nil, err
	}

	return m, nil
}

func (m *Metrics) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		method := c.Request.Method
		status := strconv.Itoa(c.Writer.Status())
		duration := time.Since(start).Seconds()

		m.httpRequestsTotal.WithLabelValues(method, path, status).Inc()
		m.requestDurationSec.WithLabelValues(method, path).Observe(duration)
	}
}

func (m *Metrics) MetricsHandler() gin.HandlerFunc {
	handler := promhttp.HandlerFor(m.gatherer, promhttp.HandlerOpts{})
	return func(c *gin.Context) {
		handler.ServeHTTP(c.Writer, c.Request)
	}
}

func (m *Metrics) IncFilterHit() {
	m.filterHitsTotal.Inc()
}
