package service

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/grpc/testutils"
	"github.com/stretchr/testify/assert"
)

func TestAuthz(t *testing.T) {
	testutils.AssertAuthzWorks(t, &serviceImpl{})
}

func TestBuildNames(t *testing.T) {
	srcImage := &storage.ImageName{FullName: "si"}

	t.Run("nil metadata", func(t *testing.T) {
		names := buildNames(srcImage, nil)
		assert.Len(t, names, 1)
		assert.Equal(t, srcImage.GetFullName(), names[0].GetFullName())
	})

	t.Run("empty metadata", func(t *testing.T) {
		names := buildNames(srcImage, &storage.ImageMetadata{})
		assert.Len(t, names, 1)
		assert.Equal(t, srcImage.GetFullName(), names[0].GetFullName())
	})

	t.Run("metadata with data source", func(t *testing.T) {
		metadata := &storage.ImageMetadata{DataSource: &storage.DataSource{}}
		names := buildNames(srcImage, metadata)
		assert.Len(t, names, 1)
		assert.Equal(t, srcImage.GetFullName(), names[0].GetFullName())
	})

	t.Run("metadata with mirror", func(t *testing.T) {
		mirror := "example.com/mirror/image:latest"
		metadata := &storage.ImageMetadata{DataSource: &storage.DataSource{Mirror: mirror}}
		names := buildNames(srcImage, metadata)
		assert.Len(t, names, 2)
		assert.Equal(t, srcImage.GetFullName(), names[0].GetFullName())
		assert.Equal(t, mirror, names[1].GetFullName())
	})

	t.Run("metadata with invalid mirror", func(t *testing.T) {
		mirror := "example.com/mirror/image@sha256:bad"
		metadata := &storage.ImageMetadata{DataSource: &storage.DataSource{Mirror: mirror}}
		names := buildNames(srcImage, metadata)
		assert.Len(t, names, 1)
		assert.Equal(t, srcImage.GetFullName(), names[0].GetFullName())
	})
}
