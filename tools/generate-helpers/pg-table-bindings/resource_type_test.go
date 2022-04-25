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

func TestGetResourceType(t *testing.T) {
	for _, tc := range []struct {
		resourceType      ResourceType
		typ               proto.Message
		permissionChecker bool
		joinTable         bool
	}{
		{typ: &storage.NamespaceMetadata{}, resourceType: directlyScoped},
		{typ: &storage.NamespaceMetadata{}, resourceType: joinTable, joinTable: true},
		{typ: &storage.NamespaceMetadata{}, resourceType: joinTable, joinTable: true, permissionChecker: true},
		{typ: &storage.NamespaceMetadata{}, resourceType: permissionChecker, joinTable: false, permissionChecker: true},
		{typ: &storage.Cluster{}, resourceType: directlyScoped},
		{typ: &storage.Deployment{}, resourceType: directlyScoped},
		{typ: &storage.Image{}, resourceType: indirectlyScoped},
		{typ: &storage.CVE{}, resourceType: indirectlyScoped},
		{typ: &storage.Policy{}, resourceType: globallyScoped},
		{typ: &storage.Email{}, resourceType: joinTable, joinTable: true},
		{typ: &storage.Email{}, resourceType: permissionChecker, permissionChecker: true},
	} {
		tc := tc
		t.Run(fmt.Sprintf("%T (join: %t, perm: %t) -> %s", tc.typ, tc.joinTable, tc.permissionChecker, tc.resourceType), func(t *testing.T) {
			actual := getResourceType(
				fmt.Sprintf("%T", tc.typ),
				walker.Walk(reflect.TypeOf(tc.typ), ""),
				tc.permissionChecker,
				tc.joinTable,
			)
			assert.Equal(t, tc.resourceType.String(), actual.String())
		})
	}

	t.Run("panics on unknown resource", func(t *testing.T) {
		email := &storage.Email{}
		assert.Panics(t, func() {
			getResourceType(fmt.Sprintf("%T", email), walker.Walk(reflect.TypeOf(email), ""), false, false)
		})
	})
}
