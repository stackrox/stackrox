package datastore

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/image/datastore/store"
	"github.com/stackrox/rox/central/image/views"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/ranking"
	riskDS "github.com/stackrox/rox/central/risk/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/images/enricher"
	imageTypes "github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/scancomponent"
	pkgSearch "github.com/stackrox/rox/pkg/search"
)

var (
	log = logging.LoggerForModule()

	imagesSAC = sac.ForResource(resources.Image)
)

type datastoreImpl struct {
	keyedMutex *concurrency.KeyedMutex

	storage store.Store

	risks riskDS.DataStore

	imageRanker          *ranking.Ranker
	imageComponentRanker *ranking.Ranker
}

func newDatastoreImpl(storage store.Store, risks riskDS.DataStore,
	imageRanker *ranking.Ranker, imageComponentRanker *ranking.Ranker) *datastoreImpl {
	ds := &datastoreImpl{
		storage: storage,

		risks: risks,

		imageRanker:          imageRanker,
		imageComponentRanker: imageComponentRanker,

		keyedMutex: concurrency.NewKeyedMutex(globaldb.DefaultDataStorePoolSize),
	}
	return ds
}

func (ds *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]pkgSearch.Result, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "Image", "Search")

	return ds.storage.Search(ctx, q)
}

// Count returns the number of search results from the query
func (ds *datastoreImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "Image", "Count")

	return ds.storage.Count(ctx, q)
}

func (ds *datastoreImpl) SearchImages(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "Image", "SearchImages")

	// TODO(ROX-29943): remove unnecessary calls to database
	results, err := ds.Search(ctx, q)
	if err != nil {
		return nil, err
	}
	var images []*storage.Image
	var newResults []pkgSearch.Result
	for _, result := range results {
		image, exists, err := ds.storage.GetImageMetadata(ctx, result.ID)
		if err != nil {
			return nil, err
		}
		// The result may not exist if the object was deleted after the search
		if !exists {
			continue
		}
		images = append(images, image)
		newResults = append(newResults, result)
	}

	if len(newResults) != len(images) {
		return nil, errors.Errorf("expected %d results, got %d", len(images), len(newResults))
	}

	protoResults := make([]*v1.SearchResult, 0, len(images))
	for i, image := range images {
		protoResults = append(protoResults, convertImage(image, newResults[i]))
	}
	return protoResults, nil
}

// SearchRawImages delegates to the underlying searcher.
func (ds *datastoreImpl) SearchRawImages(ctx context.Context, q *v1.Query) ([]*storage.Image, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "Image", "SearchRawImages")

	var imgs []*storage.Image
	err := ds.storage.WalkByQuery(ctx, q, func(img *storage.Image) error {
		imgs = append(imgs, img)
		return nil
	})
	if err != nil {
		return nil, err
	}

	ds.updateImagePriority(imgs...)

	return imgs, nil
}

func (ds *datastoreImpl) SearchListImages(ctx context.Context, q *v1.Query) ([]*storage.ListImage, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "Image", "SearchListImages")

	var imgs []*storage.ListImage
	err := ds.storage.WalkByQuery(ctx, q, func(img *storage.Image) error {
		imgs = append(imgs, imageTypes.ConvertImageToListImage(img))
		return nil
	})
	if err != nil {
		return nil, err
	}

	ds.updateListImagePriority(imgs...)

	return imgs, nil
}

func (ds *datastoreImpl) ListImage(ctx context.Context, sha string) (*storage.ListImage, bool, error) {
	img, found, err := ds.storage.GetImageMetadata(ctx, sha)
	if err != nil || !found {
		return nil, false, err
	}

	if ok, err := ds.canReadImage(ctx, sha); err != nil || !ok {
		return nil, false, err
	}

	listImage := imageTypes.ConvertImageToListImage(img)
	ds.updateListImagePriority(listImage)
	return listImage, true, nil
}

func (ds *datastoreImpl) WalkByQuery(ctx context.Context, q *v1.Query, fn func(image *storage.Image) error) error {
	wrappedFn := func(image *storage.Image) error {
		ds.updateImagePriority(image)
		return fn(image)
	}
	return ds.storage.WalkByQuery(ctx, q, wrappedFn)
}

