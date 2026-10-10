package main

import (
	"fmt"
	"log"
	"slices"
	"strconv"
	"strings"

	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/search"
)

// SerializeSchema returns Go source code that constructs the given schema.
func SerializeSchema(schema *walker.Schema, varName string) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("%s = ", varName))
	writeSchemaHeader(schema, nil, &b)

	serializeChildren(schema, varName, "", &b)

	writeSchemaFields(schema, varName, &b)

	return b.String()
}

func writeSchemaHeader(schema *walker.Schema, parentVarName *string, b *strings.Builder) {
	b.WriteString("&walker.Schema{\n")
	if parentVarName != nil {
		b.WriteString(fmt.Sprintf("\tParent: %s,\n", *parentVarName))
	}
	b.WriteString(fmt.Sprintf("\tTable: %s,\n", strconv.Quote(schema.Table)))
	b.WriteString(fmt.Sprintf("\tType: %s,\n", strconv.Quote(schema.Type)))
	b.WriteString(fmt.Sprintf("\tTypeName: %s,\n", strconv.Quote(schema.TypeName)))
	if schema.ObjectGetter != "" {
		b.WriteString(fmt.Sprintf("\tObjectGetter: %s,\n", strconv.Quote(schema.ObjectGetter)))
	}
	if schema.NoSerialized {
		b.WriteString("\tNoSerialized: true,\n")
	}
	if len(schema.SubMessages) > 0 {
		b.WriteString("\tSubMessages: map[string]string{\n")
		keys := make([]string, 0, len(schema.SubMessages))
		for k := range schema.SubMessages {
			keys = append(keys, k)
		}
		slices.Sort(keys)
		for _, k := range keys {
			b.WriteString(fmt.Sprintf("\t\t%s: %s,\n", strconv.Quote(k), strconv.Quote(schema.SubMessages[k])))
		}
		b.WriteString("\t},\n")
	}
	b.WriteString("}\n")
}

func writeSchemaFields(schema *walker.Schema, varName string, b *strings.Builder) {
	b.WriteString(fmt.Sprintf("%s.Fields = []walker.Field{\n", varName))
	for _, field := range schema.Fields {
		serializeField(&field, varName, b)
	}
	b.WriteString("}\n")
	serializeFieldReferences(schema, varName, b)
}

func serializeChildren(schema *walker.Schema, parentVarName, prefix string, b *strings.Builder) {
	for i, child := range schema.Children {
		childVarName := fmt.Sprintf("%schild%d", prefix, i)

		b.WriteString(fmt.Sprintf("%s := ", childVarName))
		writeSchemaHeader(child, &parentVarName, b)

		serializeChildren(child, childVarName, childVarName+"_", b)

		writeSchemaFields(child, childVarName, b)
	}

	if len(schema.Children) > 0 {
		b.WriteString(fmt.Sprintf("%s.Children = []*walker.Schema{", parentVarName))
		for i := range schema.Children {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(fmt.Sprintf("%schild%d", prefix, i))
		}
		b.WriteString("}\n")
	}
}

func serializeField(field *walker.Field, schemaVarName string, b *strings.Builder) {
	b.WriteString("\t{")
	b.WriteString(fmt.Sprintf("Schema: %s, ", schemaVarName))
	b.WriteString(fmt.Sprintf("Name: %s, ", strconv.Quote(field.Name)))
	b.WriteString(fmt.Sprintf("ProtoBufName: %s, ", strconv.Quote(field.ProtoBufName)))
	b.WriteString(fmt.Sprintf("ColumnName: %s, ", strconv.Quote(field.ColumnName)))
	b.WriteString(fmt.Sprintf("Type: %s, ", strconv.Quote(field.Type)))
	b.WriteString(fmt.Sprintf("DataType: %s, ", dataTypeToConst(field.DataType)))
	b.WriteString(fmt.Sprintf("SQLType: %s, ", strconv.Quote(field.SQLType)))
	b.WriteString(fmt.Sprintf("ModelType: %s, ", strconv.Quote(field.ModelType)))
	b.WriteString(fmt.Sprintf("ObjectGetter: walker.MakeObjectGetter(%s, %t), ",
		strconv.Quote(field.ObjectGetter.GetValue()),
		field.ObjectGetter.GetVariable()))

	opts := serializePostgresOptions(&field.Options)
	if opts != "" {
		b.WriteString(fmt.Sprintf("Options: walker.PostgresOptions{%s}, ", opts))
	}

	searchField := serializeSearchField(&field.Search)
	if searchField != "" {
		b.WriteString(fmt.Sprintf("Search: walker.SearchField{%s}, ", searchField))
	}

	if len(field.DerivedSearchFields) > 0 {
		b.WriteString("DerivedSearchFields: []walker.DerivedSearchField{")
		for _, dsf := range field.DerivedSearchFields {
			b.WriteString("{")
			b.WriteString(fmt.Sprintf("DerivedFrom: %s, ", strconv.Quote(dsf.DerivedFrom)))
			b.WriteString(fmt.Sprintf("DerivationType: %s, ", derivationTypeToConst(dsf.DerivationType)))
			b.WriteString(fmt.Sprintf("DerivedDataType: %s", dataTypeToConst(dsf.DerivedDataType)))
			b.WriteString("}, ")
		}
		b.WriteString("}, ")
	}

	if field.Derived {
		b.WriteString("Derived: true, ")
	}

	b.WriteString("},\n")
}

