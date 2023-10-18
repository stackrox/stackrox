package v1tostorage

import (
	"bytes"
	"testing"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
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

	assertEqual(t, config, storageConfig)
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
