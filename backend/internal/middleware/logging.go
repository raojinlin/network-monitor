package middleware

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

// ResponseWriter wrapper to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.written = true
		rw.ResponseWriter.WriteHeader(code)
	}
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.statusCode = http.StatusOK
		rw.written = true
	}
	return rw.ResponseWriter.Write(b)
}

// Hijack implements http.Hijacker interface for WebSocket support
func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := rw.ResponseWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return nil, nil, fmt.Errorf("responseWriter does not implement http.Hijacker")
}

// LoggingMiddleware logs HTTP requests and responses
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Skip logging for static files and non-API requests
		if shouldSkipLogging(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		
		// Skip wrapping for WebSocket connections to preserve Hijacker interface
		if r.URL.Path == "/ws" || r.Header.Get("Upgrade") == "websocket" {
			// Log request start but don't wrap response writer
			log.Printf("[REQUEST] %s %s from %s", r.Method, r.URL.Path, getClientIP(r))
			next.ServeHTTP(w, r)
			duration := time.Since(start)
			log.Printf("[RESPONSE] %s %s -> WebSocket (%v) from %s", 
				r.Method, r.URL.Path, duration, getClientIP(r))
			return
		}
		
		// Wrap response writer to capture status code
		wrapper := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}
		
		// Log request start
		log.Printf("[REQUEST] %s %s from %s", r.Method, r.URL.Path, getClientIP(r))
		
		// Process request
		next.ServeHTTP(wrapper, r)
		
		// Calculate duration
		duration := time.Since(start)
		
		// Log request completion
		log.Printf("[RESPONSE] %s %s -> %d (%v) from %s", 
			r.Method, r.URL.Path, wrapper.statusCode, duration, getClientIP(r))
	})
}

// shouldSkipLogging determines if a request should be logged
func shouldSkipLogging(path string) bool {
	// Skip static files and common paths
	skipPrefixes := []string{
		"/assets/",
		"/css/",
		"/js/",
		"/img/",
		"/favicon.ico",
		"/manifest.json",
	}
	
	for _, prefix := range skipPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	
	// Skip health check endpoints
	if path == "/" && path != "/api/" {
		return true
	}
	
	return false
}

// getClientIP extracts the client IP address from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}
	
	// Check X-Real-IP header
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}
	
	// Fall back to RemoteAddr
	return r.RemoteAddr
}

// WebSocketLoggingMiddleware logs WebSocket connections
func WebSocketLoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Upgrade") == "websocket" {
			log.Printf("[WEBSOCKET] Connection from %s", getClientIP(r))
		}
		next.ServeHTTP(w, r)
	})
}