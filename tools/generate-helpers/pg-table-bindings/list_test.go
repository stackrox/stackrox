package main

import (
	"reflect"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/walker"
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

func TestClusterGetter(t *testing.T) {
	assert.Panics(t, func() { clusterGetter(walker.Walk(reflect.TypeOf(&storage.CVE{}), "")) })
	assert.Equal(t, "obj.GetClusterId()", clusterGetter(walker.Walk(reflect.TypeOf(&storage.Deployment{}), "")))
}

func TestNamespaceGetter(t *testing.T) {
	assert.Empty(t, namespaceGetter(walker.Walk(reflect.TypeOf(&storage.Email{}), "")))
	assert.Empty(t, namespaceGetter(walker.Walk(reflect.TypeOf(&storage.Cluster{}), "")))
	assert.Equal(t, "obj.GetNamespace()", namespaceGetter(walker.Walk(reflect.TypeOf(&storage.Deployment{}), "")))
}
