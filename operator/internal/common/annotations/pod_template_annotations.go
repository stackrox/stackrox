package annotations

import (
	"bytes"

	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/kube"
	"helm.sh/helm/v3/pkg/postrender"
	unstructuredapi "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/cli-runtime/pkg/resource"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

// NewPodTemplateAnnotationPostRenderer creates a PostRenderer that reads config-hash annotations
// from the CR and applies them to pod templates of Deployments and DaemonSets.
func NewPodTemplateAnnotationPostRenderer(client kube.Interface, obj ctrlClient.Object) postrender.PostRenderer {
	return &podTemplateAnnotationPostRenderer{
		kubeClient: client,
		obj:        obj,
	}
}

var _ postrender.PostRenderer = &podTemplateAnnotationPostRenderer{}

type podTemplateAnnotationPostRenderer struct {
	kubeClient kube.Interface
	obj        ctrlClient.Object
}

func (pr podTemplateAnnotationPostRenderer) Run(renderedManifests *bytes.Buffer) (*bytes.Buffer, error) {
	// Read config-hash annotation from the CR
	annotations := make(map[string]string)
	if objAnnotations := pr.obj.GetAnnotations(); objAnnotations != nil {
		if configHash, exists := objAnnotations[ConfigHashAnnotation]; exists {
			annotations[ConfigHashAnnotation] = configHash
		}
	}

	if len(annotations) == 0 {
		return renderedManifests, nil
	}

	rl, err := pr.kubeClient.Build(renderedManifests, false)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build resource list for pod template annotation post renderer")
	}

	out := bytes.Buffer{}

	err = rl.Visit(func(i *resource.Info, err error) error {
		if err != nil {
			return err
		}

		// Only process Deployments and DaemonSets (i.e. Collector)
		switch obj := i.Object.(type) {
		case *unstructuredapi.Unstructured:
			kind := obj.GetKind()
			if kind == "Deployment" || kind == "DaemonSet" {
				m, _, _ := unstructuredapi.NestedStringMap(obj.Object, "spec", "template", "metadata", "annotations")
				if m == nil {
					m = map[string]string{}
				}
				for k, v := range annotations {
					m[k] = v
				}
				if err := unstructuredapi.SetNestedStringMap(obj.Object, m, "spec", "template", "metadata", "annotations"); err != nil {
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