func serializePostgresOptions(opts *walker.PostgresOptions) string {
	if opts == nil {
		return ""
	}

	var parts []string

	if opts.ID {
		parts = append(parts, "ID: true")
	}
	if opts.Ignored {
		parts = append(parts, "Ignored: true")
	}
	if opts.PrimaryKey {
		parts = append(parts, "PrimaryKey: true")
	}
	if opts.Unique {
		parts = append(parts, "Unique: true")
	}
	if opts.ColumnType != "" {
		parts = append(parts, fmt.Sprintf("ColumnType: %s", strconv.Quote(opts.ColumnType)))
	}
	if opts.RepeatedStrategy != "" {
		parts = append(parts, fmt.Sprintf("RepeatedStrategy: %s", strconv.Quote(opts.RepeatedStrategy)))
	}

	return strings.Join(parts, ", ")
}

func serializeSearchField(sf *walker.SearchField) string {
	if sf == nil || (!sf.Enabled && !sf.Ignored) {
		return ""
	}

	var parts []string

	if sf.FieldName != "" {
		parts = append(parts, fmt.Sprintf("FieldName: %s", strconv.Quote(sf.FieldName)))
	}
	if sf.Enabled {
		parts = append(parts, "Enabled: true")
	}
	if sf.Ignored {
		parts = append(parts, "Ignored: true")
	}

	return strings.Join(parts, ", ")
}

func serializeFieldReferences(schema *walker.Schema, schemaVarName string, b *strings.Builder) {
	for i := range schema.Fields {
		field := &schema.Fields[i]
		if !field.HasReference() {
			continue
		}

		if field.RefTypeName() != "" && field.RefProtoBufField() != "" {
			b.WriteString(fmt.Sprintf("%s.Fields[%d].SetReference(%s, %s, %t, %t, %t, %t)\n",
				schemaVarName, i,
				strconv.Quote(field.RefTypeName()),
				strconv.Quote(field.RefProtoBufField()),
				field.RefNoConstraint(),
				field.RefRestrictDelete(),
				field.RefDirectional(),
				field.RefNullable()))
		} else if field.RefOtherSchema() != nil {
			b.WriteString(fmt.Sprintf("%s.Fields[%d].SetParentReference(%s, %s)\n",
				schemaVarName, i,
				getSchemaVarName(schema, field.RefOtherSchema()),
				strconv.Quote(field.RefColumnName())))
		}
	}
}

func getSchemaVarName(currentSchema *walker.Schema, targetSchema *walker.Schema) string {
	if currentSchema.Parent == targetSchema {
		if targetSchema.Parent == nil {
			return "schema"
		}
		return buildSchemaVarName(targetSchema)
	}

	if targetSchema.Parent == nil {
		return "schema"
	}
	log.Fatalf("cannot resolve variable name for parent reference from table %q to table %q",
		currentSchema.Table, targetSchema.Table)
	return ""
}

func buildSchemaVarName(schema *walker.Schema) string {
	if schema.Parent == nil {
		return "schema"
	}

	var parts []string
	current := schema
	for current.Parent != nil {
		index := 0
		for i, child := range current.Parent.Children {
			if child == current {
				index = i
				break
			}
		}
		parts = append([]string{fmt.Sprintf("child%d", index)}, parts...)
		current = current.Parent
	}

	return strings.Join(parts, "_")
}

