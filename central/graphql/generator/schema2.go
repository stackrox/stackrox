package generator

import (
	"bytes"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"text/template"

	"github.com/stackrox/rox/pkg/protoreflect"
	"github.com/stackrox/rox/pkg/utils"
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
{{- else if .InterfaceFields -}}
interface {{ .Name }} {
{{- range .InterfaceFields }}
    {{ . }}
{{- end }}
}
{{- else if .Unions -}}
union {{ .Name }} =
{{- range $ui, $ud := .Unions }}{{ if $ui }} |{{ end }} {{ . }}{{- end }}
{{- else if .InputFields -}}
input {{ .Name }} {
{{- range .InputFields }}
	{{ . }}
{{- end }}
}
{{- else }}
{{- if $typeIndex }}type {{ .Name }} {{- range $ui, $ud := .Interfaces }}{{ if $ui }} &{{ else }} implements {{- end }} {{ . }}{{- end }}{{ else }}{{ .Name }} {{- end }} {
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
	// todo: add "AddScalar" API and stop hard-coding Time
)

type typeEntry struct {
	name                             string
	interfaces                       []string
	enumValues                       []string
	interfaceFields                  []string
	unionValues                      []string
	definedResolvers, extraResolvers []string
	inputFields                      []string
}

func (t *typeEntry) Enums() []string {
	return t.enumValues
}

func (t *typeEntry) Interfaces() []string {
	return t.interfaces
}

func (t *typeEntry) Unions() []string {
	return t.unionValues
}

func (t *typeEntry) Name() string {
	return t.name
}

func (t *typeEntry) InterfaceFields() []string {
	return t.interfaceFields
}

func (t *typeEntry) Fields() []string {
	return t.definedResolvers
}

func (t *typeEntry) InputFields() []string {
	return t.inputFields
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
	AddType(name string, resolvers []string, interfaces ...string) error
	AddInput(name string, fields []string) error
	AddEnumType(name string, values []string) error
	AddInterfaceType(name string, fields []string) error
	AddUnionType(name string, types []string) error
	AddExtraResolver(name string, resolver string) error
	AddExtraResolvers(name string, resolver []string) error
	AddQuery(resolver string) error
	// Deprecated: Use REST APIs instead
	AddMutation(resolver string) error
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

func (s *schemaBuilderImpl) AddType(name string, resolvers []string, interfaces ...string) error {
	if _, ok := s.entries[name]; ok {
		return fmt.Errorf("already type registered with name %q", name)
	}
	s.entries[name] = &typeEntry{
		name:             name,
		interfaces:       interfaces,
		definedResolvers: resolvers,
	}
	return nil
}

func (s *schemaBuilderImpl) AddInput(name string, fields []string) error {
	if _, ok := s.entries[name]; ok {
		return fmt.Errorf("already type registered with name %q", name)
	}
	s.entries[name] = &typeEntry{
		name:        name,
		inputFields: fields,
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

func (s *schemaBuilderImpl) AddInterfaceType(name string, fields []string) error {
	if _, ok := s.entries[name]; ok {
		return fmt.Errorf("already type registered with name %q", name)
	}
	s.entries[name] = &typeEntry{
		name:            name,
		interfaceFields: fields,
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

func (s *schemaBuilderImpl) AddExtraResolvers(name string, resolvers []string) error {
	entry, ok := s.entries[name]
	if !ok {
		return fmt.Errorf("no type data for %q (known: %v)", name, s.entries)
	}
	if entry.definedResolvers == nil {
		return fmt.Errorf("%q is an invalid type for adding resolvers", name)
	}
	entry.extraResolvers = append(entry.extraResolvers, resolvers...)
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
	return s.addBuiltin("Query", resolver)
}

// Deprecated: Use REST APIs instead
func (s *schemaBuilderImpl) AddMutation(resolver string) error {
	return s.addBuiltin("Mutation", resolver)
}

func (s *schemaBuilderImpl) addBuiltin(name, resolver string) error {
	// should be easy to extend this to support mutations and subscriptions
	entry, ok := s.entries[name]
	if !ok {
		entry = &typeEntry{name: name}
		s.entries[name] = entry
		s.entries["schema"].definedResolvers = append(s.entries["schema"].definedResolvers, fmt.Sprintf("%s: %s", strings.ToLower(name), name))
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

// RegisterProtoEnum is a utility method used by the generated code to output enums
func RegisterProtoEnum(builder SchemaBuilder, typ reflect.Type) {
	enumDesc, err := protoreflect.GetEnumDescriptor(reflect.Zero(typ).Interface().(protoreflect.ProtoEnum))
	if err != nil {
		panic(err)
	}

	values := make([]string, 0, len(enumDesc.GetValue()))
	for _, valueDesc := range enumDesc.GetValue() {
		values = append(values, valueDesc.GetName())
	}
	utils.Must(builder.AddEnumType(typ.Name(), values))
}
