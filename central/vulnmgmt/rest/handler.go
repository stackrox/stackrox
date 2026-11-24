package rest

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	imageCVEDS "github.com/stackrox/rox/central/cve/image/v2/datastore"
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	imageV2DS "github.com/stackrox/rox/central/imagev2/datastore"
	"github.com/stackrox/rox/central/views/imagecve"
	"github.com/stackrox/rox/central/views/imagecveflat"
	vulnReqDS "github.com/stackrox/rox/central/vulnmgmt/vulnerabilityrequest/datastore"
	watchedImageDS "github.com/stackrox/rox/central/watchedimage/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/parser"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const basePath = "/api/v2/vulnmgmt/"

const (
	defaultLimit int32 = 20
	maxLimit     int32 = 100
)

var log = logging.LoggerForModule()

// Deps holds the dependencies needed by the vulnerability management REST API.
type Deps struct {
	CVEView           imagecveflat.CveFlatView
	CVECountView      imagecve.CveView
	CVEDataStore      imageCVEDS.DataStore
	VulnReqStore      vulnReqDS.DataStore
	ImageStore        imageV2DS.DataStore
	DeploymentStore   deploymentDS.DataStore
	WatchedImageStore watchedImageDS.DataStore
	DB                postgres.DB
}

type handler struct {
	deps Deps
}

// NewHandler creates a new REST handler for the vulnerability management endpoints.
func NewHandler(deps Deps) http.Handler {
	return &handler{deps: deps}
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	switch {
	case r.URL.Path == basePath+"cves" || r.URL.Path == basePath+"cves/":
		h.listCVEs(w, r)
	case strings.HasPrefix(r.URL.Path, basePath+"cves/"):
		cve, ok := strings.CutSuffix(strings.TrimPrefix(r.URL.Path, basePath+"cves/"), "/detail")
		if !ok || cve == "" {
			http.NotFound(w, r)
			return
		}
		h.getCVEDetail(w, r, cve)
	case r.URL.Path == basePath+"images" || r.URL.Path == basePath+"images/":
		h.listImages(w, r)
	case r.URL.Path == basePath+"deployments" || r.URL.Path == basePath+"deployments/":
		h.listDeployments(w, r)
	case r.URL.Path == basePath+"counts" || r.URL.Path == basePath+"counts/":
		h.getCounts(w, r)
	default:
		http.NotFound(w, r)
	}
}

func parseSearchQuery(r *http.Request) (*v1.Query, error) {
	query, _, err := parser.ParseURLQuery(r.URL.Query())
	return query, err
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Errorf("failed to encode JSON response: %v", err)
	}
}

func writeError(w http.ResponseWriter, err error, status int) {
	http.Error(w, err.Error(), status)
}

func formatTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.RFC3339)
}

func timePtr(ts *timestamppb.Timestamp) *time.Time {
	if ts == nil {
		return nil
	}
	t := ts.AsTime()
	return &t
}

// queryWithoutTraversal returns the filters of q without pagination, selects or group by.
// Used for count queries that should match the same set as the list query.
func queryWithoutTraversal(q *v1.Query) *v1.Query {
	cloned := q.CloneVT()
	cloned.Selects = nil
	cloned.GroupBy = nil
	cloned.Pagination = nil
	return cloned
}

// withPaginationLimits applies the API default and maximum page size.
func withPaginationLimits(q *v1.Query) *v1.Query {
	cloned := q.CloneVT()
	pagination := cloned.GetPagination()
	if pagination == nil {
		pagination = &v1.QueryPagination{}
	} else {
		pagination = pagination.CloneVT()
	}
	if pagination.GetLimit() <= 0 {
		pagination.Limit = defaultLimit
	}
	if pagination.GetLimit() > maxLimit {
		pagination.Limit = maxLimit
	}
	cloned.Pagination = pagination
	return cloned
}

// buildPageQuery turns a filtered query into a query that returns only the page's entity
// IDs. Sort fields are restricted to entity columns and are added to the select and
// group by clauses so PostgreSQL can order grouped rows without functional-dependency
// inference on a cast primary key.
func buildPageQuery(q *v1.Query, idField search.FieldLabel, defaultSort search.FieldLabel, allowedSortFields ...search.FieldLabel) *v1.Query {
	allowed := make(map[string]bool, len(allowedSortFields))
	for _, field := range allowedSortFields {
		allowed[strings.ToLower(field.String())] = true
	}

	cloned := q.CloneVT()
	pagination := cloned.GetPagination()
	if pagination == nil {
		pagination = &v1.QueryPagination{}
	} else {
		pagination = pagination.CloneVT()
	}

	var sortOptions []*v1.QuerySortOption
	for _, sortOption := range pagination.GetSortOptions() {
		if allowed[strings.ToLower(sortOption.GetField())] {
			sortOptions = append(sortOptions, sortOption)
		}
	}
	if len(sortOptions) == 0 {
		sortOptions = []*v1.QuerySortOption{{Field: defaultSort.String()}}
	}
	pagination.SortOptions = sortOptions
	cloned.Pagination = pagination

	selects := []*v1.QuerySelect{search.NewQuerySelect(idField).Proto()}
	groupByFields := []string{idField.String()}
	seen := map[string]bool{strings.ToLower(idField.String()): true}
	for _, sortOption := range sortOptions {
		key := strings.ToLower(sortOption.GetField())
		if seen[key] {
			continue
		}
		seen[key] = true
		selects = append(selects, search.NewQuerySelect(search.FieldLabel(sortOption.GetField())).Proto())
		groupByFields = append(groupByFields, sortOption.GetField())
	}
	cloned.Selects = selects
	cloned.GroupBy = &v1.QueryGroupBy{Fields: groupByFields}

	return cloned
}

// exceptionStatusesForQuery mirrors getStatusesForExceptionCount in the UI: PENDING when
// viewing observed CVEs, APPROVED_PENDING_UPDATE otherwise.
func exceptionStatusesForQuery(q *v1.Query) []string {
	var state string
	search.ApplyFnToAllBaseQueries(q, func(bq *v1.BaseQuery) {
		if mfq := bq.GetMatchFieldQuery(); mfq != nil && strings.EqualFold(mfq.GetField(), search.VulnerabilityState.String()) {
			// Exact-match values are quoted by the query builder.
			state = strings.Trim(mfq.GetValue(), `"`)
		}
	})
	if strings.EqualFold(state, "OBSERVED") {
		return []string{"PENDING"}
	}
	return []string{"APPROVED_PENDING_UPDATE"}
}
