package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"text/template"

	// Embed is used to import the template files
	_ "embed"

	"github.com/Masterminds/sprig/v3"
	"github.com/golang/protobuf/proto"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/central/postgres/schema"
	_ "github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/pkg/utils"
	"golang.org/x/tools/imports"
)

//go:embed schema.go.tpl
var schemaFile string

//go:embed store.go.tpl
var storeFile string

//go:embed store_test.go.tpl
var storeTestFile string

//go:embed index.go.tpl
var indexFile string

//go:embed permission_checker.go.tpl
var permissionCheckerFile string

var (
	schemaTemplate            = newTemplate(schemaFile)
	storeTemplate             = newTemplate(storeFile)
	storeTestTemplate         = newTemplate(storeTestFile)
	indexTemplate             = newTemplate(indexFile)
	permissionCheckerTemplate = newTemplate(permissionCheckerFile)
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

	// Refs indicate the additional referentiol relationships. Each string is <table_name>:<proto_type>.
	// These are non-embedding relations, that is, this table is not embedded into referenced table to
	// construct the proto message.
	Refs []string

	// When set to true, it means that the schema represents a join table. The generation of mutating functions
	// such as inserts, updates, deletes, is skipped. This is because join tables should be filled from parents.
	JoinTable bool

	// Indicates whether to generate only the schema. If set to true, only the schema is generated, and not store and indexer.
	SchemaOnly bool
}

func renderFile(templateMap map[string]interface{}, temp func(s string) *template.Template, templateFileName string) error {
	buf := bytes.NewBuffer(nil)
	if err := temp(templateFileName).Execute(buf, templateMap); err != nil {
		return err
	}
	file := buf.Bytes()

	formatted, err := imports.Process(templateFileName, file, nil)
	if err != nil {
		return err
	}
	if err := os.WriteFile(templateFileName, formatted, 0644); err != nil {
		return err
	}
	return nil
}

func main() {
	c := &cobra.Command{
		Use: "generate store implementations",
	}

	var props properties
	c.Flags().StringVar(&props.Type, "type", "", "the (Go) name of the object")
	utils.Must(c.MarkFlagRequired("type"))

	c.Flags().StringVar(&props.RegisteredType, "registered-type", "", "the type this is registered in proto as storage.X")

	c.Flags().StringVar(&props.Table, "table", "", "the logical table of the objects")
	utils.Must(c.MarkFlagRequired("table"))

	c.Flags().StringVar(&props.Singular, "singular", "", "the singular name of the object")
	c.Flags().StringVar(&props.SearchCategory, "search-category", "", "the search category to index under")
	c.Flags().StringVar(&props.PermissionChecker, "permission-checker", "", "the permission checker that should be used")
	c.Flags().StringSliceVar(&props.Refs, "references", []string{}, "additional foreign key references as <table_name:type>")
	c.Flags().BoolVar(&props.JoinTable, "join-table", false, "indicates the schema represents a join table. The generation of mutating functions is skipped")
	c.Flags().BoolVar(&props.SchemaOnly, "schema-only", false, "if true, generates only the schema and not store and index")

	c.RunE = func(*cobra.Command, []string) error {
		typ := stringutils.OrDefault(props.RegisteredType, props.Type)
		fmt.Println("Generating for", typ)
		mt := proto.MessageType(typ)
		if mt == nil {
			log.Fatalf("could not find message for type: %s", typ)
		}

		schema := walker.Walk(mt, props.Table)
		if schema.NoPrimaryKey() {
			log.Fatal("No primary key defined, please check relevant proto file and ensure a primary key is specified using the \"sql:\"pk\"\" tag")
		}

		compileFKArgAndAttachToSchema(schema, props.Refs)

		permissionCheckerEnabled := props.PermissionChecker != ""
		templateMap := map[string]interface{}{
			"Type":              props.Type,
			"TrimmedType":       stringutils.GetAfter(props.Type, "."),
			"Table":             props.Table,
			"Schema":            schema,
			"SearchCategory":    fmt.Sprintf("SearchCategory_%s", props.SearchCategory),
			"JoinTable":         props.JoinTable,
			"PermissionChecker": props.PermissionChecker,
			"Obj": object{
				storageType:              props.Type,
				permissionCheckerEnabled: permissionCheckerEnabled,
				isJoinTable:              props.JoinTable,
				schema:                   schema,
			},
		}

		if err := generateSchema(schema); err != nil {
			return err
		}
		if props.SchemaOnly {
			return nil
		}
		if err := renderFile(templateMap, storeTemplate, "store.go"); err != nil {
			return err
		}
		if err := renderFile(templateMap, storeTestTemplate, "store_test.go"); err != nil {
			return err
		}

		if props.SearchCategory != "" {
			if err := renderFile(templateMap, indexTemplate, "index.go"); err != nil {
				return err
			}
		}
		if permissionCheckerEnabled {
			if err := renderFile(templateMap, permissionCheckerTemplate, "permission_checker.go"); err != nil {
				return err
			}
		}

		return nil
	}
	if err := c.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func generateSchema(s *walker.Schema) error {
	return generateSchemaRecursive(s, set.NewStringSet(), schema.SchemaGenFS)
}

func generateSchemaRecursive(schema *walker.Schema, visited set.StringSet, pkgPath string) error {
	if !visited.Add(schema.Table) {
		return nil
	}

	templateMap := map[string]interface{}{
		"Schema":         schema,
		"SearchCategory": "",
	}
	searchCategory, ok := typeToSearchCategoryMap[stringutils.GetAfter(schema.Type, ".")]
	if ok {
		templateMap["SearchCategory"] = fmt.Sprintf("v1.SearchCategory_%s", searchCategory)
	}

	if err := renderFile(templateMap, schemaTemplate, getSchemaFileName(pkgPath, schema.Table)); err != nil {
		return err
	}

	// No top-level schema has a parent unless attached synthetically.
	for _, parent := range schema.Parents {
		if err := generateSchemaRecursive(parent, visited, pkgPath); err != nil {
			return err
		}
	}

	for _, child := range schema.Children {
		// If child is the embedded one, it has already been generated in the file above.
		if child.EmbeddedIn == schema.Table {
			continue
		}
		if err := generateSchemaRecursive(child, visited, pkgPath); err != nil {
			return err
		}
	}
	return nil
}

func getSchemaFileName(dir, table string) string {
	return fmt.Sprintf("%s/%s.go", dir, table)
}

func newTemplate(tpl string) func(name string) *template.Template {
	return func(name string) *template.Template {
		return template.Must(template.New(name).Option("missingkey=error").Funcs(funcMap).Funcs(sprig.TxtFuncMap()).Parse(autogenerated + tpl))
	}
}
