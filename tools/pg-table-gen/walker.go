package main

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/stackrox/rox/pkg/protoreflect"
)

type Path struct {
	Parent       *Path
	Field        string
	RawFieldType string
	Elems        []Element
	Children     []*Path
}

func (p *Path) GetterPath() string {
	if p.Field == "" {
		return ""
	}
	getter := fmt.Sprintf("Get%s()", p.Field)
	if p.Parent == nil {
		return getter
	}
	if getterPath := p.Parent.GetterPath(); getterPath != "" {
		return getterPath + "." + getter
	}
	return getter
}

func (p *Path) SQLPath() string {
	if p.Parent == nil {
		return p.Field
	}
	if parentPath := p.Parent.SQLPath(); parentPath != "" {
		return parentPath + "_" + p.Field
	}
	return p.Field
}

func (s Path) Print(indent string) {
	fmt.Println(indent, s.Field, s.RawFieldType)
	fmt.Println(indent, "  ", "fields:")
	for _, elem := range s.Elems {
		fmt.Println(indent, "    ", elem.Field, elem.DataType.String(), elem.RawFieldType)
	}
	for _, child := range s.Children {
		child.Print("  ")
	}
}

type Element struct {
	Parent       *Path
	DataType     DataType
	Field        string
	RawFieldType string
	Slice        bool
}

func (e Element) GetterPath() string {
	getter := fmt.Sprintf("Get%s()", e.Field)
	if getterPath := e.Parent.GetterPath(); getterPath != "" {
		return e.Parent.GetterPath() + "." + getter
	}
	return getter
}

func (e Element) SQLPath() string {
	if parentPath := e.Parent.SQLPath(); parentPath != "" {
		return parentPath + "_" + e.Field
	}
	return e.Field
}

type searchWalker struct {
	table string
}

// Walk iterates over the obj and creates a search.Map object from the found struct tags
func Walk(obj interface{}) *Path {
	typ := reflect.TypeOf(obj).Elem()
	parent := &Path{
		RawFieldType: strings.TrimPrefix(typ.String(), "storage."),
	}
	walker := searchWalker{}
	walker.handleStruct(parent, typ)

	parent.Print("")
	return parent
}

type PostgresOptions struct {
	Ignored bool
}

func getPostgresOptions(tag string) *PostgresOptions {
	opts := &PostgresOptions{}

	for _, field := range strings.Split(tag, ",") {
		switch field {
		case "-":
			opts.Ignored = true
		case "":
		default:
			// ignore for just right now
			//panic(fmt.Sprintf("unknown case: %s", field))
		}
	}
	return opts
}

// handleStruct takes in a struct object and properly handles all of the fields
func (s *searchWalker) handleStruct(parent *Path, original reflect.Type) {
	for i := 0; i < original.NumField(); i++ {
		field := original.Field(i)
		if strings.HasPrefix(field.Name, "XXX") {
			continue
		}
		opts := getPostgresOptions(field.Tag.Get("postgres"))
		if opts.Ignored {
			continue
		}

		elem := Element{
			Parent:       parent,
			Field:        field.Name,
			RawFieldType: field.Type.String(),
		}

		switch field.Type.Kind() {
		case reflect.Ptr:
			child := &Path{
				Parent:       parent,
				Field:        field.Name,
				RawFieldType: field.Type.String(),
			}
			parent.Children = append(parent.Children, child)

			s.handleStruct(child, field.Type.Elem())
		case reflect.Slice:
			parent.Elems = append(parent.Elems, Element{
				Parent:   parent,
				DataType: JSONB,
				Field:    field.Name,
				Slice:    true,
			})
			continue
		case reflect.Struct:
			child := &Path{
				Parent:       parent,
				Field:        field.Name,
				RawFieldType: field.Type.String(),
			}
			parent.Children = append(parent.Children, child)
			s.handleStruct(child, field.Type)
		case reflect.Map:
			elem.DataType = MAP
			parent.Elems = append(parent.Elems, elem)
			continue
		case reflect.String:
			elem.DataType = STRING
			parent.Elems = append(parent.Elems, elem)
			continue
		case reflect.Bool:
			elem.DataType = BOOL
			parent.Elems = append(parent.Elems, elem)
			continue
		case reflect.Uint32, reflect.Uint64, reflect.Int32, reflect.Int64, reflect.Float32, reflect.Float64:
			enum, ok := reflect.Zero(original).Interface().(protoreflect.ProtoEnum)
			if !ok {
				elem.DataType = NUMERIC
				parent.Elems = append(parent.Elems, elem)
				continue
			}
			_, err := protoreflect.GetEnumDescriptor(enum)
			if err != nil {
				panic(err)
			}
			elem.DataType = ENUM
			parent.Elems = append(parent.Elems, elem)
			continue
		case reflect.Interface:
			// These are jsonb for now
			elem.DataType = JSONB
			parent.Elems = append(parent.Elems, elem)
			continue
		default:
			panic(fmt.Sprintf("Type %s for field %s is not currently handled", original.Kind(), field.Name))
		}
	}
}
