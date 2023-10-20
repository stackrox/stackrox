package v1tostorage

import (
	"testing"

	convertTestUtils "github.com/stackrox/rox/central/convert/testutils"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/require"
)

func TestAuthM2MConfig(t *testing.T) {
	config := &v1.AuthMachineToMachineConfig{}
	require.NoError(t, testutils.FullInit(config, testutils.SimpleInitializer(), testutils.JSONFieldsFilter))

	// Currently required as issuer won't be filled by FullInit.
	config.IssuerConfig = &v1.AuthMachineToMachineConfig_GenericIssuerConfig{
		GenericIssuerConfig: &v1.AuthMachineToMachineConfig_GenericIssuer{
			Issuer: "https://stackrox.io",
		},
	}
	storageConfig := AuthM2MConfig(config)

	convertTestUtils.AssertProtoMessageEqual(t, config, storageConfig)
}
