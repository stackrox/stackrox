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
	defaultTimeout = env.PostgresDefaultStatementTimeout.DurationSetting()
)

type queryTracerKey struct{}

// QueryEvent defines an event that tracks a Postgres query
type QueryEvent struct {
	query        string
	rowsAccessed *int
	args         []interface{}
	duration     time.Duration
	stack        []byte
}

// SetRowsAccessed sets the number of rows fetched from Postgres
func (qe *QueryEvent) SetRowsAccessed(n int) {
	if qe != nil {
		qe.rowsAccessed = &n
	}
}

type queryTracer struct {
	lock   sync.Mutex
	events []*QueryEvent
	id     string
}

// AddEvent adds a Postgres query to the tracer
func (qt *queryTracer) AddEvent(start time.Time, query string, args ...interface{}) *QueryEvent {
	qt.lock.Lock()
	defer qt.lock.Unlock()

	event := &QueryEvent{
		query:    query,
		args:     args,
		duration: time.Since(start),
		stack:    debug.Stack(),
	}
	qt.events = append(qt.events, event)
	return event
}

// LogTracef is a wrapper around LogTrace that provides formatting
func LogTracef(ctx context.Context, logger logging.Logger, contextString string, args ...interface{}) {
	LogTrace(ctx, logger, fmt.Sprintf(contextString, args...))
}

// LogTrace logs the queries seen in the current trace
func LogTrace(ctx context.Context, logger logging.Logger, contextString string) {
	tracer := GetTracerFromContext(ctx)

	logger.Infof("trace=%s: %s", tracer.id, contextString)
	if len(tracer.events) == 0 {
		logger.Infof("trace=%s: no queries ran", tracer.id)
		return
	}
	for _, e := range tracer.events {
		rowsAccessed := 1
		if e.rowsAccessed != nil {
			rowsAccessed = *e.rowsAccessed
		}
		if e.duration > queryThreshold {
			logger.Infof("trace=%s: returned %d rows and took(%d ms): %s %+v", tracer.id, rowsAccessed, e.duration.Milliseconds(), e.query, e.args)
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
func AddTracedQuery(ctx context.Context, start time.Time, sql string, args ...interface{}) *QueryEvent {
	tracer := GetTracerFromContext(ctx)
	if tracer == nil {
		return nil
	}
	return tracer.AddEvent(start, sql, args)
}
