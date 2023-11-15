package walker

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
)

var (
	log = logging.LoggerForModule()
)

func getSerializedField(s *Schema) Field {
	return Field{
		ObjectGetter: ObjectGetter{
			variable: true,
			value:    "serialized",
		},
		Name:       "serialized",
		ColumnName: "serialized",
		SQLType:    "bytea",
		Type:       "[]byte",
		ModelType:  "[]byte",
		Schema:     s,
	}
}

func getIdxField(s *Schema) Field {
	return Field{
		Schema: s,
		Name:   "idx",
		ObjectGetter: ObjectGetter{
			variable: true,
			value:    "idx",
		},
		Type:       reflect.TypeOf(0).String(),
		ColumnName: "idx",
		DataType:   postgres.Integer,
		SQLType:    "integer",
		ModelType:  reflect.TypeOf(0).String(),
		Options: PostgresOptions{
			Ignored: false,
			Index: []*PostgresIndexOptions{
				{IndexType: "btree"},
			},
			PrimaryKey: true,
		},
	}
}

// ColumnNamePair is a pair of column names in a SchemaRelationship.
type ColumnNamePair struct {
	ColumnNameInThisSchema  string
	ColumnNameInOtherSchema string
}

// SchemaRelationship denotes a relationship between this schema and the OtherSchema,
// via the MappedColumnNames.
type SchemaRelationship struct {
	OtherSchema       *Schema
	MappedColumnNames []ColumnNamePair

	// NoConstraint indicates that this relationship is not enforced at the SQL
	// level by a foreign key constraint.
	NoConstraint bool

	// RestrictDelete indicates that this relationship should restrict deletion rather than cascade
	RestrictDelete bool

	// CycleReference indicates that this relationship is a self reference
	// this is necessary because parent references and self references would otherwise be named the same
	CycleReference bool
}

// ThisSchemaColumnNames generates the sequence of column names for this schema
func (s *SchemaRelationship) ThisSchemaColumnNames() []string {
	var seq []string
	for _, p := range s.MappedColumnNames {
		seq = append(seq, p.ColumnNameInThisSchema)
	}
	return seq
}

// OtherSchemaColumnNames generates the list of column names for the other schema
func (s *SchemaRelationship) OtherSchemaColumnNames() []string {
	var seq []string
	for _, p := range s.MappedColumnNames {
		seq = append(seq, p.ColumnNameInOtherSchema)
	}
	return seq
}

// PermissionChecker is a permission checker that could be used by GenericStore
type PermissionChecker interface {
	ReadAllowed(ctx context.Context) (bool, error)
	WriteAllowed(ctx context.Context) (bool, error)
}

// Schema is the go representation of the schema for a table
// This is derived from walking the go struct
type Schema struct {
	Table string
	// Parent stores a link to the parent table, if any.
	// This happens when this table represents a repeated field
	// in the Parent.
	Parent *Schema
	// Children stores all Schemas for which this Schema is the Parent.
	Children     []*Schema
	Fields       []Field
	Type         string
	TypeName     string
	ObjectGetter string

	// References stores information about the other tables referenced by this schema.
	// It is grouped by referenced table.
	// It does NOT duplicate information stored in the Parent and Children fields.
	References []SchemaRelationship
	// ReferencedBy stores information about the tables that reference this schema.
	// It is just a reverse edge from the References of the other tables to enable
	// traversing the graph starting from this schema as well.
	ReferencedBy []SchemaRelationship
	// referencesResolved in an internal bool to ensure ResolveReferences is called exactly once.
	referencesResolved bool

	OptionsMap search.OptionsMap

	// SearchScope represents the search categories searchable from this schema. This can be used to limit search to only
	// some categories in cases of overlapping search fields.
	// This is optional.
	SearchScope map[v1.SearchCategory]struct{}

	ScopingResource   permissions.ResourceMetadata
	PermissionChecker PermissionChecker

	Flag string
}

// TableFieldsGroup is the group of table fields. A slice of this struct can be used where the table order is essential,
type TableFieldsGroup struct {
	Table  string
	Fields []Field
}

// SetOptionsMap sets options map for the schema.
func (s *Schema) SetOptionsMap(optionsMap search.OptionsMap) {
	s.OptionsMap = optionsMap
	for _, c := range s.Children {
		c.SetOptionsMap(optionsMap)
	}
}

