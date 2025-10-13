package v1tostorage

import (
	"testing"

	convertTestUtils "github.com/stackrox/rox/central/convert/testutils"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthM2MConfig(t *testing.T) {
	config := &v1.AuthMachineToMachineConfig{}
	require.NoError(t, testutils.FullInit(config, testutils.SimpleInitializer(), testutils.JSONFieldsFilter))

	storageConfig := AuthM2MConfig(config)

	convertTestUtils.AssertProtoMessageEqual(t, config, storageConfig)
	assert.IsType(t, &storage.AuthMachineToMachineConfig{}, storageConfig)
}
