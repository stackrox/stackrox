package derivelocalvalues

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
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
		return nil, errors.New("resource not found")
	}

	return &resource, nil
}

func newLocalK8sObjectDescriptionFromFiles(inputFiles []string) (*localK8sObjectDescription, error) {
	cache := make(map[string](map[string]unstructured.Unstructured))
	for _, file := range inputFiles {
		content, err := ioutil.ReadFile(file)
		if err != nil {
			return nil, errors.Wrapf(err, "reading input file %q", file)
		}

		resources, err := k8sutil.UnstructuredFromYAMLMulti(string(content))
		if err != nil {
			return nil, errors.Wrapf(err, "reading YAML from file %q", file)
		}
		for _, resource := range resources {
			kind := strings.ToLower(resource.GetKind())
			name := resource.GetName()
			if cache[kind] == nil {
				cache[kind] = make(map[string]unstructured.Unstructured)
			}
			cache[kind][name] = resource
		}
	}
	k := localK8sObjectDescription{cache: cache}
	return &k, nil
}

func newLocalK8sObjectDescription(input string) (*localK8sObjectDescription, error) {
	fileInfo, err := os.Stat(input)
	if err != nil {
		return nil, errors.Wrapf(err, "obtaining file info for input file or directory %q", input)
	}
	var inputFiles []string

	if fileInfo.IsDir() {
		yamls, _ := filepath.Glob(filepath.Join(input, "*.yaml")) // We can rule out the only possible error value ErrBadPattern here.
		ymls, _ := filepath.Glob(filepath.Join(input, "*.yml"))
		inputFiles = append(yamls, ymls...)
	} else {
		inputFiles = []string{input}
	}

	return newLocalK8sObjectDescriptionFromFiles(inputFiles)
}
