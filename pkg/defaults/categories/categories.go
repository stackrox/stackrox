package categories

import (
	"embed"
	"path/filepath"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/jsonutil"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	categoriesDir = "files"
)

var (
	log = logging.LoggerForModule()

	//go:embed files/*.json
	categoriesFS embed.FS
)

// DefaultPolicyCategories returns a slice of the default policy categories.
func DefaultPolicyCategories() ([]*storage.PolicyCategory, error) {
	files, err := categoriesFS.ReadDir(categoriesDir)
	// Sanity check embedded directory.
	utils.CrashOnError(err)

	var categories []*storage.PolicyCategory

	errList := errorhelpers.NewErrorList("Default policy category validation")
	for _, f := range files {
		c, err := readCategoryFile(filepath.Join(categoriesDir, f.Name()))
		if err != nil {
			errList.AddError(err)
			continue
		}
		if c.GetId() == "" {
			errList.AddStringf("category %s does not have an ID defined", c.GetName())
			continue
		}

		categories = append(categories, c)
	}

	return categories, errList.ToError()
}

func readCategoryFile(path string) (*storage.PolicyCategory, error) {
	contents, err := categoriesFS.ReadFile(path)
	// We must be able to read the embedded files.
	utils.CrashOnError(err)

	var category storage.PolicyCategory
	err = jsonutil.JSONBytesToProto(contents, &category)
	if err != nil {
		log.Errorf("Unable to unmarshal category (%s) json: %s", path, err)
		return nil, err
	}
	return &category, nil
}
