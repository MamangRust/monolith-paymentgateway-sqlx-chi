// Package apierror provides the net/http version of the shared ApiHandler:
// it wraps handlers with tracing/logging and converts returned errors into
// the gateway's JSON error response shape.
package apierror

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/MamangRust/monolith-payment-gateway-pkg/logger"
	sharederrors "github.com/MamangRust/monolith-payment-gateway-shared/errors"
	"github.com/MamangRust/monolith-payment-gateway-shared/observability"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"github.com/MamangRust/monolith-payment-gateway-apigateway/httpx"
)

type ApiHandler interface {
	Handle(method string, handler func(http.ResponseWriter, *http.Request) error) http.HandlerFunc
	HandleApiErrorWithTracing(w http.ResponseWriter, r *http.Request, err error, span trace.Span, method string)
}

type apiHandler struct {
	observability observability.TraceLoggerObservability
	logger        logger.LoggerInterface
}

func NewApiHandler(observability observability.TraceLoggerObservability, logger logger.LoggerInterface) ApiHandler {
	return &apiHandler{
		observability: observability,
		logger:        logger,
	}
}

func (h *apiHandler) Handle(method string, handler func(http.ResponseWriter, *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span, end, status, logSuccess := h.observability.StartTracingAndLogging(
			r.Context(),
			method,
			attribute.String("path", r.URL.Path),
			attribute.String("method", r.Method),
		)

		r = r.WithContext(ctx)

		defer func() {
			end(status)
		}()

		err := handler(w, r)
		if err != nil {
			status = "error"
			h.HandleApiErrorWithTracing(w, r, err, span, method)
			return
		}

		logSuccess("Request completed successfully")
	}
}

func (h *apiHandler) HandleApiErrorWithTracing(w http.ResponseWriter, r *http.Request, err error, span trace.Span, method string) {
	traceID := span.SpanContext().TraceID().String()

	h.logger.Error(
		fmt.Sprintf("API error in %s", method),
		zap.Error(err),
		zap.String("trace.id", traceID),
		zap.String("path", r.URL.Path),
		zap.String("method", r.Method),
	)

	span.SetAttributes(
		attribute.String("trace.id", traceID),
		attribute.String("error", err.Error()),
	)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())

	HandleApiError(w, err, traceID)
}

type ErrorResponse struct {
	Status      string                         `json:"status"`
	Message     string                         `json:"message"`
	Type        sharederrors.ErrorType         `json:"type,omitempty"`
	Code        int                            `json:"code"`
	TraceID     string                         `json:"trace_id,omitempty"`
	Retryable   bool                           `json:"retryable,omitempty"`
	Validations []sharederrors.ValidationError `json:"validations,omitempty"`
}

func HandleApiError(w http.ResponseWriter, err error, traceID string) {
	if err == nil {
		return
	}

	var apiErr *sharederrors.AppError
	if errors.As(err, &apiErr) {
		response := ErrorResponse{
			Status:      "error",
			Message:     apiErr.Message,
			Type:        apiErr.Type,
			Code:        apiErr.Code,
			TraceID:     traceID,
			Retryable:   apiErr.Retryable,
			Validations: apiErr.Validations,
		}
		code := apiErr.Code
		if code < 100 || code > 599 {
			code = http.StatusInternalServerError
		}
		_ = httpx.JSON(w, code, response)
		return
	}

	response := ErrorResponse{
		Status:  "error",
		Message: "An internal server error occurred",
		Type:    sharederrors.ErrorTypeInternal,
		Code:    http.StatusInternalServerError,
		TraceID: traceID,
	}
	_ = httpx.JSON(w, http.StatusInternalServerError, response)
}
