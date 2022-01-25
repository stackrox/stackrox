package localscanner

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	v1 "k8s.io/api/core/v1"
)

var (
	_ certSecretsRepo = (*certSecretsRepoImpl)(nil)
)

type certSecretsRepo interface {
	getSecrets(ctx context.Context) (map[storage.ServiceType]*v1.Secret, error)
	putSecrets(ctx context.Context, secrets map[storage.ServiceType]*v1.Secret) error
}

type certSecretsRepoImpl struct {
	// TODO secretsClient   corev1.SecretInterface
}

func (i *certSecretsRepoImpl) getSecrets(ctx context.Context) (map[storage.ServiceType]*v1.Secret, error) {
	secretsMap := make(map[storage.ServiceType]*v1.Secret, 3)
	return secretsMap, nil // TODO
}

func (i *certSecretsRepoImpl) putSecrets(ctx context.Context, secrets map[storage.ServiceType]*v1.Secret) error {
	return nil // TODO
}
