package datastore

import (
	"context"
	"strings"

	"github.com/stackrox/rox/central/secret/internal/store"
	pgStore "github.com/stackrox/rox/central/secret/internal/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sac/resources"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
)

type datastoreImpl struct {
	storage store.Store
	db      postgres.DB
}

func newPostgres(db postgres.DB) DataStore {
	dbStore := pgStore.New(db)
	return &datastoreImpl{
		storage: dbStore,
		db:      db,
	}
}

func (d *datastoreImpl) GetSecret(ctx context.Context, id string) (*storage.Secret, bool, error) {
	secret, exists, err := d.storage.Get(ctx, id)
	if err != nil || !exists {
		return nil, false, err
	}

	return secret, true, nil
}

func (d *datastoreImpl) SearchSecrets(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	if q == nil {
		q = pkgSearch.EmptyQuery()
	} else {
		q = q.CloneVT()
	}

	// Add name field to select columns
	q.Selects = append(q.GetSelects(), pkgSearch.NewQuerySelect(pkgSearch.SecretName).Proto())

	results, err := d.storage.Search(ctx, q)
	if err != nil {
		return nil, err
	}

	// Extract name from FieldValues and populate Name in search results
	searchTag := strings.ToLower(pkgSearch.SecretName.String())
	for i := range results {
		if results[i].FieldValues != nil {
			if nameVal, ok := results[i].FieldValues[searchTag]; ok {
				results[i].Name = nameVal
			}
		}
	}

	return pkgSearch.ResultsToSearchResultProtos(results, &SecretSearchResultConverter{}), nil
}

func (d *datastoreImpl) SearchListSecrets(ctx context.Context, request *v1.Query) ([]*storage.ListSecret, error) {
	// Clone the query and add field selections for ListSecret fields
	query := request.CloneVT()
	if query == nil {
		query = pkgSearch.EmptyQuery()
	}

	// Apply SAC filtering
	sacQueryFilter, err := pgSearch.GetReadSACQuery(ctx, resources.Secret)
	if err != nil {
		return nil, err
	}
	query = pkgSearch.FilterQueryByQuery(query, sacQueryFilter)

	// Specify exact fields to select - framework will detect child table fields and aggregate them
	query.Selects = []*v1.QuerySelect{
		pkgSearch.NewQuerySelect(pkgSearch.SecretID).Proto(),
		pkgSearch.NewQuerySelect(pkgSearch.SecretName).Proto(),
		pkgSearch.NewQuerySelect(pkgSearch.ClusterID).Proto(),
		pkgSearch.NewQuerySelect(pkgSearch.Cluster).Proto(),
		pkgSearch.NewQuerySelect(pkgSearch.Namespace).Proto(),
		pkgSearch.NewQuerySelect(pkgSearch.CreatedTime).Proto(),
		pkgSearch.NewQuerySelect(pkgSearch.SecretType).Proto(),
	}

	// Execute single database query using search framework
	// Framework will:
	// 1. Detect SecretType comes from secrets_files child table
	// 2. Apply array_agg(DISTINCT ...) FILTER (WHERE ... IS NOT NULL) to aggregate types
	// 3. Auto-generate GROUP BY on parent table fields (id, name, etc.)
	// 4. Use LEFT JOIN for optional relationship (secrets with no files)
	var responses []*listSecretResponse
	err = pgSearch.RunSelectRequestForSchemaFn(ctx, d.db, pkgSchema.SecretsSchema, query,
		func(r *listSecretResponse) error {
			responses = append(responses, r)
			return nil
		})
	if err != nil {
		return nil, err
	}

	// Convert response structs to protobuf ListSecret objects
	listSecrets := make([]*storage.ListSecret, 0, len(responses))
	for _, r := range responses {
		listSecrets = append(listSecrets, r.toListSecret())
	}

	return listSecrets, nil
}

func (d *datastoreImpl) SearchRawSecrets(ctx context.Context, request *v1.Query) ([]*storage.Secret, error) {
	var secrets []*storage.Secret
	err := d.storage.GetByQueryFn(ctx, request, func(secret *storage.Secret) error {
		secrets = append(secrets, secret)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return secrets, nil
}

func (d *datastoreImpl) UpsertSecret(ctx context.Context, request *storage.Secret) error {
	return d.storage.Upsert(ctx, request)
}

func (d *datastoreImpl) RemoveSecret(ctx context.Context, id string) error {
	return d.storage.Delete(ctx, id)
}

func (d *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]pkgSearch.Result, error) {
	return d.storage.Search(ctx, q)
}

// Count returns the number of search results from the query
func (d *datastoreImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	return d.storage.Count(ctx, q)
}

type SecretSearchResultConverter struct{}

func (c *SecretSearchResultConverter) BuildName(result *pkgSearch.Result) string {
	return result.Name
}

func (c *SecretSearchResultConverter) BuildLocation(result *pkgSearch.Result) string {
	// Secrets do not have a location
	return ""
}

func (c *SecretSearchResultConverter) GetCategory() v1.SearchCategory {
	return v1.SearchCategory_SECRETS
}

func (c *SecretSearchResultConverter) GetScore(result *pkgSearch.Result) float64 {
	return result.Score
}
