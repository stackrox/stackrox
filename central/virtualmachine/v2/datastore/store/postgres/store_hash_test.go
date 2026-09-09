package postgres

import (
	"testing"

	"github.com/stackrox/hashstructure"
	"github.com/stackrox/rox/central/virtualmachine/v2/datastore/store/common"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildScanHash_IncludesConversionVersion(t *testing.T) {
	parts := common.VMScanParts{
		Scan: &storage.VirtualMachineScanV2{ScanOs: "rhel9"},
		SourceComponents: []*storage.EmbeddedVirtualMachineScanComponent{
			{
				Name:    "curl",
				Version: "7.0",
				Notes: []storage.EmbeddedVirtualMachineScanComponent_Note{
					storage.EmbeddedVirtualMachineScanComponent_UNSCANNED,
				},
			},
		},
	}

	got, err := buildScanHash(parts)
	require.NoError(t, err)

	unversioned, err := hashstructure.Hash(scanHashWrapper{
		ScanOs:     parts.Scan.GetScanOs(),
		Components: parts.SourceComponents,
	}, &hashstructure.HashOptions{ZeroNil: true})
	require.NoError(t, err)

	assert.NotEqual(t, unversioned, got)
}
