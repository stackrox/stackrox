package image

import (
	"embed"
	"path/filepath"
	"testing"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	imageUtils "github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/jsonutil"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	dir = "files"
)

var (
	//go:embed files/*.json
	fs embed.FS
)

// GetTestImages returns a slice of images for testing purposes. These images contain a snapshot of image scan.
func GetTestImages(_ *testing.T) ([]*storage.Image, error) {
	files, err := fs.ReadDir(dir)
	// Sanity check embedded directory.
	utils.CrashOnError(err)

	var images []*storage.Image
	for _, f := range files {
		image, err := readContents(filepath.Join(dir, f.Name()))
		if err != nil {
			return nil, err
		}
		images = append(images, image)
	}
	return images, nil
}

// GetTestImagesV2 returns a slice of ImageV2 for testing purposes, converted from the embedded test image JSON files.
func GetTestImagesV2(t *testing.T) ([]*storage.ImageV2, error) {
	images, err := GetTestImages(t)
	if err != nil {
		return nil, err
	}
	imagesV2 := make([]*storage.ImageV2, 0, len(images))
	for _, image := range images {
		imagesV2 = append(imagesV2, imageUtils.ConvertToV2(image))
	}
	return imagesV2, nil
}

func readContents(path string) (*storage.Image, error) {
	contents, err := fs.ReadFile(path)
	// We must be able to read the embedded files.
	utils.CrashOnError(err)

	var image storage.Image
	err = jsonutil.JSONBytesToProto(contents, &image)
	if err != nil {
		return nil, errors.Errorf("unable to unmarshal image (%s) json: %s", path, err)
	}
	return &image, nil
}
