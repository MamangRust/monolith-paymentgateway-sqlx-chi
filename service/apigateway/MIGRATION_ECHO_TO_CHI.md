# Migrasi Apigateway: Echo → Chi

> **Lingkup:** `monolith-payment-gateway-grpc/service/apigateway`
> **Router:** `github.com/labstack/echo/v4` → `github.com/go-chi/chi/v5`
> **Status:** ✅ Selesai — build ✅ · vet ✅ · unit test ✅

---

## 1. Latar Belakang

Apigateway payment-gateway sebelumnya menggunakan framework Echo (labstack/echo v4) sebagai HTTP router, dengan echo-jwt untuk autentikasi dan echo-swagger untuk dokumentasi API. Migrasi ini mengganti seluruh layer HTTP router ke **chi v5.3.2** dengan handler `net/http` standar (`func(w http.ResponseWriter, r *http.Request)`), tanpa mengubah perilaku HTTP eksternal (bentuk JSON response, status code, header, whitelist JWT).

**Yang tidak disentuh:**

- Semua internal service gRPC (`service/auth`, `service/card`, dst.), `pb/`, `proto/`, deploy configs.
- `shared/errors` versi echo-typed (dipakai modul `tests/`) — sebagai gantinya apigateway punya `ApiHandler` versi net/http sendiri.
- Apigateway milik `monolith-ecommerce-grpc` dan `monolith-pointofsale-grpc` (tetap Echo).

---

## 2. Ringkasan Angka

| Item | Jumlah |
|---|---|
| File yang import echo (sebelum) | 65 |
| Paket handler domain yang dimigrasi | 11 (`auth`, `card`, `merchant`, `merchantdocument`, `role`, `saldo`, `topup`, `transaction`, `transfer`, `user`, `withdraw`) |
| Route group `/api/...` | ±45 |
| Rute terdaftar (terkonversi) | 254 |
| Middleware global | 9 |
| Middleware per-route (custom) | 4 (role, require-role, api-key, rate limiter) |
| Test file di-rewrite | 3 (apigateway) + 19 (harness modul `tests/`) |

---

## 3. Dependensi

**Ditambah:**

```
github.com/go-chi/chi/v5          v5.3.2
github.com/go-chi/cors            v1.2.2
github.com/swaggo/http-swagger    v1.3.4
```

**Tidak lagi di-import langsung:**

```
github.com/labstack/echo/v4
github.com/labstack/echo-jwt/v4
github.com/swaggo/echo-swagger
```

> Catatan: echo masih muncul sebagai *indirect dependency* karena `shared/errors` (versi echo-typed) masih dipakai modul `tests/`. Setelah `tests/` dimigrasi penuh, echo bisa hilang total dari modul ini.

---

## 4. Struktur Baru

```
service/apigateway/
├── apierror/          # NEW — ApiHandler versi net/http (tracing + JSON error)
├── httpx/             # NEW — helper JSON/Bind/RealIP, request-scoped values
├── app/client.go      # bootstrap server chi + middleware global
├── handler/           # 43 file handler + handle.go → chi.Router
├── middlewares/       # 6 middleware → func(http.Handler) http.Handler
└── ...
```

Dua package baru dibuat di luar `internal/` agar bisa diimpor modul `tests/`.

---

## 5. Pemetaan Konversi

### 5.1 API Echo → net/http

| Echo (sebelum) | Chi / net/http (sesudah) |
|---|---|
| `func(c echo.Context) error` | `func(w http.ResponseWriter, r *http.Request) error` |
| `c.QueryParam("x")` | `r.URL.Query().Get("x")` |
| `c.Param("id")` | `chi.URLParam(r, "id")` |
| `c.Bind(&body)` | `httpx.Bind(r, &body)` |
| `c.JSON(code, v)` | `httpx.JSON(w, code, v)` |
| `c.Request().Context()` | `r.Context()` |
| `c.Set(k, v)` / `c.Get(k)` | `httpx.SetValue(ctx, k, v)` / `httpx.Get(r, k)` |
| `c.RealIP()` | `httpx.RealIP(r)` |
| `*echo.Echo` (di `Deps`) | `chi.Router` (field `E` → `Router`) |

