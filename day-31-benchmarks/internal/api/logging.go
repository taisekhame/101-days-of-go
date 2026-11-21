package api

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// requestLogger writes one structured log line for every request, after it finishes
func requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		status := c.Writer.Status()
		level := slog.LevelInfo
		switch {
		case status >= 500:
			level = slog.LevelError
		case status >= 400:
			level = slog.LevelWarn
		}

		slog.Log(c.Request.Context(), level, "request",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", status,
			"duration_ms", float64(time.Since(start).Microseconds())/1000,
		)
	}
}