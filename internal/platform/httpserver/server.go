package httpserver

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/VatsalRangoonwala/AI-BOS-backend/internal/platform/observability"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

type Config struct {
	Addr              string
	FrontendOrigins   []string
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	RequestTimeout    time.Duration
	ShutdownTimeout   time.Duration
	MaxBodyBytes      int64
	Production        bool
}

type Server struct {
	httpServer      *http.Server
	shutdownTimeout time.Duration
}

func New(cfg Config, logger *slog.Logger, dependencies Dependencies, hooks observability.Hooks) *Server {
	noopHooks := observability.NoopHooks()
	if hooks.Metrics == nil {
		hooks.Metrics = noopHooks.Metrics
	}
	if hooks.Tracer == nil {
		hooks.Tracer = noopHooks.Tracer
	}

	router := chi.NewRouter()
	router.Use(requestIDMiddleware)
	router.Use(securityHeadersMiddleware(cfg.Production))
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.FrontendOrigins,
		AllowedMethods:   []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "Idempotency-Key", "If-Match", requestIDHeader},
		ExposedHeaders:   []string{requestIDHeader},
		AllowCredentials: true,
		MaxAge:           300,
	}))
	router.Use(bodyLimitMiddleware(cfg.MaxBodyBytes))
	router.Use(accessLogMiddleware(logger, hooks.Metrics))
	router.Use(recoveryMiddleware(logger))
	router.Use(tracingMiddleware(hooks.Tracer))
	router.Use(chimiddleware.Timeout(cfg.RequestTimeout))

	router.Get("/healthz", healthHandler)
	router.Get("/readyz", readinessHandler(dependencies, hooks.Metrics, logger))
	router.NotFound(func(response http.ResponseWriter, request *http.Request) {
		writeError(response, request, http.StatusNotFound, "not_found", "The requested resource was not found", nil)
	})
	router.MethodNotAllowed(func(response http.ResponseWriter, request *http.Request) {
		writeError(response, request, http.StatusMethodNotAllowed, "method_not_allowed", "The request method is not allowed", nil)
	})

	return &Server{
		httpServer: &http.Server{
			Addr:              cfg.Addr,
			Handler:           router,
			ReadHeaderTimeout: cfg.ReadHeaderTimeout,
			ReadTimeout:       cfg.ReadTimeout,
			WriteTimeout:      cfg.WriteTimeout,
			IdleTimeout:       cfg.IdleTimeout,
			ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelError),
		},
		shutdownTimeout: cfg.ShutdownTimeout,
	}
}

func (server *Server) Handler() http.Handler {
	return server.httpServer.Handler
}

func (server *Server) Run(ctx context.Context) error {
	listener, err := net.Listen("tcp", server.httpServer.Addr)
	if err != nil {
		return err
	}
	return server.Serve(ctx, listener)
}

func (server *Server) Serve(ctx context.Context, listener net.Listener) error {
	serveErrors := make(chan error, 1)
	go func() {
		serveErrors <- server.httpServer.Serve(listener)
	}()

	select {
	case err := <-serveErrors:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), server.shutdownTimeout)
		defer cancel()

		shutdownErr := server.httpServer.Shutdown(shutdownCtx)
		if shutdownErr != nil {
			_ = server.httpServer.Close()
		}
		serveErr := <-serveErrors
		if errors.Is(serveErr, http.ErrServerClosed) {
			serveErr = nil
		}
		return errors.Join(shutdownErr, serveErr)
	}
}
