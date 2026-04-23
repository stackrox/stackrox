package migratetooperator

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	if found == nil {
		return nil, errors.Errorf("Deployment %q not found in %q", name, s.dir)
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

func (s *dirSource) Route(name string) (bool, error) {
	return s.resourceExists("Route", name)
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
		for _, doc := range splitYAMLDocuments(data) {
			if match(doc) {
				return filepath.SkipAll
			}
		}
		return nil
	}), "walking directory tree")
}

func splitYAMLDocuments(data []byte) [][]byte {
	var docs [][]byte
	for _, part := range strings.Split(string(data), "\n---") {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			docs = append(docs, []byte(trimmed))
		}
	}
	return docs
}