// SetSearchScope sets search scope for the schema.
func (s *Schema) SetSearchScope(searchCategories ...v1.SearchCategory) {
	s.SearchScope = make(map[v1.SearchCategory]struct{})
	for _, cat := range searchCategories {
		s.SearchScope[cat] = struct{}{}
	}
	for _, c := range s.Children {
		c.SetSearchScope(searchCategories...)
	}
}

// AddFieldWithType adds a field to the schema with the specified data type
func (s *Schema) AddFieldWithType(field Field, dt postgres.DataType, opts PostgresOptions) {
	if !field.Include() {
		return
	}

	field.DataType = dt
	if opts.ColumnType != "" {
		field.SQLType = opts.ColumnType
	} else {
		field.SQLType = postgres.DataTypeToSQLType(dt)
	}

	field.ModelType = postgres.GetToGormModelType(field.Type, field.DataType)
	s.Fields = append(s.Fields, field)
}

// Print is a helper function to visualize the table when debugging
func (s *Schema) Print() {
	fmt.Println(s.Table)
	for _, f := range s.Fields {
		fmt.Printf("  name=%s columnName=%s getter=%+v datatype=%s searchable:%v\n", f.Name, f.ColumnName, f.ObjectGetter, f.DataType, f.Search.Enabled)
	}
	fmt.Println()
	for _, c := range s.Children {
		c.Print()
	}
}

// DBColumnFields is the set of fields that should be columns in the DB table.
func (s *Schema) DBColumnFields() []Field {
	var includedFields []Field
	for _, f := range s.Fields {
		if f.Include() {
			includedFields = append(includedFields, f)
		}
	}
	return includedFields
}

// RelationshipsToDefineAsForeignKeys returns the schema relationships which should be defined as foreign key
// constraint in this schema. If this Schema is embedded, then the relationship to the parent is also included.
func (s *Schema) RelationshipsToDefineAsForeignKeys() []SchemaRelationship {
	var out []SchemaRelationship
	// First, add the one referring to the parent, if a parent exists.
	if s.Parent != nil {
		out = append(out, s.getParentRelationship())
	}
	for _, ref := range s.References {
		if !ref.NoConstraint {
			if s.Parent != nil && s.Parent.Table == ref.OtherSchema.Table {
				ref.CycleReference = true
			}
			out = append(out, ref)
		}
	}
	return out
}

func (s *Schema) getParentRelationship() SchemaRelationship {
	rel := SchemaRelationship{
		OtherSchema: s.Parent,
	}
	for _, f := range s.Fields {
		if ref := f.Options.Reference; ref != nil && ref.OtherSchema == s.Parent {
			rel.MappedColumnNames = append(rel.MappedColumnNames, ColumnNamePair{
				ColumnNameInThisSchema:  f.ColumnName,
				ColumnNameInOtherSchema: ref.ColumnName,
			})
		}
	}
	return rel
}

// AllRelationships returns all SchemaRelationships this schema has.
// It includes relationships to everything -- schemas this schema references, other schemas that
// reference this schema, parent and children -- irrespective of whether a foreign key reference constraint
// exists.
func (s *Schema) AllRelationships() []SchemaRelationship {
	out := make([]SchemaRelationship, len(s.References)+len(s.ReferencedBy))
	copy(out, s.References)
	copy(out[len(s.References):], s.ReferencedBy)
	if s.Parent != nil {
		out = append(out, s.getParentRelationship())
	}
	for _, child := range s.Children {
		relationshipFromChild := child.getParentRelationship()
		reversedRelationship := SchemaRelationship{OtherSchema: child}
		for _, columnNamePairFromChild := range relationshipFromChild.MappedColumnNames {
			reversedRelationship.MappedColumnNames = append(reversedRelationship.MappedColumnNames, ColumnNamePair{
				ColumnNameInThisSchema:  columnNamePairFromChild.ColumnNameInOtherSchema,
				ColumnNameInOtherSchema: columnNamePairFromChild.ColumnNameInThisSchema,
			})
		}
		out = append(out, reversedRelationship)
	}
	return out
}

// FieldsDeterminedByParent returns the set of fields in this schema whose value is
// set in the context of its parent.
func (s *Schema) FieldsDeterminedByParent() []Field {
	if s.Parent == nil {
		return nil
	}
	out := s.FieldsReferringToParent()
	for _, f := range s.Fields {
		if f.ColumnName == "idx" {
			out = append(out, f)
			break
		}
	}
	return out
}