func dataTypeToConst(dt postgres.DataType) string {
	switch dt {
	case postgres.Bytes:
		return "postgres.Bytes"
	case postgres.Bool:
		return "postgres.Bool"
	case postgres.Numeric:
		return "postgres.Numeric"
	case postgres.String:
		return "postgres.String"
	case postgres.DateTime:
		return "postgres.DateTime"
	case postgres.Map:
		return "postgres.Map"
	case postgres.Enum:
		return "postgres.Enum"
	case postgres.StringArray:
		return "postgres.StringArray"
	case postgres.EnumArray:
		return "postgres.EnumArray"
	case postgres.Integer:
		return "postgres.Integer"
	case postgres.IntArray:
		return "postgres.IntArray"
	case postgres.BigInteger:
		return "postgres.BigInteger"
	case postgres.UUID:
		return "postgres.UUID"
	case postgres.CIDR:
		return "postgres.CIDR"
	case postgres.DateTimeTZ:
		return "postgres.DateTimeTZ"
	case postgres.MessageBytes:
		return "postgres.MessageBytes"
	default:
		return fmt.Sprintf("postgres.DataType(%s)", strconv.Quote(string(dt)))
	}
}

func derivationTypeToConst(dt search.DerivationType) string {
	switch dt {
	case search.CountDerivationType:
		return "search.CountDerivationType"
	case search.SimpleReverseSortDerivationType:
		return "search.SimpleReverseSortDerivationType"
	case search.MaxDerivationType:
		return "search.MaxDerivationType"
	case search.CustomFieldType:
		return "search.CustomFieldType"
	case search.MinDerivationType:
		return "search.MinDerivationType"
	case search.MaxReverseSortDerivationType:
		return "search.MaxReverseSortDerivationType"
	default:
		return fmt.Sprintf("search.DerivationType(%d)", int(dt))
	}
}

// SerializeSearchFields returns Go source code for a search.OptionsMap literal.
func SerializeSearchFields(optionsMap search.OptionsMap, categoryStr string) string {
	var b strings.Builder

	b.WriteString("map[search.FieldLabel]*search.Field{\n")

	fields := optionsMap.Original()
	labels := make([]string, 0, len(fields))
	for label := range fields {
		labels = append(labels, string(label))
	}
	slices.Sort(labels)

	for _, labelStr := range labels {
		label := search.FieldLabel(labelStr)
		field := fields[label]

		b.WriteString(fmt.Sprintf("\t%s: {", strconv.Quote(labelStr)))

		var parts []string
		parts = append(parts, fmt.Sprintf("FieldPath: %s", strconv.Quote(field.FieldPath)))

		dataTypeStr := field.Type.String()
		parts = append(parts, fmt.Sprintf("Type: v1.SearchDataType_%s", dataTypeStr))

		if field.Store {
			parts = append(parts, "Store: true")
		}
		if field.Hidden {
			parts = append(parts, "Hidden: true")
		}

		parts = append(parts, fmt.Sprintf("Category: %s", categoryStr))

		if field.Analyzer != "" {
			parts = append(parts, fmt.Sprintf("Analyzer: %s", strconv.Quote(field.Analyzer)))
		}

		b.WriteString(strings.Join(parts, ", "))
		b.WriteString("},\n")
	}

	b.WriteString("}")

	return b.String()
}

// SerializeEnumEntries returns Go source code for enumregistry.AddValues calls.
func SerializeEnumEntries(before, after map[string]map[string]int32) string {
	var b strings.Builder

	paths := make([]string, 0)
	for path := range after {
		if _, existedBefore := before[path]; !existedBefore {
			paths = append(paths, path)
		}
	}

	if len(paths) == 0 {
		return ""
	}

	slices.Sort(paths)

	for _, path := range paths {
		values := after[path]

		names := make([]string, 0, len(values))
		for name := range values {
			names = append(names, name)
		}
		slices.Sort(names)

		b.WriteString(fmt.Sprintf("enumregistry.AddValues(%s, map[string]int32{", strconv.Quote(path)))
		for i, name := range names {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(fmt.Sprintf("%s: %d", strconv.Quote(name), values[name]))
		}
		b.WriteString("})\n")
	}

	return b.String()
}

