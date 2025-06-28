package metrics

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"
)

var globalCollector *Collector

func init() {
	globalCollector = NewCollector()
}

func GetGlobalCollector() *Collector {
	return globalCollector
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    int64
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.written += int64(n)
	return n, err
}

// Hijack implements the http.Hijacker interface
func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := rw.ResponseWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return nil, nil, fmt.Errorf("responseWriter does not support hijacking")
}

// Flush implements the http.Flusher interface
func (rw *responseWriter) Flush() {
	if flusher, ok := rw.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// CloseNotify implements the http.CloseNotifier interface (deprecated but sometimes still needed)
func (rw *responseWriter) CloseNotify() <-chan bool {
	if notifier, ok := rw.ResponseWriter.(http.CloseNotifier); ok {
		return notifier.CloseNotify()
	}
	return make(chan bool)
}

func HTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		rw := &responseWriter{
			ResponseWriter: w,
			statusCode:     200,
		}

		next.ServeHTTP(rw, r)

		duration := time.Since(start)

		metric := Metric{
			Type:     RequestMetric,
			Duration: duration,
			Method:   r.Method,
			Path:     r.URL.Path,
			Status:   rw.statusCode,
			Metadata: map[string]interface{}{
				"bytes_written": rw.written,
				"user_agent":    r.UserAgent(),
				"remote_addr":   r.RemoteAddr,
			},
		}

		if rw.statusCode >= 400 {
			metric.Error = "HTTP " + strconv.Itoa(rw.statusCode)
		}

		globalCollector.RecordMetric(metric)
	})
}

func RecordDatabaseOperation(operation string, duration time.Duration, err error) {
	metric := Metric{
		Type:     DatabaseMetric,
		Duration: duration,
		Metadata: map[string]interface{}{
			"operation": operation,
		},
	}

	if err != nil {
		metric.Error = err.Error()
	}

	globalCollector.RecordMetric(metric)
}

func RecordHookExecution(collection, event string, duration time.Duration, err error) {
	metric := Metric{
		Type:     HookMetric,
		Duration: duration,
		Path:     fmt.Sprintf("/%s/%s", collection, event), // For easier filtering
		Metadata: map[string]interface{}{
			"collection": collection,
			"event":      event,
			"event_full": fmt.Sprintf("%s.%s", collection, event), // For detailed event tracking
		},
	}

	if err != nil {
		metric.Error = err.Error()
	}

	globalCollector.RecordMetric(metric)
}

func RecordError(errorType string, message string) {
	metric := Metric{
		Type:  ErrorMetric,
		Error: message,
		Metadata: map[string]interface{}{
			"error_type": errorType,
		},
	}

	globalCollector.RecordMetric(metric)
}
