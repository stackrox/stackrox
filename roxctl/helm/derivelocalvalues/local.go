package derivelocalvalues

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/k8sutil"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Retrieve Kubernetes Object Definitions from a file.

type localK8sObjectDescription struct {
	cache map[string]map[string]unstructured.Unstructured // map (kind, name) -> unstructured resource def
}

func (k localK8sObjectDescription) get(_ context.Context, kind string, name string) (*unstructured.Unstructured, error) {
	resources := k.cache[kind]
	if resources == nil {
		return nil, errors.New("resource type not found")
	}
	resource, ok := resources[name]
	if !ok {
		return nil, errox.NotFound.New("resource not found")
	}

	return &resource, nil
}

func k8sResourcesFromFile(file string) (map[string]map[string]unstructured.Unstructured, error) {
	content, err := os.ReadFile(file)
	if err != nil {
		return nil, errors.Wrapf(err, "reading input file %q", file)
	}

	k8sResources, err := k8sResourcesFromString(string(content))
	if err != nil {
		return nil, errors.Wrapf(err, "retrieving Kubernetes resource definitions from file %q", file)
	}
	return k8sResources, nil
}

func k8sResourcesFromString(input string) (map[string]map[string]unstructured.Unstructured, error) {
	cache := make(map[string]map[string]unstructured.Unstructured)
	resources, err := k8sutil.UnstructuredFromYAMLMulti(input)
	if err != nil {
		return nil, errors.Wrap(err, "parsing YAML as Unstructured")
	}
	for _, resource := range resources {
		kind := strings.ToLower(resource.GetKind())
		name := resource.GetName()
		if cache[kind] == nil {
			cache[kind] = make(map[string]unstructured.Unstructured)
		}
		cache[kind][name] = resource
	}
	return cache, nil
}

func newLocalK8sObjectDescriptionFromPath(inputPath string) (*localK8sObjectDescription, error) {
	fileInfo, err := os.Stat(inputPath)
	if err != nil {
		return nil, errors.Wrapf(err, "obtaining file info for input file or directory %q", inputPath)
	}
	var inputFiles []string

	if fileInfo.IsDir() {
		yamls, _ := filepath.Glob(filepath.Join(inputPath, "*.yaml")) // We can rule out the only possible error value ErrBadPattern here.
		ymls, _ := filepath.Glob(filepath.Join(inputPath, "*.yml"))
		inputFiles = append(yamls, ymls...)
	} else {
		inputFiles = []string{inputPath}
	}

	cache := make(map[string]map[string]unstructured.Unstructured)

	for _, inputFile := range inputFiles {
		k8sResources, err := k8sResourcesFromFile(inputFile)
		if err != nil {
			return nil, err
		}
		for kind, resources := range k8sResources {
			if cache[kind] == nil {
				cache[kind] = make(map[string]unstructured.Unstructured)
			}
			for name, u := range resources {
				cache[kind][name] = u
			}
		}
	}

	return &localK8sObjectDescription{cache: cache}, nil
}
