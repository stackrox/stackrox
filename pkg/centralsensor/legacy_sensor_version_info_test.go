package centralsensor

import (
	"context"
	"reflect"
	"testing"

	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	"github.com/stackrox/rox/pkg/version"
	"github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVersionInfoClientToServer(t *testing.T) {
	const testVersion = "2.5.27.0"
	testutils.SetMainVersion(t, testVersion)
	ctx, err := appendSensorVersionInfoToContext(context.Background(), version.GetMainVersion())
	require.NoError(t, err)
	derived, err := deriveSensorVersionInfo(metautils.ExtractOutgoing(ctx))
	require.NoError(t, err)
	require.NotNil(t, derived)
	assert.Equal(t, testVersion, derived.MainVersion)
}

func TestVersionInfoOldSensors(t *testing.T) {
	derived, err := deriveSensorVersionInfo(metautils.NiceMD{})
	assert.NoError(t, err)
	assert.Nil(t, derived)
}

// This unit test helps enforce that nobody accidentally removes a field from
// the sensorVersionInfo object. This does not (and cannot) protect against someone
// intentionally doing so; it's intended mainly as a helpful reminder.
func TestVersionInfoHasAllOldFields(t *testing.T) {
	type fieldNameWithTag struct {
		fieldName string
		jsonTag   string
	}

	allKnownFields := []fieldNameWithTag{
		{"MainVersion", "mainVersion"},
	}

	var seenFields []fieldNameWithTag

	versionInfoType := reflect.TypeOf(sensorVersionInfo{})
	for i := 0; i < versionInfoType.NumField(); i++ {
		field := versionInfoType.Field(i)
		seenFields = append(seenFields, fieldNameWithTag{field.Name, field.Tag.Get("json")})
	}

	assert.ElementsMatch(t, allKnownFields, seenFields, "We never want to remove old fields from sensorVersionInfo "+
		"so please don't do that. Also, if you added a new field, pls add that to allKnownFields above so that this test "+
		"ensures that nobody removes it later")
}
