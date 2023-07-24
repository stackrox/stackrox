package overlays

import (
	"bytes"
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/stackrox/k8s-overlay-patch/pkg/patch"
	"github.com/stackrox/k8s-overlay-patch/pkg/types"
	"github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	"helm.sh/helm/v3/pkg/postrender"
	k8sTypes "k8s.io/apimachinery/pkg/types"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

// OverlayPostRenderer is a post-renderer that applies overlays to the rendered manifests.
type OverlayPostRenderer struct {
	obj              ctrlClient.Object
	defaultNamespace string
}

// NewOverlayPostRenderer returns a new OverlayPostRenderer.
func NewOverlayPostRenderer(obj ctrlClient.Object, defaultNamespace string) *OverlayPostRenderer {
	return &OverlayPostRenderer{
		obj:              obj,
		defaultNamespace: defaultNamespace,
	}
}

var _ postrender.PostRenderer = &OverlayPostRenderer{}

type crdWithOverlays struct {
	Spec overlaySpec `json:"spec"`
}

type overlaySpec struct {
	Overlays []*v1alpha1.K8sObjectOverlay `json:"overlays"`
}

// Run applies overlays to the rendered manifests.
func (o OverlayPostRenderer) Run(renderedManifests *bytes.Buffer) (modifiedManifests *bytes.Buffer, err error) {
	var objKey = k8sTypes.NamespacedName{
		Namespace: o.obj.GetNamespace(),
		Name:      o.obj.GetName(),
	}
	var gvk = o.obj.GetObjectKind().GroupVersionKind()
	objBytes, err := json.Marshal(o.obj)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal %v %v to json", gvk, objKey)
	}
	var wo crdWithOverlays
	err = json.Unmarshal(objBytes, &wo)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal %v %v as CustomResource with overlays", gvk, objKey)
	}

	if len(wo.Spec.Overlays) == 0 {
		return renderedManifests, nil
	}

	patched, err := patch.YAMLManifestPatch(renderedManifests.String(), o.defaultNamespace, mapOverlays(wo.Spec.Overlays))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to apply overlays onto %v %v", gvk, objKey)
	}
	return bytes.NewBufferString(patched), nil
}

func mapOverlays(overlays []*v1alpha1.K8sObjectOverlay) []*types.K8sObjectOverlay {
	out := make([]*types.K8sObjectOverlay, len(overlays))
	for i, o := range overlays {
		out[i] = &types.K8sObjectOverlay{
			ApiVersion: o.APIVersion,
			Kind:       o.Kind,
			Name:       o.Name,
			Patches:    mapOverlayPatches(o.Patches),
		}
	}
	return out
}

func mapOverlayPatches(patches []*v1alpha1.K8sObjectOverlayPatch) []*types.K8sObjectOverlayPatch {
	out := make([]*types.K8sObjectOverlayPatch, len(patches))
	for i, p := range patches {
		out[i] = &types.K8sObjectOverlayPatch{
			Path:  p.Path,
			Value: p.Value,
		}
	}
	return out
}
