package storagetov1

import (
	"bytes"
	"testing"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthM2MConfig(t *testing.T) {
	config := &storage.AuthMachineToMachineConfig{}
	require.NoError(t, testutils.FullInit(config, testutils.SimpleInitializer(), testutils.JSONFieldsFilter))

	// Currently required as issuer won't be filled by FullInit.
	config.IssuerConfig = &storage.AuthMachineToMachineConfig_GenericIssuerConfig{
		GenericIssuerConfig: &storage.AuthMachineToMachineConfig_GenericIssuer{
			Issuer: "https://stackrox.io",
		},
	}
	v1Config := AuthM2MConfig(config)

	assertEqual(t, config, v1Config)
}

func assertEqual(t *testing.T, a, b proto.Message) {
	m := jsonpb.Marshaler{}

	jsonA := &bytes.Buffer{}
	jsonB := &bytes.Buffer{}

	require.NoError(t, m.Marshal(jsonA, a))
	require.NoError(t, m.Marshal(jsonB, b))

	// Use string for improved readability in case of test failures.
	assert.Equal(t, jsonA.String(), jsonB.String())
}
