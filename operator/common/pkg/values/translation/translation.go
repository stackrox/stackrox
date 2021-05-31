package translation

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	common "github.com/stackrox/rox/operator/common/api/v1alpha1"
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
func GetServiceTLS(ctx context.Context, clientSet kubernetes.Interface, namespace string, serviceTLS *corev1.LocalObjectReference) *ValuesBuilder {
	if serviceTLS == nil {
		return nil
	}

	tlsValues := NewValuesBuilder()

	secretName := serviceTLS.Name
	secret, err := clientSet.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return tlsValues.SetError(errors.Wrapf(err, "failed to fetch ServiceTLS secret %q", secretName))
	}

	if key, ok := secret.StringData["key"]; ok {
		tlsValues.SetStringValue("key", key)
	} else {
		return tlsValues.SetError(fmt.Errorf("secret %q in namespace %q does not contain member %q", secretName, namespace, "key"))
	}

	if cert, ok := secret.StringData["cert"]; ok {
		tlsValues.SetStringValue("cert", cert)
	} else {
		return tlsValues.SetError(fmt.Errorf("secret %q in namespace %q does not contain member %q", secretName, namespace, "cert"))
	}

	res := NewValuesBuilder()
	res.AddChild("serviceTLS", &tlsValues)

	return &res
}
