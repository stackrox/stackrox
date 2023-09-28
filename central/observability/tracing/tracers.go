package tracing

import (
	"github.com/exaring/otelpgx"
	"github.com/uptrace/opentelemetry-go-extra/otelgraphql"
)

func GraphQLTracer() *otelgraphql.Tracer {
	return otelgraphql.NewTracer()
}

func PostgresTracer() *otelpgx.Tracer {
	return otelpgx.NewTracer()
}
