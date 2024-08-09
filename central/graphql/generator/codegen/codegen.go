package codegen

import (
	"bytes"
	_ "embed"
	"fmt"
	"io"
	"reflect"
	"strings"
	"text/template"

	"github.com/stackrox/rox/central/graphql/generator"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/set"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

//go:embed codegen.go.tpl
var codegen string
var templates = map[string]string{
	"codegen":      codegen,
	"enum":         `value.String()`,
	"enumslice":    `stringSlice(value)`,
	"float":        `float64(value)`,
	"id":           `graphql.ID(value)`,
	"int":          `int32(value)`,
	"label":        `labelsResolver(value)`,
	"pointer":      `resolver.root.wrap{{.Type.Elem.Name}}(value, true, nil)`,
	"pointerslice": `resolver.root.wrap{{plural .Type.Elem.Elem.Name}}(value, nil)`,
	"raw":          `value`,
	"rawslice":     `value`,
	"time":         `protocompat.ConvertTimestampToGraphqlTimeOrError(value)`,
}

func listName(td typeData) string {
	split := strings.Split(td.Package, "/")
	return fmt.Sprintf("%s.List%s", split[len(split)-1], td.Name)
}

func isEnum(p reflect.Type) bool {
	if p == nil {
		return false
	}
	fullName := protoreflect.FullName(strings.ReplaceAll(importedName(p), "_", "."))
	_, err := protoregistry.GlobalTypes.FindEnumByName(fullName)
	return err == nil
}

func getFieldTransform(fd fieldData) (templateName string, returnType string) {
	switch fd.Type.Kind() {
	case reflect.String:
		if fd.Name == "Id" {
			return "id", "graphql.ID"
		}
		return "raw", "string"
	case reflect.Int32:
		if isEnum(fd.Type) {
			return "enum", "string"
		}
		return "raw", "int32"
	case reflect.Uint32:
		return "int", "int32"
	case reflect.Int64:
		return "int", "int32"
	case reflect.Float32:
		return "float", "float64"
	case reflect.Float64:
		return "raw", "float64"
	case reflect.Bool:
		return "raw", "bool"
	case reflect.Uint8:
		return "raw", "byte"
	case reflect.Map:
		if fd.Type.Key().Kind() == reflect.String && fd.Type.Elem().Kind() == reflect.String {
			return "label", "labels"
		}
	case reflect.Ptr:
		if fd.Type == protocompat.TimestampPtrType {
			return "time", "(*graphql.Time, error)"
		}
		if fd.Type.Implements(messageType) {
			if isListType(fd.Type) {
				// if a field returns a list type, we don't automatically handle this for now.
				return "", ""
			}
			return "pointer", fmt.Sprintf("(*%sResolver, error)", lower(fd.Type.Elem().Name()))
		}
	case reflect.Slice:
		template, ret := getFieldTransform(fieldData{Name: fd.Name, Type: fd.Type.Elem()})
		if len(ret) > 0 && ret[0] == '(' {
			// this converts (*fooResolver, error) into ([]*fooResolver, error)
			return template + "slice", ret[0:1] + "[]" + ret[1:]
		}
		return template + "slice", "[]" + ret
	}
	return "", ""
}

func translator(t *template.Template) func(fieldData) string {
	return func(fd fieldData) string {
		tmplName, _ := getFieldTransform(fd)
		b := &bytes.Buffer{}
		err := t.ExecuteTemplate(b, tmplName, fd)
		if err != nil {
			panic(err)
		}
		return b.String()
	}
}

func valueType(fd fieldData) string {
	_, returnType := getFieldTransform(fd)
	return returnType
}

func listField(td schemaEntry, field fieldData) bool {
	t, ok := td.ListData[field.Name]
	return ok && t == field.Type
}

// GenerateResolvers produces go code for resolvers for all the types found by the typewalk.
func GenerateResolvers(parameters generator.TypeWalkParameters, writer io.Writer) {
	typeData := typeWalk(parameters)
	rootTemplate := template.New("codegen")
	rootTemplate.Funcs(template.FuncMap{
		"importedName": importedName,
		"isEnum":       isEnum,
		"listField":    listField,
		"listName":     listName,
		"lower":        lower,
		"nonListField": func(td schemaEntry, field fieldData) bool { return !listField(td, field) },
		"plural":       plural,
		"translator":   translator(rootTemplate),
		"valueType":    valueType,
		"schemaType":   schemaType,
	})
	for name, text := range templates {
		thisTemplate := rootTemplate
		if rootTemplate.Name() != name {
			thisTemplate = rootTemplate.New(name)
		}
		_, err := thisTemplate.Parse(text)
		if err != nil {
			panic(fmt.Sprintf("Template %q: %s", name, err))
		}
	}
	imports := set.NewStringSet()
	for _, td := range typeData {
		// Input types are not directly referenced in the generated Go code.
		if td.IsInputType {
			continue
		}
		if td.Package != "" {
			imports.Add(td.Package)
		}
	}
	entries := makeSchemaEntries(typeData)
	err := rootTemplate.Execute(writer, struct {
		Entries []schemaEntry
		Imports map[string]struct{}
	}{entries, imports})
	if err != nil {
		panic(err)
	}
}
