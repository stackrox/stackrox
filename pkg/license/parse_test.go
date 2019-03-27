package license

import (
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseLicenseKey_Valid(t *testing.T) {
	t.Parallel()

	b1 := []byte{1, 3, 3, 7}
	b2 := []byte{2, 4}

	key := "AQMDBw.AgQ"

	out1, out2, err := ParseLicenseKey(key)
	assert.NoError(t, err)
	assert.Equal(t, b1, out1)
	assert.Equal(t, b2, out2)
}

func TestParseLicenseKey_InvalidLicenseB64(t *testing.T) {
	t.Parallel()

	key := "AQM?DBw.AgQ"

	_, _, err := ParseLicenseKey(key)
	assert.Error(t, err)
}

func TestParseLicenseKey_MissingLicenseB64(t *testing.T) {
	t.Parallel()

	key := ".AgQ"

	_, _, err := ParseLicenseKey(key)
	assert.Error(t, err)
}

func TestParseLicenseKey_InvalidSigB64(t *testing.T) {
	t.Parallel()

	key := "AQMDBw.Ag?Q"

	_, _, err := ParseLicenseKey(key)
	assert.Error(t, err)
}

func TestParseLicenseKey_MissingSigB64(t *testing.T) {
	t.Parallel()

	key := "AQMDBw."

	_, _, err := ParseLicenseKey(key)
	assert.Error(t, err)
}

func TestParseLicenseKey_NoDot(t *testing.T) {
	t.Parallel()

	key := "AQMDBw"

	_, _, err := ParseLicenseKey(key)
	assert.Error(t, err)
}

func TestParseLicenseKey_TooManyDots(t *testing.T) {
	t.Parallel()

	key := "AQMDBw.AgQ.AQMDBw"

	_, _, err := ParseLicenseKey(key)
	assert.Error(t, err)
}

func TestUnmarshalLicense_Valid(t *testing.T) {
	t.Parallel()

	lic := &v1.License{
		Metadata: &v1.License_Metadata{
			Id:              uuid.NewV4().String(),
			SigningKeyId:    "test/key/1",
			IssueDate:       types.TimestampNow(),
			LicensedForId:   "test",
			LicensedForName: "Test",
		},
		Restrictions: &v1.License_Restrictions{
			NotValidBefore: types.TimestampNow(),
			NotValidAfter:  types.TimestampNow(),
		},
	}

	licMarshalled, err := proto.Marshal(lic)
	require.NoError(t, err)

	licUnmarshalled, err := UnmarshalLicense(licMarshalled)
	require.NoError(t, err)
	assert.Equal(t, lic, licUnmarshalled)
}

func TestUnmarshalLicense_ExtraBytes(t *testing.T) {
	t.Parallel()

	lic := &v1.License{
		Metadata: &v1.License_Metadata{
			Id:              uuid.NewV4().String(),
			SigningKeyId:    "test/key/1",
			IssueDate:       types.TimestampNow(),
			LicensedForId:   "test",
			LicensedForName: "Test",
		},
		Restrictions: &v1.License_Restrictions{
			NotValidBefore:   types.TimestampNow(),
			NotValidAfter:    types.TimestampNow(),
			XXX_unrecognized: []byte("some extra data"),
		},
	}

	licMarshalled, err := proto.Marshal(lic)
	require.NoError(t, err)

	_, err = UnmarshalLicense(licMarshalled)
	assert.Error(t, err)
}
