package main

import (
	"fmt"
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

func TestIsGloballyScoped(t *testing.T) {
	testCases := []struct {
		storageType       string
		permissionChecker bool
		joinTable         bool
		result            bool
	}{
		{result: false, storageType: "storage.NamespaceMetadata", permissionChecker: false, joinTable: false},
		{result: true, storageType: "storage.NamespaceMetadata", permissionChecker: true, joinTable: false},
		{result: false, storageType: "*storage.NamespaceMetadata", permissionChecker: false, joinTable: false},
		{result: false, storageType: "*storage.Policy", permissionChecker: false, joinTable: true},
		{result: true, storageType: "*storage.Policy", permissionChecker: false, joinTable: false},
		{result: true, storageType: "storage.Policy", permissionChecker: false, joinTable: false},
		{result: true, storageType: "storage.SignatureIntegration", permissionChecker: false, joinTable: false},
		{result: true, storageType: "fake", permissionChecker: true, joinTable: false},
		{result: false, storageType: "fake", permissionChecker: false, joinTable: true},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(fmt.Sprintf("%+v", tc), func(t *testing.T) {
			assert.Equal(t, tc.result, isGloballyScoped(tc.storageType, tc.permissionChecker, tc.joinTable))
		})
	}

	t.Run("panics on unknown resource", func(t *testing.T) {
		assert.Panics(t, func() { isGloballyScoped("fake", false, false) })
	})
}
