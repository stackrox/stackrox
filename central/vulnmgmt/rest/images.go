package rest

import (
	"context"
	"net/http"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/search"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
	"github.com/stackrox/rox/pkg/search/postgres/aggregatefunc"
	"github.com/stackrox/rox/pkg/set"
)

const (
	watchStatusUnknown    = "UNKNOWN"
	watchStatusWatched    = "WATCHED"
	watchStatusNotWatched = "NOT_WATCHED"
)

type imageIDRow struct {
	ImageID string `db:"image_id"`
}

type imageCVECountRow struct {
	ImageID  string `db:"image_id"`
	CVECount int    `db:"cve_count"`
}

type imageDeploymentCountRow struct {
	ImageID         string `db:"image_id"`
	DeploymentCount int    `db:"deployment_id_count"`
}

func (h *handler) listImages(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	query, err := parseSearchQuery(r)
	if err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	query = withPaginationLimits(query)

	ids, err := h.pageImageIDs(ctx, query)
	if err != nil {
		writeError(w, err, http.StatusInternalServerError)
		return
	}
	if len(ids) == 0 {
		writeJSON(w, ImageListResponse{Items: []ImageListItem{}})
		return
	}

	matchedCVECounts, err := h.imageMatchedCVECounts(ctx, query, ids)
	if err != nil {
		writeError(w, err, http.StatusInternalServerError)
		return
	}
	deploymentCounts, err := h.imageDeploymentCounts(ctx, query, ids)
	if err != nil {
		writeError(w, err, http.StatusInternalServerError)
		return
	}
	images, err := h.deps.ImageStore.GetManyImageMetadata(ctx, ids)
	if err != nil {
		writeError(w, err, http.StatusInternalServerError)
		return
	}
	watchedNames, err := h.watchedImageNames(ctx)
	if err != nil {
		// Watch status is supplementary; default to UNKNOWN if it cannot be loaded.
		log.Errorf("failed to load watched images: %v", err)
		watchedNames = nil
	}

	imageByID := make(map[string]*storage.ImageV2, len(images))
	for _, image := range images {
		imageByID[image.GetId()] = image
	}

	items := make([]ImageListItem, 0, len(ids))
	for _, id := range ids {
		image := imageByID[id]
		if image == nil {
			continue
		}
		items = append(items, ImageListItem{
			ID:     image.GetId(),
			Digest: image.GetDigest(),
			Name: ImageName{
				Registry: image.GetName().GetRegistry(),
				Remote:   image.GetName().GetRemote(),
				Tag:      image.GetName().GetTag(),
				FullName: image.GetName().GetFullName(),
			},
			OperatingSystem:           image.GetScan().GetOperatingSystem(),
			ScanTime:                  formatTime(timePtr(image.GetScan().GetScanTime())),
			CreatedAt:                 formatTime(timePtr(image.GetMetadata().GetV1().GetCreated())),
			ScanNotes:                 scanNotesToStrings(image.GetScan().GetNotes()),
			Notes:                     imageNotesToStrings(image.GetNotes()),
			WatchStatus:               watchStatus(watchedNames, image.GetName().GetFullName()),
			DeploymentCount:           deploymentCounts[id],
			MatchedCVECount:           matchedCVECounts[id],
			SignatureVerificationData: signatureVerificationData(image),
		})
	}

	writeJSON(w, ImageListResponse{Items: items})
}

func (h *handler) pageImageIDs(ctx context.Context, query *v1.Query) ([]string, error) {
	pageQuery := buildPageQuery(query, search.ImageID, search.ImageName,
		search.ImageName, search.ImageOS, search.ImageCreatedTime, search.ImageScanTime)

	var ids []string
	err := pgSearch.RunSelectRequestForSchemaFn[imageIDRow](ctx, h.deps.DB, pkgSchema.ImagesV2Schema, pageQuery, func(row *imageIDRow) error {
		ids = append(ids, row.ImageID)
		return nil
	})
	return ids, err
}

