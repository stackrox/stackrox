package postgres

import (
	"context"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/uuid"
)

var (
	queryThreshold = env.PostgresQueryTracerQueryThreshold.DurationSetting()
)

type queryTracerKey struct{}

type queryEvent struct {
	query    string
	args     []interface{}
	duration time.Duration
	stack    []byte
}

type queryTracer struct {
	lock   sync.Mutex
	events []queryEvent
	id     string
}

// AddEvent adds a Postgres query to the tracer
func (qt *queryTracer) AddEvent(start time.Time, query string, args ...interface{}) {
	qt.lock.Lock()
	defer qt.lock.Unlock()

	qt.events = append(qt.events, queryEvent{
		query:    query,
		args:     args,
		duration: time.Since(start),
		stack:    debug.Stack(),
	})
}

// LogTracef is a wrapper around LogTrace that provides formatting
func LogTracef(ctx context.Context, logger *logging.Logger, contextString string, args ...interface{}) {
	LogTrace(ctx, logger, fmt.Sprintf(contextString, args...))
}

// LogTrace logs the queries seen in the current trace
func LogTrace(ctx context.Context, logger *logging.Logger, contextString string) {
	tracer := GetTracerFromContext(ctx)

	logger.Infof("trace=%s: %s", tracer.id, contextString)
	if len(tracer.events) == 0 {
		logger.Infof("trace=%s: no queries ran", tracer.id)
		return
	}
	for _, e := range tracer.events {
		if e.duration > queryThreshold {
			logger.Infof("trace=%s: took(%d ms): %s %+v", tracer.id, e.duration.Milliseconds(), e.query, e.args)
		}
	}
}

// WithTracerContext appends a query tracer to the returned context
func WithTracerContext(ctx context.Context) context.Context {
	if tracer := GetTracerFromContext(ctx); tracer != nil {
		return ctx
	}
	return context.WithValue(ctx, queryTracerKey{}, &queryTracer{
		id: uuid.NewV4().String(),
	})
}

// GetTracerFromContext returns the tracer appended to the context or nil if no tracer exists
func GetTracerFromContext(ctx context.Context) *queryTracer {
	val, ok := ctx.Value(queryTracerKey{}).(*queryTracer)
	if ok {
		return val
	}
	return nil
}

// AddTracedQuery adds a query into the tracer
func AddTracedQuery(ctx context.Context, start time.Time, sql string, args ...interface{}) {
	tracer := GetTracerFromContext(ctx)
	if tracer == nil {
		return
	}
	tracer.AddEvent(start, sql, args)
}
