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

func TLSSecretLabels() map[string]string {
	labels := DefaultLabels()
	labels["rhacs.redhat.com/tls"] = "true"
	return labels
}

// DefaultLabels defines the default labels the operator should set on resources it creates.
func DefaultLabels() map[string]string {
	labels := make(map[string]string, len(defaultLabels))
	for k, v := range defaultLabels {
		labels[k] = v
	}

	return labels
}

// MergeLabels merges labels
func MergeLabels(current, newLabels map[string]string) (map[string]string, bool) {
	updated := false
	mergedLabels := map[string]string{}

	for k, v := range current {
		mergedLabels[k] = v
	}
	for k, v := range newLabels {
		if x, exists := mergedLabels[k]; !exists || x != v {
			updated = true
			mergedLabels[k] = v
		}

	}
	return mergedLabels, updated
}

// WithDefaults return a copy of the given labels with the default labels added.
// It returns a bool as second argument to indicate whether default labels had to be added or not.
func WithDefaults(labels map[string]string) (map[string]string, bool) {
	updated := false
	newLabels := map[string]string{}
	for k, v := range labels {
		newLabels[k] = v
	}

	for k, v := range defaultLabels {
		value, hasKey := newLabels[k]
		if !hasKey || value != v {
			updated = true
			newLabels[k] = v
		}
	}

	return newLabels, updated
}

// NewLabelPostRenderer is a postrenderer for helm operator plugin kube clients to add
// given labels to each renderered object.
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
