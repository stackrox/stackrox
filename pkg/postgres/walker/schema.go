package walker

import (
	"fmt"
	"strings"

	"github.com/stackrox/rox/pkg/set"
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
	TypeName     string
	ObjectGetter string

	// This indicates the name of the parent schema in which current schema is embedded (in proto). A schema can be
	// embedded exactly one porent. For the top-most schema this field is unset.
	//
	// We use `Parents` and `Children` which mean referenced table and referencing table in SQL world,
	// but in our context it reflects the nesting of proto messages.
	EmbeddedIn string
}

// TableFieldsGroup is the group of table fields. A slice of this struct can be used where the table order is essential,
type TableFieldsGroup struct {
	Table  string
	Fields []Field
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
	// Find all the local fields that are already defined as foreign keys.
	localFKs := s.localFKs()

	var allPks []Field
	for _, parent := range s.Parents {
		pks := parent.ResolvedPrimaryKeys()
		for idx := range pks {
			pk := &pks[idx]
			if _, found := localFK(pk, localFKs); found {
				continue
			}
			tryParentify(pk, parent)
			pk.ObjectGetter = ObjectGetter{
				variable: true,
				value:    pk.Name,
			}
			allPks = append(allPks, *pk)
		}
	}

	allPks = append(allPks, s.Fields...)
	if len(s.Parents) == 0 || s.EmbeddedIn == "" {
		allPks = append(allPks, serializedField)
	}
	return allPks
}

// ParentKeys are the keys from the parent schemas that should be defined
// as foreign keys for the current schema.
func (s *Schema) ParentKeys() []Field {
	var fields []Field
	pksGrps := s.ParentKeysGroupedByTable()
	for _, pks := range pksGrps {
		fields = append(fields, pks.Fields...)
	}
	return fields
}

// ParentKeysGroupedByTable returns the keys from the parent schemas that should be defined
// as foreign keys for the current schema grouped by parent schema.
func (s *Schema) ParentKeysGroupedByTable() []TableFieldsGroup {
	pks := make([]TableFieldsGroup, 0, len(s.Parents))
	// Find all the local fields that are already defined as foreign keys.
	localFKs := s.localFKs()

	for _, parent := range s.Parents {
		currPks := parent.ResolvedPrimaryKeys()
		for idx := range currPks {
			pk := &currPks[idx]
			// If the referenced parent field is already an embedded as foriegn key in child, use the child field names.
			if field, found := localFK(pk, localFKs); found {
				pk.Name = field.Name
				pk.Reference = pk.ColumnName
				pk.ColumnName = field.ColumnName
				continue
			}
			tryParentify(pk, parent)
		}
		pks = append(pks, TableFieldsGroup{Table: parent.Table, Fields: currPks})
	}
	return pks
}

func (s *Schema) localFKs() map[foreignKeyRef]*Field {
	localFKs := make(map[foreignKeyRef]*Field)
	for idx := range s.Fields {
		f := &s.Fields[idx]
		if ref := f.Options.Reference; ref != nil {
			localFKs[foreignKeyRef{
				typeName:      strings.ToLower(ref.RefSchema.Type),
				protoBufField: strings.ToLower(ref.Reference),
			}] = f
		}
	}
	return localFKs
}

func localFK(field *Field, localFKMap map[foreignKeyRef]*Field) (*Field, bool) {
	f := localFKMap[foreignKeyRef{
		typeName:      strings.ToLower(field.Schema.Type),
		protoBufField: strings.ToLower(field.ProtoBufName),
	}]
	if f == nil {
		return nil, false
	}
	return f, true
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

	// Find all the local fields that are already defined as foreign keys.
	localFKs := s.localFKs()

	// Only get the immediate references, and not the resolved ones.
	pks := pSchema.LocalPrimaryKeys()
	for idx := range pks {
		fk := &pks[idx]
		if _, found := localFK(fk, localFKs); found {
			continue
		}
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
	// Find all the local fields that are already defined as foreign keys.
	localFKs := s.localFKs()

	var fks []Field
	for _, parent := range s.Parents {
		pks := parent.LocalPrimaryKeys()
		for idx := range pks {
			pk := &pks[idx]
			if _, found := localFK(pk, localFKs); found {
				continue
			}
			tryParentify(pk, parent)
		}
		fks = append(fks, pks...)
	}
	return fks
}

// ResolvedPrimaryKeys are all the primary keys of the current schema which is the union
// of keys from the parent schemas and also any local keys
func (s *Schema) ResolvedPrimaryKeys() []Field {
	localPKSet := set.NewStringSet()
	localPKS := s.LocalPrimaryKeys()
	for _, pk := range localPKS {
		localPKSet.Add(pk.ColumnName)
	}

	var pks []Field
	// If the resolved primary key is already present as local primary key, do not add it.
	for _, pk := range s.ParentKeys() {
		if localPKSet.Add(pk.ColumnName) {
			pks = append(pks, pk)
		}
	}
	pks = append(pks, localPKS...)
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

// WithReference adds the specified references to this schema and returns it. This enables attaching additional
// references to the schema which is not embedded within specified reference.
func (s *Schema) WithReference(refs ...*ReferenceInfo) *Schema {
	for _, ref := range refs {
		var parentFound bool
		for _, p := range s.Parents {
			if p.Table == ref.RefSchema.Table {
				parentFound = true
				break
			}
		}
		if !parentFound {
			s.Parents = append(s.Parents, ref.RefSchema)
			ref.RefSchema.Children = append(ref.RefSchema.Children, s)
		}

		for idx := range s.Fields {
			field := &s.Fields[idx]
			if field.ProtoBufName == ref.ForeignKey {
				field.Options.Reference = ref
				field.Reference = ref.Reference
			}
		}
	}
	return s
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
	Reference              *ReferenceInfo
}

type foreignKeyRef struct {
	typeName      string
	protoBufField string
}

// ReferenceInfo holds the foreign key reference information.
type ReferenceInfo struct {
	ForeignKey string
	RefSchema  *Schema
	Reference  string
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
	ProtoBufName string
	ObjectGetter ObjectGetter
	ColumnName   string
	// If set, this is the reference to
	Reference string
	// Type is the reflect.TypeOf value of the field
	Type     string
	TypeName string
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
