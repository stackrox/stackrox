package walker

import (
	"fmt"
)

var (
	serializedField = Field{
		Name: "serialized",
		ObjectGetter: ObjectGetter{
			variable: true,
			value:    "serialized",
		},
		ColumnName: "serialized",
		SQLType:    "bytea",
	}
)

// Schema is the go representation of the schema for a table
// This is derived from walking the go struct
type Schema struct {
	Table        string
	Parents      []*Schema
	Fields       []Field
	Children     []*Schema
	Type         string
	ObjectGetter string
}

// FieldsBySearchLabel returns the resulting fields in the schema by their field label
func (s *Schema) FieldsBySearchLabel() map[string]*Field {
	m := make(map[string]*Field)
	for _, f := range s.Fields {
		field := f
		if f.Search.Enabled {
			m[f.Search.FieldName] = &field
		}
	}
	for _, child := range s.Children {
		for k, v := range child.FieldsBySearchLabel() {
			m[k] = v
		}
	}
	return m
}

// AddFieldWithType adds a field to the schema with the specified data type
func (s *Schema) AddFieldWithType(field Field, dt DataType) {
	field.DataType = dt
	field.SQLType = DataTypeToSQLType(dt)
	s.Fields = append(s.Fields, field)
}

// Print is a helper function to visualize the table when debugging
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

// tryParentify attempts to convert the specified field to a reference. If the field is already a reference in
// the referenced schema, it is used as is.
func tryParentify(field *Field, parentSchema *Schema) {
	referencedColName := field.ColumnName
	if field.Reference == "" {
		field.Name = parentify(parentSchema.Table, field.Name)
		field.ColumnName = parentify(parentSchema.Table, referencedColName)
	}
	field.Reference = referencedColName
}

func parentify(parent, name string) string {
	return parent + "_" + name
}

// ResolvedFields is the total set of fields for the schema including
// fields that are derived from the parent schemas. e.g. parent primary keys, array indexes, etc.
func (s *Schema) ResolvedFields() []Field {
	var pks []Field
	for _, parent := range s.Parents {
		pks = parent.ResolvedPrimaryKeys()
		for idx := range pks {
			pk := &pks[idx]
			tryParentify(pk, parent)
			pk.ObjectGetter = ObjectGetter{
				variable: true,
				value:    pk.Name,
			}
		}
	}

	pks = append(pks, s.Fields...)
	if len(s.Parents) == 0 {
		pks = append(pks, serializedField)
	}
	return pks
}

// ParentKeys are the keys from the parent schemas that should be defined
// as foreign keys for the current schema.
func (s *Schema) ParentKeys() []Field {
	var fields []Field
	pksAsMap := s.ParentKeysAsMap()
	for _, pks := range pksAsMap {
		fields = append(fields, pks...)
	}
	return fields
}

// ParentKeysAsMap returns the keys from the parent schemas that should be defined
// as foreign keys for the current schema mapped by parent schema.
func (s *Schema) ParentKeysAsMap() map[string][]Field {
	pks := make(map[string][]Field)
	for _, parent := range s.Parents {
		currPks := parent.ResolvedPrimaryKeys()
		for idx := range currPks {
			pk := &currPks[idx]
			tryParentify(pk, parent)
		}
		pks[parent.Table] = currPks
	}
	return pks
}

// ForeignKeysReferencesTo returns the foreign keys of the current schema referencing specified schema name.
func (s *Schema) ForeignKeysReferencesTo(tableName string) []Field {
	if len(s.Parents) == 0 {
		return nil
	}

	var pSchema *Schema
	for i := 0; i < len(s.Parents); i++ {
		if s.Parents[i].Table == tableName {
			pSchema = s.Parents[i]
			break
		}
	}
	if pSchema == nil {
		return nil
	}

	// Only get the immediate references, and not the resolved ones.
	pks := pSchema.LocalPrimaryKeys()
	for idx := range pks {
		fk := &pks[idx]
		tryParentify(fk, pSchema)
	}
	// If we are here, it means all references to the required referenced table have been computed. Hence, stop.
	return pks
}

// ForeignKeys are the foreign keys in current schema.
func (s *Schema) ForeignKeys() []Field {
	if len(s.Parents) == 0 {
		return nil
	}
	var fks []Field
	for _, parent := range s.Parents {
		pks := parent.LocalPrimaryKeys()
		for idx := range pks {
			pk := &pks[idx]
			tryParentify(pk, parent)
		}
		fks = append(fks, pks...)
	}
	return fks
}

// ResolvedPrimaryKeys are all the primary keys of the current schema which is the union
// of keys from the parent schemas and also any local keys
func (s *Schema) ResolvedPrimaryKeys() []Field {
	pks := s.ParentKeys()
	pks = append(pks, s.LocalPrimaryKeys()...)
	return pks
}

// LocalPrimaryKeys are the primary keys in the current schema
func (s *Schema) LocalPrimaryKeys() []Field {
	var pks []Field
	for _, f := range s.Fields {
		if f.Options.PrimaryKey {
			pks = append(pks, f)
		}
	}
	return pks
}

// NoPrimaryKey returns true if the current schema does not have a primary key defined
func (s *Schema) NoPrimaryKey() bool {
	return len(s.LocalPrimaryKeys()) == 0
}

// SearchField is the parsed representation of the search tag on the struct field
type SearchField struct {
	FieldName string
	Analyzer  string
	Hidden    bool
	Store     bool
	Enabled   bool
	Ignored   bool
}

// PostgresOptions is the parsed representation of the sql tag on the struct field
type PostgresOptions struct {
	Ignored                bool
	Index                  string
	PrimaryKey             bool
	Unique                 bool
	IgnorePrimaryKey       bool
	IgnoreUniqueConstraint bool
}

// ObjectGetter is wrapper around determining how to represent the variable in the
// autogenerated code. If variable is true, then this is a local variable to the function
// and not a field of the struct itself so it does not need to be prefixed
type ObjectGetter struct {
	variable bool
	value    string
}

// Field is the representation of a struct field in Postgres
type Field struct {
	Schema *Schema
	// Name of the struct field
	Name         string
	ObjectGetter ObjectGetter
	ColumnName   string
	// If set, this is the reference to
	Reference string
	// Type is the reflect.TypeOf value of the field
	Type string
	// DataType is the internal type
	DataType DataType
	SQLType  string
	Options  PostgresOptions
	Search   SearchField
}

// Getter returns the path to the object. If variable is true, then the value is just
func (f Field) Getter(prefix string) string {
	value := f.ObjectGetter.value
	if f.ObjectGetter.variable {
		return value
	}
	return prefix + "." + value
}
