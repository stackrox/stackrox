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

func (s *dirSource) CentralDBDeployment() (*appsv1.Deployment, error) {
	centralDir := filepath.Join(s.dir, "central")
	entries, err := os.ReadDir(centralDir)
	if err != nil {
		return nil, errors.Wrapf(err, "reading directory %q", centralDir)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(centralDir, entry.Name()))
		if err != nil {
			return nil, errors.Wrapf(err, "reading file %q", entry.Name())
		}

		for _, doc := range splitYAMLDocuments(data) {
			var dep appsv1.Deployment
			if err := yaml.Unmarshal(doc, &dep); err != nil {
				continue
			}
			if dep.Kind == "Deployment" && dep.Name == "central-db" {
				return &dep, nil
			}
		}
	}
	return nil, errors.Errorf("central-db Deployment not found in %q", centralDir)
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
