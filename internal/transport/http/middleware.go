// middleware.go — HTTP middleware for the MCP HTTP transport.
//
// Two middleware layers are provided:
//
//  1. Logging — records method, path, remote addr, response code, and duration
//     for every request using slog.
//
//  2. APIKeyAuth — validates the API key supplied in either the
//     "Authorization: Bearer <key>" or "X-API-Key: <key>" header.
//     POST requests whose JSON-RPC method is in the publicMethods set bypass
//     authentication entirely.  GET requests (SSE stream establishment) always
//     require authentication.
package httpsvr

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// ---- logging middleware --------------------------------------------------

// responseWriter wraps http.ResponseWriter to capture the status code written
// by the downstream handler so it can be included in the access log.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.written = true
	}
	rw.ResponseWriter.WriteHeader(code)
}

// loggingMiddleware wraps next and logs one slog line per request.
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := newResponseWriter(w)
		next.ServeHTTP(rw, r)
		slog.Info("HTTP request",
			"method", r.Method,
			"path", r.URL.Path,
			"remote", r.RemoteAddr,
			"status", rw.statusCode,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}

// ---- API-key auth middleware ---------------------------------------------

// jsonRPCEnvelope holds just enough of a JSON-RPC request to read the method.
type jsonRPCEnvelope struct {
	Method string `json:"method"`
}

// apiKeyAuthMiddleware returns a middleware that enforces API-key authentication.
//
//   - Requests whose JSON-RPC method is in publicMethods skip the check.
//   - All other POST requests must supply a matching key via
//     "Authorization: Bearer <key>" or "X-API-Key: <key>".
//   - GET requests (SSE stream establishment) always require the key.
//   - If apiKey is empty the middleware is effectively disabled and all requests pass.
func apiKeyAuthMiddleware(next http.Handler, apiKey string, publicMethods map[string]struct{}) http.Handler {
	// If no API key is configured, auth is disabled.
	if apiKey == "" {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// POST: peek at the JSON-RPC method to decide if auth is needed.
		if r.Method == http.MethodPost {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "failed to read request body", http.StatusBadRequest)
				return
			}
			// Restore the body so the downstream handler can read it.
			r.Body = io.NopCloser(bytes.NewReader(body))

			var env jsonRPCEnvelope
			_ = json.Unmarshal(body, &env) // ignore unmarshal errors; auth will just be enforced

			if _, ok := publicMethods[env.Method]; ok {
				// Public method — skip auth.
				next.ServeHTTP(w, r)
				return
			}
		}

		// Enforce authentication.
		if !checkAPIKey(r, apiKey) {
			slog.Warn("Unauthorized request",
				"method", r.Method,
				"path", r.URL.Path,
				"remote", r.RemoteAddr,
			)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// checkAPIKey returns true if the request carries the expected API key in
// either the Authorization or X-API-Key header.
func checkAPIKey(r *http.Request, expected string) bool {
	// Authorization: Bearer <key>
	if auth := r.Header.Get("Authorization"); auth != "" {
		if strings.HasPrefix(auth, "Bearer ") {
			if strings.TrimPrefix(auth, "Bearer ") == expected {
				return true
			}
		}
	}
	// X-API-Key: <key>
	if r.Header.Get("X-API-Key") == expected {
		return true
	}
	return false
}

// buildPublicMethodSet converts a string slice to a set for O(1) lookup.
func buildPublicMethodSet(methods []string) map[string]struct{} {
	m := make(map[string]struct{}, len(methods))
	for _, method := range methods {
		m[method] = struct{}{}
	}
	return m
}

// NewHandler builds the full middleware chain (logging → auth → mcpHandler)
// for the given API key and public method list.
// It is exposed so tests and advanced callers can compose the chain without
// starting a real net.Listener.
func NewHandler(apiKey string, publicMethods []string, mcpHandler http.Handler) http.Handler {
	publicSet := buildPublicMethodSet(publicMethods)
	return loggingMiddleware(apiKeyAuthMiddleware(mcpHandler, apiKey, publicSet))
}
