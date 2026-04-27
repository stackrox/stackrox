package migratetooperator

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/yaml"
)

type dirSource struct {
	dir string
}

// NewDirSource creates a Source that reads resources from YAML files in a directory tree.
func NewDirSource(dir string) (*dirSource, error) {
	info, err := os.Stat(dir)
	if err != nil {
		return nil, errors.Wrapf(err, "accessing directory %q", dir)
	}
	if !info.IsDir() {
		return nil, errors.Errorf("%q is not a directory", dir)
	}
	return &dirSource{dir: dir}, nil
}

func (s *dirSource) Deployment(name string) (*appsv1.Deployment, error) {
	var found *appsv1.Deployment
	if err := s.walkYAML(func(doc []byte) bool {
		var dep appsv1.Deployment
		if err := yaml.Unmarshal(doc, &dep); err == nil && dep.Kind == "Deployment" && dep.Name == name {
			found = &dep
			return true
		}
		return false
	}); err != nil {
		return nil, err
	}
	return found, nil
}

func (s *dirSource) Service(name string) (*corev1.Service, error) {
	var found *corev1.Service
	if err := s.walkYAML(func(doc []byte) bool {
		var svc corev1.Service
		if err := yaml.Unmarshal(doc, &svc); err == nil && svc.Kind == "Service" && svc.Name == name {
			found = &svc
			return true
		}
		return false
	}); err != nil {
		return nil, err
	}
	if found == nil {
		return nil, nil
	}
	return found, nil
}

func (s *dirSource) Secret(name string) (*corev1.Secret, error) {
	var found *corev1.Secret
	if err := s.walkYAML(func(doc []byte) bool {
		var secret corev1.Secret
		if err := yaml.Unmarshal(doc, &secret); err == nil && secret.Kind == "Secret" && secret.Name == name {
			found = &secret
			return true
		}
		return false
	}); err != nil {
		return nil, err
	}
	if found == nil {
		return nil, nil
	}
	return found, nil
}

func (s *dirSource) Route(name string) (*unstructured.Unstructured, error) {
	found, err := s.resourceExists("Route", name)
	if err != nil || !found {
		return nil, err
	}
	return &unstructured.Unstructured{}, nil
}

func (s *dirSource) DaemonSet(name string) (*appsv1.DaemonSet, error) {
	var found *appsv1.DaemonSet
	if err := s.walkYAML(func(doc []byte) bool {
		var ds appsv1.DaemonSet
		if err := yaml.Unmarshal(doc, &ds); err == nil && ds.Kind == "DaemonSet" && ds.Name == name {
			found = &ds
			return true
		}
		return false
	}); err != nil {
		return nil, err
	}
	return found, nil
}

func (s *dirSource) ValidatingWebhookConfiguration(name string) (*admissionv1.ValidatingWebhookConfiguration, error) {
	var found *admissionv1.ValidatingWebhookConfiguration
	if err := s.walkYAML(func(doc []byte) bool {
		var vwc admissionv1.ValidatingWebhookConfiguration
		if err := yaml.Unmarshal(doc, &vwc); err == nil && vwc.Kind == "ValidatingWebhookConfiguration" && vwc.Name == name {
			found = &vwc
			return true
		}
		return false
	}); err != nil {
		return nil, err
	}
	return found, nil
}

func (s *dirSource) resourceExists(kind, name string) (bool, error) {
	found := false
	if err := s.walkYAML(func(doc []byte) bool {
		var meta struct {
			metav1.TypeMeta   `json:",inline"`
			metav1.ObjectMeta `json:"metadata"`
		}
		if err := yaml.Unmarshal(doc, &meta); err == nil && meta.Kind == kind && meta.Name == name {
			found = true
			return true
		}
		return false
	}); err != nil {
		return false, err
	}
	return found, nil
}

func (s *dirSource) walkYAML(match func(doc []byte) bool) error {
	return errors.Wrap(filepath.WalkDir(s.dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || (!strings.HasSuffix(d.Name(), ".yaml") && !strings.HasSuffix(d.Name(), ".yml")) {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return errors.Wrapf(err, "reading %q", path)
		}
		docs, splitErr := splitYAMLDocuments(data)
		if splitErr != nil {
			return errors.Wrapf(splitErr, "parsing YAML in %q", path)
		}
		for _, doc := range docs {
			if match(doc) {
				return filepath.SkipAll
			}
		}
		return nil
	}), "walking directory tree")
}

func splitYAMLDocuments(data []byte) ([][]byte, error) {
	var docs [][]byte
	reader := utilyaml.NewYAMLReader(bufio.NewReader(bytes.NewReader(data)))
	for {
		doc, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(string(doc)) != "" {
			docs = append(docs, doc)
		}
	}
	return docs, nil
}
