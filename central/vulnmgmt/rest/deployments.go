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
)

type deploymentIDRow struct {
	DeploymentID string `db:"deployment_id"`
}

type deploymentCVECountRow struct {
	DeploymentID string `db:"deployment_id"`
	CVECount     int    `db:"cve_count"`
}

type deploymentImageCountRow struct {
	DeploymentID string `db:"deployment_id"`
	ImageCount   int    `db:"image_id_count"`
}

func (h *handler) listDeployments(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	query, err := parseSearchQuery(r)
	if err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	query = withPaginationLimits(query)

	ids, err := h.pageDeploymentIDs(ctx, query)
	if err != nil {
		writeError(w, err, http.StatusInternalServerError)
		return
	}
	if len(ids) == 0 {
		writeJSON(w, DeploymentListResponse{Items: []DeploymentListItem{}})
		return
	}

	cveCounts, err := h.deploymentCVECounts(ctx, query, ids)
	if err != nil {
		writeError(w, err, http.StatusInternalServerError)
		return
	}
	imageCounts, err := h.deploymentImageCounts(ctx, query, ids)
	if err != nil {
		writeError(w, err, http.StatusInternalServerError)
		return
	}
	deployments, err := h.deps.DeploymentStore.GetDeployments(ctx, ids)
	if err != nil {
		writeError(w, err, http.StatusInternalServerError)
		return
	}

	deploymentByID := make(map[string]*storage.Deployment, len(deployments))
	for _, deployment := range deployments {
		deploymentByID[deployment.GetId()] = deployment
	}

	items := make([]DeploymentListItem, 0, len(ids))
	for _, id := range ids {
		deployment := deploymentByID[id]
		if deployment == nil {
			continue
		}
		items = append(items, DeploymentListItem{
			ID:              deployment.GetId(),
			Name:            deployment.GetName(),
			Type:            deployment.GetType(),
			Cluster:         deployment.GetClusterName(),
			Namespace:       deployment.GetNamespace(),
			CreatedAt:       formatTime(timePtr(deployment.GetCreated())),
			ImageCount:      imageCounts[id],
			MatchedCVECount: cveCounts[id],
		})
	}

	writeJSON(w, DeploymentListResponse{Items: items})
}

func (h *handler) pageDeploymentIDs(ctx context.Context, query *v1.Query) ([]string, error) {
	pageQuery := buildPageQuery(query, search.DeploymentID, search.DeploymentName,
		search.DeploymentName, search.Cluster, search.Namespace, search.Created)

	var ids []string
	err := pgSearch.RunSelectRequestForSchemaFn[deploymentIDRow](ctx, h.deps.DB, pkgSchema.DeploymentsSchema, pageQuery, func(row *deploymentIDRow) error {
		ids = append(ids, row.DeploymentID)
		return nil
	})
	return ids, err
}

// deploymentCVECounts returns the number of matching CVEs per deployment.
func (h *handler) deploymentCVECounts(ctx context.Context, base *v1.Query, ids []string) (map[string]int, error) {
	countQuery := search.ConjunctionQuery(
		queryWithoutTraversal(base),
		search.NewQueryBuilder().AddExactMatches(search.DeploymentID, ids...).ProtoQuery(),
	)
	countQuery.Selects = []*v1.QuerySelect{
		search.NewQuerySelect(search.DeploymentID).Proto(),
		search.NewQuerySelect(search.CVE).Distinct().AggrFunc(aggregatefunc.Count).Proto(),
	}
	countQuery.GroupBy = &v1.QueryGroupBy{Fields: []string{search.DeploymentID.String()}}

	counts := make(map[string]int, len(ids))
	err := pgSearch.RunSelectRequestForSchemaFn[deploymentCVECountRow](ctx, h.deps.DB, pkgSchema.DeploymentsSchema, countQuery, func(row *deploymentCVECountRow) error {
		counts[row.DeploymentID] = row.CVECount
		return nil
	})
	return counts, err
}

// deploymentImageCounts returns the number of matching images per deployment. This is a
// separate query from the CVE counts so deployments without CVEs still report their
// image count. It is rooted on the images schema so the image ID is unambiguous.
func (h *handler) deploymentImageCounts(ctx context.Context, base *v1.Query, ids []string) (map[string]int, error) {
	countQuery := search.ConjunctionQuery(
		queryWithoutTraversal(base),
		search.NewQueryBuilder().AddExactMatches(search.DeploymentID, ids...).ProtoQuery(),
	)
	countQuery.Selects = []*v1.QuerySelect{
		search.NewQuerySelect(search.DeploymentID).Proto(),
		search.NewQuerySelect(search.ImageID).Distinct().AggrFunc(aggregatefunc.Count).Proto(),
	}
	countQuery.GroupBy = &v1.QueryGroupBy{Fields: []string{search.DeploymentID.String()}}

	counts := make(map[string]int, len(ids))
	err := pgSearch.RunSelectRequestForSchemaFn[deploymentImageCountRow](ctx, h.deps.DB, pkgSchema.ImagesV2Schema, countQuery, func(row *deploymentImageCountRow) error {
		counts[row.DeploymentID] = row.ImageCount
		return nil
	})
	return counts, err
}
