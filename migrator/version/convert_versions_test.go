package version

import (
	"testing"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stretchr/testify/assert"
)

func TestVersionSerialization(t *testing.T) {
	obj := &storage.Version{}
	assert.NoError(t, testutils.FullInit(obj, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
	obj.LastPersisted = types.TimestampNow()
	m, err := ConvertVersionFromProto(obj)
	assert.NoError(t, err)
	conv, err := ConvertVersionToProto(m)
	assert.NoError(t, err)
	// ConvertVersionFromProto and ConvertVersionToProto rounds up ts to microseconds, so make sure obj field is also rounded up.
	timestamp.RoundTimestamp(obj.LastPersisted, time.Microsecond)
	assert.Equal(t, obj, conv)
}