// FieldsReferringToParent are the keys from the (proto-)parent schemas that should be defined
// as foreign keys for the current schema.
func (s *Schema) FieldsReferringToParent() []Field {
	if s.Parent == nil {
		return nil
	}
	var fieldsReferringToParent []Field
	for _, f := range s.Fields {
		if ref := f.Options.Reference; ref != nil && ref.OtherSchema == s.Parent {
			fieldsReferringToParent = append(fieldsReferringToParent, f)
		}
	}
	return fieldsReferringToParent
}

// PrimaryKeys are the primary keys in the current schema
func (s *Schema) PrimaryKeys() []Field {
	var pks []Field
	for _, f := range s.Fields {
		if f.Options.PrimaryKey {
			pks = append(pks, f)
		}
	}
	return pks
}

// ID is the id field in the current schema, if any.
func (s *Schema) ID() Field {
	for _, f := range s.Fields {
		if f.Options.ID {
			return f
		}
	}
	// If there is only one primary key, that is considered ID column by default even if not specified explicitly.
	pks := s.PrimaryKeys()
	if len(pks) == 1 {
		return pks[0]
	}
	log.Errorf("No ID column defined for %s", s.Table)
	return Field{}
}

func (s *Schema) findTableAndColumnName(protoBufName string) (*Schema, string) {
	for _, f := range s.Fields {
		if f.ProtoBufName == protoBufName {
			return s, f.ColumnName
		}
	}
	for _, s := range s.Children {
		if table, columnName := s.findTableAndColumnName(protoBufName); table != nil && columnName != "" {
			return table, columnName
		}
	}
	return nil, ""
}

// ResolveReferences resolves references from this schema to other schemas, using the passed function that
// returns peer Schemas, in order to populate relationship info in this Schema and peer schemas.
// Until this function is called, the References and ReferencedBy fields in the Schema will be blank.
func (s *Schema) ResolveReferences(schemaProvider func(messageTypeName string) *Schema) {
	if s.referencesResolved {
		log.Panicf("Duplicate call to ResolveReferences for schema %+v", s)
	}
	s.referencesResolved = true
	for i := range s.Fields {
		f := &s.Fields[i]
		fieldRef := f.Options.Reference
		if fieldRef == nil {
			continue
		}
		// Reference is resolved already.
		if fieldRef.OtherSchema != nil && fieldRef.ColumnName != "" {
			continue
		}
		referencedSchema := schemaProvider(fieldRef.TypeName)
		if referencedSchema == nil {
			log.Panicf("Couldn't resolve reference in field %+v (ref: %v): type not provided", f, *fieldRef)
		}
		otherTable, columnNameInOtherSchema := referencedSchema.findTableAndColumnName(fieldRef.ProtoBufField)
		if otherTable == nil || columnNameInOtherSchema == "" {
			log.Panicf("Couldn't resolve reference in field %+v: no field with protobuf name found", f)
		}
		fieldRef.OtherSchema = otherTable
		fieldRef.ColumnName = columnNameInOtherSchema

		addColumnPairToRelationshipsSlice(&s.References, s, otherTable, f.ColumnName, columnNameInOtherSchema, fieldRef.NoConstraint, fieldRef.RestrictDelete)
		if !fieldRef.Directional {
			addColumnPairToRelationshipsSlice(&otherTable.ReferencedBy, otherTable, s, columnNameInOtherSchema, f.ColumnName, fieldRef.NoConstraint, fieldRef.RestrictDelete)
		}
	}

	for _, child := range s.Children {
		child.ResolveReferences(schemaProvider)
	}
}

func addColumnPairToRelationshipsSlice(relationshipsSlice *[]SchemaRelationship, thisSchema, otherSchema *Schema, columnNameInThisSchema, columnNameInOtherSchema string, noConstraint bool, restrictDelete bool) {
	refIdxToModify := -1
	for i, ref := range *relationshipsSlice {
		if ref.OtherSchema == otherSchema {
			if ref.NoConstraint != noConstraint {
				log.Panicf("This reference from %s (%s)to %s (%s) has a noConstraint value inconsistent with the other reference(s) (%+v)",
					thisSchema.Table, columnNameInThisSchema, otherSchema.Table, columnNameInOtherSchema, ref.MappedColumnNames)
			}
			refIdxToModify = i
			break
		}
	}
	// This is the first column mapping for this particular schema.
	if refIdxToModify == -1 {
		*relationshipsSlice = append(*relationshipsSlice, SchemaRelationship{OtherSchema: otherSchema, NoConstraint: noConstraint, RestrictDelete: restrictDelete})
		refIdxToModify = len(*relationshipsSlice) - 1
	}
	(*relationshipsSlice)[refIdxToModify].MappedColumnNames = append((*relationshipsSlice)[refIdxToModify].MappedColumnNames, ColumnNamePair{
		ColumnNameInThisSchema:  columnNameInThisSchema,
		ColumnNameInOtherSchema: columnNameInOtherSchema,
	})
}

