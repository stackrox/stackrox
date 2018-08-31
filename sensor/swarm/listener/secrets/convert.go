package secrets

import (
	"github.com/docker/docker/api/types/swarm"
	"github.com/stackrox/rox/generated/api/v1"
)

type secretWrap swarm.Secret

func (s secretWrap) asSecret() *v1.Secret {
	return &v1.Secret{
		Id:          s.ID,
		Name:        s.Spec.Name,
		Namespace:   "default",
		Type:        "Secret",
		Labels:      s.Spec.Labels,
		Annotations: s.Spec.Annotations.Labels,
	}
}
