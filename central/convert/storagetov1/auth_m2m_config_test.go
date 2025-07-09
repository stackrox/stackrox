package storagetov1

import (
	"testing"

	convertTestUtils "github.com/stackrox/rox/central/convert/testutils"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/require"
)

func TestAuthM2MConfig(t *testing.T) {
	config := &storage.AuthMachineToMachineConfig{}
	require.NoError(t, testutils.FullInit(config, testutils.SimpleInitializer(), testutils.JSONFieldsFilter))

	v1Config := AuthM2MConfig(config)
	expectedV1Config := config.CloneVT()
	expectedV1Config.Traits = nil

	convertTestUtils.AssertProtoMessageEqual(t, expectedV1Config, v1Config)
}
