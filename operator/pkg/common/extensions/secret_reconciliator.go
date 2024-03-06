package extensions

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	commonLabels "github.com/stackrox/rox/operator/pkg/common/labels"
	"github.com/stackrox/rox/operator/pkg/types"
	"github.com/stackrox/rox/operator/pkg/utils"
	coreV1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

// OwnershipStrategy is the strategy used to manage the ownership of a secret.
type OwnershipStrategy string

const (
	// OwnershipStrategyOwnerReference is the strategy to use metadata.ownerReferences to manage the ownership of a secret.
	OwnershipStrategyOwnerReference OwnershipStrategy = "owner-reference"
	// OwnershipStrategyLabel is the strategy to use annotation to manage the ownership of a secret.
	OwnershipStrategyLabel OwnershipStrategy = "label"
)

const (
	// managedByOperatorLabel is the label used to manage the ownership of a secret.
	managedByOperatorLabel = "app.kubernetes.io/managed-by"
	// managedByOperatorValue is the value of the label used to manage the ownership of a secret.
	managedByOperatorValue = "rhacs-operator"
)

// validateSecretDataFunc validates a secret to determine if the generation should run. The boolean parameter is true if the
// secret is managed by the running operator.
type validateSecretDataFunc func(types.SecretDataMap, bool) error

// generateSecretDataFunc generates new content of a secret.
// The input data map contains the pre-existing secret content (if any) - in rare cases
// it is needed in order to preserve selected fields rather than regenerate them.
type generateSecretDataFunc func(types.SecretDataMap) (types.SecretDataMap, error)

// NewSecretReconciliator creates a new SecretReconciliator. It takes a context and controller client.
// The obj parameter is the owner object (i.e. a custom resource).
func NewSecretReconciliator(client ctrlClient.Client, direct ctrlClient.Reader, obj types.K8sObject, ownershipStrategy OwnershipStrategy) *SecretReconciliator {
	return &SecretReconciliator{
		client: client,
		obj:    obj,
		direct: direct,
	}
}

// SecretReconciliator reconciles a secret.
type SecretReconciliator struct {
	client ctrlClient.Client
	obj    types.K8sObject
	direct ctrlClient.Reader
}

// Client returns the (cached) controller-runtime client used by the extension.
func (r *SecretReconciliator) Client() ctrlClient.Client {
	return r.client
}

// UncachedClient returns the uncached controller-runtime client used by the extension.
func (r *SecretReconciliator) UncachedClient() ctrlClient.Reader {
	return r.direct
}

// DeleteSecret makes sure a secret with the given name does NOT exist.
// NOTE that this function will never touch a secret which is not managed.
func (r *SecretReconciliator) DeleteSecret(ctx context.Context, name string) error {
	secret := &coreV1.Secret{}
	key := ctrlClient.ObjectKey{Namespace: r.obj.GetNamespace(), Name: name}
	if err := utils.GetWithFallbackToUncached(ctx, r.Client(), r.UncachedClient(), key, secret); err != nil {
		if !apiErrors.IsNotFound(err) {
			return errors.Wrapf(err, "checking existence of %s secret", name)
		}
		secret = nil
	}
	if secret == nil {
		return nil
	}
	if !isSecretManaged(secret, r.obj) {
		return nil // do not touch unmanaged secrets
	}
	if err := utils.DeleteExact(ctx, r.Client(), secret); err != nil && !apiErrors.IsNotFound(err) {
		return errors.Wrapf(err, "deleting secret %s", key)
	}
	return nil
}

// EnsureSecret makes sure a secret with the given name exists.
// If the validateSecretDataFunc returns an error, then this function calls generateSecretDataFunc to get new secret data and updates the secret to "fix" it.
// Also note that this function will refuse to touch a secret which is not managed.
func (r *SecretReconciliator) EnsureSecret(ctx context.Context, name string, validate validateSecretDataFunc, generate generateSecretDataFunc) error {
	secret := &coreV1.Secret{}
	key := ctrlClient.ObjectKey{Namespace: r.obj.GetNamespace(), Name: name}

	// Fallback to read directly from API server as oposed to a cache client
	// to make sure we recognize old secrets that have not been labeled properly
	// to match the cache selector.
	if err := utils.GetWithFallbackToUncached(ctx, r.Client(), r.UncachedClient(), key, secret); err != nil {
		if !apiErrors.IsNotFound(err) {
			return errors.Wrapf(err, "checking existence of %s secret", name)
		}
		secret = nil
	}

	if secret != nil {
		return r.updateExisting(ctx, secret, validate, generate)
	}

	// Try to generate the secret, in order to fix it.
	data, err := generate(nil)
	if err != nil {
		return generateError(err, name, "new")
	}
	newSecret := &coreV1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: r.obj.GetNamespace(),
			Labels: map[string]string{
				managedByOperatorLabel: managedByOperatorValue,
			},
			Labels: commonLabels.DefaultLabels(),
		},
		Data: data,
	}
	if r.ownershipStrategy == OwnershipStrategyOwnerReference {
		newSecret.SetOwnerReferences([]metav1.OwnerReference{*metav1.NewControllerRef(r.obj, r.obj.GroupVersionKind())})
	}

	return errors.Wrapf(r.Client().Create(ctx, newSecret), "creating new %s secret failed", name)
}

