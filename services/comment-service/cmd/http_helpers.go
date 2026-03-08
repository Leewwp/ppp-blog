package main

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	traceIDHeader     = "X-Trace-Id"
	requestIDHeader   = "X-Request-Id"
	traceIDContextKey = "trace_id"
)

func traceIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := strings.TrimSpace(c.GetHeader(traceIDHeader))
		if traceID == "" {
			traceID = strings.TrimSpace(c.GetHeader(requestIDHeader))
		}
		if traceID == "" {
			traceID = newTraceID()
		}

		c.Set(traceIDContextKey, traceID)
		c.Writer.Header().Set(traceIDHeader, traceID)
		c.Request.Header.Set(traceIDHeader, traceID)
		c.Next()
	}
}

func requestLogger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		logger.Info("http request",
			"method", c.Request.Method,
			"path", path,
			"status", c.Writer.Status(),
			"duration_ms", time.Since(start).Milliseconds(),
			"client_ip", c.ClientIP(),
			"trace_id", getTraceID(c),
		)
	}
}

func getTraceID(c *gin.Context) string {
	value, ok := c.Get(traceIDContextKey)
	if !ok {
		return ""
	}
	traceID, ok := value.(string)
	if !ok {
		return ""
	}
	return traceID
}

func newTraceID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 16)
	}
	return hex.EncodeToString(bytes)
}
