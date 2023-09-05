package csv

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/pkg/grpc/errors"
)

// Utility functions to be used for CSV exporting.

// WriteError responds with the error message and HTTP status code deduced from
// the error class. Appropriate response headers are set.
func WriteError(w http.ResponseWriter, err error) {
	WriteErrorWithCode(w, errors.ErrToHTTPStatus(err), err)
}

// WriteErrorWithCode is similar to WriteError but uses the provided HTTP status
// code. If err is a known internal error, use WriteError to ensure consistency
// of HTTP status codes.
func WriteErrorWithCode(w http.ResponseWriter, code int, err error) {
	http.Error(w, err.Error(), code)
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
