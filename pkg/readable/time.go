package readable

import (
	"time"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

const (
	// ISO-8601 format.
	layout = "2006-01-02 15:04:05"
)

// ProtoTime takes a proto time type and converts it to a human readable string down to seconds.
// It always prints a UTC time.
func ProtoTime(ts *ptypes.Timestamp) string {
	t, err := ptypes.TimestampFromProto(ts)
	if err != nil {
		log.Error(err)
		return "<malformed time>"
	}
	return Time(t)
}

// Time takes a golang time type and converts it to a human readable string down to seconds
// It always print the UTC time.
func Time(t time.Time) string {
	return t.UTC().Format(layout)
}