// SerializeEnumRegistryFile generates the complete enum_registry_list.go file content.
func SerializeEnumRegistryFile(allEnums map[string]map[string]int32) string {
	var b strings.Builder

	// File header
	b.WriteString("// Code generated by pg-table-bindings. DO NOT EDIT.\n\n")
	b.WriteString("package enumregistry\n\n")
	b.WriteString("func init() {\n")

	// Sort paths for stable output
	paths := make([]string, 0, len(allEnums))
	for path := range allEnums {
		paths = append(paths, path)
	}
	slices.Sort(paths)

	// Generate enumMap
	b.WriteString("\tenumMap = map[string]map[string]int32{\n")
	for _, path := range paths {
		values := allEnums[path]

		// Get names sorted by value number
		names := make([]string, 0, len(values))
		for name := range values {
			names = append(names, name)
		}
		slices.SortFunc(names, func(a, b string) int {
			return int(values[a] - values[b])
		})

		b.WriteString(fmt.Sprintf("\t\t%s: {", strconv.Quote(path)))
		for i, name := range names {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(fmt.Sprintf("%s: %d", strconv.Quote(strings.ToLower(name)), values[name]))
		}
		b.WriteString("},\n")
	}
	b.WriteString("\t}\n\n")

	// Generate reverseEnumMap
	b.WriteString("\treverseEnumMap = map[string]map[int32]string{\n")
	for _, path := range paths {
		values := allEnums[path]

		// Get names sorted by value number
		names := make([]string, 0, len(values))
		for name := range values {
			names = append(names, name)
		}
		slices.SortFunc(names, func(a, b string) int {
			return int(values[a] - values[b])
		})

		b.WriteString(fmt.Sprintf("\t\t%s: {", strconv.Quote(path)))
		for i, name := range names {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(fmt.Sprintf("%d: %s", values[name], strconv.Quote(name)))
		}
		b.WriteString("},\n")
	}
	b.WriteString("\t}\n")
	b.WriteString("}\n")

	return b.String()
}

// SerializeSearchFieldsFile generates a complete *_search.go file with search field definitions.
func SerializeSearchFieldsFile(tableName, searchCategory string, fields map[search.FieldLabel]*search.Field) string {
	var b strings.Builder

	// File header
	b.WriteString("// Code generated by pg-table-bindings. DO NOT EDIT.\n\n")
	b.WriteString("package schema\n\n")
	b.WriteString("import (\n")
	b.WriteString("\tv1 \"github.com/stackrox/rox/generated/api/v1\"\n")
	b.WriteString("\t\"github.com/stackrox/rox/pkg/search\"\n")
	b.WriteString(")\n\n")

	// Variable name: {tableName}SearchFields
	varName := strings.ReplaceAll(tableName, "_", "") + "SearchFields"

	// Generate map literal
	b.WriteString(fmt.Sprintf("var %s = map[search.FieldLabel]*search.Field{\n", varName))

	// Sort field labels for stable output
	labels := make([]search.FieldLabel, 0, len(fields))
	for label := range fields {
		labels = append(labels, label)
	}
	slices.SortFunc(labels, func(a, b search.FieldLabel) int {
		return strings.Compare(string(a), string(b))
	})

	for _, label := range labels {
		field := fields[label]
		b.WriteString(fmt.Sprintf("\t%s: {\n", strconv.Quote(string(label))))
		b.WriteString(fmt.Sprintf("\t\tFieldPath: %s,\n", strconv.Quote(field.FieldPath)))
		b.WriteString(fmt.Sprintf("\t\tType: v1.SearchDataType_%s,\n", field.Type.String()))
		b.WriteString(fmt.Sprintf("\t\tCategory: v1.SearchCategory_%s,\n", searchCategory))

		if field.Store {
			b.WriteString("\t\tStore: true,\n")
		}
		if field.Hidden {
			b.WriteString("\t\tHidden: true,\n")
		}
		if field.Analyzer != "" {
			b.WriteString(fmt.Sprintf("\t\tAnalyzer: %s,\n", strconv.Quote(field.Analyzer)))
		}

		b.WriteString("\t},\n")
	}

	b.WriteString("}\n")

	return b.String()
}
