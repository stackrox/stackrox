package walker

import (
	"fmt"
	"log"
	"reflect"
	"strings"

	"github.com/stackrox/rox/pkg/protoreflect"
	"github.com/stackrox/rox/pkg/stringutils"
)

type Table struct {
	Parent       *Table
	TopLevel 	 bool
	Field        string
	RawFieldType string
	Elems    []Element
	Embedded []*Table
	Children []*Table
	OneOf 	bool
	SearchField string
}

func (t *Table) Elements() []Element {
	var elems []Element
	for _, elem := range t.Elems {
		if !elem.IsSearchable() {
			continue
		}
		elems = append(elems, elem)
	}
	for _, child := range t.Embedded {
		childPairs := child.Elements()
		elems = append(elems, childPairs...)
	}
	return elems
}

func (p *Table) SearchFieldsToElement() map[string]Element {
	m := make(map[string]Element)
	for _, elem := range p.Elements() {
		if elem.IsSearchable() {
			if _, ok := m[elem.SearchField]; !ok {
				m[elem.SearchField] = elem
			}
		}
	}
	for _, child := range p.Embedded {
		for k, v := range child.SearchFieldsToElement() {
			if _, ok := m[k]; !ok {
				m[k] = v
			}
		}
	}
	for _, child := range p.Children {
		for k, v := range child.SearchFieldsToElement() {
			m[k] = v
		}
	}
	return m
}

func (p *Table) GetInsertComposer(level int) *InsertComposer {
	ic := &InsertComposer{
		Table: p.TableName(),
	}

	for _, elem := range p.Elements() {
		ic.AddSQL(elem.SQLPath())
		ic.AddExcluded(elem.SQLPath())
		getterPath := fmt.Sprintf("obj%d.", level) + elem.GetterPath()
		if elem.DataType == DATETIME {
			getterPath = fmt.Sprintf("nilOrStringTimestamp(%s)", getterPath)
		} else if elem.DataType == INT_ARRAY {
			getterPath = fmt.Sprintf("convertEnumSliceToIntArray(%s)", getterPath)
		}
		ic.AddGetters(getterPath)
	}
	return ic
}

func (p *Table) AbsGetterPath() string {
	if p.Field == "" {
		return ""
	}
	getter := fmt.Sprintf("Get%s()", p.Field)
	if p.Parent == nil {
		return getter
	}
	if p.Parent.TopLevel {
		return getter
	}
	if getterPath := p.Parent.GetterPath(); getterPath != "" {
		return getterPath + "." + getter
	}
	return getter
}

func (p *Table) GetterPath() string {
	if p.Field == "" || p.TopLevel {
		return ""
	}
	getter := fmt.Sprintf("Get%s()", p.Field)
	if p.Parent == nil {
		return getter
	}
	if p.OneOf {
		return p.Parent.GetterPath()
	}
	if getterPath := p.Parent.GetterPath(); getterPath != "" {
		return getterPath + "." + getter
	}
	return getter
}

func (p *Table) PrimaryKeyElements() []Element {
	// This means top level
	if p.Parent == nil {
		var pks []Element
		for _, elem := range p.Elems {
			if elem.Options.PrimaryKey {
				pks = append(pks, elem)
			}
		}
		return pks
	}
	if !p.TopLevel {
		return p.Parent.PrimaryKeyElements()
	}
	parentKeys := p.Parent.PrimaryKeyElements()
	for i := range parentKeys {
		parentKeys[i].Field = "parent_"+parentKeys[i].Field
	}

	parentKeys = append(parentKeys, Element{
		DataType:     INTEGER,
		Field:        "idx",
		Parent:       p,
	})
	return parentKeys
}

func (p *Table) TableName() string {
	if p.Parent == nil {
		return p.RawFieldType
	}
	if !p.TopLevel {
		return p.Parent.TableName()
	}
	if parentPath := p.Parent.TableName(); parentPath != "" {
		return parentPath + "_" + p.Field
	}
	return p.RawFieldType
}

func (p *Table) SQLPath() string {
	if p.Parent == nil {
		return p.Field
	}
	if p.TopLevel {
		return ""
	}
	if parentPath := p.Parent.SQLPath(); parentPath != "" {
		return parentPath + "_" + p.Field
	}
	return p.Field
}

func (s Table) Print(indent string, searchOnly bool) {
	fmt.Println(indent, s.Field, s.RawFieldType, s.OneOf)
	fmt.Println(indent, "  ", "fields:")
	for _, elem := range s.Elems {
		if !searchOnly || elem.IsSearchable() {
			fmt.Println(indent, "    ", elem.Field, elem.DataType.String(), elem.RawFieldType)
		}
	}
	if len(s.Embedded) > 0 {
		fmt.Println(indent, "embedded:")
	}
	for _, child := range s.Embedded {
		child.Print("    ", searchOnly)
	}
	if len(s.Children) > 0 {
		fmt.Println(indent, "tables:")
	}
	for _, table := range s.Children {
		table.Print("    ", searchOnly)
	}
	fmt.Println()
}

type Element struct {
	Parent       *Table
	DataType     DataType
	Field        string
	RawFieldType string
	Slice        bool
	SearchField string
	Options PostgresOptions
}

func (e Element) TableName() string {
	return e.Parent.TableName()
}

func (e Element) IsSearchable() bool {
	return e.SearchField != ""
}

func (e Element) TablePrefixed() string {
	return fmt.Sprintf("%s.%s", e.Parent.TableName(), e.SQLPath())
}

