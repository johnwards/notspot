package api

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

type contextKey int

const correlationIDKey contextKey = iota

// CorrelationID returns the correlation ID from the request context.
func CorrelationID(ctx context.Context) string {
	if id, ok := ctx.Value(correlationIDKey).(string); ok {
		return id
	}
	return ""
}

// Recovery returns middleware that recovers from panics and returns a 500 error
// in HubSpot error format.
func Recovery() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					slog.Error("panic recovered",
						"error", rec,
						"method", r.Method,
						"path", r.URL.Path,
					)
					corrID := CorrelationID(r.Context())
					WriteError(w, http.StatusInternalServerError, &Error{
						Status:        "error",
						Message:       "Internal Server Error",
						CorrelationID: corrID,
						Category:      "INTERNAL_ERROR",
					})
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// RequestID returns middleware that generates a UUID v4 correlation ID, stores
// it in the request context, and adds it to the response headers.
func RequestID() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := newUUID()
			ctx := context.WithValue(r.Context(), correlationIDKey, id)
			w.Header().Set("X-Correlation-Id", id)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// Auth returns middleware that validates the Bearer token if authToken is
// non-empty. If authToken is empty, all requests pass through.
func Auth(authToken string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/_ui/") || strings.HasPrefix(r.URL.Path, "/_ui") {
				next.ServeHTTP(w, r)
				return
			}

			if authToken == "" {
				next.ServeHTTP(w, r)
				return
			}

			header := r.Header.Get("Authorization")
			token := strings.TrimPrefix(header, "Bearer ")
			if header == "" || token != authToken {
				corrID := CorrelationID(r.Context())
				WriteError(w, http.StatusUnauthorized, &Error{
					Status:        "error",
					Message:       "Authentication credentials not found. This API supports OAuth 2.0 authentication and you can find more details at https://developers.hubspot.com/docs/methods/auth/oauth-overview",
					CorrelationID: corrID,
					Category:      CategoryValidationError,
				})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// JSONContentType returns middleware that sets the Content-Type header to
// application/json on all responses.
func JSONContentType() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/_ui/") || strings.HasPrefix(r.URL.Path, "/_ui") {
				next.ServeHTTP(w, r)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			next.ServeHTTP(w, r)
		})
	}
}

// statusWriter wraps http.ResponseWriter to capture the status code.
type statusWriter struct {
	http.ResponseWriter
	code int
}

// WriteHeader captures the status code and delegates to the wrapped writer.
func (sw *statusWriter) WriteHeader(code int) {
	sw.code = code
	sw.ResponseWriter.WriteHeader(code)
}

// Logging returns middleware that logs each request with slog.
func Logging() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sw := &statusWriter{ResponseWriter: w, code: http.StatusOK}
			next.ServeHTTP(sw, r)
			slog.Info("request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", sw.code,
				"duration", time.Since(start).String(),
			)
		})
	}
}

// Chain applies middleware in order so that the first middleware is the
// outermost handler.
func Chain(handler http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}

// newUUID generates a UUID v4 using crypto/rand.
func newUUID() string {
	var uuid [16]byte
	_, err := rand.Read(uuid[:])
	if err != nil {
		// Fallback — extremely unlikely to fail.
		return "00000000-0000-4000-8000-000000000000"
	}
	// Set version (4) and variant (RFC 4122).
	uuid[6] = (uuid[6] & 0x0f) | 0x40
	uuid[8] = (uuid[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:16])
}
