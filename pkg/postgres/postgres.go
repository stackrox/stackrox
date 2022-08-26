package postgres

import (
	"context"
	"runtime/debug"
	"time"

	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/uuid"
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

func LogTrace(logger *logging.Logger, ctx context.Context, contextString string) {
	tracer := ctx.Value(queryTracerKey{}).(*queryTracer)

	logger.Infof("%s: %s", tracer.id, contextString)
	if len(tracer.events) == 0 {
		logger.Infof("%s: no queries ran", tracer.id)
		return
	}
	for _, e := range tracer.events {
		logger.Infof("%s: took(%d ms): %s %+v", tracer.id, e.duration.Milliseconds(), e.query, e.args)
	}
}

func WithTracerContext(ctx context.Context) context.Context {
	if tracer := GetTracerFromContext(ctx); tracer != nil {
		return ctx
	}
	return context.WithValue(ctx, queryTracerKey{}, &queryTracer{
		id: uuid.NewV4().String(),
	})
}

func GetTracerFromContext(ctx context.Context) *queryTracer {
	val, ok := ctx.Value(queryTracerKey{}).(*queryTracer)
	if ok {
		return val
	}
	return nil
}

func AddTracedQuery(ctx context.Context, start time.Time, sql string, args ...interface{}) {
	tracer := GetTracerFromContext(ctx)
	if tracer == nil {
		return
	}
	tracer.AddEvent(start, sql, args)
}