### 5.2 Registrasi Route

```go
// ── SEBELUM (echo) ──────────────────────────────────────────
routerRole := params.router.Group("/api/role-query")
roleMiddlewareChain := roleMiddleware.Middleware()
requireAdmin := middlewares.RequireRoles("Admin_Role_10")

routerRole.GET("", roleMiddlewareChain(requireAdmin(roleQueryHandler.FindAll)))
routerRole.GET("/:id", roleMiddlewareChain(requireAdmin(roleQueryHandler.FindById)))
```

```go
// ── SESUDAH (chi) ───────────────────────────────────────────
params.router.Route("/api/role-query", func(routerRole chi.Router) {
	roleMiddlewareChain := roleMiddleware.Middleware()
	requireAdmin := middlewares.RequireRoles("Admin_Role_10")

	routerRole.With(roleMiddlewareChain, requireAdmin).Get("/", httpx.Handler(roleQueryHandler.FindAll))
	routerRole.With(roleMiddlewareChain, requireAdmin).Get("/{id}", httpx.Handler(roleQueryHandler.FindById))
})
```

Aturan konversi path & method:

| Echo | Chi |
|---|---|
| `router.GET("/x")` | `router.Get("/x")` |
| `router.POST(...)`, `.DELETE(...)`, `.PUT(...)` | `router.Post(...)`, `.Delete(...)`, `.Put(...)` |
| Path params `:id`, `:user_id` | `{id}`, `{user_id}` |
| Wildcard `/swagger/*` | `/swagger/*` (sama) |
| `Group("")` + `.GET("")` | `Route(...)` + `.Get("/")` (match dengan & tanpa trailing slash) |

Tiga gaya endpoint yang dipertahankan:

1. **Dibungkus tracing:** `router.Get(path, params.apiHandler.Handle("name", h.Method))` → error jadi JSON `AppError`.
2. **Langsung (unwrapped):** `router.Get(path, httpx.Handler(h.Method))` → error apa pun menjadi JSON 500 generik (perilaku lama dipertahankan).
3. **Dengan middleware chain:** `router.With(m1, m2).Get(path, ...)` — urutan chain sama seperti nesting echo.

### 5.3 Middleware Global (`app/client.go`)

| Echo | Chi |
|---|---|
| `middleware.Recover()` | `chi_mw.Recoverer` |
| `middleware.RequestID()` | `chi_mw.RequestID` |
| `middleware.LoggerWithConfig` (JSON) | `chi_mw.RequestLogger` + `jsonLogFormatter` custom (field JSON identik: time, id, remote_ip, host, method, uri, status, error, latency, bytes_in/out) |
| `PyroscopeMiddleware()` | rewrite ke `func(http.Handler) http.Handler` |
| `middleware.CORSWithConfig` | `go-chi/cors` (origins/methods/headers/credentials/maxAge identik) |
| `middleware.Gzip()` | `chi_mw.Compress(5)` |
| `middleware.SecureWithConfig` | custom `createSecureMiddleware()` (XSS, nosniff, X-Frame-Options, HSTS+preload, Referrer-Policy, CSP — chi tidak punya CSP bawaan) |
| `echojwt.WithConfig` + `Skipper` | `middlewares.JWTAuth()` custom |
| 404-hack middleware + custom `HTTPErrorHandler` | `r.NotFound(...)` + `r.MethodNotAllowed(...)` (JSON eksplisit) |

Urutan global dipertahankan: `Recoverer → RequestID → Logger → Pyroscope → CORS → Compress → Secure → JWTAuth`.

### 5.4 JWT (pengganti echo-jwt)

- Parse `Authorization: Bearer <token>` manual dengan `jwt.Parse` + `SigningMethodHMAC` check (setara default HS256 echo-jwt).
- Validasi `exp` tetap otomatis oleh `golang-jwt/v5`.
- Claim `sub` disimpan ke context dengan **dua key** (`userId` dan `user_id`) — sama seperti `SuccessHandler` echo-jwt, agar handler lama tetap jalan.
- Whitelist path & prefix `/swagger`, `/metrics` dipindah apa adanya ke `skipAuth(r)`.
- 401 → JSON `{"status":"error","message":"Unauthorized","code":401}` (bentuk sama dengan error handler echo sebelumnya).

