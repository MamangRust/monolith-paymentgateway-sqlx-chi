package app

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MamangRust/monolith-payment-gateway-apigateway/httpx"
	"github.com/MamangRust/monolith-payment-gateway-apigateway/middlewares"
	"github.com/go-chi/chi/v5"
	chi_mw "github.com/go-chi/chi/v5/middleware"
	"github.com/spf13/viper"
)

func TestMiddlewareErrorsWriteJSON(t *testing.T) {
	viper.Set("SECRET_KEY", "test-secret")

	router, _ := createChiServer(&ClientConfig{}, nil, nil)
	router.Get("/api/test-protected", func(w http.ResponseWriter, r *http.Request) {
		_ = httpx.JSON(w, http.StatusOK, map[string]string{"ok": "1"})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/test-protected", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	t.Logf("status=%d body=%q", rec.Code, rec.Body.String())

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for missing token, got %d with body %q", rec.Code, rec.Body.String())
	}
	if rec.Body.Len() == 0 {
		t.Fatal("expected a JSON error body, got empty response")
	}
}

// TestErrorIsolation builds the chi chain incrementally to find the middleware
// that swallows error responses. Every case registers a route handler that
// writes a 401 response, simulating any route-level middleware rejection.
func TestErrorIsolation(t *testing.T) {
	viper.Set("SECRET_KEY", "test-secret")

	// The route handler writes a 401 JSON body directly, mirroring the way the
	// chi middleware stack rejects requests (no error propagation, response is
	// always written by whoever rejects).
	rejectingHandler := func(w http.ResponseWriter, r *http.Request) {
		_ = httpx.JSON(w, http.StatusUnauthorized, map[string]interface{}{
			"status":  "error",
			"message": "no token",
			"code":    http.StatusUnauthorized,
		})
	}

	cases := []struct {
		name  string
		setup func(r chi.Router)
	}{
		{"baseline", func(r chi.Router) {}},
		{"recover", func(r chi.Router) { r.Use(chi_mw.Recoverer) }},
		{"requestid", func(r chi.Router) { r.Use(chi_mw.RequestID) }},
		{"logger", func(r chi.Router) { r.Use(createLoggerMiddleware()) }},
		{"cors", func(r chi.Router) {
			r.Use(createCORSMiddleware(nil))
		}},
		{"compress", func(r chi.Router) { r.Use(chi_mw.Compress(5)) }},
		{"secure", func(r chi.Router) { r.Use(createSecureMiddleware()) }},
		{"recover+compress+cors+secure", func(r chi.Router) {
			r.Use(chi_mw.Recoverer)
			r.Use(chi_mw.RequestID)
			r.Use(chi_mw.Compress(5))
			r.Use(createCORSMiddleware(nil))
			r.Use(createSecureMiddleware())
		}},
		{"pyroscope", func(r chi.Router) { r.Use(middlewares.PyroscopeMiddleware()) }},
		{"jwtauth", func(r chi.Router) { r.Use(middlewares.JWTAuth()) }},
		{"all-with-jwtauth", func(r chi.Router) {
			r.Use(chi_mw.Recoverer)
			r.Use(chi_mw.RequestID)
			r.Use(chi_mw.Compress(5))
			r.Use(createCORSMiddleware(nil))
			r.Use(createSecureMiddleware())
			r.Use(middlewares.PyroscopeMiddleware())
			r.Use(middlewares.JWTAuth())
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// The JWT middleware skips whitelisted paths, so protected paths are
			// used for the rejection scenario; the JWT case itself rejects with
			// its own 401 before the handler runs.
			r := chi.NewRouter()
			tc.setup(r)

			r.Get("/api/test-protected", rejectingHandler)

			req := httptest.NewRequest(http.MethodGet, "/api/test-protected", nil)
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			t.Logf("status=%d body=%q", rec.Code, rec.Body.String())
			if rec.Code != http.StatusUnauthorized || rec.Body.Len() == 0 {
				t.Errorf("expected 401 with body, got status=%d body=%q", rec.Code, rec.Body.String())
			}
		})
	}
}
