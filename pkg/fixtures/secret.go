package fixtures

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac/testconsts"
	"github.com/stackrox/rox/pkg/uuid"
)

// GetSecret returns a mock Secret
func GetSecret() *storage.Secret {
	return GetScopedSecret("ID", "clusterid", "")
}

// GetScopedSecret returns a mock Secret belonging to the input scope
func GetScopedSecret(ID string, clusterID string, namespace string) *storage.Secret {
	return &storage.Secret{
		Id:          ID,
		Name:        "secretName",
		ClusterId:   clusterID,
		ClusterName: "clustername",
		Namespace:   namespace,
		Files: []*storage.SecretDataFile{
			{
				Name: "foo",
				Type: storage.SecretType_IMAGE_PULL_SECRET,
			},
		},
	}
}

// GetSACTestSecretSet returns a set of mock secrets that can be used for scoped access control tests
func GetSACTestSecretSet() []*storage.Secret {
	secrets := make([]*storage.Secret, 0, 18)
	secrets = append(secrets, GetScopedSecret(uuid.NewV4().String(), testconsts.Cluster1, testconsts.NamespaceA))
	secrets = append(secrets, GetScopedSecret(uuid.NewV4().String(), testconsts.Cluster1, testconsts.NamespaceA))
	secrets = append(secrets, GetScopedSecret(uuid.NewV4().String(), testconsts.Cluster1, testconsts.NamespaceA))
	secrets = append(secrets, GetScopedSecret(uuid.NewV4().String(), testconsts.Cluster1, testconsts.NamespaceA))
	secrets = append(secrets, GetScopedSecret(uuid.NewV4().String(), testconsts.Cluster1, testconsts.NamespaceA))
	secrets = append(secrets, GetScopedSecret(uuid.NewV4().String(), testconsts.Cluster1, testconsts.NamespaceA))
	secrets = append(secrets, GetScopedSecret(uuid.NewV4().String(), testconsts.Cluster1, testconsts.NamespaceA))
	secrets = append(secrets, GetScopedSecret(uuid.NewV4().String(), testconsts.Cluster1, testconsts.NamespaceA))
	secrets = append(secrets, GetScopedSecret(uuid.NewV4().String(), testconsts.Cluster1, testconsts.NamespaceB))
	secrets = append(secrets, GetScopedSecret(uuid.NewV4().String(), testconsts.Cluster1, testconsts.NamespaceB))
	secrets = append(secrets, GetScopedSecret(uuid.NewV4().String(), testconsts.Cluster1, testconsts.NamespaceB))
	secrets = append(secrets, GetScopedSecret(uuid.NewV4().String(), testconsts.Cluster1, testconsts.NamespaceB))
	secrets = append(secrets, GetScopedSecret(uuid.NewV4().String(), testconsts.Cluster1, testconsts.NamespaceB))
	secrets = append(secrets, GetScopedSecret(uuid.NewV4().String(), testconsts.Cluster2, testconsts.NamespaceB))
	secrets = append(secrets, GetScopedSecret(uuid.NewV4().String(), testconsts.Cluster2, testconsts.NamespaceB))
	secrets = append(secrets, GetScopedSecret(uuid.NewV4().String(), testconsts.Cluster2, testconsts.NamespaceB))
	secrets = append(secrets, GetScopedSecret(uuid.NewV4().String(), testconsts.Cluster2, testconsts.NamespaceC))
	secrets = append(secrets, GetScopedSecret(uuid.NewV4().String(), testconsts.Cluster2, testconsts.NamespaceC))
	return secrets
}
