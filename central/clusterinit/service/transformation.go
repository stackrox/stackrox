package service

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// InitBundleMetaStorageToV1 transforms the internal storage representation of
// init bundle metadata to the exposed data model in the v1 API.
func InitBundleMetaStorageToV1(meta *storage.InitBundleMeta) *v1.InitBundleMeta {
	return &v1.InitBundleMeta{
		Id:        meta.GetId(),
		Name:      meta.GetName(),
		CreatedAt: meta.GetCreatedAt(),
		CreatedBy: meta.GetCreatedBy(),
		ExpiresAt: meta.GetExpiresAt(),
	}
}
