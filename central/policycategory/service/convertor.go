package service

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// ToStorageProto converts the v1 representation to its corresponding storage representation
func ToStorageProto(category *v1.PolicyCategory) *storage.PolicyCategory {
	pc := &storage.PolicyCategory{}
	pc.SetId(category.GetId())
	pc.SetName(category.GetName())
	pc.SetIsDefault(category.GetIsDefault())
	return pc
}

// ToV1Proto converts the storage representation to its corresponding v1 representation
func ToV1Proto(category *storage.PolicyCategory) *v1.PolicyCategory {
	pc := &v1.PolicyCategory{}
	pc.SetId(category.GetId())
	pc.SetName(category.GetName())
	pc.SetIsDefault(category.GetIsDefault())
	return pc
}
