package migratetooperator

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/yaml"
)

type dirSource struct {
	dir string
}

func newDirSource(dir string) (*dirSource, error) {
	info, err := os.Stat(dir)
	if err != nil {
		return nil, errors.Wrapf(err, "accessing directory %q", dir)
	}
	if !info.IsDir() {
		return nil, errors.Errorf("%q is not a directory", dir)
	}
	return &dirSource{dir: dir}, nil
}

func (s *dirSource) CentralDeployment() (*appsv1.Deployment, error) {
	return s.findDeployment("central")
}

func (s *dirSource) CentralDBDeployment() (*appsv1.Deployment, error) {
	return s.findDeployment("central-db")
}

func (s *dirSource) findDeployment(name string) (*appsv1.Deployment, error) {
	var found *appsv1.Deployment
	err := filepath.WalkDir(s.dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || (!strings.HasSuffix(d.Name(), ".yaml") && !strings.HasSuffix(d.Name(), ".yml")) {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return errors.Wrapf(err, "reading %q", path)
		}
		for _, doc := range splitYAMLDocuments(data) {
			var dep appsv1.Deployment
			if err := yaml.Unmarshal(doc, &dep); err != nil {
				continue
			}
			if dep.Kind == "Deployment" && dep.Name == name {
				found = &dep
				return filepath.SkipAll
			}
		}
		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "walking directory tree")
	}
	if found == nil {
		return nil, errors.Errorf("%s Deployment not found in %q", name, s.dir)
	}
	return found, nil
}

func (s *dirSource) ResourceByKindAndName(kind, name string) (bool, map[string]interface{}, error) {
	var found map[string]interface{}
	err := filepath.WalkDir(s.dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || (!strings.HasSuffix(d.Name(), ".yaml") && !strings.HasSuffix(d.Name(), ".yml")) {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return errors.Wrapf(err, "reading %q", path)
		}
		for _, doc := range splitYAMLDocuments(data) {
			var obj map[string]interface{}
			if err := yaml.Unmarshal(doc, &obj); err != nil {
				continue
			}
			objKind, _ := obj["kind"].(string)
			meta, _ := obj["metadata"].(map[string]interface{})
			objName, _ := meta["name"].(string)
			if objKind == kind && objName == name {
				found = obj
				return filepath.SkipAll
			}
		}
		return nil
	})
	if err != nil {
		return false, nil, errors.Wrap(err, "walking directory tree")
	}
	if found == nil {
		return false, nil, nil
	}
	return true, found, nil
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
