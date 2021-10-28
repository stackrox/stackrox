package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/tools/generate-helpers/common/packagenames"
)

const indexFile = `
package postgres

import (
	mappings "{{.OptionsPath}}"
	metrics "github.com/stackrox/rox/central/metrics"
	v1 "github.com/stackrox/rox/generated/api/v1"
	storage "github.com/stackrox/rox/generated/storage"
	ops "github.com/stackrox/rox/pkg/metrics"
	search "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/postgres/mapping"
	"github.com/stackrox/rox/pkg/search/postgres"
	"time"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/pkg/search/blevesearch"
	"github.com/stackrox/rox/pkg/logging"
)

var log = logging.LoggerForModule()

const table = "{{.Table}}"

func init() {
	mapping.RegisterCategoryToTable(v1.{{.SearchCategory}}, table)
}

func NewIndexer(db *pgxpool.Pool) *indexerImpl {
	return &indexerImpl {
		db: db,
	}
}

type indexerImpl struct {
	db *pgxpool.Pool
}

func (b *indexerImpl) Add{{.Type}}(deployment *storage.{{.Type}}) error {
	// Added as a part of normal DB op
	return nil
}

func (b *indexerImpl) Add{{.Type}}s(_ []*storage.{{.Type}}) error {
	// Added as a part of normal DB op
	return nil
}

func (b *indexerImpl) Delete{{.Type}}(id string) error {
	// Removed as a part of normal DB op
	return nil
}

func (b *indexerImpl) Delete{{.Type}}s(_ []string) error {
	// Added as a part of normal DB op
	return nil
}

func (b *indexerImpl) MarkInitialIndexingComplete() error {
	return nil
}

func (b *indexerImpl) NeedsInitialIndexing() (bool, error) {
	return false, nil
}

func (b *indexerImpl) Count(q *v1.Query, opts ...blevesearch.SearchOption) (int, error) {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.Count, "{{.Type}}")
	return postgres.RunCountRequest(v1.{{.SearchCategory}}, q, b.db, mappings.OptionsMap)
}

func (b *indexerImpl) Search(q *v1.Query, opts ...blevesearch.SearchOption) ([]search.Result, error) {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.Search, "{{.Type}}")
	return postgres.RunSearchRequest(v1.{{.SearchCategory}}, q, b.db, mappings.OptionsMap)
}
`

type properties struct {
	Type           string
	Table          string
	NoKeyField     bool
	KeyFunc        string
	UniqKeyFunc    string
	SearchCategory string
	ObjectPathName string
	Singular       string
	WriteOptions   bool
	OptionsPath    string
	Object         string
}

func main() {
	c := &cobra.Command{
		Use: "generate store implementations",
	}

	/*
		//go:generate pgsearchbindings-wrapper --object-path-name alert --write-options=false --options-path mappings --object ListAlert --singular ListAlert --search-category ALERTS

	*/

	var props properties
	c.Flags().StringVar(&props.Type, "type", "", "the (Go) name of the object")
	utils.Must(c.MarkFlagRequired("type"))

	c.Flags().StringVar(&props.Table, "table", "", "the logical table of the objects")
	utils.Must(c.MarkFlagRequired("table"))

	c.Flags().StringVar(&props.SearchCategory, "search-category", "", fmt.Sprintf("the search category to index under (supported - %s)", renderSearchCategories()))
	utils.Must(c.MarkFlagRequired("search-category"))

	//c.Flags().StringVar(&props.Pkg, "package", "github.com/stackrox/rox/generated/storage", "the package of the object being indexed")
	//
	//c.Flags().StringVar(&props.Object, "object", "", "the (Go) name of the object being indexed")
	//utils.Must(c.MarkFlagRequired("object"))
	//
	c.Flags().StringVar(&props.Singular, "singular", "", "the singular name of the object")
	//
	//c.Flags().StringVar(&props.Plural, "plural", "", "the plural name of the object (optional; appends 's' to singular by default")
	//
	//c.Flags().StringVar(&props.IDFunc, "id-func", "GetId", "the method to invoke on the proto object to get an id out")
	//
	//c.Flags().StringVar(&props.SearchCategory, "search-category", "", fmt.Sprintf("the search category to index under (supported - %s)", renderSearchCategories()))
	//utils.Must(c.MarkFlagRequired("search-category"))
	//
	c.Flags().BoolVar(&props.WriteOptions, "write-options", true, "enable writing out the options map")
	c.Flags().StringVar(&props.OptionsPath, "options-path", "/index/mappings", "path to write out the options to")
	//c.Flags().StringVar(&props.Tag, "tag", "", "use the specified json tag")

	/*
		props := operations.GeneratorProperties{}
		c.Flags().StringVar(&props.Pkg, "package", "github.com/stackrox/rox/generated/storage", "the package of the object being indexed")

		c.Flags().StringVar(&props.Object, "object", "", "the (Go) name of the object being indexed")
		utils.Must(c.MarkFlagRequired("object"))

		c.Flags().StringVar(&props.SearchCategory, "search-category", "", fmt.Sprintf("the search category to index under (supported - %s)", renderSearchCategories()))
		utils.Must(c.MarkFlagRequired("search-category"))

		c.Flags().StringVar(&props.ObjectPathName, "object-path-name", "", "overwrite the object path underneath Central")
		c.Flags().StringVar(&props.Tag, "tag", "", "use the specified json tag")

		c.RunE = func(*cobra.Command, []string) error {
			if props.Plural == "" {
				props.Plural = fmt.Sprintf("%ss", props.Singular)
			}
			if err := checkSupported(props.SearchCategory); err != nil {
				return err
			}
			props.SearchCategory = fmt.Sprintf("SearchCategory_%s", props.SearchCategory)
			return generate(props)
		}

		if err := c.Execute(); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	*/

	c.RunE = func(*cobra.Command, []string) error {
		props.SearchCategory = fmt.Sprintf("SearchCategory_%s", props.SearchCategory)
		templateMap := map[string]interface{}{
			"Type":           props.Type,
			"Table":          props.Table,
			"SearchCategory": props.SearchCategory,
			"OptionsPath":    path.Join(packagenames.Rox, props.OptionsPath),
		}

		t := template.Must(template.New("gen").Parse(autogenerated + indexFile))
		buf := bytes.NewBuffer(nil)
		if err := t.Execute(buf, templateMap); err != nil {
			return err
		}
		if err := ioutil.WriteFile("index.go", buf.Bytes(), 0644); err != nil {
			return err
		}
		return nil
	}
	if err := c.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func renderSearchCategories() string {
	allCategories := make([]string, 0, len(v1.SearchCategory_value))

	for category := range v1.SearchCategory_value {
		allCategories = append(allCategories, category)
	}
	return strings.Join(allCategories, ",")
}
