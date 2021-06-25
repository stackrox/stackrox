package translation

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	common "github.com/stackrox/rox/operator/api/common/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	// ResourcesKey is a key for most resources chart values.
	ResourcesKey = "resources"
)

// GetResources converts common.Resources to chart values builder.
func GetResources(resources *common.Resources) *ValuesBuilder {
	if resources == nil || resources.Override == nil {
		return nil
	}
	// TODO(ROX-7146): take care of sizing guidelines
	res := NewValuesBuilder()

	if len(resources.Override.Requests) > 0 {
		res.SetResourceList("requests", resources.Override.Requests.DeepCopy())
	}
	if len(resources.Override.Limits) > 0 {
		res.SetResourceList("limits", resources.Override.Limits.DeepCopy())
	}
	return &res
}

// GetCustomize converts common.CustomizeSpec to chart values builder.
func GetCustomize(customizeSpec *common.CustomizeSpec) *ValuesBuilder {
	if customizeSpec == nil {
		return nil
	}

	res := NewValuesBuilder()
	res.SetStringMap("labels", customizeSpec.Labels)
	res.SetStringMap("annotations", customizeSpec.Annotations)
	res.SetStringMap("podLabels", customizeSpec.PodLabels)
	res.SetStringMap("podAnnotations", customizeSpec.PodAnnotations)
	res.SetStringMap("envVars", customizeSpec.EnvVars)
	return &res
}

// GetServiceTLS reads given secret and returns "serviceTLS" chart values.
func GetServiceTLS(ctx context.Context, clientSet kubernetes.Interface, namespace string, serviceTLS *corev1.LocalObjectReference, crPath string) *ValuesBuilder {
	return GetServiceTLSWithKey(ctx, clientSet, namespace, serviceTLS, crPath, "serviceTLS")
}

// GetServiceTLSWithKey reads given secret and returns chart values with given key.
func GetServiceTLSWithKey(ctx context.Context, clientSet kubernetes.Interface, namespace string, serviceTLS *corev1.LocalObjectReference, crPath string, key string) *ValuesBuilder {
	if serviceTLS == nil {
		return nil
	}

	res := NewValuesBuilder()
	res.AddChild(key, NewBuilderFromSecret(ctx, clientSet, namespace, serviceTLS, map[string]string{"key": "key", "cert": "cert"}, crPath))
	return &res
}

// GetImagePullSecrets converts corev1.LocalObjectReference to a *ValuesBuilder with an "imagePullSecrets" field.
func GetImagePullSecrets(imagePullSecrets []corev1.LocalObjectReference) *ValuesBuilder {
	res := NewValuesBuilder()
	// TODO(ROX-7179): support imagePullSecrets.allowNone and/or disabling fromDefaultServiceAccount?
	if len(imagePullSecrets) > 0 {
		var ps []string
		for _, secret := range imagePullSecrets {
			ps = append(ps, secret.Name)
		}
		existing := NewValuesBuilder()
		existing.SetStringSlice("useExisting", ps)
		res.AddChild("imagePullSecrets", &existing)
	}
	return &res
}

// GetTLSValues converts common.TLSConfig to a *ValuesBuilder with an "additionalCAs" field.
func GetTLSValues(tls *common.TLSConfig) *ValuesBuilder {
	if tls == nil {
		return nil
	}
	if len(tls.AdditionalCAs) == 0 {
		return nil
	}
	cas := NewValuesBuilder()
	for _, ca := range tls.AdditionalCAs {
		cas.SetStringValue(ca.Name, ca.Content)
	}
	res := NewValuesBuilder()
	res.AddChild("additionalCAs", &cas)
	return &res
}

// NewBuilderFromSecret returns a *ValuesBuilder with string values with given keys based on components in the referred kubernetes Secret.
func NewBuilderFromSecret(ctx context.Context, clientSet kubernetes.Interface, namespace string, secret *corev1.LocalObjectReference, membersToKeys map[string]string, crPath string) *ValuesBuilder {
	v := NewValuesBuilder()
	if secret == nil {
		return &v
	}
	secretObj, err := clientSet.CoreV1().Secrets(namespace).Get(ctx, secret.Name, metav1.GetOptions{})
	if err != nil {
		return v.SetError(errors.Wrapf(err, "failed to retrieve secret %q in namespace %q configured in %s", secret.Name, namespace, crPath))
	}
	for secretMember, builderKey := range membersToKeys {
		value, ok := secretObj.Data[secretMember]
		if ok {
			v.SetStringValue(builderKey, string(value))
		} else {
			// Check all items in map before returning.
			v.SetError(fmt.Errorf("secret %q in namespace %q configured in %s does not contain member %q", secret.Name, namespace, crPath, secretMember))
		}
	}
	return &v
}
