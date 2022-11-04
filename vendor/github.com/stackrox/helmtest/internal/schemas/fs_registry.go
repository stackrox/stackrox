package schemas

import (
	"io/fs"
	"sync"

	openapi_v2 "github.com/google/gnostic/openapiv2"
	"github.com/pkg/errors"
	"github.com/stackrox/helmtest/internal/rox-imported/gziputil"
)

type fsBasedRegistry struct {
	fs fs.FS

	mutex   sync.RWMutex
	schemas map[string]*schemaEntry
}

func newFSBasedRegistry(fs fs.FS) *fsBasedRegistry {
	return &fsBasedRegistry{
		fs:      fs,
		schemas: make(map[string]*schemaEntry),
	}
}

type schemaEntry struct {
	schema  Schema
	loadErr error
}

func (r *fsBasedRegistry) GetSchema(name string) (Schema, error) {
	if e := r.getCachedSchema(name); e != nil {
		return e.schema, e.loadErr
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()
	if e := r.schemas[name]; e != nil {
		return e.schema, e.loadErr
	}
	s, err := r.loadSchema(name)
	r.schemas[name] = &schemaEntry{
		schema:  s,
		loadErr: err,
	}
	return s, err
}

func (r *fsBasedRegistry) getCachedSchema(name string) *schemaEntry {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return r.schemas[name]
}

func (r *fsBasedRegistry) loadSchema(name string) (Schema, error) {
	schemaBytes, err := fs.ReadFile(r.fs, name+".json.gz")
	if err != nil {
		return nil, errors.Wrapf(err, "no schema found for name %q", name)
	}

	openapiDoc, err := gziputil.Decompress(schemaBytes)
	if err != nil {
		return nil, errors.Wrapf(err, "failed reading openapi docs for schema %s", name)
	}

	doc, err := openapi_v2.ParseDocument(openapiDoc)
	if err != nil {
		return nil, errors.Wrapf(err, "parsing OpenAPI doc for schema %s", name)
	}
	schema, err := newSchema(doc)
	if err != nil {
		return nil, errors.Wrapf(err, "creating OpenAPI schema from document for schema %s", name)
	}

	return schema, nil
}
