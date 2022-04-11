package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStorageToResource(t *testing.T) {
	assert.Equal(t, "Namespace", storageToResource("storage.NamespaceMetadata"))
	assert.Equal(t, "Namespace", storageToResource("*storage.NamespaceMetadata"))
	assert.Equal(t, "Cluster", storageToResource("*storage.Cluster"))
	assert.Equal(t, "Cluster", storageToResource("storage.Cluster"))
	assert.Equal(t, "SignatureIntegration", storageToResource("storage.SignatureIntegration"))
	assert.Equal(t, "*fake", storageToResource("fake"))
	assert.Equal(t, "fake", storageToResource("storage.fake"))
	assert.Equal(t, "fake", storageToResource("*storage.fake"))
}
