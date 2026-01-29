package datastore

import (
	"context"
	"strings"
	"time"

	"github.com/pkg/errors"
	imageCVEInfoDS "github.com/stackrox/rox/central/cve/image/info/datastore"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/image/datastore/store"
	"github.com/stackrox/rox/central/image/views"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/ranking"
	riskDS "github.com/stackrox/rox/central/risk/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/cve"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/images/enricher"
	imageTypes "github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protocompat"
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

	imageCVEInfoDS imageCVEInfoDS.DataStore
}

func newDatastoreImpl(storage store.Store, risks riskDS.DataStore,
	imageRanker *ranking.Ranker, imageComponentRanker *ranking.Ranker,
	imageCVEInfo imageCVEInfoDS.DataStore) *datastoreImpl {
	ds := &datastoreImpl{
		storage: storage,

		risks: risks,

		imageRanker:          imageRanker,
		imageComponentRanker: imageComponentRanker,

		imageCVEInfoDS: imageCVEInfo,

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

	if q == nil {
		q = pkgSearch.EmptyQuery()
	}

	// Clone the query and add select fields for SearchResult construction
	clonedQuery := q.CloneVT()
	clonedQuery.Selects = append(q.GetSelects(), pkgSearch.NewQuerySelect(pkgSearch.ImageName).Proto())

	results, err := ds.Search(ctx, q)
	if err != nil {
		return nil, err
	}
	for i := range results {
		if results[i].FieldValues != nil {
			if nameVal, ok := results[i].FieldValues[strings.ToLower(pkgSearch.ImageName.String())]; ok {
				results[i].Name = nameVal
			}
		}
		results[i].ID = imageTypes.NewDigest(results[i].ID).Digest()
	}

	return pkgSearch.ResultsToSearchResultProtos(results, &ImageSearchResultConverter{}), nil
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
	err := ds.storage.WalkMetadataByQuery(ctx, q, func(img *storage.Image) error {
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

	if features.CVEFixTimestampCriteria.Enabled() {
		// Populate the ImageCVEInfo table with CVE timing metadata
		if err := ds.upsertImageCVEInfos(ctx, image); err != nil {
			log.Warnf("Failed to upsert ImageCVEInfo: %v", err)
			// Non-fatal, continue with image upsert
		}

		// Enrich the CVEs with accurate timestamps from lookup table
		if err := ds.enrichCVEsFromImageCVEInfo(ctx, image); err != nil {
			log.Warnf("Failed to enrich CVEs from ImageCVEInfo: %v", err)
			// Non-fatal, continue with image upsert
		}
	}

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
		for index, component := range image.GetScan().GetComponents() {
			componentID := scancomponent.ComponentIDV2(component, image.GetId(), index)
			component.Priority = ds.imageComponentRanker.GetRankForID(componentID)
		}
	}
}

func (ds *datastoreImpl) updateComponentRisk(image *storage.Image) {
	for index, component := range image.GetScan().GetComponents() {
		componentID := scancomponent.ComponentIDV2(component, image.GetId(), index)
		component.RiskScore = ds.imageComponentRanker.GetScoreForID(componentID)
	}
}

// upsertImageCVEInfos populates the ImageCVEInfo lookup table with CVE timing metadata.
func (ds *datastoreImpl) upsertImageCVEInfos(ctx context.Context, image *storage.Image) error {
	if !features.CVEFixTimestampCriteria.Enabled() {
		return nil
	}

	infos := make([]*storage.ImageCVEInfo, 0)
	now := protocompat.TimestampNow()

	for _, component := range image.GetScan().GetComponents() {
		for _, vuln := range component.GetVulns() {
			// Determine fix available timestamp: use scanner-provided value if available,
			// otherwise fabricate from scan time if the CVE is fixable (has a fix version).
			// This handles non-Red Hat data sources that don't provide fix timestamps.
			fixAvailableTimestamp := vuln.GetFixAvailableTimestamp()
			if fixAvailableTimestamp == nil && vuln.GetFixedBy() != "" {
				fixAvailableTimestamp = now
			}

			info := &storage.ImageCVEInfo{
				Id:                    cve.ImageCVEInfoID(vuln.GetCve(), component.GetName(), vuln.GetDatasource()),
				FixAvailableTimestamp: fixAvailableTimestamp,
				FirstSystemOccurrence: now, // Smart upsert in ImageCVEInfo datastore preserves existing
			}
			infos = append(infos, info)
		}
	}

	return ds.imageCVEInfoDS.UpsertMany(ctx, infos)
}

// enrichCVEsFromImageCVEInfo enriches the image's CVEs with accurate timestamps from the lookup table.
func (ds *datastoreImpl) enrichCVEsFromImageCVEInfo(ctx context.Context, image *storage.Image) error {
	if !features.CVEFixTimestampCriteria.Enabled() {
		return nil
	}

	// Collect all IDs
	ids := make([]string, 0)
	for _, component := range image.GetScan().GetComponents() {
		for _, vuln := range component.GetVulns() {
			ids = append(ids, cve.ImageCVEInfoID(vuln.GetCve(), component.GetName(), vuln.GetDatasource()))
		}
	}

	if len(ids) == 0 {
		return nil
	}

	// Batch fetch
	infos, err := ds.imageCVEInfoDS.GetBatch(ctx, ids)
	if err != nil {
		return err
	}

	// Build lookup map
	infoMap := make(map[string]*storage.ImageCVEInfo)
	for _, info := range infos {
		infoMap[info.GetId()] = info
	}

	// Enrich CVEs and blank out datasource after using it
	for _, component := range image.GetScan().GetComponents() {
		for _, vuln := range component.GetVulns() {
			id := cve.ImageCVEInfoID(vuln.GetCve(), component.GetName(), vuln.GetDatasource())
			if info, ok := infoMap[id]; ok {
				vuln.FixAvailableTimestamp = info.GetFixAvailableTimestamp()
				vuln.FirstSystemOccurrence = info.GetFirstSystemOccurrence()
			}
			// Blank out datasource after using it - this is internal scanner data not meant for end users
			vuln.Datasource = ""
		}
	}

	return nil
}

// ImageSearchResultConverter converts image search results to proto search results
type ImageSearchResultConverter struct{}

func (c *ImageSearchResultConverter) BuildName(result *pkgSearch.Result) string {

	return result.Name
}

func (c *ImageSearchResultConverter) BuildLocation(result *pkgSearch.Result) string {
	// Images do not have a location
	return ""
}

func (c *ImageSearchResultConverter) GetCategory() v1.SearchCategory {
	return v1.SearchCategory_IMAGES
}

func (c *ImageSearchResultConverter) GetScore(result *pkgSearch.Result) float64 {
	return result.Score
}
