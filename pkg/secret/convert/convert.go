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

	ls := &storage.ListSecret{}
	ls.SetId(s.GetId())
	ls.SetName(s.GetName())
	ls.SetClusterId(s.GetClusterId())
	ls.SetClusterName(s.GetClusterName())
	ls.SetNamespace(s.GetNamespace())
	ls.SetTypes(typeSlice)
	ls.SetCreatedAt(s.GetCreatedAt())
	return ls
}
