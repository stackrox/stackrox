package csv

import (
	"fmt"
	"net/http"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/pkg/transitional/protocompat/types"
)

// Utility functions to be used for CSV exporting.

// WriteError Writes the given error to the given http.ResponseWriter.
func WriteError(w http.ResponseWriter, code int, err error) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(code)
	_, _ = fmt.Fprint(w, err)
}

// FromTimestamp creates a string representation of the given timestamp.
func FromTimestamp(timestamp *types.Timestamp) string {
	if timestamp == nil {
		return "N/A"
	}
	ts, err := types.TimestampFromProto(timestamp)
	if err != nil {
		return "ERR"
	}
	return ts.Format(time.RFC1123)
}

// FromGraphQLTime create a string representation of the given graphQL.Time.
func FromGraphQLTime(timestamp *graphql.Time) string {
	if timestamp == nil {
		return "-"
	}
	return timestamp.Time.Format(time.RFC1123)
}
