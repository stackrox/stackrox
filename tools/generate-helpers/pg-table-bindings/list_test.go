package main

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stretchr/testify/assert"
)

func TestStorageToResource(t *testing.T) {
	assert.Equal(t, "Namespace", storageToResource("storage.NamespaceMetadata"))
	assert.Equal(t, "Namespace", storageToResource("*storage.NamespaceMetadata"))
	assert.Equal(t, "Cluster", storageToResource("*storage.Cluster"))
	assert.Equal(t, "Cluster", storageToResource("storage.Cluster"))
	assert.Equal(t, "*fake", storageToResource("fake"))
	assert.Equal(t, "fake", storageToResource("storage.fake"))
	assert.Equal(t, "fake", storageToResource("*storage.fake"))
}

func TestIsGloballyScoped(t *testing.T) {
	assert.False(t, isGloballyScoped("storage.NamespaceMetadata"))
	assert.False(t, isGloballyScoped("*storage.NamespaceMetadata"))
	assert.True(t, isGloballyScoped("*storage.Policy"))
	assert.True(t, isGloballyScoped("storage.Policy"))
	assert.True(t, isGloballyScoped("fake"))
}

func TestIsDirectlyScoped(t *testing.T) {
	for typ, directlyScoped := range map[proto.Message]bool{
		&storage.NamespaceMetadata{}:    true,
		&storage.Cluster{}:              true,
		&storage.Deployment{}:           true,
		&storage.Image{}:                false,
		&storage.CVE{}:                  false,
		&storage.Policy{}:               true,
		&storage.VulnerabilityRequest{}: true, // no resource metadata
		&storage.Email{}:                true, // no resource metadata
	} {
		t.Run(fmt.Sprintf("%T directly scoped: %t", typ, directlyScoped), func(t *testing.T) {
			assert.Equal(t, directlyScoped, isDirectlyScoped(walker.Walk(reflect.TypeOf(typ), "")))
		})
	}
}
