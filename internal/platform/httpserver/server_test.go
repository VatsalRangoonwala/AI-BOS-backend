package httpserver

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/VatsalRangoonwala/AI-BOS-backend/internal/platform/observability"
)

type fakeChecker func(context.Context) error

func (check fakeChecker) Check(ctx context.Context) error {
	return check(ctx)
}

func TestHealthEndpoint(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	response := httptest.NewRecorder()
	testServer(nil, nil).Handler().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	requestID := response.Header().Get(requestIDHeader)
	if requestID == "" {
		t.Fatal("response did not include X-Request-Id")
	}
	var body struct {
		Data healthData   `json:"data"`
		Meta responseMeta `json:"meta"`
	}
	decodeResponse(t, response, &body)
	if body.Data.Status != "ok" || body.Meta.RequestID != requestID {
		t.Fatalf("body = %+v, request ID = %q", body, requestID)
	}
}

func TestReadinessEndpoint(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	response := httptest.NewRecorder()
	testServer(nil, nil).Handler().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	var body struct {
		Data readinessData `json:"data"`
	}
	decodeResponse(t, response, &body)
	if body.Data.Status != "ready" || body.Data.Dependencies.Postgres != "ok" || body.Data.Dependencies.Redis != "ok" {
		t.Fatalf("body = %+v", body)
	}
}

func TestReadinessEndpointReportsDependencyFailures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		postgresErr error
		redisErr    error
		postgres    string
		redis       string
	}{
		{name: "PostgreSQL failure", postgresErr: errors.New("database offline"), postgres: "unavailable", redis: "ok"},
		{name: "Redis failure", redisErr: errors.New("cache offline"), postgres: "ok", redis: "unavailable"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			request := httptest.NewRequest(http.MethodGet, "/readyz", nil)
			response := httptest.NewRecorder()
			testServer(test.postgresErr, test.redisErr).Handler().ServeHTTP(response, request)

			if response.Code != http.StatusServiceUnavailable {
				t.Fatalf("status = %d, want 503", response.Code)
			}
			var body errorResponse
			decodeResponse(t, response, &body)
			if body.Error.Code != "dependency_unavailable" {
				t.Fatalf("error code = %q", body.Error.Code)
			}
			details, ok := body.Error.Details.(map[string]any)
			if !ok {
				t.Fatalf("details type = %T", body.Error.Details)
			}
			dependencies, ok := details["dependencies"].(map[string]any)
			if !ok {
				t.Fatalf("dependencies type = %T", details["dependencies"])
			}
			if dependencies["postgres"] != test.postgres || dependencies["redis"] != test.redis {
				t.Fatalf("dependencies = %v, want postgres=%s redis=%s", dependencies, test.postgres, test.redis)
			}
			if body.Error.Message == "database offline" || body.Error.Message == "cache offline" {
				t.Fatal("response exposed an internal dependency error")
			}
		})
	}
}

func TestRequestIDMiddlewarePreservesValidID(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	request.Header.Set(requestIDHeader, "client-request_123")
	response := httptest.NewRecorder()
	testServer(nil, nil).Handler().ServeHTTP(response, request)

	if got := response.Header().Get(requestIDHeader); got != "client-request_123" {
		t.Fatalf("X-Request-Id = %q, want client-request_123", got)
	}
}

func TestRequestIDMiddlewareReplacesInvalidID(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	request.Header.Set(requestIDHeader, "invalid request id")
	response := httptest.NewRecorder()
	testServer(nil, nil).Handler().ServeHTTP(response, request)

	if got := response.Header().Get(requestIDHeader); got == "invalid request id" || !validRequestID(got) {
		t.Fatalf("X-Request-Id = %q, want a generated valid ID", got)
	}
}

func TestBodyLimitRejectsKnownOversizedRequest(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodPost, "/healthz", io.NopCloser(&fixedReader{remaining: 2048}))
	request.ContentLength = 2048
	response := httptest.NewRecorder()
	testServer(nil, nil).Handler().ServeHTTP(response, request)

	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413", response.Code)
	}
	var body errorResponse
	decodeResponse(t, response, &body)
	if body.Error.Code != "payload_too_large" {
		t.Fatalf("error code = %q, want payload_too_large", body.Error.Code)
	}
}

