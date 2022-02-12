package walker

import (
	"fmt"
)

type Schema struct {
	Table               string
	ParentSchema        *Schema
	Fields              []Field
	FieldsBySearchField map[string]Field
	Children            []*Schema
	Relationships       []Relationship
	Type                string
	ObjectGetter        string
	ForeignKeys         []Field
}

func (s *Schema) AddFieldWithType(field Field, dt DataType) {
	field.DataType = dt
	field.SQLType = DataTypeToSQLType(dt)
	s.Fields = append(s.Fields, field)
}

func (s *Schema) Print() {
	fmt.Println(s.Table)
	for _, f := range s.Fields {
		fmt.Printf("  name=%s columnName=%s getter=%+v datatype=%s\n", f.Name, f.ColumnName, f.ObjectGetter, f.DataType)
	}
	fmt.Println()
	for _, c := range s.Children {
		c.Print()
	}
}

func parent(name string) string {
	return "parent_" + name
}

func (s *Schema) ResolvedFields() []Field {
	var pks []Field
	if s.ParentSchema != nil {
		pks = s.ParentSchema.ResolvedPrimaryKeys()
	}
	for idx := range pks {
		pk := &pks[idx]
		pk.Reference = pk.ColumnName
		pk.Name = parent(pk.Name)
		pk.ObjectGetter = ObjectGetter{
			variable: true,
			value:    pk.Name,
		}
		pk.ColumnName = parent(pk.ColumnName)
	}
	pks = append(pks, s.Fields...)
	if s.ParentSchema == nil {
		pks = append(pks, Field{
			Schema:       s,
			Name:         "serialized",
			ObjectGetter: ObjectGetter{
				variable: true,
				value: "serialized",
			},
			ColumnName:   "serialized",
			SQLType:      "bytea",
		})
	}
	return pks
}

func (s *Schema) ParentKeys() []Field {
	if s.ParentSchema == nil {
		return nil
	}
	pks := s.ParentSchema.ResolvedPrimaryKeys()
	for idx := range pks {
		pk := &pks[idx]
		pk.Reference = pk.ColumnName
		pk.Name = parent(pk.Name)
		pk.ColumnName = parent(pk.ColumnName)
	}
	return pks
}

func (s *Schema) ResolvedPrimaryKeys() []Field {
	pks := s.ParentKeys()
	for _, f := range s.Fields {
		if f.Options.PrimaryKey {
			pks = append(pks, f)
		}
	}
	return pks
}

func (s *Schema) LocalPrimaryKeys() []Field {
	var pks []Field
	for _, f := range s.Fields {
		if f.Options.PrimaryKey {
			pks = append(pks, f)
		}
	}
	return pks
}

type Relationship struct{}

type SearchField struct {
	FieldName string
	Analyzer  string
	Hidden    bool
	Store     bool
	Enabled   bool
}

type PrimaryKey struct {
	LocalKey  string
	ParentKey string
}

type PostgresOptions struct {
	Ignored    bool
	Index      string
	PrimaryKey bool
	Unique     bool
}

type ObjectGetter struct {
	variable bool
	value   string
}

type Field struct {
	Schema       *Schema
	Name         string
	ObjectGetter ObjectGetter
	ColumnName   string
	Reference    string
	Type         string
	DataType     DataType
	SQLType      string
	Options      PostgresOptions
	Search       SearchField
}

func (f Field) Getter(prefix string) string {
	value := f.ObjectGetter.value
	if f.ObjectGetter.variable {
		return value
	}
	return prefix + "." + value
}
