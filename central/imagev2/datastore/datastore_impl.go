package datastore

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/imagev2/datastore/store"
	"github.com/stackrox/rox/central/imagev2/views"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/ranking"
	riskDS "github.com/stackrox/rox/central/risk/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/scancomponent"
	"github.com/stackrox/rox/pkg/search"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/signatureintegration"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	log = logging.LoggerForModule()

	imagesSAC = sac.ForResource(resources.Image)
)

type datastoreImpl struct {
	keyedMutex *concurrency.KeyedMutex

	storage store.Store

	risks riskDS.DataStore

	imageRanker                    *ranking.Ranker
	imageComponentRanker           *ranking.Ranker
	signatureIntegrationGetterFunc signatureintegration.GetterFunc
	signatureIntegrationMutex      sync.RWMutex
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
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "ImageV2", "Search")

	return ds.storage.Search(ctx, q)
}

// Count returns the number of search results from the query
func (ds *datastoreImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "ImageV2", "Count")

	return ds.storage.Count(ctx, q)
}

// TODO(ROX-29943): Eliminate unnecessary 2 pass database queries
func (ds *datastoreImpl) SearchImages(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "ImageV2", "SearchImages")

	results, err := ds.Search(ctx, q)
	if err != nil {
		return nil, err
	}
	var images []*storage.ImageV2
	var existing []search.Result
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
		existing = append(existing, result)
	}

	return convertMany(images, existing)
}

// TODO(ROX-29943): Eliminate unnecessary 2 pass database queries
// SearchRawImages delegates to the underlying searcher.
func (ds *datastoreImpl) SearchRawImages(ctx context.Context, q *v1.Query) ([]*storage.ImageV2, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "ImageV2", "SearchRawImages")

	results, err := ds.Search(ctx, q)
	if err != nil {
		return nil, err
	}

	images, err := ds.storage.GetByIDs(ctx, search.ResultsToIDs(results))
	if err != nil {
		return nil, err
	}

	ds.updateImagePriority(images...)
	ds.injectSignatureIntegrationName(ctx, images...)

	return images, nil
}

func (ds *datastoreImpl) WalkByQuery(ctx context.Context, q *v1.Query, fn func(image *storage.ImageV2) error) error {
	wrappedFn := func(image *storage.ImageV2) error {
		ds.updateImagePriority(image)
		ds.injectSignatureIntegrationName(ctx, image)
		return fn(image)
	}
	return ds.storage.WalkByQuery(ctx, q, wrappedFn)
}

func (ds *datastoreImpl) canReadImage(ctx context.Context, id string) (bool, error) {
	if ok, err := imagesSAC.ReadAllowed(ctx); err != nil {
		return false, err
	} else if ok {
		return true, nil
	}

	queryForImage := pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.ImageID, id).ProtoQuery()
	if results, err := ds.storage.Search(ctx, queryForImage); err != nil {
		return false, err
	} else if len(results) > 0 {
		return true, nil
	}

	return false, nil
}

// GetManyImageMetadata gets the image data without the scan.
func (ds *datastoreImpl) GetManyImageMetadata(ctx context.Context, ids []string) ([]*storage.ImageV2, error) {
	imgs, err := ds.storage.GetManyImageMetadata(ctx, ids)
	if err != nil {
		return nil, err
	}
	if len(imgs) != len(ids) {
		log.Errorf("Could not fetch %d/%d images", len(ids)-len(imgs), len(ids))
	}
	for _, img := range imgs {
		ds.updateImagePriority(img)
		ds.injectSignatureIntegrationName(ctx, img)
	}
	return imgs, nil
}

// GetImageMetadata gets the image data without the scan
func (ds *datastoreImpl) GetImageMetadata(ctx context.Context, id string) (*storage.ImageV2, bool, error) {
	img, found, err := ds.storage.GetImageMetadata(ctx, id)
	if err != nil || !found {
		return nil, false, err
	}
	if ok, err := ds.canReadImage(ctx, id); err != nil || !ok {
		return nil, false, err
	}
	ds.updateImagePriority(img)
	ds.injectSignatureIntegrationName(ctx, img)

	return img, true, nil
}

// GetImage delegates to the underlying store.
func (ds *datastoreImpl) GetImage(ctx context.Context, id string) (*storage.ImageV2, bool, error) {
	img, found, err := ds.storage.Get(ctx, id)
	if err != nil || !found {
		return nil, false, err
	}
	if ok, err := ds.canReadImage(ctx, id); err != nil || !ok {
		return nil, false, err
	}

	ds.updateImagePriority(img)
	ds.injectSignatureIntegrationName(ctx, img)

	return img, true, nil
}

