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
}

func TestIsGloballyScoped(t *testing.T) {
	assert.False(t, isGloballyScoped("storage.NamespaceMetadata"))
	assert.False(t, isGloballyScoped("*storage.NamespaceMetadata"))
	assert.True(t, isGloballyScoped("*storage.Policy"))
	assert.True(t, isGloballyScoped("storage.Policy"))
}
