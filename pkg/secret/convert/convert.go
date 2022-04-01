package convert

import (
	mapset "github.com/deckarep/golang-set"
	"github.com/stackrox/rox/generated/storage"
)

// SecretToSecretList converts a secret to list secret
func SecretToSecretList(s *storage.Secret) *storage.ListSecret {
	typeSet := mapset.NewSet()
	var typeSlice []storage.SecretType
	for _, f := range s.GetFiles() {
		if !typeSet.Contains(f.GetType()) {
			typeSlice = append(typeSlice, f.GetType())
			typeSet.Add(f.GetType())
		}
	}
	if len(typeSlice) == 0 {
		typeSlice = append(typeSlice, storage.SecretType_UNDETERMINED)
	}

	return &storage.ListSecret{
		Id:          s.GetId(),
		Name:        s.GetName(),
		ClusterId:   s.GetClusterId(),
		ClusterName: s.GetClusterName(),
		Namespace:   s.GetNamespace(),
		Types:       typeSlice,
		CreatedAt:   s.GetCreatedAt(),
	}
}