// CountImages delegates to the underlying store.
func (ds *datastoreImpl) CountImages(ctx context.Context) (int, error) {
	if _, err := imagesSAC.ReadAllowed(ctx); err != nil {
		return 0, err
	}

	return ds.Count(ctx, pkgSearch.EmptyQuery())
}

func (ds *datastoreImpl) canReadImage(ctx context.Context, sha string) (bool, error) {
	if ok, err := imagesSAC.ReadAllowed(ctx); err != nil {
		return false, err
	} else if ok {
		return true, nil
	}

	queryForImage := pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.ImageSHA, sha).ProtoQuery()
	if results, err := ds.Search(ctx, queryForImage); err != nil {
		return false, err
	} else if len(results) > 0 {
		return true, nil
	}

	return false, nil
}

// GetManyImageMetadata gets the image data without the scan.
func (ds *datastoreImpl) GetManyImageMetadata(ctx context.Context, ids []string) ([]*storage.Image, error) {
	imgs, err := ds.storage.GetManyImageMetadata(ctx, ids)
	if err != nil {
		return nil, err
	}
	if len(imgs) != len(ids) {
		log.Errorf("Could not fetch %d/%d some images", len(ids)-len(imgs), len(ids))
	}
	for _, img := range imgs {
		ds.updateImagePriority(img)
	}
	return imgs, nil
}

// GetImageMetadata gets the image data without the scan
func (ds *datastoreImpl) GetImageMetadata(ctx context.Context, id string) (*storage.Image, bool, error) {
	img, found, err := ds.storage.GetImageMetadata(ctx, id)
	if err != nil || !found {
		return nil, false, err
	}
	if ok, err := ds.canReadImage(ctx, id); err != nil || !ok {
		return nil, false, err
	}
	ds.updateImagePriority(img)

	return img, true, nil
}

// GetImage delegates to the underlying store.
func (ds *datastoreImpl) GetImage(ctx context.Context, sha string) (*storage.Image, bool, error) {
	img, found, err := ds.storage.Get(ctx, sha)
	if err != nil || !found {
		return nil, false, err
	}
	if ok, err := ds.canReadImage(ctx, sha); err != nil || !ok {
		return nil, false, err
	}

	ds.updateImagePriority(img)

	return img, true, nil
}

// GetImagesBatch delegates to the underlying store.
func (ds *datastoreImpl) GetImagesBatch(ctx context.Context, shas []string) ([]*storage.Image, error) {
	var imgs []*storage.Image
	if ok, err := imagesSAC.ReadAllowed(ctx); err != nil {
		return nil, err
	} else if ok {
		imgs, err = ds.storage.GetByIDs(ctx, shas)
		if err != nil {
			return nil, err
		}
	} else {
		shasQuery := pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.ImageSHA, shas...).ProtoQuery()
		imgs, err = ds.SearchRawImages(ctx, shasQuery)
		if err != nil {
			return nil, err
		}
	}

	ds.updateImagePriority(imgs...)

	return imgs, nil
}

// UpsertImage dedupes the image with the underlying storage and adds the image to the index.
func (ds *datastoreImpl) UpsertImage(ctx context.Context, image *storage.Image) error {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "Image", "UpsertImage")

	if image.GetId() == "" {
		return errors.New("cannot upsert an image without an id")
	}

	if ok, err := imagesSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	ds.keyedMutex.Lock(image.GetId())
	defer ds.keyedMutex.Unlock(image.GetId())

	ds.updateComponentRisk(image)
	enricher.FillScanStats(image)

	if err := ds.storage.Upsert(ctx, image); err != nil {
		return err
	}
	// If the image in db is latest, this image object will be carrying its risk score
	ds.imageRanker.Add(image.GetId(), image.GetRiskScore())
	return nil
}

