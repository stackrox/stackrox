package schemas

import (
	"github.com/stackrox/helmtest/internal/rox-imported/set"

	"helm.sh/helm/v3/pkg/chartutil"
	k8sSchema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/kube-openapi/pkg/util/proto"
)

// Schemas is a list of schemas to be combined.
type Schemas []Schema

// LookupResource locates a given GVK in the schema.
func (s Schemas) LookupResource(gvk k8sSchema.GroupVersionKind) proto.Schema {
	for _, subSchema := range s {
		if protoSchema := subSchema.LookupResource(gvk); protoSchema != nil {
			return protoSchema
		}
	}
	return nil
}

// VersionSet returns the set of all API versions (Group, Group/Version, Group/Version/Kind) supported by the schemas.
func (s Schemas) VersionSet() chartutil.VersionSet {
	allVersions := set.NewStringSet()
	for _, subSchema := range s {
		allVersions.AddAll(subSchema.VersionSet()...)
	}
	return allVersions.AsSortedSlice(func(a, b string) bool { return a < b })
}
