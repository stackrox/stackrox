package convert

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
)

// SecretToSecretList converts a secret to list secret
func SecretToSecretList(s *storage.Secret) *storage.ListSecret {
	typeSet := set.NewSet[storage.SecretType]()
	var typeSlice []storage.SecretType
	for _, f := range s.GetFiles() {
		if ty := f.GetType(); typeSet.Add(ty) {
			typeSlice = append(typeSlice, ty)
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
