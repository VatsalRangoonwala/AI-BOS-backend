package observability

import (
	"context"
	"time"
)

type Metrics interface {
	ObserveHTTPRequest(ctx context.Context, method, route string, status int, duration time.Duration)
	ObserveDependencyCheck(ctx context.Context, dependency string, healthy bool, duration time.Duration)
}

type Tracer interface {
	Start(ctx context.Context, name string) (context.Context, Span)
}

type Span interface {
	RecordError(err error)
	End()
}

type Hooks struct {
	Metrics Metrics
	Tracer  Tracer
}

func NoopHooks() Hooks {
	return Hooks{
		Metrics: NoopMetrics{},
		Tracer:  NoopTracer{},
	}
}

type NoopMetrics struct{}

func (NoopMetrics) ObserveHTTPRequest(context.Context, string, string, int, time.Duration) {}

func (NoopMetrics) ObserveDependencyCheck(context.Context, string, bool, time.Duration) {}

type NoopTracer struct{}

func (NoopTracer) Start(ctx context.Context, _ string) (context.Context, Span) {
	return ctx, NoopSpan{}
}

type NoopSpan struct{}

func (NoopSpan) RecordError(error) {}

func (NoopSpan) End() {}
