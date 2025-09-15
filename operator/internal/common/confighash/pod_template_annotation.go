package confighash

import (
	"bytes"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/operator/internal/common/rendercache"
	"helm.sh/helm/v3/pkg/kube"
	"helm.sh/helm/v3/pkg/postrender"
	unstructuredapi "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/cli-runtime/pkg/resource"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

// NewPodTemplateAnnotationPostRenderer creates a PostRenderer that reads config-hash data
// from the render cache and applies them to pod templates of Deployments and DaemonSets.
func NewPodTemplateAnnotationPostRenderer(client kube.Interface, obj ctrlClient.Object, renderCache *rendercache.RenderCache) postrender.PostRenderer {
	return &podTemplateAnnotationPostRenderer{
		kubeClient:  client,
		obj:         obj,
		renderCache: renderCache,
	}
}

var _ postrender.PostRenderer = &podTemplateAnnotationPostRenderer{}

type podTemplateAnnotationPostRenderer struct {
	kubeClient  kube.Interface
	obj         ctrlClient.Object
	renderCache *rendercache.RenderCache
}

func (pr podTemplateAnnotationPostRenderer) Run(renderedManifests *bytes.Buffer) (*bytes.Buffer, error) {
	configHash, found := pr.renderCache.GetCAHash(pr.obj)
	if !found || configHash == "" {
		return renderedManifests, nil
	}

	rl, err := pr.kubeClient.Build(renderedManifests, false)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build resource list for pod template annotation post renderer")
	}

	return applyPodTemplateAnnotationAndSerialize(rl, configHash)
}

func applyPodTemplateAnnotationAndSerialize(rl kube.ResourceList, configHash string) (*bytes.Buffer, error) {
	out := bytes.Buffer{}
	err := rl.Visit(func(i *resource.Info, err error) error {
		if err != nil {
			return err
		}

		// Only process Deployments and DaemonSets (i.e. Collector)
		switch obj := i.Object.(type) {
		case *unstructuredapi.Unstructured:
			kind := obj.GetKind()
			// ROX-30875: Add a test to detect when new pod-creating resource types are introduced.
			if kind == "Deployment" || kind == "DaemonSet" {
				annotations, found, err := unstructuredapi.NestedStringMap(obj.Object, "spec", "template", "metadata", "annotations")
				if err != nil {
					return errors.Wrapf(err, "failed to get annotations on %s %s/%s pod template", kind, i.Namespace, i.Name)
				}

				if !found || annotations == nil {
					annotations = map[string]string{}
				}
				annotations[AnnotationKey] = configHash

				if err := unstructuredapi.SetNestedStringMap(obj.Object, annotations, "spec", "template", "metadata", "annotations"); err != nil {
					return errors.Wrapf(err, "failed to set annotations on %s %s/%s pod template", kind, i.Namespace, i.Name)
				}
			}
		}

		outData, err := yaml.Marshal(i.Object)
		if err != nil {
			return errors.Wrapf(err, "failed to marshal object to yaml for %s/%s", i.Namespace, i.Name)
		}

		if _, err := out.WriteString("---\n" + string(outData)); err != nil {
			return errors.Wrapf(err, "failed to write object to output buffer for %s/%s", i.Namespace, i.Name)
		}
		return nil
	})

	if err != nil {
		return nil, errors.Wrap(err, "failed to visit resources for pod template annotation post renderer")
	}

	return &out, nil
}