func (e Element) GetterPath() string {
	getter := fmt.Sprintf("Get%s()", e.Field)
	if e.Parent.TopLevel {
		return getter
	}
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

type SQLWalker struct {
	table string
}

// Walk iterates over the obj and creates a search.Map object from the found struct tags
func Walk(obj reflect.Type, table string) *Table {
	typ := obj.Elem()
	parent := &Table{
		RawFieldType: table,
		TopLevel: true,
	}
	walker := SQLWalker{}
	walker.handleStruct(parent, typ)

	// Validate there is a pk
	if len(parent.PrimaryKeyElements()) == 0 {
		log.Printf("table %s needs PK", table)
	}

	return parent
}

type PostgresOptions struct {
	Ignored bool
	Index   string
	PrimaryKey bool
}

const defaultIndex = "btree"

func getPostgresOptions(tag string) PostgresOptions {
	var opts PostgresOptions

	for _, field := range strings.Split(tag, ",") {
		switch {
		case field == "-":
			opts.Ignored = true
		case strings.HasPrefix(field, "index"):
			if strings.Contains(field, "=") {
				opts.Index = stringutils.GetAfter(field, "=")
			} else {
				opts.Index = defaultIndex
			}
		case field == "pk":
			opts.PrimaryKey = true
		case field == "":
		default:
			// ignore for just right now
			panic(fmt.Sprintf("unknown case: %s", field))
		}
	}
	return opts
}

func hasSearchField(tag string) bool {
	return tag != "" && tag != "-"
}

// handleStruct takes in a struct object and properly handles all of the fields
func (s *SQLWalker) handleStruct(parent *Table, original reflect.Type) {
	for i := 0; i < original.NumField(); i++ {
		field := original.Field(i)
		if strings.HasPrefix(field.Name, "XXX") {
			continue
		}
		opts := getPostgresOptions(field.Tag.Get("sql"))
		if opts.Ignored {
			continue
		}

		var searchField string
		searchTag := field.Tag.Get("search")
		if searchTag == "-" {
			continue
		} else if searchTag != "" {
			fields := strings.Split(searchTag, ",")
			searchField = fields[0]
		} else if opts.PrimaryKey {
			searchField = field.Name
		}
		elem := Element{
			Parent:       parent,
			Field:        field.Name,
			RawFieldType: field.Type.String(),
			SearchField: searchField,
			Options: opts,
		}
		switch field.Type.Kind() {
		case reflect.Ptr:
			if field.Type.String() == "*types.Timestamp" {
				elem.DataType = DATETIME
				parent.Elems = append(parent.Elems, elem)
				continue
			}
			child := &Table{
				Parent:       parent,
				Field:        field.Name,
				RawFieldType: field.Type.String(),
			}
			parent.Embedded = append(parent.Embedded, child)

			s.handleStruct(child, field.Type.Elem())
		case reflect.Slice:
			elemType := field.Type.Elem()

			switch elemType.Kind() {
			case reflect.String:
				parent.Elems = append(parent.Elems, Element{
					Parent:       parent,
					DataType:     STRING_ARRAY,
					Field:        field.Name,
					Slice:        true,
					SearchField: searchField,
				})
				continue
			case reflect.Uint32, reflect.Uint64, reflect.Int32, reflect.Int64:
				parent.Elems = append(parent.Elems, Element{
					Parent:       parent,
					DataType:     INT_ARRAY,
					Field:        field.Name,
					Slice:        true,
					SearchField: searchField,
				})
				continue
			}

			table := &Table{
				Parent:       parent,
				Field:        field.Name,
				RawFieldType: field.Type.String(),
				TopLevel: true,
			}
			parent.Children = append(parent.Children, table)

			s.handleStruct(table, field.Type.Elem().Elem())
			continue
		case reflect.Struct:
			child := &Table{
				Parent:       parent,
				Field:        field.Name,
				RawFieldType: field.Type.String(),
			}
			parent.Embedded = append(parent.Embedded, child)
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
			enum, ok := reflect.Zero(field.Type).Interface().(protoreflect.ProtoEnum)
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
			// If it is a oneof then call XXX_OneofWrappers to get the types.
			// The return values is a slice of interfaces that are nil type pointers
			if field.Tag.Get("protobuf_oneof") != "" {
				ptrToOriginal := reflect.PtrTo(original)

				methodName := fmt.Sprintf("Get%s", field.Name)
				oneofGetter, ok := ptrToOriginal.MethodByName(methodName)
				if !ok {
					panic("didn't find oneof function, did the naming change?")
				}
				oneofInterfaces := oneofGetter.Func.Call([]reflect.Value{reflect.New(original)})
				if len(oneofInterfaces) != 1 {
					panic(fmt.Sprintf("found %d interfaces returned from oneof getter", len(oneofInterfaces)))
				}

				oneofInterface := oneofInterfaces[0].Type()

				method, ok := ptrToOriginal.MethodByName("XXX_OneofWrappers")
				if !ok {
					panic(fmt.Sprintf("XXX_OneofWrappers should exist for all protobuf oneofs, not found for %s", original.Name()))
				}
				out := method.Func.Call([]reflect.Value{reflect.New(original)})
				actualOneOfFields := out[0].Interface().([]interface{})
				for _, f := range actualOneOfFields {
					typ := reflect.TypeOf(f)
					if typ.Implements(oneofInterface) {
						child := &Table{
							Parent:       parent,
							Field:        field.Name,
							RawFieldType: field.Type.String(),
							OneOf: true,
						}
						parent.Embedded = append(parent.Embedded, child)
						s.handleStruct(child, typ.Elem())
					}
				}
				continue
			}
			panic("non-oneof interface is not handled")
		default:
			panic(fmt.Sprintf("Type %s for field %s is not currently handled", original.Kind(), field.Name))
		}
	}
}
