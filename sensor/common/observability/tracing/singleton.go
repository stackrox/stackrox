package tracing

import (
	"github.com/stackrox/rox/pkg/observability/tracing"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once    sync.Once
	handler tracing.TracerHandler
)

func initialize() {
	handler = tracing.NewHandler()
}

// Singleton returns the tracer handler instance.
func Singleton() tracing.TracerHandler {
	once.Do(initialize)
	return handler
}