### 5.5 Server Lifecycle

| Echo | Chi |
|---|---|
| `e.Start(port)` | `http.Server{Handler: r}.ListenAndServe()` |
| `e.Shutdown(ctx)` | `srv.Shutdown(ctx)` |
| `e.HideBanner/HidePort` | — (tidak diperlukan) |
| `Client.Echo *echo.Echo` | `Client.Router chi.Router` + `Client.Server *http.Server` |

### 5.6 Swagger

```go
// sebelum: e.GET("/swagger/*", echoSwagger.WrapHandler)
r.Get("/swagger/*", httpSwagger.WrapHandler)
```

Anotasi `@Router` / swag tidak berubah sama sekali.

---

## 6. File Baru

| File | Isi |
|---|---|
| `httpx/httpx.go` | `JSON`, `Bind`, `RealIP`, `SetValue`/`Get` (request-scoped), `Handler` (adapter error→500 JSON), `NewHTTPError`/`WriteHTTPError` (pengganti `echo.NewHTTPError`) |
| `apierror/api_error.go` | Interface `ApiHandler` versi net/http: `Handle(method, handler) http.HandlerFunc` + `HandleApiErrorWithTracing` — replika `shared/errors/api_error.go` dengan `AppError` dari shared |

---

## 7. Verifikasi

```bash
cd monolith-payment-gateway-grpc/service/apigateway
go build ./...   # ✅
go vet ./...     # ✅
go test -count=1 ./...   # ✅ (app, middlewares)
```

- Unit test yang di-rewrite: `app/client_test.go`, `app/errorhandler_test.go`, `middlewares/auth_test.go` (semua pass, termasuk isolasi error per-middleware dan 6 skenario JWT).
- Harness integration `tests/` (19 file) dikonversi mekanis ke chi agar workspace tetap build: `Deps{E: …}` → `{Router: …}`, middleware bypass `c.Set(...)` → `context.WithValue`, `*echo.Echo` → `chi.Router`. Menjalankan suite-nya sendiri tetap butuh Docker/testcontainers (di luar lingkup verifikasi ini).
- Modul `shared` tetap build tanpa perubahan.

---

## 8. Catatan Perilaku (Behavioral Notes)

1. **Bug lama dipertahankan sesuai prinsip "perilaku identik":** handler yang tidak dibungkus `apiHandler.Handle` (mis. `role-query`, `merchant-document`, sebagian `transaction`) tetap mengembalikan **500 generik** saat gRPC error, bukan JSON `AppError`. Di echo dulu error `*AppError` juga tidak dikenali oleh custom `HTTPErrorHandler` (hanya `*echo.HTTPError`). Perbaikan lanjutan bisa dilakukan terpisah bila diinginkan.
2. **Echo masih indirect dependency** via `shared/errors` (versi echo-typed untuk modul `tests/`).
3. Label Pyroscope `endpoint` kini memakai `chi.RouteContext.RoutePattern()` dengan fallback ke `r.URL.Path` — saat middleware berjalan sebelum dispatch, pattern bisa kosong sehingga label memakai path aktual.

---

## 9. Follow-up (Opsional)

- [ ] Smoke test end-to-end dengan skenario `hurl/` atau jalankan full stack (docker compose).
- [ ] `swag init` regenerasi docs jika perlu (anotasi tidak berubah).
- [ ] Migrasi modul `tests/` + `shared/errors` ke net/http secara penuh → echo hilang total.
- [ ] Pertimbangkan bungkus semua handler dengan `apiHandler.Handle` agar error gRPC konsisten menghasilkan JSON `AppError` (bukan 500 generik).
- [ ] Migrasi serupa untuk apigateway `monolith-ecommerce-grpc` (76 file echo) dan `monolith-pointofsale-grpc` (16 file echo) bila diinginkan — pola konversi di dokumen ini berlaku sama.
