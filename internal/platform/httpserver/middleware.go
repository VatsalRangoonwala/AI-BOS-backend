package httpserver

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/VatsalRangoonwala/AI-BOS-backend/internal/platform/observability"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

func bodyLimitMiddleware(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			if request.ContentLength > maxBytes {
				writeError(response, request, http.StatusRequestEntityTooLarge, "payload_too_large", "The request body is too large", nil)
				return
			}
			request.Body = http.MaxBytesReader(response, request.Body, maxBytes)
			next.ServeHTTP(response, request)
		})
	}
}

func securityHeadersMiddleware(production bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			response.Header().Set("X-Content-Type-Options", "nosniff")
			response.Header().Set("X-Frame-Options", "DENY")
			response.Header().Set("Referrer-Policy", "no-referrer")
			if production {
				response.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			}
			next.ServeHTTP(response, request)
		})
	}
}

func accessLogMiddleware(logger *slog.Logger, metrics observability.Metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			startedAt := time.Now()
			wrapped := chimiddleware.NewWrapResponseWriter(response, request.ProtoMajor)
			next.ServeHTTP(wrapped, request)

			status := wrapped.Status()
			if status == 0 {
				status = http.StatusOK
			}
			route := chi.RouteContext(request.Context()).RoutePattern()
			if route == "" {
				route = "unmatched"
			}
			duration := time.Since(startedAt)
			metrics.ObserveHTTPRequest(request.Context(), request.Method, route, status, duration)
			logger.InfoContext(request.Context(), "http request completed",
				"request_id", RequestIDFromContext(request.Context()),
				"method", request.Method,
				"route", route,
				"status", status,
				"bytes", wrapped.BytesWritten(),
				"duration_ms", duration.Milliseconds(),
			)
		})
	}
}

func recoveryMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					logger.ErrorContext(request.Context(), "panic recovered",
						"request_id", RequestIDFromContext(request.Context()),
						"stack", string(debug.Stack()),
					)
					if wrapped, ok := response.(chimiddleware.WrapResponseWriter); !ok || wrapped.Status() == 0 {
						writeError(response, request, http.StatusInternalServerError, "internal_error", "An unexpected error occurred", nil)
					}
				}
			}()
			next.ServeHTTP(response, request)
		})
	}
}

func tracingMiddleware(tracer observability.Tracer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			ctx, span := tracer.Start(request.Context(), "http.request")
			defer span.End()
			next.ServeHTTP(response, request.WithContext(ctx))
		})
	}
}
