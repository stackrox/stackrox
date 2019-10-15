package grpc

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMaxResponseMsgSize_Unset(t *testing.T) {
	require.NoError(t, os.Unsetenv(maxResponseMsgSizeSetting.EnvVar()))

	assert.Equal(t, defaultMaxResponseMsgSize, maxResponseMsgSize())
}

func TestMaxResponseMsgSize_Empty(t *testing.T) {
	require.NoError(t, os.Setenv(maxResponseMsgSizeSetting.EnvVar(), ""))

	assert.Equal(t, defaultMaxResponseMsgSize, maxResponseMsgSize())
}

func TestMaxResponseMsgSize_Invalid(t *testing.T) {
	require.NoError(t, os.Setenv(maxResponseMsgSizeSetting.EnvVar(), "notAnInt"))

	assert.Equal(t, defaultMaxResponseMsgSize, maxResponseMsgSize())
}

func TestMaxResponseMsgSize_Valid(t *testing.T) {
	require.NoError(t, os.Setenv(maxResponseMsgSizeSetting.EnvVar(), "1337"))

	assert.Equal(t, 1337, maxResponseMsgSize())
}
