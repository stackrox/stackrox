package convert

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
	"google.golang.org/protobuf/proto"
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

	return storage.ListSecret_builder{
		Id:          proto.String(s.GetId()),
		Name:        proto.String(s.GetName()),
		ClusterId:   proto.String(s.GetClusterId()),
		ClusterName: proto.String(s.GetClusterName()),
		Namespace:   proto.String(s.GetNamespace()),
		Types:       typeSlice,
		CreatedAt:   s.GetCreatedAt(),
	}.Build()
}
