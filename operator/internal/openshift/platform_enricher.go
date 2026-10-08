package openshift

import (
	"context"
	"fmt"

	"github.com/stackrox/rox/operator/internal/values/translation"
	"github.com/stackrox/rox/pkg/k8sutil"
	"helm.sh/helm/v3/pkg/chartutil"
	corev1 "k8s.io/api/core/v1"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const namespaceUIDRangeAnnotation = "openshift.io/sa.scc.uid-range"

type PlatformEnricher struct {
	reader ctrlClient.Reader
}

var _ translation.Enricher = (*PlatformEnricher)(nil)

// NewEnricher detects OpenShift from the namespace's allocated UID range,
// avoiding the OpenShift API discovery check that can fail during upgrades.
func NewEnricher(reader ctrlClient.Reader) *PlatformEnricher {
	return &PlatformEnricher{reader: reader}
}

func (e *PlatformEnricher) Enrich(ctx context.Context, obj k8sutil.Object, vals chartutil.Values) (chartutil.Values, error) {
	namespaceName := obj.GetNamespace()
	namespace := &corev1.Namespace{}
	if err := e.reader.Get(ctx, ctrlClient.ObjectKey{Name: namespaceName}, namespace); err != nil {
		return nil, fmt.Errorf("reading namespace %q for OpenShift detection: %w", namespaceName, err)
	}

	if namespace.Annotations[namespaceUIDRangeAnnotation] == "" {
		return vals, nil
	}

	if vals == nil {
		vals = chartutil.Values{}
	}
	if err := setOpenShiftValue(vals); err != nil {
		return nil, err
	}
	return vals, nil
}

func setOpenShiftValue(vals chartutil.Values) error {
	switch env := vals["env"].(type) {
	case nil:
		vals["env"] = map[string]interface{}{"openshift": true}
	case map[string]interface{}:
		if env == nil {
			env = make(map[string]interface{})
			vals["env"] = env
		}
		env["openshift"] = true
	case chartutil.Values:
		if env == nil {
			vals["env"] = map[string]interface{}{"openshift": true}
			return nil
		}
		env["openshift"] = true
	default:
		return fmt.Errorf("expected Helm env values to be a map, got %T", vals["env"])
	}
	return nil
}
