package postgres

// WhereInterceptor is a callback that can modify the WHERE clause and
// parameter values of a compiled query before execution.
type WhereInterceptor func(where string, values []any) (string, []any)

// SelectRequestOption configures RunSelectRequestForSchemaFn and RunSelectOneForSchema.
type SelectRequestOption func(*selectRequestConfig)

type selectRequestConfig struct {
	whereInterceptor WhereInterceptor
}

// WithWhereInterceptor returns an option that injects a WHERE-clause
// interceptor into the select execution pipeline.
func WithWhereInterceptor(fn WhereInterceptor) SelectRequestOption {
	return func(cfg *selectRequestConfig) {
		cfg.whereInterceptor = fn
	}
}
