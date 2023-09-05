package csv

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/pkg/grpc/errors"
)

const (
	// utf8BOM is the UTF-8 BOM byte sequence.
	utf8BOM = "\uFEFF"
)

// Utility functions to be used for CSV exporting.

// WriteError responds with the error message and HTTP status code deduced from
// the error class. Appropriate response headers are set. Note that once data
// have been written to ResponseWriter, depending on its implementation, headers
// and the status code might have already been sent over HTTP. In such case,
// calling WriteError will not have the desired effect.
func WriteError(w http.ResponseWriter, err error) {
	http.Error(w, err.Error(), errors.ErrToHTTPStatus(err))
}

// writeHeaders sets appropriate HTTP headers for CSV response.
func writeHeaders(w http.ResponseWriter, filename string) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.WriteHeader(http.StatusOK)
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
