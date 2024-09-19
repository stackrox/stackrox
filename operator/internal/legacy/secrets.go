package legacy

import (
	"context"
	"fmt"

	"github.com/stackrox/rox/operator/internal/values/translation"
	"github.com/stackrox/rox/pkg/k8sutil"
	"helm.sh/helm/v3/pkg/chartutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/strings/slices"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

// NewImagePullSecretReferenceInjector returns an object which enriches helm values
// by appending "secrets" to vals[key]["useExisting"].
//
// It exists to provide minimum backward compatibility to installations which depended on
// references to image pull secrets being added unconditionally to ServiceAccounts.
func NewImagePullSecretReferenceInjector(client ctrlClient.Reader, commonSecretsKey string, commonSecrets ...string) *injector {
	return &injector{
		client:          client,
		commonTableName: commonSecretsKey,
		commonSecrets:   commonSecrets,
	}
}

// WithExtraImagePullSecrets configures the injector to additionally inject "secrets" into
// a helm values table keyed with "key".
//
// However, unlike with the common key and secrets supplied to the constructor, this one will be given the following
// special treatment. Before injection, the list of secrets provided in this call is first extended by prepending
// the list of secrets in vals[common key].useExisting (i.e. in the *input* helm values), if any.
//
// This is done such that the useExisting secrets in the table at the common key get properly propagated to the extra table.
func (i *injector) WithExtraImagePullSecrets(key string, secrets ...string) *injector {
	if i.extraSecretMap == nil {
		i.extraSecretMap = map[string][]string{key: secrets}
	} else {
		i.extraSecretMap[key] = secrets
	}
	return i
}

type injector struct {
	client          ctrlClient.Reader
	commonTableName string
	commonSecrets   []string
	extraSecretMap  map[string][]string
}

var _ translation.Enricher = &injector{}

// Enrich modifies vals to append - for each {key,secrets} entry in secret map - secrets to vals[key]["useExisting"].
func (i *injector) Enrich(ctx context.Context, obj k8sutil.Object, vals chartutil.Values) (chartutil.Values, error) {
	namespaceName := obj.GetNamespace()
	upstreamCommonSecrets, err := getUseExisting(vals, i.commonTableName)
	if err != nil {
		return nil, err
	}
	vals, err = i.enrich(ctx, vals, i.commonTableName, i.commonSecrets, namespaceName)
	if err != nil {
		return nil, err
	}
	for key, secretNames := range i.extraSecretMap {
		vals, err = i.enrich(ctx, vals, key, append(upstreamCommonSecrets, secretNames...), namespaceName)
		if err != nil {
			return nil, err
		}
	}
	return vals, nil
}

// getUseExisting returns a slice of secret names at vals.key.useExisting.
// If no secret names are specified, returns an empty slice and nil error.
// However, if an unexpected type is encountered, returns an error.
func getUseExisting(vals chartutil.Values, key string) ([]string, error) {
	parentValue, ok := vals[key]
	if !ok {
		return nil, nil
	}
	parentTable, ok := parentValue.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("key %q maps to a %T, table expected", key, parentValue)
	}
	value, ok := parentTable["useExisting"]
	if !ok {
		return nil, nil
	}
	secretsSlice, ok := value.([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected value %q type: %T", key+".useExisting", value)
	}
	secretStringSlice := make([]string, 0, len(secretsSlice))
	for i, e := range secretsSlice {
		var secret string
		if secret, ok = e.(string); ok {
			secretStringSlice = append(secretStringSlice, secret)
		} else {
			return nil, fmt.Errorf("unexpected %q element %d type: %T", key+".useExisting", i, e)
		}
	}
	return secretStringSlice, nil
}

func (i *injector) enrich(ctx context.Context, vals chartutil.Values, key string, secretNames []string, namespaceName string) (chartutil.Values, error) {
	var secretNamesToAdd []string

	existingSecretNames, err := i.getExistingSecretNames(ctx, namespaceName)
	if err != nil {
		return nil, err
	}

	for _, existingName := range existingSecretNames {
		if slices.Contains(secretNames, existingName) {
			secretNamesToAdd = append(secretNamesToAdd, existingName)
		}
	}

	if len(secretNamesToAdd) == 0 {
		return vals, nil
	}
	return appendUseExisting(vals, key, secretNamesToAdd)
}

func (i *injector) getExistingSecretNames(ctx context.Context, namespaceName string) ([]string, error) {
	existingSecrets := corev1.SecretList{}

	err := i.client.List(ctx, &existingSecrets, &ctrlClient.ListOptions{Namespace: namespaceName})
	if err != nil {
		return []string{}, fmt.Errorf("failed to list secrets in namespace: %s: %w", namespaceName, err)
	}

	secretNames := make([]string, len(existingSecrets.Items))
	for i, secret := range existingSecrets.Items {
		secretNames[i] = secret.Name
	}

	return secretNames, nil
}

// appendUseExisting creates or appends secretNamesToAdd to vals[key]["useExisting"] slice, with error checking.
func appendUseExisting(vals chartutil.Values, key string, secretNamesToAdd []string) (chartutil.Values, error) {
	if vals == nil {
		vals = chartutil.Values{}
	}
	useExistingSlice, err := getUseExisting(vals, key)
	if err != nil {
		return nil, err
	}
	if _, ok := vals[key]; !ok {
		vals[key] = map[string]interface{}{}
	}
	table := vals[key].(map[string]interface{})
	// conversion for consistency with ToHelmValues()
	table["useExisting"] = toInterfaceSlice(append(useExistingSlice, secretNamesToAdd...))
	return vals, nil
}

func toInterfaceSlice(secrets []string) []interface{} {
	secretsAsInterface := make([]interface{}, 0, len(secrets))
	for _, secret := range secrets {
		secretsAsInterface = append(secretsAsInterface, secret)
	}
	return secretsAsInterface
}