// NoPrimaryKey returns true if the current schema does not have a primary key defined
func (s *Schema) NoPrimaryKey() bool {
	return len(s.PrimaryKeys()) == 0
}

// SearchField is the parsed representation of the search tag on the struct field
type SearchField struct {
	FieldName string
	Enabled   bool
	Ignored   bool
}

// PostgresIndexOptions is the parsed representation of the index subpart of the sql tag in the struct field
type PostgresIndexOptions struct {
	IndexName     string
	IndexType     string
	IndexCategory string
	IndexPriority string
}

// PostgresOptions is the parsed representation of the sql tag on the struct field
type PostgresOptions struct {
	ID                     bool
	Ignored                bool
	Index                  []*PostgresIndexOptions
	PrimaryKey             bool
	Unique                 bool
	IgnorePrimaryKey       bool
	IgnoreUniqueConstraint bool
	IgnoreSearchLabels     set.StringSet
	Reference              *foreignKeyRef

	// Which database type will be used to store this value
	ColumnType string

	// IgnoreChildFKs is an option used to tell the walker that
	// foreign keys of children of this field should be ignored.
	IgnoreChildFKs bool

	// IgnoreChildIndexes is an option used to tell the walker that
	// index options of children of this field should be ignored.
	IgnoreChildIndexes bool

	// Flag is an option used to tell the walker that the child fields
	// are dependent on a feature flag and should not be included if the flag is false
	Flag string
}

type foreignKeyRef struct {
	TypeName      string
	ProtoBufField string
	// If true, this column (foreign key) depends on a column in other table, but does not have a constraint.
	NoConstraint bool

	// If true, the constraint on this foreign key reference should restrict deletion
	RestrictDelete bool

	// If true, this means that we only want to create a graph edge out from this field and not have it be bi-directional
	Directional bool

	// The referenced schema and column name are what we ultimately need for the foreign key reference.
	// However, we don't want to put this information in the proto message itself, since we
	// don't want to bleed that level of detail from the  SQL implementation into the proto.
	// Therefore, these are filled later, based on the parameters provided at code generation
	// time.
	OtherSchema *Schema
	ColumnName  string
}

// FieldInOtherSchema returns the `Field` in the other schema that has the specific column name.
func (f *foreignKeyRef) FieldInOtherSchema() (Field, error) {
	if f.OtherSchema == nil {
		return Field{}, errors.New("OtherSchema is nil, please call ResolveReferences first")
	}
	for _, field := range f.OtherSchema.Fields {
		if field.ColumnName == f.ColumnName {
			return field, nil
		}
	}
	return Field{}, fmt.Errorf("no field found in schema %s with column %s", f.OtherSchema.Table, f.ColumnName)
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

	// Type is the reflect.TypeOf value of the field
	Type string

	// DataType is the internal type
	DataType  postgres.DataType
	SQLType   string
	ModelType string
	Options   PostgresOptions
	Search    SearchField
	// DerivedSearchFields represents the search fields derived from this search field.
	DerivedSearchFields []DerivedSearchField
	// Derived indicates whether the search field (if valid search field) is derived from other search field.
	Derived bool
}

// DerivedSearchField represents a search field that's derived.
// It includes the name of the derived field, as well as the derivation type.
type DerivedSearchField struct {
	DerivedFrom    string
	DerivationType search.DerivationType
}

// Getter returns the path to the object. If variable is true, then the value is just
func (f Field) Getter(prefix string) string {
	value := f.ObjectGetter.value
	if f.ObjectGetter.variable {
		return value
	}
	return prefix + "." + value
}

// Include returns if the field should be included in the schema
func (f Field) Include() bool {
	return f.Options.PrimaryKey || f.Options.Unique || f.Search.Enabled || f.ColumnName == "serialized" || f.Options.Reference != nil
}
