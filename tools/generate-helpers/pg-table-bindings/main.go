package main

import (
	"fmt"
	"log"
	"os"
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

var (
	schemaTemplate            = newTemplate(schemaFile)
	singletonTemplate         = newTemplate(strings.Join([]string{"\npackage postgres", singletonFile}, "\n"))
	singletonTestTemplate     = newTemplate(singletonTestFile)
	storeTemplate             = newTemplate(storeFile)
	storeTestTemplate         = newTemplate(storeTestFile)
	migrationToolTemplate     = newTemplate(migrationToolFile)
	migrationToolTestTemplate = newTemplate(migrationToolTestFile)
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

	PermissionChecker string

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
	c.Flags().StringVar(&props.PermissionChecker, "permission-checker", "", "the permission checker that should be used")
	c.Flags().StringSliceVar(&props.Refs, "references", []string{}, "additional foreign key references, comma seperated of <[table_name:]type>")
	c.Flags().BoolVar(&props.JoinTable, "read-only-store", false, "if set to true, creates read-only store")
	c.Flags().BoolVar(&props.NoCopyFrom, "no-copy-from", false, "if true, indicates that the store should not use Postgres copyFrom operation")
	c.Flags().BoolVar(&props.SchemaOnly, "schema-only", false, "if true, generates only the schema and not store and index")
	c.Flags().StringVar(&props.SchemaDirectory, "schema-directory", "", "the directory in which to generate the schema")
	c.Flags().BoolVar(&props.SingletonStore, "singleton", false, "indicates that we should just generate the singleton store")
	c.Flags().StringSliceVar(&props.SearchScope, "search-scope", []string{}, "if set, the search is scoped to specified search categories. comma seperated of search categories")
	c.Flags().BoolVar(&props.CachedStore, "cached-store", false, "if true, ensure the store is mirrored in a memory cache (can be dangerous on high cardinality stores, use with care)")
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

		permissionCheckerEnabled := props.PermissionChecker != ""
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
			"Type":              props.Type,
			"TrimmedType":       trimmedType,
			"Table":             props.Table,
			"Schema":            schema,
			"SearchCategory":    searchCategory,
			"JoinTable":         props.JoinTable,
			"PermissionChecker": props.PermissionChecker,
			"Obj": object{
				storageType:              props.Type,
				permissionCheckerEnabled: permissionCheckerEnabled,
				schema:                   schema,
			},
			"NoCopyFrom":     props.NoCopyFrom,
			"Cycle":          embeddedFK != "",
			"EmbeddedFK":     embeddedFK,
			"References":     filteredReferences,
			"SearchScope":    searchScope,
			"RegisterSchema": !props.ConversionFuncs,
			"FeatureFlag":    props.FeatureFlag,
			"CachedStore":    props.CachedStore,
		}

		if err := common.RenderFile(templateMap, schemaTemplate, getSchemaFileName(props.SchemaDirectory, schema.Table)); err != nil {
			return err
		}

		if props.ConversionFuncs {
			if err := generateConverstionFuncs(schema, props.SchemaDirectory); err != nil {
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

func generateConverstionFuncs(s *walker.Schema, dir string) error {
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
