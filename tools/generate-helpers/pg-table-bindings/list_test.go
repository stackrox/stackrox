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
	assert.Equal(t, "Integration", storageToResource("storage.SignatureIntegration"))
	assert.Equal(t, "*fake", storageToResource("fake"))
	assert.Equal(t, "fake", storageToResource("storage.fake"))
	assert.Equal(t, "fake", storageToResource("*storage.fake"))
}

func TestClusterGetter(t *testing.T) {
	for typ, getter := range map[proto.Message]string{
		&storage.Deployment{}:      "obj.GetClusterId()",
		&storage.Cluster{}:         "obj.GetId()",
		&storage.Risk{}:            "obj.GetSubject().GetClusterId()",
		&storage.ProcessBaseline{}: "obj.GetKey().GetClusterId()",
	} {
		t.Run(fmt.Sprintf("%T -> %s", typ, getter), func(t *testing.T) {
			assert.Equal(t, getter, clusterGetter("obj", walker.Walk(reflect.TypeOf(typ), "")))
		})
	}

	t.Run("panics for not directly scoped type", func(t *testing.T) {
		assert.Panics(t, func() { clusterGetter("obj", walker.Walk(reflect.TypeOf(&storage.CVE{}), "")) })
		assert.Panics(t, func() { clusterGetter("obj", walker.Walk(reflect.TypeOf(&storage.Email{}), "")) })
	})
}

func TestNamespaceGetter(t *testing.T) {
	for typ, getter := range map[proto.Message]string{
		&storage.NamespaceMetadata{}: "obj.GetName()",
		&storage.Deployment{}:        "obj.GetNamespace()",
		&storage.ProcessBaseline{}:   "obj.GetKey().GetNamespace()",
		&storage.Risk{}:              "obj.GetSubject().GetNamespace()",
	} {
		t.Run(fmt.Sprintf("%T -> %s", typ, getter), func(t *testing.T) {
			assert.Equal(t, getter, namespaceGetter("obj", walker.Walk(reflect.TypeOf(typ), "")))
		})
	}

	t.Run("panics for not directly & ns scoped type", func(t *testing.T) {
		assert.Panics(t, func() { namespaceGetter("obj", walker.Walk(reflect.TypeOf(&storage.Email{}), "")) })
		assert.Panics(t, func() { namespaceGetter("obj", walker.Walk(reflect.TypeOf(&storage.Cluster{}), "")) })
	})
}