func (r *SecretReconciliator) updateExisting(ctx context.Context, secret *coreV1.Secret, validate validateSecretDataFunc, generate generateSecretDataFunc) error {
	isManaged := isSecretManaged(secret, r.obj)
	validateErr := validate(secret.Data, isManaged)
	needsUpdate := r.applyOwnershipStrategy(secret)

	if validateErr == nil {
		if needsUpdate {
			if err := r.Client().Update(ctx, secret); err != nil {
				return errors.Wrapf(err, "updating %s secret", name)
			}
		}
		return nil // validation of existing secret successful - no reconciliation needed
	}

	// If the secret is unmanaged, we cannot fix it, so we should fail.
	if validateErr != nil && !isManaged {
		return errors.Wrapf(validateErr,
			"existing %s secret is invalid (%s), but not owned by the CR, please delete the secret to allow fixing the issue",
			validateErr.Error(), secret.Name)
	}

	if validateErr != nil {
		oldData := secret.Data
		data, err := generate(oldData)
		if err != nil {
			return generateError(err, secret.Name, fmt.Sprintf("invalid (%s)", validateErr.Error()))
		}
		secret.Data = data
		needsUpdate = true
	}

	labels, needsLabelUpdate := commonLabels.WithDefaults(secret.Labels)
	secret.Labels = labels
	needsUpdate = needsUpdate || needsLabelUpdate
	if !needsUpdate || !isManaged {
		return nil
	}

	return errors.Wrapf(r.client.Update(ctx, secret), "updating secret %s/%s", secret.Namespace, secret.Name)
}

func generateError(err error, secretName, extraInfo string) error {
	return errors.Wrapf(err, "error generating data for %s %s secret", extraInfo, secretName)
}

// applyOwnershipStrategy ensures that the secret uses the desired ownership strategy.
// it will return true if the secret needs to be updated
func (r *SecretReconciliator) applyOwnershipStrategy(secret *coreV1.Secret) bool {
	if !isSecretManaged(secret, r.obj) {
		// do not touch unmanaged secrets
		return false
	}

	shouldUpdate := false

	// We always want managed secrets to have the label/value pair.
	// Secrets created in previous versions will only have the ownerReference set.
	// So we need to migrate them to also have the label/value pair.
	if secret.Labels == nil || secret.Labels[managedByOperatorLabel] != managedByOperatorValue {
		if secret.Labels == nil {
			secret.Labels = make(map[string]string)
		}
		secret.Labels[managedByOperatorLabel] = managedByOperatorValue
		shouldUpdate = true
	}

	if r.ownershipStrategy == OwnershipStrategyLabel && metav1.IsControlledBy(secret, r.obj) {
		// Secret should be using label, but is using ownerReference, so remove it
		var newOwnerReferences []metav1.OwnerReference
		for _, ref := range secret.GetOwnerReferences() {
			if ref.UID != r.obj.GetUID() {
				newOwnerReferences = append(newOwnerReferences, ref)
			}
		}
		secret.SetOwnerReferences(newOwnerReferences)
		shouldUpdate = true
	} else if r.ownershipStrategy == OwnershipStrategyOwnerReference && !metav1.IsControlledBy(secret, r.obj) {
		// Secret should be using ownerReference, but doesn't have one, so set it
		ownerRef := metav1.NewControllerRef(r.obj, r.obj.GroupVersionKind())
		secret.SetOwnerReferences([]metav1.OwnerReference{*ownerRef})
		shouldUpdate = true
	}
	return shouldUpdate
}

func getIsManagedByOwnerRef(secret *coreV1.Secret, obj types.K8sObject) bool {
	return metav1.IsControlledBy(secret, obj)
}

func getIsManagedByAnnotation(secret *coreV1.Secret) bool {
	return secret.Labels != nil && secret.Labels[managedByOperatorLabel] == managedByOperatorValue
}

func isSecretManaged(secret *coreV1.Secret, obj types.K8sObject) bool {
	return getIsManagedByOwnerRef(secret, obj) || getIsManagedByAnnotation(secret)
}
