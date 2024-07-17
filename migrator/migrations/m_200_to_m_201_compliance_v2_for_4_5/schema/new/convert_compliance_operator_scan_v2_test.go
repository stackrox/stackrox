package new

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
)

func TestComplianceOperatorScanV2Serialization(t *testing.T) {
	obj := &storage.ComplianceOperatorScanV2{}
	assert.NoError(t, testutils.FullInit(obj, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
	m, err := ConvertComplianceOperatorScanV2FromProto(obj)
	assert.NoError(t, err)
	conv, err := ConvertComplianceOperatorScanV2ToProto(m)
	assert.NoError(t, err)
	protoassert.Equal(t, obj, conv)
}
