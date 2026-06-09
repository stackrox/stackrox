package reconcile

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	v1alpha1 "github.com/stackrox/rox/config-controller/api/v1alpha1"
	"sigs.k8s.io/yaml"
)

func loadPoliciesFromDir(dir string) ([]v1alpha1.SecurityPolicySpec, error) {
	var specs []v1alpha1.SecurityPolicySpec
	seen := make(map[string]string) // policyName -> filename

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}

		fileSpecs, err := loadPoliciesFromFile(path)
		if err != nil {
			return errors.Wrapf(err, "loading %s", path)
		}

		for _, spec := range fileSpecs {
			if prev, ok := seen[spec.PolicyName]; ok {
				return fmt.Errorf("duplicate policy name %q found in %s and %s", spec.PolicyName, prev, path)
			}
			seen[spec.PolicyName] = path
			specs = append(specs, spec)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return specs, nil
}

func loadPoliciesFromFile(path string) ([]v1alpha1.SecurityPolicySpec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Wrap(err, "reading file")
	}
	return parsePolicyDocuments(data)
}

func parsePolicyDocuments(data []byte) ([]v1alpha1.SecurityPolicySpec, error) {
	documents := splitYAMLDocuments(data)
	var specs []v1alpha1.SecurityPolicySpec
	for _, doc := range documents {
		trimmed := bytes.TrimSpace(doc)
		if len(trimmed) == 0 {
			continue
		}
		var spec v1alpha1.SecurityPolicySpec
		if err := yaml.Unmarshal(trimmed, &spec); err != nil {
			return nil, errors.Wrap(err, "decoding YAML document")
		}
		if spec.PolicyName == "" {
			continue
		}
		specs = append(specs, spec)
	}
	return specs, nil
}

func splitYAMLDocuments(data []byte) [][]byte {
	sep := []byte("\n---")
	parts := bytes.Split(data, sep)
	return parts
}
