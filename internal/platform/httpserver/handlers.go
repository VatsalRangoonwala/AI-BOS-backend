package httpserver

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/VatsalRangoonwala/AI-BOS-backend/internal/platform/observability"
	"github.com/go-chi/chi/v5"
)

type HealthChecker interface {
	Check(context.Context) error
}

type Dependencies struct {
	Postgres   HealthChecker
	Redis      HealthChecker
	Identity   chi.Router
	Businesses chi.Router
	Me         http.HandlerFunc
}

type healthData struct {
	Status string `json:"status"`
}

type readinessData struct {
	Status       string             `json:"status"`
	Dependencies dependencyStatuses `json:"dependencies"`
}

type dependencyStatuses struct {
	Postgres string `json:"postgres"`
	Redis    string `json:"redis"`
}

type readinessErrorDetails struct {
	Dependencies dependencyStatuses `json:"dependencies"`
}

func healthHandler(response http.ResponseWriter, request *http.Request) {
	writeSuccess(response, request, http.StatusOK, healthData{Status: "ok"})
}

func readinessHandler(dependencies Dependencies, metrics observability.Metrics, logger *slog.Logger) http.HandlerFunc {
	type checkResult struct {
		name     string
		healthy  bool
		duration time.Duration
	}

	return func(response http.ResponseWriter, request *http.Request) {
		results := make(chan checkResult, 2)
		checks := []struct {
			name    string
			checker HealthChecker
		}{
			{name: "postgres", checker: dependencies.Postgres},
			{name: "redis", checker: dependencies.Redis},
		}

		for _, check := range checks {
			go func() {
				startedAt := time.Now()
				err := check.checker.Check(request.Context())
				result := checkResult{name: check.name, healthy: err == nil, duration: time.Since(startedAt)}
				metrics.ObserveDependencyCheck(request.Context(), result.name, result.healthy, result.duration)
				results <- result
			}()
		}

		statuses := dependencyStatuses{Postgres: "unavailable", Redis: "unavailable"}
		ready := true
		for range checks {
			result := <-results
			status := "unavailable"
			if result.healthy {
				status = "ok"
			} else {
				ready = false
				logger.WarnContext(request.Context(), "dependency health check failed",
					"request_id", RequestIDFromContext(request.Context()),
					"dependency", result.name,
				)
			}
			switch result.name {
			case "postgres":
				statuses.Postgres = status
			case "redis":
				statuses.Redis = status
			}
		}

		if !ready {
			writeError(
				response,
				request,
				http.StatusServiceUnavailable,
				"dependency_unavailable",
				"One or more required dependencies are unavailable",
				readinessErrorDetails{Dependencies: statuses},
			)
			return
		}

		writeSuccess(response, request, http.StatusOK, readinessData{
			Status:       "ready",
			Dependencies: statuses,
		})
	}
}
