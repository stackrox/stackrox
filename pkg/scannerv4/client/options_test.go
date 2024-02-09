package client

import (
	"testing"

	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stretchr/testify/assert"
)

func TestOptions(t *testing.T) {
	t.Run("When no setters then options should be default", func(t *testing.T) {
		o, err := makeOptions()
		assert.NoError(t, err)
		assert.Equal(t, defaultOptions, o)
	})

	t.Run("When non-default setters for both indexer and matcher then options should be set", func(t *testing.T) {
		// Randomly choosing scanner TLS.
		subject := mtls.ScannerSubject
		address := "localhost:9000"
		serverName := "newServer"

		o, err := makeOptions(
			WithSubject(subject),
			WithAddress(address),
			WithServerName(serverName),
		)

		assert.NoError(t, err)
		assert.Equal(t, subject, o.indexerOpts.mTLSSubject)
		assert.Equal(t, subject, o.matcherOpts.mTLSSubject)
		assert.Equal(t, address, o.indexerOpts.address)
		assert.Equal(t, address, o.matcherOpts.address)
		assert.Equal(t, serverName, o.indexerOpts.serverName)
		assert.Equal(t, serverName, o.matcherOpts.serverName)
		assert.False(t, o.matcherOpts.skipTLSVerify)
		assert.False(t, o.indexerOpts.skipTLSVerify)
		assert.True(t, o.comboMode)
	})

	t.Run("When skip TLS is set, then both indexer and matcher comply", func(t *testing.T) {
		o, err := makeOptions(SkipTLSVerification)
		assert.NoError(t, err)
		assert.True(t, o.indexerOpts.skipTLSVerify)
		assert.True(t, o.matcherOpts.skipTLSVerify)
	})

	t.Run("When different options are set for indexer and matcher, then comboMode should be false", func(t *testing.T) {
		o, err := makeOptions(
			WithIndexerAddress("localhost:9001"),
			WithMatcherAddress("localhost:9002"),
		)
		assert.NoError(t, err)
		assert.False(t, o.comboMode)
	})

	t.Run("When the same options are set for indexer and matcher, then comboMode should be true", func(t *testing.T) {
		o, err := makeOptions(
			WithSubject(mtls.ScannerV4IndexerSubject), // Doesn't matter.
			WithIndexerAddress("localhost:9001"),
			WithMatcherAddress("localhost:9001"),
			WithServerName("scanner-v4-combo"),
		)
		assert.NoError(t, err)
		assert.True(t, o.comboMode)
	})

	t.Run("When indexer address is not valid host:port, should error", func(t *testing.T) {
		_, err := makeOptions(
			WithIndexerAddress("https://localhost:9001"),
		)
		assert.Error(t, err)
	})

	t.Run("When matcher address is not valid host:port, should error", func(t *testing.T) {
		_, err := makeOptions(
			WithMatcherAddress("https://localhost:9001"),
		)
		assert.Error(t, err)
	})
}
