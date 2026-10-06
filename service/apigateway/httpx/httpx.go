// Package httpx provides small helpers shared by the chi-based handlers:
// JSON response writing, request body binding, client IP extraction, and
// request-scoped values that were previously stored on echo.Context.
package httpx

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"
)

// JSON writes v as a JSON-encoded response with the given status code.
func JSON(w http.ResponseWriter, code int, v interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	return json.NewEncoder(w).Encode(v)
}

// Bind decodes the JSON request body into v.
func Bind(r *http.Request, v interface{}) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}

// RealIP resolves the client IP, honoring the same proxy headers as the
// previous echo implementation.
func RealIP(r *http.Request) string {
	if fw := r.Header.Get("X-Forwarded-For"); fw != "" {
		if i := strings.IndexByte(fw, ','); i >= 0 {
			fw = fw[:i]
		}
		fw = strings.TrimSpace(fw)
		if fw != "" {
			return fw
		}
	}

	if ip := strings.TrimSpace(r.Header.Get("X-Real-Ip")); ip != "" {
		return ip
	}

	if ip, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return ip
	}
	return r.RemoteAddr
}

// SetValue returns a copy of ctx carrying a request-scoped value. Keys are
// plain strings so handlers can keep addressing them by the same names that
// were used on echo.Context ("userId", "user_id", "role_names", ...).
func SetValue(ctx context.Context, key string, value interface{}) context.Context {
	return context.WithValue(ctx, key, value)
}

// Get reads a request-scoped value previously stored with SetValue.
func Get(r *http.Request, key string) interface{} {
	return r.Context().Value(key)
}

// Handler adapts an error-returning handler to http.HandlerFunc. It reproduces
// the gateway's former echo HTTPErrorHandler behavior: any error that reaches
// the top of an unwrapped route becomes a JSON 500 with a fixed body, while
// routes wrapped by the ApiHandler handle their own errors before returning.
func Handler(h func(http.ResponseWriter, *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := h(w, r); err != nil {
			_ = JSON(w, http.StatusInternalServerError, map[string]interface{}{
				"status":  "error",
				"message": "Internal server error",
				"code":    http.StatusInternalServerError,
			})
		}
	}
}

// HTTPError is an error carrying a status code and message, written as JSON by
// WriteHTTPError. It replaces echo.NewHTTPError in middleware rejections.
type HTTPError struct {
	Code    int
	Message string
}

func (e *HTTPError) Error() string { return e.Message }

// NewHTTPError builds an *HTTPError with the given status code and message.
func NewHTTPError(code int, message string) error {
	return &HTTPError{Code: code, Message: message}
}

// WriteHTTPError writes err as a JSON error response. Non-*HTTPError errors
// fall back to a generic 500, matching the former echo error handler.
func WriteHTTPError(w http.ResponseWriter, err error) {
	var httpErr *HTTPError
	if errors.As(err, &httpErr) {
		_ = JSON(w, httpErr.Code, map[string]interface{}{
			"status":  "error",
			"message": httpErr.Message,
			"code":    httpErr.Code,
		})
		return
	}

	_ = JSON(w, http.StatusInternalServerError, map[string]interface{}{
		"status":  "error",
		"message": "Internal server error",
		"code":    http.StatusInternalServerError,
	})
}