func (ds *datastoreImpl) DeleteImages(ctx context.Context, ids ...string) error {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "Image", "DeleteImages")

	if ok, err := imagesSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	errorList := errorhelpers.NewErrorList("deleting images")
	deleteRiskCtx := sac.WithGlobalAccessScopeChecker(ctx,
		sac.AllowFixedScopes(sac.AccessModeScopeKeys(storage.Access_READ_WRITE_ACCESS), sac.ResourceScopeKeys(resources.DeploymentExtension)))

	for _, id := range ids {
		if err := ds.storage.Delete(ctx, id); err != nil {
			errorList.AddError(err)
			continue
		}
		if err := ds.risks.RemoveRisk(deleteRiskCtx, id, storage.RiskSubjectType_IMAGE); err != nil {
			return err
		}
	}
	// removing component risk handled by pruning
	return errorList.ToError()
}

func (ds *datastoreImpl) Exists(ctx context.Context, id string) (bool, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "Image", "Exists")

	if ok, err := ds.canReadImage(ctx, id); err != nil || !ok {
		return false, err
	}
	return ds.storage.Exists(ctx, id)
}

func (ds *datastoreImpl) UpdateVulnerabilityState(ctx context.Context, cve string, images []string, state storage.VulnerabilityState) error {
	if ok, err := imagesSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	if err := ds.storage.UpdateVulnState(ctx, cve, images, state); err != nil {
		return err
	}
	return nil
}

func (ds *datastoreImpl) initializeRankers() {
	readCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS), sac.ResourceScopeKeys(resources.Image)))

	query := pkgSearch.NewQueryBuilder().AddSelectFields(pkgSearch.NewQuerySelect(pkgSearch.ImageSHA),
		pkgSearch.NewQuerySelect(pkgSearch.ImageRiskScore)).ProtoQuery()

	// The entire image is not needed to initialize the ranker.  We only need the image id and risk score.
	var results []*views.ImageRiskView
	results, err := ds.storage.GetImagesRiskView(readCtx, query)
	if err != nil {
		log.Errorf("unable to initialize image ranking: %v", err)
		return
	}

	for _, result := range results {
		ds.imageRanker.Add(result.ImageID, result.ImageRiskScore)
	}

	log.Infof("Initialized image ranking with %d images", len(results))
}

func (ds *datastoreImpl) updateListImagePriority(images ...*storage.ListImage) {
	for _, image := range images {
		image.Priority = ds.imageRanker.GetRankForID(image.GetId())
	}
}

func (ds *datastoreImpl) updateImagePriority(images ...*storage.Image) {
	for _, image := range images {
		image.Priority = ds.imageRanker.GetRankForID(image.GetId())
		for _, component := range image.GetScan().GetComponents() {
			if features.FlattenCVEData.Enabled() {
				componentID, err := scancomponent.ComponentIDV2(component, image.GetId())
				if err != nil {
					log.Error(err)
					continue
				}
				component.Priority = ds.imageComponentRanker.GetRankForID(componentID)
			} else {
				component.Priority = ds.imageComponentRanker.GetRankForID(scancomponent.ComponentID(component.GetName(), component.GetVersion(), image.GetScan().GetOperatingSystem()))
			}
		}
	}
}

func (ds *datastoreImpl) updateComponentRisk(image *storage.Image) {
	for _, component := range image.GetScan().GetComponents() {
		if features.FlattenCVEData.Enabled() {
			componentID, err := scancomponent.ComponentIDV2(component, image.GetId())
			if err != nil {
				log.Error(err)
				continue
			}
			component.RiskScore = ds.imageComponentRanker.GetScoreForID(componentID)
		} else {
			component.RiskScore = ds.imageComponentRanker.GetScoreForID(scancomponent.ComponentID(component.GetName(), component.GetVersion(), image.GetScan().GetOperatingSystem()))
		}
	}
}

// convertImage returns proto search result from an image object and the internal search result
func convertImage(image *storage.Image, result pkgSearch.Result) *v1.SearchResult {
	return &v1.SearchResult{
		Category:       v1.SearchCategory_IMAGES,
		Id:             imageTypes.NewDigest(image.GetId()).Digest(),
		Name:           image.GetName().GetFullName(),
		FieldToMatches: pkgSearch.GetProtoMatchesMap(result.Matches),
		Score:          result.Score,
	}
}
