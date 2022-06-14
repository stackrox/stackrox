package service

import (
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
)

// initBundleMetaStorageToV1 transforms the internal storage representation of
// init bundle metadata to the exposed data model in the v1 API.
func initBundleMetaStorageToV1(meta *storage.InitBundleMeta) *v1.InitBundleMeta {
	return initBundleMetaStorageToV1WithImpactedClusters(meta, nil)
}

func initBundleMetaStorageToV1WithImpactedClusters(meta *storage.InitBundleMeta, clusters []*v1.InitBundleMeta_ImpactedCluster) *v1.InitBundleMeta {
	return &v1.InitBundleMeta{
		Id:               meta.GetId(),
		Name:             meta.GetName(),
		CreatedAt:        meta.GetCreatedAt(),
		CreatedBy:        meta.GetCreatedBy(),
		ExpiresAt:        meta.GetExpiresAt(),
		ImpactedClusters: clusters,
	}
}
