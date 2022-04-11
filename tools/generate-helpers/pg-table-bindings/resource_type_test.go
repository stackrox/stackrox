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
		{typ: &storage.NamespaceMetadata{}, resourceType: DirectlyScoped},
		{typ: &storage.NamespaceMetadata{}, resourceType: JoinTable, joinTable: true},
		{typ: &storage.NamespaceMetadata{}, resourceType: JoinTable, joinTable: true, permissionChecker: true},
		{typ: &storage.NamespaceMetadata{}, resourceType: PermissionChecker, joinTable: false, permissionChecker: true},
		{typ: &storage.Cluster{}, resourceType: DirectlyScoped},
		{typ: &storage.Deployment{}, resourceType: DirectlyScoped},
		{typ: &storage.Image{}, resourceType: IndirectlyScoped},
		{typ: &storage.CVE{}, resourceType: IndirectlyScoped},
		{typ: &storage.Policy{}, resourceType: GloballyScoped},
		{typ: &storage.Email{}, resourceType: JoinTable, joinTable: true},
		{typ: &storage.Email{}, resourceType: PermissionChecker, permissionChecker: true},
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