// GetImagesBatch delegates to the underlying store.
func (ds *datastoreImpl) GetImagesBatch(ctx context.Context, ids []string) ([]*storage.ImageV2, error) {
	var imgs []*storage.ImageV2
	if ok, err := imagesSAC.ReadAllowed(ctx); err != nil {
		return nil, err
	} else if ok {
		imgs, err = ds.storage.GetByIDs(ctx, ids)
		if err != nil {
			return nil, err
		}
	} else {
		idsQuery := pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.ImageID, ids...).ProtoQuery()
		imgs, err = ds.SearchRawImages(ctx, idsQuery)
		if err != nil {
			return nil, err
		}
	}

	ds.updateImagePriority(imgs...)
	ds.injectSignatureIntegrationName(ctx, imgs...)

	return imgs, nil
}

// UpsertImage dedupes the image with the underlying storage and adds the image to the index.
func (ds *datastoreImpl) UpsertImage(ctx context.Context, image *storage.ImageV2) error {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "ImageV2", "UpsertImage")

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
	utils.FillScanStatsV2(image)

	if err := ds.storage.Upsert(ctx, image); err != nil {
		return err
	}
	// If the image in db is latest, this image object will be carrying its risk score
	ds.imageRanker.Add(image.GetId(), image.GetRiskScore())
	return nil
}

func (ds *datastoreImpl) DeleteImages(ctx context.Context, ids ...string) error {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "ImageV2", "DeleteImages")

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
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "ImageV2", "Exists")

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

	selects := []*v1.QuerySelect{
		pkgSearch.NewQuerySelect(pkgSearch.ImageID).Proto(),
		pkgSearch.NewQuerySelect(pkgSearch.ImageRiskScore).Proto(),
	}
	query := pkgSearch.EmptyQuery()
	query.Selects = selects

	// The entire image is not needed to initialize the ranker.  We only need the image id and risk score.
	var results []*views.ImageV2RiskView
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

func (ds *datastoreImpl) updateImagePriority(images ...*storage.ImageV2) {
	for _, image := range images {
		image.Priority = ds.imageRanker.GetRankForID(image.GetId())
		for _, component := range image.GetScan().GetComponents() {
			componentID, err := scancomponent.ComponentIDV2(component, image.GetId())
			if err != nil {
				log.Error(err)
				continue
			}
			component.Priority = ds.imageComponentRanker.GetRankForID(componentID)
		}
	}
}

func (ds *datastoreImpl) updateComponentRisk(image *storage.ImageV2) {
	for _, component := range image.GetScan().GetComponents() {
		componentID, err := scancomponent.ComponentIDV2(component, image.GetId())
		if err != nil {
			log.Error(err)
			continue
		}
		component.RiskScore = ds.imageComponentRanker.GetScoreForID(componentID)
	}
}

func convertMany(images []*storage.ImageV2, results []search.Result) ([]*v1.SearchResult, error) {
	if len(images) != len(results) {
		return nil, errors.New("mismatch between search results and retrieved images")
	}

	searchResults := make([]*v1.SearchResult, 0, len(images))
	for i, image := range images {
		searchResults = append(searchResults, convertOne(image, &results[i]))
	}
	return searchResults, nil
}

func convertOne(image *storage.ImageV2, result *search.Result) *v1.SearchResult {
	return &v1.SearchResult{
		Category:       v1.SearchCategory_IMAGES,
		Id:             image.GetId(),
		Name:           image.GetName().GetFullName(),
		FieldToMatches: search.GetProtoMatchesMap(result.Matches),
		Score:          result.Score,
	}
}

func (ds *datastoreImpl) SetSignatureIntegrationGetterFunc(fn signatureintegration.GetterFunc) {
	ds.signatureIntegrationMutex.Lock()
	defer ds.signatureIntegrationMutex.Unlock()
	ds.signatureIntegrationGetterFunc = fn
}

func (ds *datastoreImpl) injectSignatureIntegrationName(ctx context.Context, images ...*storage.ImageV2) {
	// Early exit if the signature integration getter has not been set up yet.
	ds.signatureIntegrationMutex.RLock()
	defer ds.signatureIntegrationMutex.RUnlock()

	if ds.signatureIntegrationGetterFunc == nil {
		log.Debug("Signature integration getter has not been set.")
		return
	}

	signatureIntegrationCtx := sac.WithGlobalAccessScopeChecker(ctx,
		sac.AllowFixedScopes(
			sac.ResourceScopeKeys(resources.Integration),
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
		),
	)
	for _, image := range images {
		for _, result := range image.GetSignatureVerificationData().GetResults() {
			verifierName, err := signatureintegration.GetVerifierName(signatureIntegrationCtx,
				ds.signatureIntegrationGetterFunc(), result)
			if err != nil {
				log.Warnf("Failed to get signature integration name: %v", err)
				continue
			}
			result.VerifierName = verifierName
		}
	}
}
