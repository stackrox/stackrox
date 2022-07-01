package service

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// ToStorageProto converts the v1 representation to its corresponding storage representation
func ToStorageProto(category *v1.PolicyCategory) *storage.PolicyCategory {
	return &storage.PolicyCategory{
		Id:        category.GetId(),
		Name:      category.GetName(),
		IsDefault: category.GetIsDefault(),
	}
}

// ToV1Proto converts the storage representation to its corresponding v1 representation
func ToV1Proto(category *storage.PolicyCategory) *v1.PolicyCategory {
	return &v1.PolicyCategory{
		Id:        category.GetId(),
		Name:      category.GetName(),
		IsDefault: category.GetIsDefault(),
	}
}
