package datastore

import (
	"context"

	imageDatastore "github.com/stackrox/rox/central/image/datastore"
	imageV2Datastore "github.com/stackrox/rox/central/imagev2/datastore"
	"github.com/stackrox/rox/central/imagev2/datastore/mapper"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/images/types"
	searchPkg "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
)

type datastoreImpl struct {
	imageDataStore   imageDatastore.DataStore
	imageV2DataStore imageV2Datastore.DataStore

	flattenImageData bool
}

func newDatastoreImpl(datastoreV1 imageDatastore.DataStore, datastoreV2 imageV2Datastore.DataStore) *datastoreImpl {
	ds := &datastoreImpl{
		imageDataStore:   datastoreV1,
		imageV2DataStore: datastoreV2,

		flattenImageData: features.FlattenImageData.Enabled(),
	}
	return ds
}

func (ds *datastoreImpl) SearchListImages(ctx context.Context, q *v1.Query) ([]*storage.ListImage, error) {
	if !ds.flattenImageData {
		return ds.imageDataStore.SearchListImages(ctx, q)
	}

	// Get v2 images
	images, err := ds.imageV2DataStore.SearchRawImages(ctx, q)
	if err != nil {
		return nil, err
	}

	// Convert to list images
	listImages := make([]*storage.ListImage, 0, len(images))
	for _, image := range images {
		// Convert v2 to v1
		v1Image := mapper.ConvertToV1(image)
		// Convert v1 to ListImage
		listImage := types.ConvertImageToListImage(v1Image)
		listImages = append(listImages, listImage)
	}
	return listImages, nil
}

func (ds *datastoreImpl) ListImage(ctx context.Context, sha string) (*storage.ListImage, bool, error) {
	if !ds.flattenImageData {
		return ds.imageDataStore.ListImage(ctx, sha)
	}

	// Get v2 image
	image, found, err := ds.GetImageMetadata(ctx, sha)
	if err != nil {
		return nil, false, err
	}
	if !found {
		return nil, false, nil
	}
	return types.ConvertImageToListImage(image), true, nil
}

func (ds *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error) {
	if !ds.flattenImageData {
		return ds.imageDataStore.Search(ctx, q)
	}
	return ds.imageV2DataStore.Search(ctx, q)
}

func (ds *datastoreImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	if !ds.flattenImageData {
		return ds.imageDataStore.Count(ctx, q)
	}
	return ds.imageV2DataStore.Count(ctx, q)
}

func (ds *datastoreImpl) SearchImages(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	if !ds.flattenImageData {
		return ds.imageDataStore.SearchImages(ctx, q)
	}
	return ds.imageV2DataStore.SearchImages(ctx, q)
}

func (ds *datastoreImpl) SearchRawImages(ctx context.Context, q *v1.Query) ([]*storage.Image, error) {
	if !ds.flattenImageData {
		return ds.imageDataStore.SearchRawImages(ctx, q)
	}
	images, err := ds.imageV2DataStore.SearchRawImages(ctx, q)
	if err != nil {
		return nil, err
	}
	results := make([]*storage.Image, 0, len(images))
	for _, image := range images {
		results = append(results, mapper.ConvertToV1(image))
	}
	return results, nil
}

func (ds *datastoreImpl) CountImages(ctx context.Context) (int, error) {
	if !ds.flattenImageData {
		return ds.imageDataStore.CountImages(ctx)
	}
	return ds.imageV2DataStore.Count(ctx, searchPkg.EmptyQuery())
}

func (ds *datastoreImpl) GetImage(ctx context.Context, sha string) (*storage.Image, bool, error) {
	if !ds.flattenImageData {
		return ds.imageDataStore.GetImage(ctx, sha)
	}
	// If the string passed in was a uuid, we can just use the V2 datastore function directly
	if _, err := uuid.FromString(sha); err == nil {
		image, found, err := ds.imageV2DataStore.GetImage(ctx, sha)
		return mapper.ConvertToV1(image), found, err
	}
	// Otherwise, we need to find the image we're looking for with a query
	images, err := ds.imageV2DataStore.SearchRawImages(ctx, searchPkg.NewQueryBuilder().AddExactMatches(searchPkg.ImageSHA, sha).ProtoQuery())
	if err != nil {
		return nil, false, err
	}
	if len(images) == 0 {
		return nil, false, nil
	}
	return mapper.ConvertToV1(images[0]), true, nil
}

