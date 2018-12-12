package secrets

import (
	"github.com/docker/docker/api/types/swarm"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoconv"
)

type secretWrap swarm.Secret

func (s secretWrap) asSecret() *storage.Secret {
	return &storage.Secret{
		Id:          s.ID,
		Name:        s.Spec.Name,
		Namespace:   "default",
		Type:        "Secret",
		Labels:      s.Spec.Labels,
		Annotations: s.Spec.Annotations.Labels,
		CreatedAt:   protoconv.ConvertTimeToTimestamp(s.CreatedAt),
	}
}