func TestCORSMiddlewareAllowsConfiguredOrigin(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodOptions, "/healthz", nil)
	request.Header.Set("Origin", "http://localhost:3000")
	request.Header.Set("Access-Control-Request-Method", http.MethodGet)
	response := httptest.NewRecorder()
	testServer(nil, nil).Handler().ServeHTTP(response, request)

	if got := response.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Fatalf("Access-Control-Allow-Origin = %q", got)
	}
}

func TestServerGracefullyDrainsActiveRequest(t *testing.T) {
	serverConnection, clientConnection := net.Pipe()
	listener := newSingleConnectionListener(serverConnection)
	defer func() {
		if err := clientConnection.Close(); err != nil {
			t.Errorf("close client connection: %v", err)
		}
	}()

	server := testServer(nil, nil)
	requestStarted := make(chan struct{})
	releaseRequest := make(chan struct{})
	server.httpServer.Handler = http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		close(requestStarted)
		<-releaseRequest
		response.WriteHeader(http.StatusNoContent)
	})

	ctx, cancel := context.WithCancel(context.Background())
	serveDone := make(chan error, 1)
	go func() { serveDone <- server.Serve(ctx, listener) }()

	requestDone := make(chan error, 1)
	go func() {
		if _, requestErr := fmt.Fprint(clientConnection, "GET / HTTP/1.1\r\nHost: test\r\nConnection: close\r\n\r\n"); requestErr != nil {
			requestDone <- requestErr
			return
		}
		response, requestErr := http.ReadResponse(bufio.NewReader(clientConnection), &http.Request{Method: http.MethodGet})
		if requestErr == nil {
			_ = response.Body.Close()
		}
		requestDone <- requestErr
	}()

	select {
	case <-requestStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("request did not reach server")
	}

	cancel()
	select {
	case err := <-serveDone:
		t.Fatalf("Serve() returned before active request drained: %v", err)
	case <-time.After(50 * time.Millisecond):
	}

	close(releaseRequest)
	if err := <-requestDone; err != nil {
		t.Fatalf("active request error = %v", err)
	}
	select {
	case err := <-serveDone:
		if err != nil {
			t.Fatalf("Serve() error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Serve() did not return after active request completed")
	}
}

type singleConnectionListener struct {
	connection net.Conn
	closed     chan struct{}
	acceptOnce sync.Once
	closeOnce  sync.Once
}

type fixedReader struct {
	remaining int
}

func (reader *fixedReader) Read(buffer []byte) (int, error) {
	if reader.remaining == 0 {
		return 0, io.EOF
	}
	count := min(len(buffer), reader.remaining)
	for index := range count {
		buffer[index] = 'x'
	}
	reader.remaining -= count
	return count, nil
}

func newSingleConnectionListener(connection net.Conn) *singleConnectionListener {
	return &singleConnectionListener{connection: connection, closed: make(chan struct{})}
}

func (listener *singleConnectionListener) Accept() (net.Conn, error) {
	var connection net.Conn
	listener.acceptOnce.Do(func() { connection = listener.connection })
	if connection != nil {
		return connection, nil
	}
	<-listener.closed
	return nil, net.ErrClosed
}

func (listener *singleConnectionListener) Close() error {
	listener.closeOnce.Do(func() { close(listener.closed) })
	return nil
}

func (listener *singleConnectionListener) Addr() net.Addr {
	return listener.connection.LocalAddr()
}

func testServer(postgresErr, redisErr error) *Server {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	return New(Config{
		Addr:              "127.0.0.1:0",
		FrontendOrigins:   []string{"http://localhost:3000"},
		ReadHeaderTimeout: time.Second,
		ReadTimeout:       time.Second,
		WriteTimeout:      2 * time.Second,
		IdleTimeout:       time.Second,
		RequestTimeout:    time.Second,
		ShutdownTimeout:   time.Second,
		MaxBodyBytes:      1024,
	}, logger, Dependencies{
		Postgres: fakeChecker(func(context.Context) error { return postgresErr }),
		Redis:    fakeChecker(func(context.Context) error { return redisErr }),
	}, observability.NoopHooks())
}

func decodeResponse(t *testing.T, response *httptest.ResponseRecorder, destination any) {
	t.Helper()
	if err := json.NewDecoder(response.Body).Decode(destination); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}
