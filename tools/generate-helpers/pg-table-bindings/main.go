package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"

	// Embed is used to import the template files
	_ "embed"

	"github.com/Masterminds/sprig/v3"
	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stackrox/rox/pkg/readable"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/tools/generate-helpers/common"
)

//go:embed schema.go.tpl
var schemaFile string

//go:embed singleton.go.tpl
var singletonFile string

//go:embed singleton_test.go.tpl
var singletonTestFile string

//go:embed store.go.tpl
var storeFile string

//go:embed store_test.go.tpl
var storeTestFile string

//go:embed migration_tool.go.tpl
var migrationToolFile string

//go:embed migration_tool_test.go.tpl
var migrationToolTestFile string

//go:embed optimized_schema.go.tpl
var optimizedSchemaFile string

var (
	schemaTemplate            = newTemplate(schemaFile)
	singletonTemplate         = newTemplate(strings.Join([]string{"\npackage postgres", singletonFile}, "\n"))
	singletonTestTemplate     = newTemplate(singletonTestFile)
	storeTemplate             = newTemplate(storeFile)
	storeTestTemplate         = newTemplate(storeTestFile)
	migrationToolTemplate     = newTemplate(migrationToolFile)
	migrationToolTestTemplate = newTemplate(migrationToolTestFile)
	optimizedSchemaTemplate   = newTemplate(optimizedSchemaFile)
)

type properties struct {
	Type           string
	TrimmedType    string
	Table          string
	RegisteredType string

	SearchCategory string
	ObjectPathName string
	Singular       string
	WriteOptions   bool

	// Refs indicate the additional referentiol relationships. Each string is [<table_name>:]<proto_type>.
	// These are non-embedding relations, that is, this table is not embedded into referenced table to
	// construct the proto message.
	Refs []string

	// When set to true, it means that the schema represents a join table. The generation of mutating functions
	// such as inserts, updates, deletes, is skipped. This is because join tables should be filled from parents.
	JoinTable bool

	// Indicates whether to generate only the schema. If set to true, only the schema is generated, and not store and indexer.
	SchemaOnly bool

	// Indicates the directory in which the generated schema file must go.
	SchemaDirectory string

	// Indicates that we should just generate the singleton store.
	SingletonStore bool

	// Indicates the scope of search. Set this field to limit search to only some categories in case of overlapping
	// search fields.
	SearchScope []string

	// Indicates whether stores should use Postgres copyFrom operation or not.
	NoCopyFrom bool

	// Generate conversion functions with schema.
	ConversionFuncs bool

	// Indicates that there is a foreign key cycle relationship. Should be defined as <Embedded FK Field>:<Referenced Field>.
	Cycle string

	// The feature flag that specifies if the schema should be registered.
	FeatureFlag string

	// Indicates the store should be mirrored in memory.
	CachedStore bool

	// Provides default sort option field
	DefaultSortField string

	// Informs to reverse the default sort option
	ReverseDefaultSort bool

	// Provides options map for sort option transforms
	TransformSortOptions string

	// Generate optimized schema files
	GenerateOptimizedSchema bool
}

type parsedReference struct {
	TypeName string
	Table    string
}

