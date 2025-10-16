package fixtures

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/sac/testconsts"
	"github.com/stackrox/rox/pkg/uuid"
)

// GetSecret returns a mock Secret
func GetSecret() *storage.Secret {
	return GetScopedSecret(uuid.NewDummy().String(), fixtureconsts.Cluster1, "")
}

// GetScopedSecret returns a mock Secret belonging to the input scope
func GetScopedSecret(ID string, clusterID string, namespace string) *storage.Secret {
	sdf := &storage.SecretDataFile{}
	sdf.SetName("foo")
	sdf.SetType(storage.SecretType_IMAGE_PULL_SECRET)
	secret := &storage.Secret{}
	secret.SetId(ID)
	secret.SetName("secretName")
	secret.SetClusterId(clusterID)
	secret.SetClusterName("clustername")
	secret.SetNamespace(namespace)
	secret.SetFiles([]*storage.SecretDataFile{
		sdf,
	})
	return secret
}

// GetSACTestSecretSet returns a set of mock secrets that can be used for scoped access control tests
func GetSACTestSecretSet() []*storage.Secret {
	secrets := []*storage.Secret{
		scopedSecret(testconsts.Cluster1, testconsts.NamespaceA),
		scopedSecret(testconsts.Cluster1, testconsts.NamespaceA),
		scopedSecret(testconsts.Cluster1, testconsts.NamespaceA),
		scopedSecret(testconsts.Cluster1, testconsts.NamespaceA),
		scopedSecret(testconsts.Cluster1, testconsts.NamespaceA),
		scopedSecret(testconsts.Cluster1, testconsts.NamespaceA),
		scopedSecret(testconsts.Cluster1, testconsts.NamespaceA),
		scopedSecret(testconsts.Cluster1, testconsts.NamespaceA),
		scopedSecret(testconsts.Cluster1, testconsts.NamespaceB),
		scopedSecret(testconsts.Cluster1, testconsts.NamespaceB),
		scopedSecret(testconsts.Cluster1, testconsts.NamespaceB),
		scopedSecret(testconsts.Cluster1, testconsts.NamespaceB),
		scopedSecret(testconsts.Cluster1, testconsts.NamespaceB),
		scopedSecret(testconsts.Cluster2, testconsts.NamespaceB),
		scopedSecret(testconsts.Cluster2, testconsts.NamespaceB),
		scopedSecret(testconsts.Cluster2, testconsts.NamespaceB),
		scopedSecret(testconsts.Cluster2, testconsts.NamespaceC),
		scopedSecret(testconsts.Cluster2, testconsts.NamespaceC),
	}
	return secrets
}

func scopedSecret(clusterID string, namespace string) *storage.Secret {
	return GetScopedSecret(uuid.NewV4().String(), clusterID, namespace)
}
