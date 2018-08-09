package readable

import (
	"fmt"
	"time"

	ptypes "github.com/gogo/protobuf/types"
	timestamp "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// ProtoTime takes a proto time type and converts it to a human readable string down to seconds
func ProtoTime(ts *timestamp.Timestamp) string {
	t, err := ptypes.TimestampFromProto(ts)
	if err != nil {
		log.Error(err)
		return "<malformed time>"
	}
	return Time(t)
}

// Time takes a golang time type and converts it to a human readable string down to seconds
func Time(t time.Time) string {
	return fmt.Sprintf("%d-%02d-%02d %02d:%02d:%02d",
		t.Year(), t.Month(), t.Day(),
		t.Hour(), t.Minute(), t.Second())
}
