package main

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strings"

	"github.com/stackrox/rox/pkg/protoreflect"
)

type function struct {
	t1      reflect.Type
	t2      reflect.Type
	written bool
}

func sliceFuncName(t1, t2 reflect.Type) string {
	return fmt.Sprintf("ConvertSlice%sTo%s", normalizeName(t1.String()), normalizeName(t2.String()))
}

func funcName(t1, t2 reflect.Type) string {
	return fmt.Sprintf("Convert%sTo%s", normalizeName(t1.String()), normalizeName(t2.String()))
}

func (s *converter) addFunc(t1, t2 reflect.Type) string {
	name := funcName(t1, t2)
	if _, ok := s.neededFunctions[name]; ok {
		return name
	}
	s.neededFunctions[name] = &function{
		t1: t1,
		t2: t2,
	}
	return name
}

type converter struct {
	numIndents      int
	neededFunctions map[string]*function
}

func (s *converter) sortedFunctions() []string {
	var names []string
	for k := range s.neededFunctions {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

// walk iterates over the obj and creates the necessary functions to properly convert
func walk(w io.Writer, t1, t2 reflect.Type) {
	sw := converter{
		neededFunctions: make(map[string]*function),
	}

	buf := bytes.NewBuffer(nil)

	sw.createSliceFunc(buf, t1, t2)
	sw.createFunc(buf, t1, t2)

	// Keep iterating until all required functions are written
	for {
		allWritten := true
		for _, k := range sw.sortedFunctions() {
			v := sw.neededFunctions[k]
			if v.written {
				continue
			}
			allWritten = false
			sw.createSliceFunc(w, v.t1, v.t2)
			sw.createFunc(w, v.t1, v.t2)
			v.written = true
		}
		if allWritten {
			break
		}
	}

	if _, err := io.Copy(w, buf); err != nil {
		panic(err)
	}
}

func normalizeName(v string) string {
	v = strings.TrimPrefix(v, "*")
	v = strings.ReplaceAll(v, ".", "")
	return strings.Title(v)
}

func (s *converter) createSliceFunc(w io.Writer, t1, t2 reflect.Type) {
	name := sliceFuncName(t1, t2)
	singleFuncName := funcName(t1, t2)
	src, dst := t1.String(), t2.String()
	s.printf(w, "// %s converts a slice of %s to a slice of %s", name, src, dst)
	s.printf(w, "func %s(p1 []%s) []%s {", name, src, dst)
	s.indent()
	s.printf(w, "if p1 == nil {")
	s.indent()
	s.printf(w, "return nil")
	s.unindent()
	s.printf(w, "}")
	s.printf(w, "p2 := make([]%s, 0, len(p1))", t2.String())
	s.printf(w, "for _, v := range p1 {")
	s.indent()
	s.printf(w, "p2 = append(p2, %s(v))", singleFuncName)
	s.unindent()
	s.printf(w, "}")
	s.printf(w, "return p2")
	s.unindent()
	s.printf(w, "}")
	s.printf(w, "")
}

func (s *converter) createFunc(w io.Writer, t1, t2 reflect.Type) {
	name := funcName(t1, t2)
	src, dst := t1.String(), t2.String()
	s.printf(w, "// %s converts from %s to %s", name, src, dst)
	s.printf(w, "func %s(p1 %s) %s {", name, src, dst)
	s.indent()
	s.printf(w, "if p1 == nil {")
	s.indent()
	s.printf(w, "return nil")
	s.unindent()
	s.printf(w, "}")
	s.printf(w, "p2 := new(%s)", t2.Elem().String())
	s.handleStruct(w, t1.Elem(), t2.Elem())
	s.printf(w, "return p2")
	s.unindent()
	s.printf(w, "}")
	s.printf(w, "")
}

func (s *converter) printf(w io.Writer, template string, args ...interface{}) {
	fmt.Fprintf(w, "%s%s\n", strings.Repeat("  ", s.numIndents), fmt.Sprintf(template, args...))
}

func (s *converter) indent() {
	s.numIndents += 1
}

func (s *converter) unindent() {
	s.numIndents -= 1
}

// handleStruct takes in a struct object and properly handles all of the fields
func (s *converter) handleStruct(w io.Writer, original reflect.Type, new reflect.Type) {
	for i := 0; i < original.NumField(); i++ {
		field := original.Field(i)
		if strings.HasPrefix(field.Name, "XXX") {
			continue
		}
		newField := new.Field(i)
		switch field.Type.Kind() {
		case reflect.Ptr:
			if field.Type.String() == "*types.Timestamp" {
				s.printf(w, "p2.%s = p1.%s.Clone()", field.Name, field.Name)
				continue
			}
			if field.Type.String() == "*types.Any" {
				s.printf(w, "p2.%s = p1.%s.Clone()", field.Name, field.Name)
				continue
			}
			s.printf(w, "p2.%s = %s(p1.%s)", field.Name, s.addFunc(field.Type, newField.Type), field.Name)
		case reflect.Slice:
			s.printf(w, "if p1.%s != nil {", field.Name)
			s.indent()
			s.printf(w, "p2.%s = make(%s, len(p1.%s))", field.Name, newField.Type, field.Name)
			s.printf(w, "for idx := range p1.%s {", field.Name)
			s.indent()
			switch field.Type.Elem().Kind() {
			case reflect.Struct:
				panic("shouldn't be possible in proto")
			case reflect.Ptr:
				s.printf(w, "p2.%s[idx] = %s(p1.%s[idx])", field.Name, s.addFunc(field.Type.Elem(), newField.Type.Elem()), field.Name)
			case reflect.Int32:
				_, ok := reflect.Zero(field.Type.Elem()).Interface().(protoreflect.ProtoEnum)
				if !ok {
					s.printf(w, "p2.%s[idx] = p1.%s[idx]", field.Name, field.Name)
				} else {
					s.printf(w, "p2.%s[idx] = %s(p1.%s[idx])", field.Name, newField.Type.Elem().String(), field.Name)
				}
			case reflect.Slice:
				switch field.Type.Elem().Elem().Kind() {
				case reflect.Uint8:
					s.printf(w, "p2.%s[idx] = make([]byte, len(p1.%s[idx]))", field.Name, field.Name)
					s.printf(w, "copy(p2.%s[idx], p1.%s[idx])", field.Name, field.Name)
				default:
					panic("expect slice of slice to only be uint8 in proto")
				}
			default:
				s.printf(w, "p2.%s[idx] = p1.%s[idx]", field.Name, field.Name)
			}
			s.unindent()
			s.printf(w, "}")
			s.unindent()
			s.printf(w, "}")
		case reflect.Struct:
			s.handleStruct(w, field.Type, newField.Type)
		case reflect.Map:
			s.printf(w, "if p1.%s != nil {", field.Name)
			s.indent()
			s.printf(w, "p2.%s = make(map[%s]%s, len(p1.%s))", field.Name, field.Type.Key().String(), newField.Type.Elem().String(), field.Name)
			s.printf(w, "for k, v := range p1.%s {", field.Name)
			s.indent()
			switch field.Type.Elem().Kind() {
			case reflect.Ptr:
				s.printf(w, "p2.%s[k] = %s(v)", field.Name, s.addFunc(field.Type.Elem(), newField.Type.Elem()))
			case reflect.Struct:
				panic("shouldn't happen in protobuf")
			case reflect.Slice:
				switch field.Type.Elem().Elem().Kind() {
				case reflect.Uint8:
					s.printf(w, "p2.%s[k] = make([]byte, len(v))", field.Name)
					s.printf(w, "copy(p2.%s[k], v)", field.Name)
				default:
					panic("expecting only slice of bytes as map value")
				}
			default:
				s.printf(w, "p2.%s[k] = v", field.Name)
			}
			s.unindent()
			s.printf(w, "}")
			s.unindent()
			s.printf(w, "}")
		case reflect.String, reflect.Bool:
			s.printf(w, "p2.%s = p1.%s", field.Name, field.Name)
		case reflect.Uint32, reflect.Uint64, reflect.Int32, reflect.Int64, reflect.Float32, reflect.Float64:
			_, ok := reflect.Zero(field.Type).Interface().(protoreflect.ProtoEnum)
			if !ok {
				s.printf(w, "p2.%s = p1.%s", field.Name, field.Name)
				continue
			}
			s.printf(w, "p2.%s = %s(p1.%s)", field.Name, newField.Type.String(), field.Name)
		case reflect.Interface:
			//If it is a oneof then call XXX_OneofWrappers to get the types.
			//The return values is a slice of interfaces that are nil type pointers
			if field.Tag.Get("protobuf_oneof") != "" {
				s.printf(w, "if p1.%s != nil {", field.Name)
				s.indent()

				oneofWrappers := reflect.Zero(reflect.PtrTo(original)).Interface().(interface{ XXX_OneofWrappers() []interface{} }).XXX_OneofWrappers()
				newOneofWrappers := reflect.Zero(reflect.PtrTo(new)).Interface().(interface{ XXX_OneofWrappers() []interface{} }).XXX_OneofWrappers()

				for idx, wrapper := range oneofWrappers {
					wrapperTy := reflect.TypeOf(wrapper)
					if !wrapperTy.Implements(field.Type) {
						continue
					}
					newWrapperTy := reflect.TypeOf(newOneofWrappers[idx])
					s.printf(w, "if val, ok := p1.%s.(%s); ok {", field.Name, wrapperTy.String())
					s.indent()
					s.printf(w, "p2.%s = %s(val)", field.Name, s.addFunc(wrapperTy, newWrapperTy))
					s.unindent()
					s.printf(w, "}")
				}

				s.unindent()
				s.printf(w, "}")
			}
		}
	}
}
