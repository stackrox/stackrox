package translation

import (
	"context"
	"fmt"

	common "github.com/stackrox/rox/operator/common/api/v1alpha1"
	"helm.sh/helm/v3/pkg/chartutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ResourcesKey describes the key for a resources dict.
type ResourcesKey string

const (
	// ResourcesLabel is a key for most resources specifications.
	ResourcesLabel ResourcesKey = "resources"
	// ResourcesComplianceLabel is a key for compliance container resource specification.
	ResourcesComplianceLabel ResourcesKey = "complianceResources"
)

// SetResources optionally sets a "resources" field in values based on resources.
func SetResources(resources *common.Resources, values chartutil.Values, key ResourcesKey) {
	if resources == nil || resources.Override == nil {
		return
	}
	// TODO(ROX-7146): take care of sizing guidelines
	res := chartutil.Values{}

	if len(resources.Override.Requests) > 0 {
		res["requests"] = resources.Override.Requests.DeepCopy()
	}
	if len(resources.Override.Limits) > 0 {
		res["limits"] = resources.Override.Limits.DeepCopy()
	}
	if len(res) > 0 {
		values[string(key)] = res
	}
}

// CustomizeComponent describes the key for a customize dict.
type CustomizeComponent string

// These constants are used for specifying customize dicts for various components.
const (
	CustomizeSensor           CustomizeComponent = "sensor"
	CustomizeAdmissionControl CustomizeComponent = "admission-control"
	CustomizeCollector        CustomizeComponent = "collector"
	CustomizeTopLevel         CustomizeComponent = ""
)

// SetCustomize optionally populates a "customize" dict in values based on customizeSpec.
// If component is not empty, sub-dict of that name is populated. Otherwise, set properties directly in passed "values".
func SetCustomize(customizeSpec *common.CustomizeSpec, values chartutil.Values, component CustomizeComponent) {
	if customizeSpec == nil {
		return
	}

	cv := chartutil.Values{}

	if len(customizeSpec.Labels) > 0 {
		cv["labels"] = customizeSpec.Labels
	}
	if len(customizeSpec.Annotations) > 0 {
		cv["annotations"] = customizeSpec.Annotations
	}
	if len(customizeSpec.PodLabels) > 0 {
		cv["podLabels"] = customizeSpec.PodLabels
	}
	if len(customizeSpec.PodAnnotations) > 0 {
		cv["podAnnotations"] = customizeSpec.PodAnnotations
	}
	if len(customizeSpec.EnvVars) > 0 {
		cv["envVars"] = customizeSpec.EnvVars
	}

	if len(cv) == 0 {
		return
	}

	if component == CustomizeTopLevel {
		if _, ok := values["customize"]; !ok {
			values["customize"] = cv
		} else {
			// Careful not to overwrite the whole top-level dict
			dest := values["customize"].(chartutil.Values)
			for name, val := range cv {
				dest[name] = val
			}
		}
	} else {
		if _, ok := values["customize"]; !ok {
			values["customize"] = chartutil.Values{string(component): cv}
		} else {
			values["customize"].(chartutil.Values)[string(component)] = cv
		}
	}
}

// SetServiceTLS optionally sets a "serviceTLS" field in values based on serviceTLS.
func SetServiceTLS(ctx context.Context, clientSet kubernetes.Interface, namespace string, serviceTLS *corev1.LocalObjectReference, values chartutil.Values) error {
	if serviceTLS == nil {
		return nil
	}

	secretName := serviceTLS.Name
	secret, err := clientSet.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	key, ok := secret.StringData["key"]
	if !ok {
		return fmt.Errorf("secret %q in namespace %q does not contain member %q", secretName, namespace, "key")
	}
	cert, ok := secret.StringData["cert"]
	if !ok {
		return fmt.Errorf("secret %q in namespace %q does not contain member %q", secretName, namespace, "cert")
	}
	values["serviceTLS"] = chartutil.Values{
		"cert": cert,
		"key":  key,
	}
	return nil
}

// SetBool optionally sets label field in values based on b.
func SetBool(b *bool, label string, values chartutil.Values) {
	if b != nil {
		values[label] = *b
	}
}
