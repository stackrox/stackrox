package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"runtime"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/stackrox/central/graphql/resolvers"
	"github.com/stackrox/stackrox/central/graphql/resolvers/loaders"
	"github.com/stackrox/stackrox/central/metrics"
	"github.com/stackrox/stackrox/pkg/buildinfo"
	"github.com/stackrox/stackrox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
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
			log.Errorf("Full query:\n%s\nVariables:\n%+v\n", params.Query, params.Variables)
		}
	}

	// Adds the data loader intermediates so that we can stop ourselves from loading the same data from the store
	// many time.
	ctx = loaders.WithLoaderContext(ctx)

	defer metrics.SetGraphQLQueryDurationTime(time.Now(), params.Query)

	response := h.Schema.Exec(ctx, params.Query, params.OperationName, params.Variables)
	responseJSON, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
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
