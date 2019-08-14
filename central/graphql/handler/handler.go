package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"runtime"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/central/graphql/resolvers"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/pkg/logging"
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

	ctx := context.WithValue(r.Context(), paramsContextKey{}, params)

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
