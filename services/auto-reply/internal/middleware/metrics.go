package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	httpRequestsTotal   *prometheus.CounterVec
	replyDecisionsTotal *prometheus.CounterVec
	matchedRuleTotal    *prometheus.CounterVec
	requestDurationSec  *prometheus.HistogramVec
	gatherer            prometheus.Gatherer
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
		replyDecisionsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "reply_decisions_total",
				Help: "Total reply decisions by should_reply value.",
			},
			[]string{"should_reply"},
		),
		matchedRuleTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "matched_rule_total",
				Help: "Total matched rules for reply decisions.",
			},
			[]string{"rule_id", "rule_name"},
		),
		requestDurationSec: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "request_duration_seconds",
				Help:    "HTTP request latency in seconds.",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "path"},
		),
		gatherer: registry,
	}

	collectors := []prometheus.Collector{
		m.httpRequestsTotal,
		m.replyDecisionsTotal,
		m.matchedRuleTotal,
		m.requestDurationSec,
	}
	for _, collector := range collectors {
		if err := registry.Register(collector); err != nil {
			return nil, err
		}
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

		m.httpRequestsTotal.WithLabelValues(method, path, status).Inc()
		m.requestDurationSec.WithLabelValues(method, path).Observe(time.Since(start).Seconds())
	}
}

func (m *Metrics) MetricsHandler() gin.HandlerFunc {
	handler := promhttp.HandlerFor(m.gatherer, promhttp.HandlerOpts{})
	return func(c *gin.Context) {
		handler.ServeHTTP(c.Writer, c.Request)
	}
}

func (m *Metrics) RecordReplyDecision(shouldReply bool, ruleID, ruleName string) {
	replyValue := "false"
	if shouldReply {
		replyValue = "true"
	}
	m.replyDecisionsTotal.WithLabelValues(replyValue).Inc()

	if ruleID == "" {
		ruleID = "none"
	}
	if ruleName == "" {
		ruleName = "none"
	}
	m.matchedRuleTotal.WithLabelValues(ruleID, ruleName).Inc()
}
