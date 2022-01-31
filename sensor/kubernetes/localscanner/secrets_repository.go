package localscanner

import (
	"context"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	v1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/util/retry"
)

var (
	_ certSecretsRepo = (*certSecretsRepoImpl)(nil)
)

// certSecretsRepo is in charge of persisting and retrieving a set of secrets corresponding to service types
// into some permanent storage system.
type certSecretsRepo interface {
	// getSecrets retrieves the secrets from permanent storage.
	getSecrets(ctx context.Context) (map[storage.ServiceType]*v1.Secret, error)
	// putSecrets persists the secrets on permanent storage.
	putSecrets(ctx context.Context, secrets map[storage.ServiceType]*v1.Secret) error
}

type certSecretsRepoImpl struct {
	secretNames   map[storage.ServiceType]string
	backoff       wait.Backoff
	secretsClient corev1.SecretInterface
}

// NewCertSecretsRepo creates a new certSecretsRepo that handles secrets with the specified names and
// for the specified service types, and uses the k8s API for persistence.
func NewCertSecretsRepo(secretNames map[storage.ServiceType]string,
	backoff wait.Backoff, secretsClient corev1.SecretInterface) certSecretsRepo {
	return &certSecretsRepoImpl{
		secretNames:   secretNames,
		backoff:       backoff,
		secretsClient: secretsClient,
	}
}

func (r *certSecretsRepoImpl) getSecrets(ctx context.Context) (map[storage.ServiceType]*v1.Secret, error) {
	secretsMap := make(map[storage.ServiceType]*v1.Secret, len(r.secretNames))
	var getErr error
	for serviceType, secretName := range r.secretNames {
		var (
			secret *v1.Secret
			err    error
		)
		retryErr := retry.OnError(r.backoff,
			func(err error) bool {
				return (ctx.Err() == nil) && !k8sErrors.IsNotFound(err)
			},
			func() error {
				secret, err = r.secretsClient.Get(ctx, secretName, metav1.GetOptions{})
				return err
			},
		)
		if retryErr != nil {
			getErr = multierror.Append(getErr, errors.Wrapf(retryErr, "for secret %s", secretName))
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
	for serviceType, secretName := range r.secretNames {
		secret := secrets[serviceType]
		if secret == nil {
			putErr =
				multierror.Append(putErr, errors.Errorf("no secret found for service type %s", serviceType))
		} else {
			retryErr := retry.OnError(r.backoff,
				func(err error) bool {
					return ctx.Err() == nil
				},
				func() error {
					_, err := r.secretsClient.Update(ctx, secret, metav1.UpdateOptions{})
					return err
				},
			)
			if retryErr != nil {
				putErr = multierror.Append(putErr, errors.Wrapf(retryErr, "for secret %s", secretName))
			}
		}
		// on context cancellation abort putting other secrets.
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}

	return putErr
}
