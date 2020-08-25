package contextutil

import (
	"context"
)

type contextWithValuesFrom struct {
	context.Context
	valueFunc func(interface{}) interface{}
}

func (c contextWithValuesFrom) Value(key interface{}) interface{} {
	return c.valueFunc(key)
}

// WithValuesFrom returns a context that takes its deadline, errors and cancellation signal
// from ctx, but its values from valueProvider.
func WithValuesFrom(ctx, valueProvider context.Context) context.Context {
	return contextWithValuesFrom{
		Context:   ctx,
		valueFunc: valueProvider.Value,
	}
}
