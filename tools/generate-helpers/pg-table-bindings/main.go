package main

import (
	"bytes"
	"fmt"
	"go/scanner"
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
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	_ "github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/mathutil"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/readable"
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

	// Indicates the directory in which the generated schema file must go.
	SchemaDirectory string

	// Indicates that we want to generate a GetAll function. Defaults to false because this can be dangerous on high cardinality stores
	GetAll bool
}

func renderFile(templateMap map[string]interface{}, temp func(s string) *template.Template, templateFileName string) error {
	buf := bytes.NewBuffer(nil)
	if err := temp(templateFileName).Execute(buf, templateMap); err != nil {
		return err
	}
	file := buf.Bytes()

	importProcessingStart := time.Now()
	formatted, err := imports.Process(templateFileName, file, nil)
	importProcessingDuration := time.Since(importProcessingStart)

	if err != nil {
		target := scanner.ErrorList{}
		if !errors.As(err, &target) {
			fmt.Println(string(file))
			return err
		}
		e := target[0]
		fileLines := strings.Split(string(file), "\n")
		fmt.Printf("There is an error in following snippet: %s\n", e.Msg)
		fmt.Println(strings.Join(fileLines[mathutil.MaxInt(0, e.Pos.Line-2):mathutil.MinInt(len(fileLines), e.Pos.Line+1)], "\n"))
		return err
	}
	if err := os.WriteFile(templateFileName, formatted, 0644); err != nil {
		return err
	}
	if importProcessingDuration > time.Second {
		absTemplatePath, err := filepath.Abs(templateFileName)
		if err != nil {
			absTemplatePath = templateFileName
		}
		log.Panicf("Import processing for file %q took more than 1 second (%s). This typically indicates that an import was "+
			"not added to the Go template, which forced import processing to search through all types and magically "+
			"add the import. Please add the import to the template; you can compare the imports in the generated file "+
			"with the ones in the template, and add the missing one(s)", absTemplatePath, importProcessingDuration)
	}
	return nil
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

	c.Flags().StringVar(&props.RegisteredType, "registered-type", "", "the type this is registered in proto as storage.X")

	c.Flags().StringVar(&props.Table, "table", "", "the logical table of the objects")
	utils.Must(c.MarkFlagRequired("table"))

	c.Flags().StringVar(&props.Singular, "singular", "", "the singular name of the object")
	c.Flags().StringVar(&props.SearchCategory, "search-category", "", "the search category to index under")
	c.Flags().StringVar(&props.PermissionChecker, "permission-checker", "", "the permission checker that should be used")
	c.Flags().StringSliceVar(&props.Refs, "references", []string{}, "additional foreign key references as <table_name:type>")
	c.Flags().BoolVar(&props.JoinTable, "join-table", false, "indicates the schema represents a join table. The generation of mutating functions is skipped")
	c.Flags().BoolVar(&props.SchemaOnly, "schema-only", false, "if true, generates only the schema and not store and index")
	c.Flags().BoolVar(&props.GetAll, "get-all-func", false, "if true, generates a GetAll function")
	c.Flags().StringVar(&props.SchemaDirectory, "schema-directory", "", "the directory in which to generate the schema")
	utils.Must(c.MarkFlagRequired("schema-directory"))

	c.RunE = func(*cobra.Command, []string) error {
		typ := stringutils.OrDefault(props.RegisteredType, props.Type)
		fmt.Println(readable.Time(time.Now()), "Generating for", typ)
		mt := proto.MessageType(typ)
		if mt == nil {
			log.Fatalf("could not find message for type: %s", typ)
		}

		schema := walker.Walk(mt, props.Table)
		if schema.NoPrimaryKey() {
			log.Fatal("No primary key defined, please check relevant proto file and ensure a primary key is specified using the \"sql:\"pk\"\" tag")
		}
		if len(schema.PrimaryKeys()) > 1 {
			for _, pk := range schema.PrimaryKeys() {
				if pk.Search.FieldName == "" {
					log.Printf("%s:%s is not searchable and is PK", props.Type, pk.Name)
				}
			}
		}

		parsedReferences := parseReferencesAndInjectPeerSchemas(schema, props.Refs)

		permissionCheckerEnabled := props.PermissionChecker != ""
		var searchCategory string
		if props.SearchCategory != "" {
			if asInt, err := strconv.Atoi(props.SearchCategory); err == nil {
				searchCategory = fmt.Sprintf("SearchCategory(%d)", asInt)
			} else {
				searchCategory = fmt.Sprintf("SearchCategory_%s", props.SearchCategory)
			}
		}
		templateMap := map[string]interface{}{
			"Type":              props.Type,
			"TrimmedType":       stringutils.GetAfter(props.Type, "."),
			"Table":             props.Table,
			"Schema":            schema,
			"SearchCategory":    searchCategory,
			"JoinTable":         props.JoinTable,
			"PermissionChecker": props.PermissionChecker,
			"GetAll":            props.GetAll,
			"Obj": object{
				storageType:              props.Type,
				permissionCheckerEnabled: permissionCheckerEnabled,
				isJoinTable:              props.JoinTable,
				schema:                   schema,
			},
		}

		if err := generateSchema(schema, searchCategory, parsedReferences, props.SchemaDirectory); err != nil {
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

func generateSchema(s *walker.Schema, searchCategory string, parsedReferences []parsedReference, dir string) error {
	templateMap := map[string]interface{}{
		"Schema":         s,
		"SearchCategory": searchCategory,
		"References":     parsedReferences,
	}

	if err := renderFile(templateMap, schemaTemplate, getSchemaFileName(dir, s.Table)); err != nil {
		return err
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
