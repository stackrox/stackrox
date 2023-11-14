package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/central/graphql/resolvers"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres"
)

var (
	log = logging.LoggerForModule()

	queryTracerEnabled    = env.PostgresQueryTracer.BooleanSetting()
	graphQLQueryThreshold = env.PostgresQueryTracerGraphQLThreshold.DurationSetting()
)

type logger struct {
}

func (*logger) LogPanic(ctx context.Context, value interface{}) {
	const size = 64 << 10
	buf := make([]byte, size)
	buf = buf[:runtime.Stack(buf, false)]
	log.Errorf("graphql: panic occurred: query was %+v; %v\n%s", ctx.Value(paramsContextKey{}), value, buf)
}

type relayHandler struct {
	Schema *graphql.Schema
}

type params struct {
	Query         string                 `json:"query"`
	OperationName string                 `json:"operationName"`
	Variables     map[string]interface{} `json:"variables"`
}

type paramsContextKey struct{}

// Copied from github.com/graph-gophers/graphql-go/relay/relay.go, but with minor modifications
// so we have the request in the context and can inject our custom logger.
func (h *relayHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var params params
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Adds the params for the framework.
	ctx := context.WithValue(r.Context(), paramsContextKey{}, params)

	if !buildinfo.ReleaseBuild {
		if validationErrs := h.Schema.ValidateWithVariables(params.Query, params.Variables); len(validationErrs) > 0 {
			log.Errorf("UNEXPECTED: GraphQL operation %s: received a query failing schema validation: %v", params.OperationName, validationErrs)
			log.Errorf("Full query:\n%s\nVariables:\n%+v", params.Query, params.Variables)
		}
	}

	// Adds the data loader intermediates so that we can stop ourselves from loading the same data from the store
	// many time.
	ctx = loaders.WithLoaderContext(ctx)
	if queryTracerEnabled {
		ctx = postgres.WithTracerContext(ctx)
	}

	startTime := time.Now()
	defer metrics.SetGraphQLQueryDurationTime(startTime, params.Query)

	response := h.Schema.Exec(ctx, params.Query, params.OperationName, params.Variables)
	responseJSON, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if queryTracerEnabled && time.Since(startTime) > graphQLQueryThreshold {
		singleLineQuery := strings.ReplaceAll(params.Query, "\n", " ")
		postgres.LogTracef(ctx, log, "GraphQL Op %s took %d ms: %s vars=%+v", params.OperationName, time.Since(startTime).Milliseconds(), singleLineQuery, params.Variables)
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(responseJSON)
}

// Handler returns an HTTP handler for the graphql api endpoint
func Handler() http.Handler {
	opts := []graphql.SchemaOpt{graphql.Logger(&logger{})}
	s := resolvers.Schema()
	ourSchema, err := graphql.ParseSchema(s, resolvers.New(), opts...)
	if err != nil {
		log.Errorf("Unable to parse schema:\n%q", s)
		panic(err)
	}
	return &relayHandler{Schema: ourSchema}
}
