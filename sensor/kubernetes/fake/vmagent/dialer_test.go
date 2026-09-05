package vmagent_test

import (
	"context"
	"io"
	"testing"

	"github.com/stackrox/rox/sensor/kubernetes/fake/vmagent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDialer_Dial_ReturnsCloseableStream(t *testing.T) {
	t.Parallel()

	stream, err := vmagent.NewDialer().Dial(t.Context(), "default", "vm-1", 1, true)
	require.NoError(t, err)
	require.NotNil(t, stream)

	n, err := stream.Write([]byte("ignored"))
	assert.NoError(t, err)
	assert.Equal(t, len("ignored"), n)

	_, err = stream.Read(make([]byte, 1))
	assert.ErrorIs(t, err, io.EOF)

	require.NoError(t, stream.Close())
	require.NoError(t, stream.Close())
}

func TestDialer_Dial_CancelledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := vmagent.NewDialer().Dial(ctx, "default", "vm-1", 1, true)
	assert.ErrorIs(t, err, context.Canceled)
}
