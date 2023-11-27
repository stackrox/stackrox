package datastore

import (
	"context"

	edge "github.com/stackrox/rox/central/complianceoperator/v2/profiles/profileclusteredge/store/postgres"
	"github.com/stackrox/rox/central/complianceoperator/v2/profiles/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
)

var (
	complianceOperatorSAC = sac.ForResource(resources.ComplianceOperator)
)

type datastoreImpl struct {
	store            postgres.Store
	profileEdgeStore edge.Store
}

// UpsertProfile adds the profile to the database
func (d *datastoreImpl) UpsertProfile(ctx context.Context, profile *storage.ComplianceOperatorProfileV2, clusterID string, profileUID string) error {
	if ok, err := complianceOperatorSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	if err := d.store.Upsert(ctx, profile); err != nil {
		return err
	}

	profileEdge := &storage.ComplianceOperatorProfileClusterEdge{
		Id:         uuid.NewV4().String(),
		ProfileId:  profile.GetId(),
		ProfileUid: profileUID,
		ClusterId:  clusterID,
	}
	return d.profileEdgeStore.Upsert(ctx, profileEdge)
}

// DeleteProfileForCluster removes a profile from the database
func (d *datastoreImpl) DeleteProfileForCluster(ctx context.Context, uid string, clusterID string) error {
	if ok, err := complianceOperatorSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	return d.profileEdgeStore.DeleteByQuery(ctx, search.NewQueryBuilder().
		AddExactMatches(search.ClusterID, clusterID).
		AddExactMatches(search.ComplianceOperatorProfileUID, uid).ProtoQuery())
}

// GetProfileEdgesByCluster gets the list of profile edges for a given cluster
func (d *datastoreImpl) GetProfileEdgesByCluster(ctx context.Context, clusterID string) ([]*storage.ComplianceOperatorProfileClusterEdge, error) {
	if ok, err := complianceOperatorSAC.ReadAllowed(ctx); err != nil {
		return nil, err
	} else if !ok {
		return nil, sac.ErrResourceAccessDenied
	}

	return d.profileEdgeStore.GetByQuery(ctx, search.NewQueryBuilder().
		AddExactMatches(search.ClusterID, clusterID).ProtoQuery())
}
