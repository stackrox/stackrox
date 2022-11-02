package schemas

import (
	"strings"

	openapi_v2 "github.com/google/gnostic/openapiv2"
	"github.com/pkg/errors"
	"github.com/stackrox/helmtest/internal/rox-imported/set"
	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/chartutil"
	k8sSchema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/kubectl/pkg/util/openapi"
)

// Schema is a schema that can be used for validation.
type Schema interface {
	openapi.Resources
	VersionSet() chartutil.VersionSet
}

type schema struct {
	openapi.Resources
	allGVKs map[k8sSchema.GroupVersionKind]struct{}
}

func newSchema(doc *openapi_v2.Document) (*schema, error) {
	resources, err := openapi.NewOpenAPIData(doc)
	if err != nil {
		return nil, errors.Wrap(err, "parsing OpenAPI document")
	}
	allGVKs := make(map[k8sSchema.GroupVersionKind]struct{})
	for _, def := range doc.GetDefinitions().GetAdditionalProperties() {
		for _, vendorExt := range def.GetValue().GetVendorExtension() {
			if vendorExt.GetName() != "x-kubernetes-group-version-kind" {
				continue
			}
			var gvks []k8sSchema.GroupVersionKind
			yamlDec := yaml.NewDecoder(strings.NewReader(vendorExt.GetValue().GetYaml()))
			yamlDec.KnownFields(true)
			if err := yamlDec.Decode(&gvks); err != nil {
				return nil, errors.Wrap(err, "decoding x-kubernetes-group-version-kind vendor extension")
			}
			for _, gvk := range gvks {
				allGVKs[gvk] = struct{}{}
			}
		}
	}
	return &schema{
		Resources: resources,
		allGVKs:   allGVKs,
	}, nil
}

func (s *schema) VersionSet() chartutil.VersionSet {
	allVersions := set.NewStringSet()
	for gvk := range s.allGVKs {
		prefix := ""
		if gvk.Group != "" {
			allVersions.Add(gvk.Group)
			prefix = gvk.Group + "/"
		}
		allVersions.Add(prefix + gvk.Version)
		allVersions.Add(prefix + gvk.Version + "/" + gvk.Kind)
	}
	return allVersions.AsSortedSlice(func(i, j string) bool { return i < j })
}
