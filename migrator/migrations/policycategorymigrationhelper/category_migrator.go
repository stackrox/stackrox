package policycategorymigrationhelper

import (
	"context"
	"embed"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/jsonutil"
	"github.com/stackrox/rox/pkg/sac"
)

const (
	policyCategoryParentDirName = "categories_add_and_remove"
	addDirName                  = policyCategoryParentDirName + "/add"
	removeDirName               = policyCategoryParentDirName + "/remove"
)

// ReadPolicyCategoryFromFile reads categories from file given the path and the collection of files.
func ReadPolicyCategoryFromFile(fs embed.FS, filePath string) (*storage.PolicyCategory, error) {
	contents, err := fs.ReadFile(filePath)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to read file %s", filePath)
	}
	var category storage.PolicyCategory
	err = jsonutil.JSONBytesToProto(contents, &category)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to unmarshal policy json at path %s", filePath)
	}
	return &category, nil
}

// AddNewCategoriesToDB adds new default categories to the store
func AddNewCategoriesToDB(fs embed.FS, upsertCategory func(ctx context.Context, category *storage.PolicyCategory) error) error {
	ctx := sac.WithAllAccess(context.Background())
	entries, err := fs.ReadDir(filepath.Join(policyCategoryParentDirName, "add"))
	if err != nil {
		return err
	}
	for _, entry := range entries {
		filePath := filepath.Join(policyCategoryParentDirName, "add", entry.Name())
		category, err := ReadPolicyCategoryFromFile(fs, filePath)
		if err != nil {
			return errors.Wrapf(err, "unable to read file %s", filePath)
		}
		err = upsertCategory(ctx, category)
		if err != nil {
			return err
		}
	}
	return nil
}
