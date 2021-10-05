package main

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoreflect"
)

//go:generate stringer -type=DataType
type DataType int

const (
	BOOL     			DataType = 0
	NUMERIC  			DataType = 1
	STRING 				DataType = 2
	DATETIME  		    DataType = 3
	MAP       			DataType = 4
	ENUM      			DataType = 5
	ARRAY 				DataType = 6
	STRING_ARRAY 		DataType = 7
	STRUCT 				DataType = 8
)

func validateTable(t *Table) {
	var foundPK bool
	for _, f := range t.Fields {
		if f.pk {
			foundPK = true
			break
		}
	}
	if !foundPK {
		fmt.Printf("ERROR: did not find PK in fields for table %s\n", t.Name)
	}
	for _, child := range t.ChildTables {
		validateTable(child)
	}
}

func printTable(t *Table) {
	fmt.Println(t.Name, t.FieldName)
	for _, field := range t.Fields {
		fmt.Printf("\tname=%s, type=%s, pk=%t\n", field.Name, field.DataType, field.pk)
	}
	for _, child := range t.ChildTables {
		printTable(child)
	}
}

func main() {
	obj := (*storage.Deployment)(nil)

	table := &Table{
		Name: "Deployment",
		Type: reflect.ValueOf(obj).Type().String(),
	}
	walk(table, obj)

	// Enrich with foreign key fields

	genInsertion(table)

	//printTable(table)
	//enrichTableWithFKs(table)
	//validateTable(table)
	//generateTableDeclarations(nil, table)
}

func enrichTableWithFKs(table *Table) {
	var pkFields []Field
	for _, field := range table.Fields {
		if field.pk {
			pkFields = append(pkFields, field)
		}
	}

	for _, field := range table.ForeignKeys {
		field.Name = normalizeName("parent_" + field.Name)
		pkFields = append(pkFields, field)
	}

	for _, child := range table.ChildTables {
		child.ForeignKeys = pkFields
		enrichTableWithFKs(child)
	}
}

type Field struct {
	Name     string
	DataType DataType
	pk       bool
}

func (f Field) NormalizedName() string {
	return normalizeName(f.Name)
}

type ForeignKeyField struct {
	Field
	parentFieldName string
}

type Table struct {
	Name      string
	FieldName string
	Type      string
	ChildTables []*Table
	Fields 		[]Field
	ForeignKeys []Field
}

func (t Table) NormalizedName() string {
	return normalizeName(t.Name)
}

type walker struct {
	Tables map[string]*Table
}

// Walk iterates over the obj and creates a search.Map object from the found struct tags
func walk(table *Table, obj interface{}) {
	sw := walker{
		Tables: make(map[string]*Table),
	}
	sw.walkRecursive("", table, reflect.TypeOf(obj))
}

func normalizeName(name string) string {
	name =  strings.ToLower(strings.ReplaceAll(name,"-", "_"))
	return strings.ReplaceAll(name, ".", "_")
}

// handleStruct takes in a struct object and properly handles all of the fields
func (s *walker) handleStruct(prefix string, table *Table, original reflect.Type) {
	for i := 0; i < original.NumField(); i++ {
		field := original.Field(i)

		if strings.HasPrefix(field.Name, "XXX") {
			continue
		}

		sqlTag := field.Tag.Get("sql")
		isPK := strings.Contains(sqlTag, "pk")

		fieldName := field.Name
		if prefix != "" {
			fieldName = prefix +"."+fieldName
		}

		// Special case proto timestamp because we actually want to index seconds
		if field.Type.String() == "*types.Timestamp" {
			table.Fields = append(table.Fields, Field{
				Name:     fieldName,
				DataType: DATETIME,
				pk:       isPK,
			})
			continue
		}
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
					s.walkRecursive(prefix, table, typ)
				}
			}
			continue
		}

		dataType := s.walkRecursive(fieldName, table, field.Type)
		if dataType == ARRAY {
			childTable := &Table{
				FieldName: fieldName,
				Type: field.Type.Elem().String(),
				Name: table.Name + "_" + fieldName, // normalizeName(table.Name + "_" + fieldName),
			}

			table.ChildTables = append(table.ChildTables, childTable)
			// new table so can reset the prefix
			s.walkRecursive("", childTable, field.Type.Elem())
			continue
		} else if dataType == STRUCT {
			continue
		}

		table.Fields = append(table.Fields, Field{
			Name:     fieldName,
			DataType: dataType,
			pk:       isPK,
		})
	}
}

func (s *walker) walkRecursive(prefix string, table *Table, original reflect.Type) DataType {
	switch original.Kind() {
	case reflect.Ptr:
		return s.walkRecursive(prefix, table, original.Elem())
	case reflect.Slice:
		if original.Elem().Kind() == reflect.String {
			return STRING_ARRAY
		}
		return ARRAY
	case reflect.Struct:
		s.handleStruct(prefix, table, original)
		return STRUCT
	case reflect.Map:
		return MAP
	case reflect.String:
		return STRING
	case reflect.Bool:
		return BOOL
	case reflect.Uint32, reflect.Uint64, reflect.Int32, reflect.Int64, reflect.Float32, reflect.Float64:
		_, ok := reflect.Zero(original).Interface().(protoreflect.ProtoEnum)
		if !ok {
			return NUMERIC
		}
		//enumDesc, err := protoreflect.GetEnumDescriptor(enum)
		//if err != nil {
		//	panic(err)
		//}
		//enumregistry.Add(prefix, enumDesc)
		return ENUM
	case reflect.Interface:
	default:
		panic(fmt.Sprintf("Type %s on table %s is not currently handled", original.Kind(), table.Name))
	}
	return STRING
}
