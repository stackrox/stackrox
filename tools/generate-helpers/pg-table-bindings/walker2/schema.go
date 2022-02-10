package walker

import (
	"fmt"
)

type Schema struct {
	Table 				string
	ParentTable 	    *Schema
	Fields 				[]*Field
	FieldsBySearchField map[string]*Field
	Children 			[]*Schema
	Relationships 		[]Relationship
}

func (s *Schema) AddFieldWithType(field *Field, dt DataType) {
	field.DataType = dt
	s.Fields = append(s.Fields, field)
}

func (s *Schema) Print() {
	fmt.Println(s.Table)
	for _, f := range s.Fields {
		fmt.Printf("  name=%s columnName=%s getter=%s datatype=%s\n", f.Name, f.ColumnName, f.ObjectGetter, f.DataType)
	}
	fmt.Println()
	for _, c := range s.Children {
		c.Print()
	}
}

type Relationship struct {}

type SearchField struct {
	FieldName string
	Analyzer  string
	Hidden    bool
	Store     bool
}

type IndexConfig struct {
	Using string
}

type PrimaryKey struct {
	LocalKey  string
	ParentKey string
}

type PostgresOptions struct {
	Ignored    bool
	Index      string
	PrimaryKey bool
}

type Field struct {
	Schema 		 *Schema
	Name 		 string
	ObjectGetter string
	ColumnName   string
	DataType     DataType
	Options     *PostgresOptions
	Search	    *SearchField
}





/*
	FieldType              reflect.Type
	IndirectFieldType      reflect.Type
	StructField            reflect.StructField
	Tag                    reflect.StructTag
	TagSettings            map[string]string
	Schema                 *Schema
	EmbeddedSchema         *Schema
	OwnerSchema            *Schema
	ReflectValueOf         func(reflect.Value) reflect.Value
	ValueOf                func(reflect.Value) (value interface{}, zero bool)
	Set                    func(reflect.Value, interface{}) error
	IgnoreMigration        bool
 */

