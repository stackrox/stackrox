package generator

import (
	"bytes"
	"fmt"
	"sort"
	"text/template"
)

var (
	wellKnownTypes = map[string]int{
		"schema":       -4,
		"Query":        -3,
		"Mutation":     -2,
		"Subscription": -1,
	}
)

const (
	schema2template = `
{{ range $typeIndex, $type := .Entries }}
{{- if .Enums }}
enum {{ .Name }} {
{{- range .Enums }}
    {{ . }}
{{- end }}
}
{{- else if .Unions -}}
union {{ .Name }} =
{{- range $ui, $ud := .Unions }}{{if $ui }} |{{ end }} {{ . }}{{- end }}
{{- else }}
{{- if $typeIndex }}type {{ end }}{{ .Name }} {
{{- range .Fields }}
	{{ . }}
{{- end }}
{{- range .ExtraFields }}
	{{ . }}
{{- end }}
}
{{- end }}

{{end -}}
scalar Time
`
)

type typeEntry struct {
	name                             string
	enumValues                       []string
	unionValues                      []string
	definedResolvers, extraResolvers []string
	listResolvers                    map[string]struct{}
}

func (t *typeEntry) Enums() []string {
	return t.enumValues
}

func (t *typeEntry) Unions() []string {
	return t.unionValues
}

func (t *typeEntry) Name() string {
	return t.name
}

func (t *typeEntry) Fields() []string {
	return t.definedResolvers
}

func (t *typeEntry) ExtraFields() []string {
	sort.Slice(t.extraResolvers, func(i, j int) bool {
		return t.extraResolvers[i] < t.extraResolvers[j]
	})
	return t.extraResolvers
}

type schemaBuilderImpl struct {
	entries map[string]*typeEntry
}

// SchemaBuilder is a builder for schemas
type SchemaBuilder interface {
	AddType(name string, resolvers []string) error
	AddListType(name string, resolvers []string, listResolvers map[string]struct{}) error
	AddEnumType(name string, values []string) error
	AddUnionType(name string, types []string) error
	AddExtraResolver(name string, resolver string) error
	AddQuery(resolver string) error
	Render() (string, error)
}

// NewSchemaBuilder returns an empty schema builder for creating schemas
func NewSchemaBuilder() SchemaBuilder {
	val := &schemaBuilderImpl{
		entries: make(map[string]*typeEntry),
	}
	val.entries["schema"] = &typeEntry{name: "schema"}
	return val
}

func (s *schemaBuilderImpl) AddType(name string, resolvers []string) error {
	return s.AddListType(name, resolvers, nil)
}

func (s *schemaBuilderImpl) AddListType(name string, resolvers []string, listResolvers map[string]struct{}) error {
	if _, ok := s.entries[name]; ok {
		return fmt.Errorf("already type registered with name %q", name)
	}
	s.entries[name] = &typeEntry{
		name:             name,
		definedResolvers: resolvers,
		listResolvers:    listResolvers,
	}
	return nil
}

func (s *schemaBuilderImpl) AddEnumType(name string, values []string) error {
	if _, ok := s.entries[name]; ok {
		return fmt.Errorf("already type registered with name %q", name)
	}
	s.entries[name] = &typeEntry{
		name:       name,
		enumValues: values,
	}
	return nil
}

func (s *schemaBuilderImpl) AddUnionType(name string, types []string) error {
	if _, ok := s.entries[name]; ok {
		return fmt.Errorf("already type registered with name %q", name)
	}
	s.entries[name] = &typeEntry{
		name:        name,
		unionValues: types,
	}
	return nil
}

func (s *schemaBuilderImpl) AddExtraResolver(name string, resolver string) error {
	entry, ok := s.entries[name]
	if !ok {
		return fmt.Errorf("no type data for %q (known: %v)", name, s.entries)
	}
	if entry.definedResolvers == nil {
		return fmt.Errorf("%q is an invalid type for adding resolvers", name)
	}
	entry.extraResolvers = append(entry.extraResolvers, resolver)
	return nil
}

func (s *schemaBuilderImpl) AddQuery(resolver string) error {
	const q = "Query"
	return s.addBuiltin(q, resolver)
}

func (s *schemaBuilderImpl) addBuiltin(name, resolver string) error {
	// should be easy to extend this to support mutations and subscriptions
	entry, ok := s.entries[name]
	if !ok {
		entry = &typeEntry{name: name}
		s.entries[name] = entry
		s.entries["schema"].definedResolvers = append(s.entries["schema"].definedResolvers, "query: Query")
	}
	entry.extraResolvers = append(entry.extraResolvers, resolver)
	return nil
}

func (s *schemaBuilderImpl) Render() (string, error) {
	entries := make([]*typeEntry, 0, len(s.entries))
	for _, v := range s.entries {
		entries = append(entries, v)
	}
	sort.Slice(entries, func(i, j int) bool {
		ni, nj := entries[i].name, entries[j].name
		wi, wj := wellKnownTypes[ni], wellKnownTypes[nj]
		if wi == wj {
			return entries[i].name < entries[j].name
		}
		return wi < wj
	})
	t, err := template.New("schema").Parse(
		schema2template)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	err = t.Execute(&buf, struct{ Entries []*typeEntry }{entries})
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}
