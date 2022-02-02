package localscanner

import (
	"context"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

var (
	_ certSecretsRepo = (*certSecretsRepoImpl)(nil)
)

// certSecretsRepo is in charge of persisting and retrieving a set of secrets corresponding to a fixed
// set of service types into k8s, thus implementing the
// [repository pattern](https://martinfowler.com/eaaCatalog/repository.html) for a map from service types
// to secrets and using the k8s API as persistence.
type certSecretsRepo interface {
	// getSecrets retrieves the secrets from permanent storage.
	getSecrets(ctx context.Context) (map[storage.ServiceType]*v1.Secret, error)
	// putSecrets persists the secrets on permanent storage.
	// - Returns an error in case some service in `secret` is not in the set of service types handled by the repository.
	// - `secrets` may miss an entry for some service type handled by the repository, in that case this only updates
	//   the secrets for the service types in `secrets`.
	// - This operation is idempotent but not atomic in sense that on error some of the secrets might be persisted
	//   while others are not.
	putSecrets(ctx context.Context, secrets map[storage.ServiceType]*v1.Secret) error
}

type certSecretsRepoImpl struct {
	secretNames   map[storage.ServiceType]string
	secretsClient corev1.SecretInterface
}

// newCertSecretsRepo creates a new certSecretsRepo that handles secrets with the specified names and
// for the specified service types, and uses the k8s API for persistence.
func newCertSecretsRepo(secretNames map[storage.ServiceType]string,
	secretsClient corev1.SecretInterface) certSecretsRepo {
	return &certSecretsRepoImpl{
		secretNames:   secretNames,
		secretsClient: secretsClient,
	}
}

func (r *certSecretsRepoImpl) getSecrets(ctx context.Context) (map[storage.ServiceType]*v1.Secret, error) {
	secretsMap := make(map[storage.ServiceType]*v1.Secret, len(r.secretNames))
	var getErr error
	for serviceType, secretName := range r.secretNames {
		secret, err := r.secretsClient.Get(ctx, secretName, metav1.GetOptions{})
		if err != nil {
			getErr = multierror.Append(getErr, errors.Wrapf(err, "for secret %s", secretName))
		} else {
			secretsMap[serviceType] = secret
		}
		// on context cancellation abort getting other secrets.
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
	}
	if getErr != nil {
		return nil, getErr
	}
	return secretsMap, nil
}

func (r *certSecretsRepoImpl) putSecrets(ctx context.Context, secrets map[storage.ServiceType]*v1.Secret) error {
	var putErr error

	for serviceType, secret := range secrets {
		secretName, ok := r.secretNames[serviceType]
		if !ok {
			putErr = multierror.Append(putErr, errors.Errorf("unkown service type %s", serviceType))
		} else {
			if secret != nil {
				_, err := r.secretsClient.Update(ctx, secret, metav1.UpdateOptions{})
				if err != nil {
					putErr = multierror.Append(putErr, errors.Wrapf(err, "for secret %s", secretName))
				}
			}
		}
		// on context cancellation abort putting other secrets.
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}

	return putErr
}
