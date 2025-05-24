package middleware

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/sirupsen/logrus"
)

// Logger is a middleware that logs HTTP requests
func Logger(logger *logrus.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Create a wrapped ResponseWriter to capture status code
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			// Process request
			next.ServeHTTP(ww, r)

			// Calculate request duration
			duration := time.Since(start)

			// Extract user info from context if available
			userID := ""
			if uid := r.Context().Value("user_id"); uid != nil {
				userID = uid.(string)
			}

			// Log the request
			logger.WithFields(logrus.Fields{
				"method":     r.Method,
				"path":       r.URL.Path,
				"query":      r.URL.RawQuery,
				"status":     ww.Status(),
				"bytes":      ww.BytesWritten(),
				"duration":   duration.Milliseconds(),
				"user_id":    userID,
				"user_agent": r.UserAgent(),
				"remote_ip":  r.RemoteAddr,
			}).Info("HTTP request")
		})
	}
}

// SetupLogger creates and configures a logrus logger
func SetupLogger(isDevelopment bool) *logrus.Logger {
	logger := logrus.New()

	if isDevelopment {
		logger.SetLevel(logrus.DebugLevel)
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
			ForceColors:   true,
		})
	} else {
		logger.SetLevel(logrus.InfoLevel)
		logger.SetFormatter(&logrus.JSONFormatter{})
	}

	return logger
}
