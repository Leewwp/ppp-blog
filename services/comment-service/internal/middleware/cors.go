package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// CORS 允许博客前端跨域访问 comment-service。
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Trace-Id, X-Request-Id")
		c.Header("Access-Control-Expose-Headers", "X-Trace-Id, X-RateLimit-Remaining, Retry-After")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
