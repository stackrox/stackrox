package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"time"

	// Embed is used to import the template files
	_ "embed"

	"github.com/Masterminds/sprig/v3"
	"github.com/spf13/cobra"
	_ "github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stackrox/rox/pkg/readable"
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
}

type OptimizedSchemaField struct {
	Name         string
	ColumnName   string
	Type         string
	SQLType      string
	DataType     string
	IsPrimaryKey bool
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

	// Extract fields from walker schema
	var fields []OptimizedSchemaField
	for _, field := range schema.Fields {
		optimizedField := OptimizedSchemaField{
			Name:         field.Name,
			ColumnName:   field.ColumnName,
			Type:         field.Type,
			SQLType:      field.SQLType,
			DataType:     getDataTypeName(field.DataType),
			IsPrimaryKey: field.Options.PrimaryKey,
		}
		fields = append(fields, optimizedField)
	}

	// Generate search fields (simplified approach - use empty list for now)
	var searchFields []OptimizedSearchField
	// Note: Search field generation is complex and will be handled by the build system
	// For now, we generate the schema structure and search fields will be empty

	// Clean search category name
	cleanSearchCategory := strings.TrimPrefix(searchCategory, "v1.")

	data := OptimizedSchemaData{
		TypeName:       trimmedType,
		Table:          schema.Table,
		Type:           props.Type,
		SearchCategory: cleanSearchCategory,
		Fields:         fields,
		SearchFields:   searchFields,
	}

	templateMap := map[string]interface{}{
		"TypeName":       data.TypeName,
		"Table":          data.Table,
		"Type":           data.Type,
		"SearchCategory": data.SearchCategory,
		"Fields":         data.Fields,
		"SearchFields":   data.SearchFields,
	}

	fileName := filepath.Join(props.SchemaDirectory, fmt.Sprintf("generated_%s.go", schema.Table))
	return common.RenderFile(templateMap, optimizedSchemaTemplate, fileName)
}

func getDataTypeName(dataType interface{}) string {
	switch dataType {
	case nil:
		return "String" // default
	default:
		// Try to extract the type name from the interface
		dataTypeStr := fmt.Sprintf("%v", dataType)
		if strings.Contains(dataTypeStr, "String") {
			return "String"
		} else if strings.Contains(dataTypeStr, "Integer") {
			return "Integer"
		} else if strings.Contains(dataTypeStr, "Bool") {
			return "Bool"
		} else if strings.Contains(dataTypeStr, "DateTime") {
			return "DateTime"
		} else if strings.Contains(dataTypeStr, "Map") {
			return "Map"
		} else if strings.Contains(dataTypeStr, "Array") {
			return "StringArray"
		} else {
			return "String" // fallback
		}
	}
}