func (ds *datastoreImpl) GetImageMetadata(ctx context.Context, id string) (*storage.Image, bool, error) {
	if !ds.flattenImageData {
		return ds.imageDataStore.GetImageMetadata(ctx, id)
	}
	// If the id passed in was a uuid and not a sha, we can avoid looking up the ID by the sha
	if _, err := uuid.FromString(id); err != nil {
		// Since it wasn't a uuid, we need to get the uuid from the sha by calling ds.GetImage
		foundImage, found, err := ds.GetImage(ctx, id)
		if !found || err != nil {
			return nil, false, err
		}
		id = foundImage.GetId()
	}
	// Now that we have the uuid, we can just look up the metadata from the uuid using the v2 datastore
	image, found, err := ds.imageV2DataStore.GetImageMetadata(ctx, id)
	if err != nil {
		return nil, false, err
	}
	if !found {
		return nil, false, nil
	}
	return mapper.ConvertToV1(image), true, nil
}

func (ds *datastoreImpl) GetManyImageMetadata(ctx context.Context, ids []string) ([]*storage.Image, error) {
	if !ds.flattenImageData {
		return ds.imageDataStore.GetManyImageMetadata(ctx, ids)
	}
	images, err := ds.imageV2DataStore.GetManyImageMetadata(ctx, ids)
	if err != nil {
		return nil, err
	}
	v1Images := make([]*storage.Image, 0, len(images))
	for _, image := range images {
		v1Images = append(v1Images, mapper.ConvertToV1(image))
	}
	return v1Images, nil
}

func (ds *datastoreImpl) GetImagesBatch(ctx context.Context, shas []string) ([]*storage.Image, error) {
	if !ds.flattenImageData {
		return ds.imageDataStore.GetImagesBatch(ctx, shas)
	}
	images, err := ds.imageV2DataStore.GetImagesBatch(ctx, shas)
	if err != nil {
		return nil, err
	}
	v1Images := make([]*storage.Image, 0, len(images))
	for _, image := range images {
		v1Images = append(v1Images, mapper.ConvertToV1(image))
	}
	return v1Images, nil
}

func (ds *datastoreImpl) WalkByQuery(ctx context.Context, q *v1.Query, fn func(image *storage.Image) error) error {
	if !ds.flattenImageData {
		return ds.imageDataStore.WalkByQuery(ctx, q, fn)
	}
	return ds.imageV2DataStore.WalkByQuery(ctx, q, func(image *storage.ImageV2) error {
		return fn(mapper.ConvertToV1(image))
	})
}

func (ds *datastoreImpl) UpsertImage(ctx context.Context, image *storage.Image) error {
	if !ds.flattenImageData {
		return ds.imageDataStore.UpsertImage(ctx, image)
	}
	return ds.imageV2DataStore.UpsertImage(ctx, mapper.ConvertToV2(image))
}

func (ds *datastoreImpl) UpdateVulnerabilityState(ctx context.Context, cve string, images []string, state storage.VulnerabilityState) error {
	if !ds.flattenImageData {
		return ds.imageDataStore.UpdateVulnerabilityState(ctx, cve, images, state)
	}
	return ds.imageV2DataStore.UpdateVulnerabilityState(ctx, cve, images, state)
}

func (ds *datastoreImpl) DeleteImages(ctx context.Context, ids ...string) error {
	if !ds.flattenImageData {
		return ds.imageDataStore.DeleteImages(ctx, ids...)
	}
	return ds.imageV2DataStore.DeleteImages(ctx, ids...)
}

func (ds *datastoreImpl) Exists(ctx context.Context, id string) (bool, error) {
	if !ds.flattenImageData {
		return ds.imageDataStore.Exists(ctx, id)
	}
	return ds.imageV2DataStore.Exists(ctx, id)
}
