package version

import (
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
)

func TestVersionSerialization(t *testing.T) {
	obj := &storage.Version{}
	assert.NoError(t, testutils.FullInit(obj, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
	obj.LastPersisted = protocompat.TimestampNow()
	m, err := ConvertVersionFromProto(obj)
	assert.NoError(t, err)
	conv, err := ConvertVersionToProto(m)
	assert.NoError(t, err)
	// ConvertVersionFromProto and ConvertVersionToProto rounds up ts to microseconds, so make sure obj field is also rounded up.
	obj.LastPersisted = protoutils.RoundTimestamp(obj.LastPersisted, time.Microsecond)
	protoassert.Equal(t, obj, conv)
}