// imageMatchedCVECounts returns the number of CVEs matching the current filters per image.
// The severity tab filter is part of the query, so only the active severity is counted.
func (h *handler) imageMatchedCVECounts(ctx context.Context, base *v1.Query, ids []string) (map[string]int, error) {
	countQuery := search.ConjunctionQuery(
		queryWithoutTraversal(base),
		search.NewQueryBuilder().AddExactMatches(search.ImageID, ids...).ProtoQuery(),
	)
	countQuery.Selects = []*v1.QuerySelect{
		search.NewQuerySelect(search.ImageID).Proto(),
		search.NewQuerySelect(search.CVE).Distinct().AggrFunc(aggregatefunc.Count).Proto(),
	}
	countQuery.GroupBy = &v1.QueryGroupBy{Fields: []string{search.ImageID.String()}}

	counts := make(map[string]int, len(ids))
	err := pgSearch.RunSelectRequestForSchemaFn[imageCVECountRow](ctx, h.deps.DB, pkgSchema.ImagesV2Schema, countQuery, func(row *imageCVECountRow) error {
		counts[row.ImageID] = row.CVECount
		return nil
	})
	return counts, err
}

func (h *handler) imageDeploymentCounts(ctx context.Context, base *v1.Query, ids []string) (map[string]int, error) {
	countQuery := search.ConjunctionQuery(
		queryWithoutTraversal(base),
		search.NewQueryBuilder().AddExactMatches(search.ImageID, ids...).ProtoQuery(),
	)
	countQuery.Selects = []*v1.QuerySelect{
		search.NewQuerySelect(search.ImageID).Proto(),
		search.NewQuerySelect(search.DeploymentID).Distinct().AggrFunc(aggregatefunc.Count).Proto(),
	}
	countQuery.GroupBy = &v1.QueryGroupBy{Fields: []string{search.ImageID.String()}}

	counts := make(map[string]int, len(ids))
	err := pgSearch.RunSelectRequestForSchemaFn[imageDeploymentCountRow](ctx, h.deps.DB, pkgSchema.ImagesV2Schema, countQuery, func(row *imageDeploymentCountRow) error {
		counts[row.ImageID] = row.DeploymentCount
		return nil
	})
	return counts, err
}

func (h *handler) watchedImageNames(ctx context.Context) (set.StringSet, error) {
	watched, err := h.deps.WatchedImageStore.GetAllWatchedImages(ctx)
	if err != nil {
		return nil, err
	}
	names := set.NewStringSet()
	for _, image := range watched {
		names.Add(image.GetName())
	}
	return names, nil
}

func watchStatus(watchedNames set.StringSet, imageName string) string {
	if watchedNames == nil {
		return watchStatusUnknown
	}
	if watchedNames.Contains(imageName) {
		return watchStatusWatched
	}
	return watchStatusNotWatched
}

func imageNotesToStrings(notes []storage.ImageV2_Note) []string {
	out := make([]string, 0, len(notes))
	for _, note := range notes {
		out = append(out, note.String())
	}
	return out
}

func signatureVerificationData(image *storage.ImageV2) *ImageSignatureVerificationData {
	data := image.GetSignatureVerificationData()
	if data == nil {
		return nil
	}
	results := make([]SignatureVerificationResult, 0, len(data.GetResults()))
	for _, result := range data.GetResults() {
		results = append(results, SignatureVerificationResult{
			Description:             result.GetDescription(),
			Status:                  result.GetStatus().String(),
			VerificationTime:        formatTime(timePtr(result.GetVerificationTime())),
			VerifiedImageReferences: nonNilStrings(result.GetVerifiedImageReferences()),
			VerifierID:              result.GetVerifierId(),
			VerifierName:            result.GetVerifierName(),
		})
	}
	return &ImageSignatureVerificationData{Results: results}
}

func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func scanNotesToStrings(notes []storage.ImageScan_Note) []string {
	out := make([]string, 0, len(notes))
	for _, note := range notes {
		out = append(out, note.String())
	}
	return out
}
