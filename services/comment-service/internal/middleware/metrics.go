package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics 聚合 comment-service 的 Prometheus 指标。
type Metrics struct {
	httpRequestsTotal   *prometheus.CounterVec
	httpDurationSeconds *prometheus.HistogramVec

	kafkaProducedTotal prometheus.Counter
	kafkaConsumedTotal *prometheus.CounterVec
	kafkaErrorTotal    prometheus.Counter

	rateLimitRejectedTotal *prometheus.CounterVec
	cacheHitsTotal         *prometheus.CounterVec
	cacheMissesTotal       *prometheus.CounterVec

	gatherer prometheus.Gatherer
}

// NewMetrics 创建并注册指标。
func NewMetrics(registry *prometheus.Registry) (*Metrics, error) {
	metrics := &Metrics{
		httpRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "comment_service_http_requests_total",
				Help: "Total number of HTTP requests.",
			},
			[]string{"method", "path", "status"},
		),
		httpDurationSeconds: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "comment_service_http_request_duration_seconds",
				Help:    "Duration of HTTP requests in seconds.",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "path"},
		),
		kafkaProducedTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "comment_service_kafka_messages_produced_total",
				Help: "Total Kafka messages produced.",
			},
		),
		kafkaConsumedTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "comment_service_kafka_messages_consumed_total",
				Help: "Total Kafka messages consumed.",
			},
			[]string{"event_type"},
		),
		kafkaErrorTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "comment_service_kafka_consume_errors_total",
				Help: "Total Kafka consume errors.",
			},
		),
		rateLimitRejectedTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "comment_service_ratelimit_rejected_total",
				Help: "Total rate limit rejections.",
			},
			[]string{"dimension"},
		),
		cacheHitsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "comment_service_cache_hits_total",
				Help: "Total cache hits.",
			},
			[]string{"cache_type"},
		),
		cacheMissesTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "comment_service_cache_misses_total",
				Help: "Total cache misses.",
			},
			[]string{"cache_type"},
		),
		gatherer: registry,
	}

	collectors := []prometheus.Collector{
		metrics.httpRequestsTotal,
		metrics.httpDurationSeconds,
		metrics.kafkaProducedTotal,
		metrics.kafkaConsumedTotal,
		metrics.kafkaErrorTotal,
		metrics.rateLimitRejectedTotal,
		metrics.cacheHitsTotal,
		metrics.cacheMissesTotal,
	}
	for _, collector := range collectors {
		if err := registry.Register(collector); err != nil {
			return nil, err
		}
	}
	return metrics, nil
}

// HTTPMiddleware 记录接口请求总量与耗时。
func (m *Metrics) HTTPMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		status := strconv.Itoa(c.Writer.Status())
		m.httpRequestsTotal.WithLabelValues(c.Request.Method, path, status).Inc()
		m.httpDurationSeconds.WithLabelValues(c.Request.Method, path).Observe(time.Since(start).Seconds())
	}
}

// MetricsHandler 返回 /metrics 处理器。
func (m *Metrics) MetricsHandler() gin.HandlerFunc {
	handler := promhttp.HandlerFor(m.gatherer, promhttp.HandlerOpts{})
	return func(c *gin.Context) {
		handler.ServeHTTP(c.Writer, c.Request)
	}
}

// IncKafkaProduced 增加生产计数。
func (m *Metrics) IncKafkaProduced() {
	m.kafkaProducedTotal.Inc()
}

// IncKafkaConsumed 增加消费计数。
func (m *Metrics) IncKafkaConsumed(eventType string) {
	m.kafkaConsumedTotal.WithLabelValues(eventType).Inc()
}

// IncKafkaConsumeError 增加 Kafka 消费错误计数。
func (m *Metrics) IncKafkaConsumeError() {
	m.kafkaErrorTotal.Inc()
}

// IncRateLimitRejected 增加限流拒绝计数。
func (m *Metrics) IncRateLimitRejected(dimension string) {
	m.rateLimitRejectedTotal.WithLabelValues(dimension).Inc()
}

// IncCacheHit 增加缓存命中计数。
func (m *Metrics) IncCacheHit(cacheType string) {
	m.cacheHitsTotal.WithLabelValues(cacheType).Inc()
}

// IncCacheMiss 增加缓存未命中计数。
func (m *Metrics) IncCacheMiss(cacheType string) {
	m.cacheMissesTotal.WithLabelValues(cacheType).Inc()
}
