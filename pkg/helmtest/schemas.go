package helmtest

import (
	"embed"
	"io/fs"
	"path"
	"strings"

	openapi_v2 "github.com/googleapis/gnostic/openapiv2"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/gziputil"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/chartutil"
	schema2 "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/kube-openapi/pkg/util/proto"
	"k8s.io/kubectl/pkg/util/openapi"
)

//go:embed openapi-schemas/*
var openAPISchemaFS embed.FS

var (
	allSchemas      = map[string]*schemaEntry{}
	allSchemasMutex sync.Mutex
)

type schema struct {
	openapi.Resources
	allGVKs map[schema2.GroupVersionKind]struct{}
}

func newSchema(doc *openapi_v2.Document) (*schema, error) {
	resources, err := openapi.NewOpenAPIData(doc)
	if err != nil {
		return nil, errors.Wrap(err, "parsing OpenAPI document")
	}
	allGVKs := make(map[schema2.GroupVersionKind]struct{})
	for _, def := range doc.GetDefinitions().GetAdditionalProperties() {
		for _, vendorExt := range def.GetValue().GetVendorExtension() {
			if vendorExt.GetName() != "x-kubernetes-group-version-kind" {
				continue
			}
			var gvks []schema2.GroupVersionKind
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

type schemaEntry struct {
	name     string
	schema   *schema
	loadErr  error
	loadOnce sync.Once
}

func (e *schemaEntry) get() (*schema, error) {
	e.loadOnce.Do(func() {
		e.schema, e.loadErr = e.load()
	})
	return e.schema, e.loadErr
}

func (e *schemaEntry) load() (*schema, error) {
	schemaBytes, err := fs.ReadFile(openAPISchemaFS, path.Join("openapi-schemas", e.name+".json.gz"))
	if err != nil {
		return nil, errors.Wrapf(err, "invalid name %q", e.name)
	}

	openapiDoc, err := gziputil.Decompress(schemaBytes)
	if err != nil {
		return nil, errors.Wrapf(err, "failed reading openapi docs %s", e.name)
	}

	doc, err := openapi_v2.ParseDocument(openapiDoc)
	if err != nil {
		return nil, errors.Wrapf(err, "parsing OpenAPI doc for %s", e.name)
	}
	schema, err := newSchema(doc)
	if err != nil {
		return nil, errors.Wrapf(err, "creating OpenAPI schema from document for %s", e.name)
	}

	return schema, nil
}

func getSchemaEntry(name string) *schemaEntry {
	name = strings.ToLower(name)

	allSchemasMutex.Lock()
	defer allSchemasMutex.Unlock()

	entry := allSchemas[name]
	if entry != nil {
		return entry
	}
	entry = &schemaEntry{
		name: name,
	}
	allSchemas[name] = entry
	return entry
}

func getSchema(name string) (*schema, error) {
	return getSchemaEntry(name).get()
}

type schemas []*schema

func (s schemas) LookupResource(gvk schema2.GroupVersionKind) proto.Schema {
	for _, subSchema := range s {
		if protoSchema := subSchema.LookupResource(gvk); protoSchema != nil {
			return protoSchema
		}
	}
	return nil
}

func (s schemas) versionSet() chartutil.VersionSet {
	allVersions := set.NewStringSet()
	for _, subSchema := range s {
		for gvk := range subSchema.allGVKs {
			prefix := ""
			if gvk.Group != "" {
				allVersions.Add(gvk.Group)
				prefix = gvk.Group + "/"
			}
			allVersions.Add(prefix + gvk.Version)
			allVersions.Add(prefix + gvk.Version + "/" + gvk.Kind)
		}
	}
	return allVersions.AsSortedSlice(func(a, b string) bool { return a < b })
}
