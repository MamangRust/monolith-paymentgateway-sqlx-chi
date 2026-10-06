package middlewares

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/spf13/viper"

	"github.com/MamangRust/monolith-payment-gateway-apigateway/httpx"
)

var whiteListPaths = []string{
	"/api/auth/login",
	"/api/auth/register", "/api/auth/hello",
	"/api/auth/verify-code",
	"/api/auth/refresh-token",
	"/api/auth/forgot-password",
	"/api/auth/reset-password",
	"/docs/",
	"/docs",
	"/swagger",
	"/metrics",
	"/health",
	"/live",
	"/ready",
}

func skipAuth(r *http.Request) bool {
	path := r.URL.Path

	for _, p := range whiteListPaths {
		if path == p ||
			strings.HasPrefix(path, "/swagger") ||
			strings.HasPrefix(path, "/metrics") {
			return true
		}
	}

	return false
}

// JWTAuth validates the Bearer access token and stores the token subject in
// the request context. Both "userId" and "user_id" keys are populated to keep
// the same request-scoped values that echojwt previously set on echo.Context.
func JWTAuth() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if skipAuth(r) {
				next.ServeHTTP(w, r)
				return
			}

			tokenString, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
			if !ok || tokenString == "" {
				unauthorized(w)
				return
			}

			token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
				}
				return []byte(viper.GetString("SECRET_KEY")), nil
			})
			if err != nil || !token.Valid {
				unauthorized(w)
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				unauthorized(w)
				return
			}
			subject := claims["sub"]

			// Keep both keys during the transition: existing handlers use
			// userId while role-protected handlers use user_id.
			ctx := httpx.SetValue(r.Context(), "userId", subject)
			ctx = httpx.SetValue(ctx, "user_id", subject)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func unauthorized(w http.ResponseWriter) {
	_ = httpx.JSON(w, http.StatusUnauthorized, map[string]interface{}{
		"status":  "error",
		"message": "Unauthorized",
		"code":    http.StatusUnauthorized,
	})
}
