package fixtures

import (
	"github.com/stackrox/rox/generated/storage"
)

// GetSecret returns a mock Secret
func GetSecret() *storage.Secret {
	return &storage.Secret{
		Id:          "ID",
		ClusterId:   "clusterid",
		ClusterName: "clustername",
		Files: []*storage.SecretDataFile{
			{
				Name: "foo",
				Type: storage.SecretType_IMAGE_PULL_SECRET,
			},
		},
	}
}