func main() {
	c := &cobra.Command{
		Use: "generate store implementations",
	}

	var props properties
	c.Flags().StringVar(&props.Type, "type", "", "the (Go) name of the object")
	utils.Must(c.MarkFlagRequired("type"))

	c.Flags().StringVar(&props.FeatureFlag, "feature-flag", "", "the feature flag that registers the schema")
	c.Flags().StringVar(&props.RegisteredType, "registered-type", "", "the type this is registered in proto as storage.X")

	c.Flags().StringVar(&props.Table, "table", "", "the logical table of the objects, default to lower snake_case of type")

	c.Flags().StringVar(&props.Singular, "singular", "", "the singular name of the object")
	c.Flags().StringVar(&props.SearchCategory, "search-category", "", "the search category to index under")
	c.Flags().StringSliceVar(&props.Refs, "references", []string{}, "additional foreign key references, comma seperated of <[table_name:]type>")
	c.Flags().BoolVar(&props.JoinTable, "read-only-store", false, "if set to true, creates read-only store")
	c.Flags().BoolVar(&props.NoCopyFrom, "no-copy-from", false, "if true, indicates that the store should not use Postgres copyFrom operation")
	c.Flags().BoolVar(&props.SchemaOnly, "schema-only", false, "if true, generates only the schema and not store and index")
	c.Flags().StringVar(&props.SchemaDirectory, "schema-directory", "", "the directory in which to generate the schema")
	c.Flags().BoolVar(&props.SingletonStore, "singleton", false, "indicates that we should just generate the singleton store")
	c.Flags().StringSliceVar(&props.SearchScope, "search-scope", []string{}, "if set, the search is scoped to specified search categories. comma seperated of search categories")
	c.Flags().BoolVar(&props.CachedStore, "cached-store", false, "if true, ensure the store is mirrored in a memory cache (can be dangerous on high cardinality stores, use with care)")
	c.Flags().StringVar(&props.DefaultSortField, "default-sort", "", "if set, provides a default sort for search if one is not present")
	c.Flags().BoolVar(&props.ReverseDefaultSort, "reverse-default-sort", false, "if true, reverses the default sort")
	c.Flags().StringVar(&props.TransformSortOptions, "transform-sort-options", "", "if set, provides an option map for sort transforms")
	c.Flags().BoolVar(&props.GenerateOptimizedSchema, "generate-optimized-schema", true, "if true, generates optimized schema files with pre-computed search fields")
	utils.Must(c.MarkFlagRequired("schema-directory"))

	c.Flags().StringVar(&props.Cycle, "cycle", "", "indicates that there is a cyclical foreign key reference, should be the path to the embedded foreign key")
	c.Flags().BoolVar(&props.ConversionFuncs, "conversion-funcs", false, "indicates that we should generate conversion functions between protobuf types to/from Gorm model")
	c.RunE = func(*cobra.Command, []string) error {
		typ := stringutils.OrDefault(props.RegisteredType, props.Type)
		fmt.Println(readable.Time(time.Now()), "Generating for", typ)
		mt := protoutils.MessageType(typ)
		if mt == nil {
			log.Fatalf("could not find message for type: %s", typ)
		}
		trimmedType := stringutils.GetAfter(props.Type, ".")
		if props.Table == "" {
			props.Table = pgutils.NamingStrategy.TableName(trimmedType)
		}
		schema := walker.Walk(mt, props.Table)
		if schema.NoPrimaryKey() && !props.SingletonStore {
			log.Fatal("No primary key defined, please check relevant proto file and ensure a primary key is specified using the \"sql:\"pk\"\" tag")
		}
		if schema.MultiplePrimaryKeys() {
			log.Fatal("Multiple primary keys defined, please check relevant proto file and ensure a primary key is specified once using the \"sql:\"pk\"\" tag")
		}

		var searchCategory string
		if props.SearchCategory != "" {
			if asInt, err := strconv.Atoi(props.SearchCategory); err == nil {
				searchCategory = fmt.Sprintf("SearchCategory(%d)", asInt)
			} else {
				searchCategory = fmt.Sprintf("SearchCategory_%s", props.SearchCategory)
			}
		}

		searchScope := make([]string, 0, len(props.SearchScope))
		if len(props.SearchScope) > 0 {
			for _, category := range props.SearchScope {
				searchScope = append(searchScope, v1SearchCategoryString(category))
			}
		}

		defaultSort := props.DefaultSortField

		var embeddedFK string
		if props.Cycle != "" {
			embeddedFK = props.Cycle
		}

		// remove any self references
		parsedReferences := parseReferencesAndInjectPeerSchemas(schema, props.Refs)
		filteredReferences := make([]parsedReference, 0, len(parsedReferences))
		for _, ref := range parsedReferences {
			if ref.Table != props.Table {
				filteredReferences = append(filteredReferences, ref)
			}
		}

		templateMap := map[string]interface{}{
			"Type":           props.Type,
			"TrimmedType":    trimmedType,
			"Table":          props.Table,
			"Schema":         schema,
			"SearchCategory": searchCategory,
			"JoinTable":      props.JoinTable,
			"Obj": object{
				storageType: props.Type,
				schema:      schema,
			},
			"NoCopyFrom":           props.NoCopyFrom,
			"Cycle":                embeddedFK != "",
			"EmbeddedFK":           embeddedFK,
			"References":           filteredReferences,
			"SearchScope":          searchScope,
			"RegisterSchema":       !props.ConversionFuncs,
			"FeatureFlag":          props.FeatureFlag,
			"CachedStore":          props.CachedStore,
			"DefaultSortStore":     defaultSort != "",
			"DefaultSort":          defaultSort,
			"ReverseDefaultSort":   props.ReverseDefaultSort,
			"TransformSortOptions": props.TransformSortOptions,
			"DefaultTransform":     props.TransformSortOptions != "",
			"Singleton":            props.SingletonStore,
		}

		if err := common.RenderFile(templateMap, schemaTemplate, getSchemaFileName(props.SchemaDirectory, schema.Table)); err != nil {
			return err
		}

		if props.GenerateOptimizedSchema {
			if err := generateOptimizedSchema(schema, props, trimmedType, searchCategory); err != nil {
				return err
			}
		}

		if props.ConversionFuncs {
			if err := generateConversionFuncs(schema, props.SchemaDirectory); err != nil {
				return err
			}
		}
		if !props.SchemaOnly {
			if props.SingletonStore {
				if err := common.RenderFile(templateMap, singletonTemplate, "store.go"); err != nil {
					return err
				}
				if err := common.RenderFile(templateMap, singletonTestTemplate, "store_test.go"); err != nil {
					return err
				}
			} else {
				if err := common.RenderFile(templateMap, storeTemplate, "store.go"); err != nil {
					return err
				}
				if err := common.RenderFile(templateMap, storeTestTemplate, "store_test.go"); err != nil {
					return err
				}
			}
		}

		return nil
	}
	if err := c.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func generateConversionFuncs(s *walker.Schema, dir string) error {
	templateMap := map[string]interface{}{
		"Schema": s,
	}

	if err := common.RenderFile(templateMap, migrationToolTemplate, getConversionToolFileName(dir, s.Table)); err != nil {
		return err
	}
	if err := common.RenderFile(templateMap, migrationToolTestTemplate, getConversionTestFileName(dir, s.Table)); err != nil {
		return err
	}
	return nil
}

func getSchemaFileName(dir, table string) string {
	return fmt.Sprintf("%s/%s.go", dir, table)
}

func getConversionToolFileName(dir, table string) string {
	return fmt.Sprintf("%s/convert_%s.go", dir, table)
}

func getConversionTestFileName(dir, table string) string {
	return fmt.Sprintf("%s/convert_%s_test.go", dir, table)
}

func newTemplate(tpl string) func(name string) *template.Template {
	return func(name string) *template.Template {
		return template.Must(template.New(name).Option("missingkey=error").Funcs(funcMap).Funcs(sprig.TxtFuncMap()).Parse(autogenerated + tpl))
	}
}

func v1SearchCategoryString(category string) string {
	if asInt, err := strconv.Atoi(category); err == nil {
		return fmt.Sprintf("v1.SearchCategory(%d)", asInt)
	}
	return fmt.Sprintf("v1.SearchCategory_%s", category)
}

// OptimizedSchemaData represents the data for generating optimized schema files
type OptimizedSchemaData struct {
	TypeName       string
	Table          string
	Type           string
	SearchCategory string
	Fields         []OptimizedSchemaField
	SearchFields   []OptimizedSearchField
	ChildSchemas   []OptimizedSchemaData
}

type OptimizedSchemaField struct {
	Name         string
	ColumnName   string
	Type         string
	SQLType      string
	DataType     string
	IsPrimaryKey bool
	SearchFieldName string // The field name for search if this field is searchable
}

type OptimizedSearchField struct {
	FieldLabel     string
	FieldPath      string
	Store          bool
	Hidden         bool
	SearchCategory string
	Analyzer       string
}

func generateOptimizedSchema(schema *walker.Schema, props properties, trimmedType, searchCategory string) error {
	// Generate the optimized schema file in the same directory

	// Generate search fields using reflection on the protobuf type first
	// Only include fields that actually exist in this type (walker.Walk parity)
	searchFields := generateSearchFields(props.Type, trimmedType)

	// Create a map from field path to search field name for lookup
	fieldPathToSearchName := make(map[string]string)
	for _, searchField := range searchFields {
		fieldPathToSearchName[searchField.FieldPath] = searchField.FieldLabel
	}

	// Extract fields from walker schema
	var fields []OptimizedSchemaField
	for _, field := range schema.Fields {
		// Find the search field name for this field by matching the field path
		searchFieldName := ""
		// Try multiple variations of the field path to match
		possiblePaths := []string{
			// Direct field name matching for root level fields
			"." + strings.ToLower(field.Name),
			"." + pgutils.NamingStrategy.ColumnName("", field.ColumnName),
			// Entity-prefixed paths for root level fields
			trimmedType + "." + strings.ToLower(field.Name),
			trimmedType + "." + strings.ToLower(field.ColumnName),
		}
		for _, path := range possiblePaths {
			if fieldName, exists := fieldPathToSearchName[path]; exists {
				searchFieldName = fieldName
				break
			}
		}

		optimizedField := OptimizedSchemaField{
			Name:            field.Name,
			ColumnName:      field.ColumnName,
			Type:            field.Type,
			SQLType:         field.SQLType,
			DataType:        getDataTypeName(field.DataType),
			IsPrimaryKey:    field.Options.PrimaryKey,
			SearchFieldName: searchFieldName,
		}
		fields = append(fields, optimizedField)
	}

	// Clean search category name
	cleanSearchCategory := strings.TrimPrefix(searchCategory, "v1.")

	// Extract child schemas recursively
	var childSchemas []OptimizedSchemaData
	for _, childSchema := range schema.Children {
		childData := extractSchemaDataRecursively(childSchema, cleanSearchCategory, fieldPathToSearchName, strings.ToLower(trimmedType))
		childSchemas = append(childSchemas, childData)
	}

	data := OptimizedSchemaData{
		TypeName:       trimmedType,
		Table:          schema.Table,
		Type:           schema.Type, // Use schema.Type instead of props.Type to preserve pointer asterisk
		SearchCategory: cleanSearchCategory,
		Fields:         fields,
		SearchFields:   searchFields,
		ChildSchemas:   childSchemas,
	}

	templateMap := map[string]interface{}{
		"TypeName":       data.TypeName,
		"Table":          data.Table,
		"Type":           data.Type,
		"SearchCategory": data.SearchCategory,
		"Fields":         data.Fields,
		"SearchFields":   data.SearchFields,
		"ChildSchemas":   data.ChildSchemas,
	}

	internalDir := filepath.Join(props.SchemaDirectory, "internal")
	if err := os.MkdirAll(internalDir, 0755); err != nil {
		return fmt.Errorf("failed to create internal directory: %w", err)
	}
	fileName := filepath.Join(internalDir, fmt.Sprintf("%s.go", schema.Table))
	return common.RenderFile(templateMap, optimizedSchemaTemplate, fileName)
}

func extractFieldsFromSchema(schema *walker.Schema, fieldPathToSearchName map[string]string, entityPrefix string) []OptimizedSchemaField {
	var fields []OptimizedSchemaField
	for _, field := range schema.Fields {
		// Find the search field name for this field by matching possible paths
		searchFieldName := ""

		// For child schema fields, we need to construct the full path
		// For example: deployment.containers.image.name.full_name should map to FullName field
		possiblePaths := []string{
			// Direct field name matching
			"." + strings.ToLower(field.Name),
			"." + strings.ToLower(field.ColumnName),
			// Entity-prefixed paths
			entityPrefix + "." + strings.ToLower(field.Name),
			entityPrefix + "." + strings.ToLower(field.ColumnName),
		}

		// For child schemas, also try to match by constructing the full hierarchical path
		if strings.Contains(schema.Table, "_") {
			// This is a child table, try to construct full path
			// For deployments_containers, we want paths like deployment.containers.xxx
			tableParts := strings.Split(schema.Table, "_")
			if len(tableParts) >= 2 {
				// Build hierarchical path: deployment.containers.field_name
				hierarchicalPath := entityPrefix
				for i := 1; i < len(tableParts); i++ {
					hierarchicalPath += "." + tableParts[i]
				}

				// Add variations with field names
				possiblePaths = append(possiblePaths,
					hierarchicalPath + "." + strings.ToLower(field.Name),
					hierarchicalPath + "." + strings.ToLower(field.ColumnName),
				)

				// Special handling for common field patterns
				if field.Name == "FullName" && strings.Contains(field.ColumnName, "Image_Name_FullName") {
					possiblePaths = append(possiblePaths,
						".containers.image.name.full_name",
					)
				}
				if field.Name == "Registry" && strings.Contains(field.ColumnName, "Image_Name_Registry") {
					possiblePaths = append(possiblePaths,
						".containers.image.name.registry",
					)
				}
				if field.Name == "Remote" && strings.Contains(field.ColumnName, "Image_Name_Remote") {
					possiblePaths = append(possiblePaths,
						".containers.image.name.remote",
					)
				}
				if field.Name == "Tag" && strings.Contains(field.ColumnName, "Image_Name_Tag") {
					possiblePaths = append(possiblePaths,
						".containers.image.name.tag",
					)
				}
				if field.Name == "Id" && strings.Contains(field.ColumnName, "Image_Id") {
					possiblePaths = append(possiblePaths,
						".containers.image.id",
					)
				}
				if field.Name == "IdV2" && strings.Contains(field.ColumnName, "Image_IdV2") {
					possiblePaths = append(possiblePaths,
						".containers.image.id_v2",
					)
				}
				if field.Name == "Privileged" && strings.Contains(field.ColumnName, "SecurityContext_Privileged") {
					possiblePaths = append(possiblePaths,
						".containers.security_context.privileged",
					)
				}
				if field.Name == "DropCapabilities" && strings.Contains(field.ColumnName, "SecurityContext_DropCapabilities") {
					possiblePaths = append(possiblePaths,
						".containers.security_context.drop_capabilities",
					)
				}
				if field.Name == "AddCapabilities" && strings.Contains(field.ColumnName, "SecurityContext_AddCapabilities") {
					possiblePaths = append(possiblePaths,
						".containers.security_context.add_capabilities",
					)
				}
				if field.Name == "ReadOnlyRootFilesystem" && strings.Contains(field.ColumnName, "SecurityContext_ReadOnlyRootFilesystem") {
					possiblePaths = append(possiblePaths,
						".containers.security_context.read_only_root_filesystem",
					)
				}
				if field.Name == "CpuCoresRequest" && strings.Contains(field.ColumnName, "Resources_CpuCoresRequest") {
					possiblePaths = append(possiblePaths,
						".containers.resources.cpu_cores_request",
					)
				}
				if field.Name == "CpuCoresLimit" && strings.Contains(field.ColumnName, "Resources_CpuCoresLimit") {
					possiblePaths = append(possiblePaths,
						".containers.resources.cpu_cores_limit",
					)
				}
				if field.Name == "MemoryMbRequest" && strings.Contains(field.ColumnName, "Resources_MemoryMbRequest") {
					possiblePaths = append(possiblePaths,
						".containers.resources.memory_mb_request",
					)
				}
				if field.Name == "MemoryMbLimit" && strings.Contains(field.ColumnName, "Resources_MemoryMbLimit") {
					possiblePaths = append(possiblePaths,
						".containers.resources.memory_mb_limit",
					)
				}

				// Environment variable patterns
				if field.Name == "Key" && strings.Contains(schema.Table, "envs") {
					possiblePaths = append(possiblePaths,
						".containers.config.env.key",
					)
				}
				if field.Name == "Value" && strings.Contains(schema.Table, "envs") {
					possiblePaths = append(possiblePaths,
						".containers.config.env.value",
					)
				}
				if field.Name == "EnvVarSource" && strings.Contains(schema.Table, "envs") {
					possiblePaths = append(possiblePaths,
						".containers.config.env.env_var_source",
					)
				}

				// Volume patterns
				if field.Name == "Name" && strings.Contains(schema.Table, "volumes") {
					possiblePaths = append(possiblePaths,
						".containers.volumes.name",
					)
				}
				if field.Name == "Source" && strings.Contains(schema.Table, "volumes") {
					possiblePaths = append(possiblePaths,
						".containers.volumes.source",
					)
				}
				if field.Name == "Destination" && strings.Contains(schema.Table, "volumes") {
					possiblePaths = append(possiblePaths,
						".containers.volumes.destination",
					)
				}
				if field.Name == "ReadOnly" && strings.Contains(schema.Table, "volumes") {
					possiblePaths = append(possiblePaths,
						".containers.volumes.read_only",
					)
				}
				if field.Name == "Type" && strings.Contains(schema.Table, "volumes") {
					possiblePaths = append(possiblePaths,
						".containers.volumes.type",
					)
				}

				// Secret patterns
				if field.Name == "Name" && strings.Contains(schema.Table, "secrets") {
					possiblePaths = append(possiblePaths,
						".containers.secrets.name",
					)
				}
				if field.Name == "Path" && strings.Contains(schema.Table, "secrets") {
					possiblePaths = append(possiblePaths,
						".containers.secrets.path",
					)
				}

				// Port patterns
				if field.Name == "ContainerPort" && strings.Contains(schema.Table, "ports") {
					possiblePaths = append(possiblePaths,
						".ports.container_port",
					)
				}
				if field.Name == "Protocol" && strings.Contains(schema.Table, "ports") && !strings.Contains(schema.Table, "exposure") {
					possiblePaths = append(possiblePaths,
						".ports.protocol",
					)
				}
				if field.Name == "Exposure" && strings.Contains(schema.Table, "ports") {
					possiblePaths = append(possiblePaths,
						".ports.exposure",
					)
				}

				// Port exposure info patterns
				if field.Name == "Level" && strings.Contains(schema.Table, "exposure_infos") {
					possiblePaths = append(possiblePaths,
						".ports.exposure_infos.level",
					)
				}
				if field.Name == "ServiceName" && strings.Contains(schema.Table, "exposure_infos") {
					possiblePaths = append(possiblePaths,
						".ports.exposure_infos.service_name",
					)
				}
				if field.Name == "ServicePort" && strings.Contains(schema.Table, "exposure_infos") {
					possiblePaths = append(possiblePaths,
						".ports.exposure_infos.service_port",
					)
				}
				if field.Name == "NodePort" && strings.Contains(schema.Table, "exposure_infos") {
					possiblePaths = append(possiblePaths,
						".ports.exposure_infos.node_port",
					)
				}
				if field.Name == "ExternalIps" && strings.Contains(schema.Table, "exposure_infos") {
					possiblePaths = append(possiblePaths,
						".ports.exposure_infos.external_ips",
					)
				}
				if field.Name == "ExternalHostnames" && strings.Contains(schema.Table, "exposure_infos") {
					possiblePaths = append(possiblePaths,
						".ports.exposure_infos.external_hostnames",
					)
				}
			}
		}

		for _, path := range possiblePaths {
			if fieldName, exists := fieldPathToSearchName[path]; exists {
				searchFieldName = fieldName
				break
			}
		}

		optimizedField := OptimizedSchemaField{
			Name:            field.Name,
			ColumnName:      field.ColumnName,
			Type:            field.Type,
			SQLType:         field.SQLType,
			DataType:        getDataTypeName(field.DataType),
			IsPrimaryKey:    field.Options.PrimaryKey,
			SearchFieldName: searchFieldName,
		}
		fields = append(fields, optimizedField)
	}
	return fields
}

func extractSchemaDataRecursively(schema *walker.Schema, searchCategory string, fieldPathToSearchName map[string]string, entityPrefix string) OptimizedSchemaData {
	fields := extractFieldsFromSchema(schema, fieldPathToSearchName, entityPrefix)

	// Extract child schemas recursively
	var childSchemas []OptimizedSchemaData
	for _, childSchema := range schema.Children {
		childData := extractSchemaDataRecursively(childSchema, searchCategory, fieldPathToSearchName, entityPrefix)
		childSchemas = append(childSchemas, childData)
	}

	return OptimizedSchemaData{
		TypeName:       schema.TypeName,
		Table:          schema.Table,
		Type:           schema.Type,
		SearchCategory: searchCategory,
		Fields:         fields,
		SearchFields:   []OptimizedSearchField{}, // Child schemas don't have separate search fields
		ChildSchemas:   childSchemas,
	}
}

func getDataTypeName(dataType interface{}) string {
	if dataType == nil {
		return "" // nil should map to empty string, not "String"
	}

	// Convert the DataType value to string and map to Go constant names
	dataTypeStr := fmt.Sprintf("%v", dataType)
	switch dataTypeStr {
	case "":
		return "" // empty DataType should remain empty
	case "bytes":
		return "Bytes"
	case "bool":
		return "Bool"
	case "numeric":
		return "Numeric"
	case "string":
		return "String"
	case "datetime":
		return "DateTime"
	case "map":
		return "Map"
	case "enum":
		return "Enum"
	case "stringarray":
		return "StringArray"
	case "enumarray":
		return "EnumArray"
	case "integer":
		return "Integer"
	case "intarray":
		return "IntArray"
	case "biginteger":
		return "BigInteger"
	case "uuid":
		return "UUID"
	case "cidr":
		return "CIDR"
	default:
		return "String" // fallback
	}
}

func generateSearchFieldsWithScope(protoType, trimmedType string, searchScope []string) []OptimizedSearchField {
	// Generate fields for the primary type
	searchFields := generateSearchFields(protoType, trimmedType)

	// Create a map to track field labels and avoid duplicates
	fieldLabelMap := make(map[string]bool)
	for _, field := range searchFields {
		fieldLabelMap[field.FieldLabel] = true
	}

	// Add fields from other types in the search scope
	for _, scopeCategory := range searchScope {
		// Skip the primary category to avoid duplicates
		primaryCategory := getSearchCategoryName(getSearchCategoryForType(trimmedType))
		if scopeCategory == primaryCategory {
			continue
		}

		// Get the type name for this search category
		scopeTypeName := getTypeNameForSearchCategory(scopeCategory)
		if scopeTypeName != "" && scopeTypeName != trimmedType {
			// Generate search fields for this scope type
			scopeFields := generateSearchFields("", scopeTypeName)
			// Add only unique fields to our list
			for _, field := range scopeFields {
				if !fieldLabelMap[field.FieldLabel] {
					searchFields = append(searchFields, field)
					fieldLabelMap[field.FieldLabel] = true
				}
			}
		}
	}

	return searchFields
}

func generateSearchFields(protoType, trimmedType string) []OptimizedSearchField {
	if protoType == "" && trimmedType == "" {
		return []OptimizedSearchField{}
	}

	// Get the protobuf type using reflection from the storage package

	// Try to find the actual type by name in the storage package
	var protoObj interface{}
	switch trimmedType {
	case "Alert":
		protoObj = &storage.Alert{}
	case "Cluster":
		protoObj = &storage.Cluster{}
	case "Deployment":
		protoObj = &storage.Deployment{}
	case "Image":
		protoObj = &storage.Image{}
	case "Policy":
		protoObj = &storage.Policy{}
	case "Node":
		protoObj = &storage.Node{}
	case "Secret":
		protoObj = &storage.Secret{}
	case "Role":
		protoObj = &storage.Role{}
	case "K8SRoleBinding":
		protoObj = &storage.K8SRoleBinding{}
	case "K8sRoleBinding":
		protoObj = &storage.K8SRoleBinding{}
	// Add more cases as needed for other types
	default:
		// For types we don't have explicit cases for, return empty for now
		log.Printf("No explicit case for type %s, returning empty search fields", trimmedType)
		return []OptimizedSearchField{}
	}

	if protoObj == nil {
		return []OptimizedSearchField{}
	}

	// Use the existing search.Walk functionality to get search fields
	searchCategory := getSearchCategoryForType(trimmedType)
	optionsMap := search.Walk(searchCategory, "", protoObj)

	// Convert OptionsMap to our OptimizedSearchField format
	var searchFields []OptimizedSearchField
	for fieldLabel, field := range optionsMap.Original() {
		// Handle field path prefixes based on entity type
		// Most entities should use dot-prefixed paths (e.g., ".policy.name")
		// Only specific entities like K8SRoleBinding should use entity prefixes (e.g., "k8srolebinding.role_id")
		fieldPath := field.FieldPath
		if strings.HasPrefix(fieldPath, ".") && len(fieldPath) > 1 {
			// Only add entity prefix for specific types that require it
			if shouldUseEntityPrefix(trimmedType) {
				// Remove the leading dot and prefix with table name
				tableName := getTableNameFromType(trimmedType)
				fieldPath = tableName + fieldPath
			}
			// For other types, keep the dot prefix as-is
		}

		searchField := OptimizedSearchField{
			FieldLabel:     string(fieldLabel),
			FieldPath:      fieldPath,
			Store:          field.Store,
			Hidden:         field.Hidden,
			SearchCategory: getSearchCategoryName(searchCategory),
		}

		// Add analyzer if present
		if field.Analyzer != "" {
			searchField.Analyzer = field.Analyzer
		}

		searchFields = append(searchFields, searchField)
	}

	// Sort searchFields by FieldLabel to ensure stable order between generations
	sort.Slice(searchFields, func(i, j int) bool {
		return searchFields[i].FieldLabel < searchFields[j].FieldLabel
	})

	return searchFields
}

func getSearchCategoryForType(typeName string) v1.SearchCategory {
	switch typeName {
	case "Alert":
		return v1.SearchCategory_ALERTS
	case "Cluster":
		return v1.SearchCategory_CLUSTERS
	case "Deployment":
		return v1.SearchCategory_DEPLOYMENTS
	case "Image":
		return v1.SearchCategory_IMAGES
	case "Policy":
		return v1.SearchCategory_POLICIES
	case "Node":
		return v1.SearchCategory_NODES
	case "Secret":
		return v1.SearchCategory_SECRETS
	case "Role":
		return v1.SearchCategory_ROLES
	case "K8SRoleBinding":
		return v1.SearchCategory_ROLEBINDINGS
	case "K8sRoleBinding":
		return v1.SearchCategory_ROLEBINDINGS
	// Add more mappings as needed
	default:
		return v1.SearchCategory_SEARCH_UNSET
	}
}

func getSearchCategoryName(category v1.SearchCategory) string {
	// Convert category back to string name for template
	switch category {
	case v1.SearchCategory_ALERTS:
		return "SearchCategory_ALERTS"
	case v1.SearchCategory_CLUSTERS:
		return "SearchCategory_CLUSTERS"
	case v1.SearchCategory_DEPLOYMENTS:
		return "SearchCategory_DEPLOYMENTS"
	case v1.SearchCategory_IMAGES:
		return "SearchCategory_IMAGES"
	case v1.SearchCategory_POLICIES:
		return "SearchCategory_POLICIES"
	case v1.SearchCategory_NODES:
		return "SearchCategory_NODES"
	case v1.SearchCategory_SECRETS:
		return "SearchCategory_SECRETS"
	case v1.SearchCategory_ROLES:
		return "SearchCategory_ROLES"
	case v1.SearchCategory_ROLEBINDINGS:
		return "SearchCategory_ROLEBINDINGS"
	default:
		return "SearchCategory_SEARCH_UNSET"
	}
}

func getTableNameFromType(typeName string) string {
	// Map type names to their corresponding table names for backward compatibility
	switch typeName {
	case "Alert":
		return "alert"
	case "Cluster":
		return "cluster"
	case "Deployment":
		return "deployment"
	case "Image":
		return "image"
	case "Policy":
		return "policy"
	case "Node":
		return "node"
	case "Secret":
		return "secret"
	case "Role":
		return "k8srole"
	case "K8SRoleBinding":
		return "k8srolebinding"
	case "K8sRoleBinding":
		return "k8srolebinding"
	// Add more mappings as needed
	default:
		// Fallback: convert to lowercase and snake_case
		return pgutils.NamingStrategy.TableName(typeName)
	}
}

func getTypeNameForSearchCategory(searchCategory string) string {
	// Map search categories back to type names
	switch searchCategory {
	case "SearchCategory_ALERTS", "ALERTS":
		return "Alert"
	case "SearchCategory_CLUSTERS", "CLUSTERS":
		return "Cluster"
	case "SearchCategory_DEPLOYMENTS", "DEPLOYMENTS":
		return "Deployment"
	case "SearchCategory_IMAGES", "IMAGES":
		return "Image"
	case "SearchCategory_POLICIES", "POLICIES":
		return "Policy"
	case "SearchCategory_NODES", "NODES":
		return "Node"
	case "SearchCategory_SECRETS", "SECRETS":
		return "Secret"
	case "SearchCategory_ROLES", "ROLES":
		return "Role"
	case "SearchCategory_ROLEBINDINGS", "ROLEBINDINGS":
		return "K8SRoleBinding"
	default:
		return ""
	}
}

func shouldUseEntityPrefix(typeName string) bool {
	// Only role bindings need entity prefixes for backward compatibility
	// All other types should use dot prefixes to match search.Walk behavior
	switch typeName {
	case "K8SRoleBinding", "K8sRoleBinding":
		return true
	default:
		return false
	}
}

