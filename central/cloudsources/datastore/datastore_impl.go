package datastore

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/cloudsources/datastore/internal/search"
	"github.com/stackrox/rox/central/cloudsources/datastore/internal/store"
	discoveredClustersDS "github.com/stackrox/rox/central/discoveredclusters/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/endpoints"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

var (
	_ DataStore = (*datastoreImpl)(nil)

	log = logging.LoggerForModule()

	discoveredClusterCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.ResourceScopeKeys(resources.Administration),
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
		),
	)
)

type datastoreImpl struct {
	searcher            search.Searcher
	store               store.Store
	discoveredClusterDS discoveredClustersDS.DataStore
}

func (ds *datastoreImpl) CountCloudSources(ctx context.Context, query *v1.Query) (int, error) {
	count, err := ds.searcher.Count(ctx, query)
	if err != nil {
		return 0, errors.Wrap(err, "failed to count cloud sources")
	}
	return count, nil
}

func (ds *datastoreImpl) GetCloudSource(ctx context.Context, id string) (*storage.CloudSource, error) {
	cloudSource, exists, err := ds.store.Get(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cloud source")
	}
	if !exists {
		return nil, errox.NotFound.Newf("cloud source %q not found", id)
	}
	return cloudSource, nil
}

func (ds *datastoreImpl) ProcessCloudSources(ctx context.Context, fn func(obj *storage.CloudSource) error) error {
	return ds.store.Walk(ctx, fn)
}

func (ds *datastoreImpl) ListCloudSources(ctx context.Context, query *v1.Query) ([]*storage.CloudSource, error) {
	cloudSources, err := ds.store.GetByQuery(ctx, query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list cloud sources")
	}
	return cloudSources, nil
}

func (ds *datastoreImpl) UpsertCloudSource(ctx context.Context, cloudSource *storage.CloudSource) error {
	if err := validateCloudSource(cloudSource); err != nil {
		return errox.InvalidArgs.CausedBy(err)
	}
	if err := ds.store.Upsert(ctx, cloudSource); err != nil {
		return errors.Wrapf(err, "failed to upsert cloud source %q", cloudSource.GetId())
	}
	return nil
}

func (ds *datastoreImpl) DeleteCloudSource(ctx context.Context, id string) error {
	if err := ds.store.Delete(ctx, id); err != nil {
		return errors.Wrapf(err, "failed to delete cloud source %q", id)
	}

	_, err := ds.discoveredClusterDS.DeleteDiscoveredClusters(discoveredClusterCtx,
		searchPkg.NewQueryBuilder().AddExactMatches(searchPkg.IntegrationID, id).ProtoQuery())
	return err
}

func validateCloudSource(cloudSource *storage.CloudSource) error {
	if cloudSource == nil {
		return errors.New("empty cloud source")
	}

	errorList := errorhelpers.NewErrorList("Validation")
	if cloudSource.GetId() == "" {
		errorList.AddString("cloud source id must be defined")
	}
	if cloudSource.GetName() == "" {
		errorList.AddString("cloud source name must be defined")
	}
	if err := validateCredentials(cloudSource); err != nil {
		errorList.AddError(err)
	}
	if err := validateType(cloudSource); err != nil {
		errorList.AddError(err)
	}
	if err := endpoints.ValidateEndpoints(cloudSource.GetConfig()); err != nil {
		errorList.AddWrap(err, "invalid endpoint")
	}
	return errorList.ToError()
}

func validateCredentials(cloudSource *storage.CloudSource) error {
	creds := cloudSource.GetCredentials()
	switch cloudSource.GetConfig().(type) {
	case *storage.CloudSource_PaladinCloud:
		if creds.GetSecret() == "" {
			return errors.New("cloud source credentials must be defined")
		}
		return nil
	case *storage.CloudSource_Ocm:
		// TODO(ROX-25633): fail validation if token is used for authentication.
		if creds.GetSecret() != "" {
			log.Warn("secret is deprecated for type OCM - use clientId and clientSecret instead")
		}
		if creds.GetSecret() == "" && (creds.GetClientId() == "" || creds.GetClientSecret() == "") {
			return errors.New("either secret or both clientId and clientSecret must be defined")
		}
		return nil
	}
	return errors.New("invalid cloud source config type")
}

func validateType(cloudSource *storage.CloudSource) error {
	cloudSourceType := cloudSource.GetType()
	if cloudSourceType == storage.CloudSource_TYPE_UNSPECIFIED {
		return errors.New("cloud source type must be specified")
	}
	switch cloudSource.GetConfig().(type) {
	case *storage.CloudSource_PaladinCloud:
		if cloudSourceType != storage.CloudSource_TYPE_PALADIN_CLOUD {
			return errors.Errorf("invalid cloud source type %q", cloudSourceType.String())
		}
		return nil
	case *storage.CloudSource_Ocm:
		if cloudSourceType != storage.CloudSource_TYPE_OCM {
			return errors.Errorf("invalid cloud source type %q", cloudSourceType.String())
		}
		return nil
	}
	return errors.New("invalid cloud source config type")
}
