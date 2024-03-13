package labels

import (
	"bytes"

	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/kube"
	"helm.sh/helm/v3/pkg/postrender"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/cli-runtime/pkg/resource"
	"sigs.k8s.io/yaml"
)

var defaultLabels = map[string]string{
	"app.stackrox.io/managed-by": "operator",
}

// DefaultLabels defines the default labels the operator should set on resources it creates
func DefaultLabels() map[string]string {
	labels := make(map[string]string, len(defaultLabels))
	for k, v := range defaultLabels {
		labels[k] = v
	}

	return labels
}

// SetDefaultLabels sets the labels defined in DefaultLabels on the given map.
// It returns a bool to indicate whether the given map was updated or not
func SetDefaultLabels(labels map[string]string) bool {
	updated := false
	for k, v := range defaultLabels {
		value, hasKey := labels[k]
		if !hasKey || value != v {
			updated = true
			if labels == nil {
				labels = map[string]string{}
			}
			labels[k] = v
		}
	}

	return updated
}

// NewLabelPostRenderer is a postrenderer for helm operator plugin kube clients to add
// given labels to each renderered object
func NewLabelPostRenderer(client kube.Interface, labels map[string]string) postrender.PostRenderer {
	return &labelPostRenderer{
		kubeClient: client,
		labels:     labels,
	}
}

var _ postrender.PostRenderer = &labelPostRenderer{}

type labelPostRenderer struct {
	kubeClient kube.Interface
	labels     map[string]string
}

func (lpr labelPostRenderer) Run(renderedManifests *bytes.Buffer) (*bytes.Buffer, error) {
	rl, err := lpr.kubeClient.Build(renderedManifests, false)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build resource list for label post renderer")
	}

	out := bytes.Buffer{}

	err = rl.Visit(func(i *resource.Info, err error) error {
		if err != nil {
			return err
		}

		objMeta, err := meta.Accessor(i.Object)
		if err != nil {
			return errors.Wrapf(err, "failed to access metadata for %s/%s", i.Namespace, i.Name)
		}

		labels := objMeta.GetLabels()
		if labels == nil {
			labels = map[string]string{}
		}

		for k, v := range lpr.labels {
			labels[k] = v
		}

		objMeta.SetLabels(labels)

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
		return nil, errors.Wrap(err, "failed to visit resources for label post renderer")
	}

	return &out, nil
}
