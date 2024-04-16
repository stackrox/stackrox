package datastore

import (
	"context"

	// "github.com/jackc/pgx/v5"
	"github.com/stackrox/rox/central/runtimeconfiguration/store"
	runtimeCollectionsStore "github.com/stackrox/rox/central/runtimeconfigurationcollection/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/uuid"
)

type datastoreImpl struct {
	storage   store.Store
	rcStorage runtimeCollectionsStore.Store
	pool      postgres.DB
}

var (
	// Add in SAC later
	// rcSAC = sac.ForResource(resources.Administration)
	log = logging.LoggerForModule()
)

func newDatastoreImpl(
	storage store.Store,
	rcStorage runtimeCollectionsStore.Store,
	pool postgres.DB,
) *datastoreImpl {
	return &datastoreImpl{
		storage:   storage,
		rcStorage: rcStorage,
		pool:      pool,
	}
}

func (ds *datastoreImpl) getResourceCollections(ctx context.Context) ([]*storage.ResourceCollection, error) {
	resourceCollections := make([]*storage.ResourceCollection, 0)

	err := ds.rcStorage.Walk(ctx,
		func(resourceCollection *storage.ResourceCollection) error {
			resourceCollections = append(resourceCollections, resourceCollection)
			return nil
		})

	return resourceCollections, err
}

func (ds *datastoreImpl) GetRuntimeConfiguration(ctx context.Context) (*storage.RuntimeFilteringConfiguration, error) {
	runtimeFilters := make([]*storage.RuntimeFilter, 0)
	runtimeFilteringMap := make(map[storage.RuntimeFilterFeatures]*storage.RuntimeFilter)

	err := ds.storage.Walk(ctx,
		func(runtimeConfigurationRow *storage.RuntimeFilterData) error {
			if runtimeConfigurationRow.ResourceCollectionId == "" {
				runtimeFilteringMap[runtimeConfigurationRow.Feature] = &storage.RuntimeFilter{
					DefaultStatus: runtimeConfigurationRow.Status,
					Feature:       runtimeConfigurationRow.Feature,
					Rules:         make([]*storage.RuntimeFilter_RuntimeFilterRule, 0),
				}
			} else {
				rule := storage.RuntimeFilter_RuntimeFilterRule{
					ResourceCollectionId: runtimeConfigurationRow.ResourceCollectionId,
					Status:               runtimeConfigurationRow.Status,
				}
				rules := runtimeFilteringMap[runtimeConfigurationRow.Feature].Rules
				rules = append(rules, &rule)
				runtimeFilter := runtimeFilteringMap[runtimeConfigurationRow.Feature]
				runtimeFilter.Rules = rules
				runtimeFilteringMap[runtimeConfigurationRow.Feature] = runtimeFilter

			}

			return nil
		})

	if err != nil {
		return nil, err
	}

	for _, runtimeFilter := range runtimeFilteringMap {
		runtimeFilters = append(runtimeFilters, runtimeFilter)
	}

	resourceCollections, err := ds.getResourceCollections(ctx)

	runtimeFilteringConfiguration := storage.RuntimeFilteringConfiguration{
		RuntimeFilters:      runtimeFilters,
		ResourceCollections: resourceCollections,
	}

	return &runtimeFilteringConfiguration, err
}

func convertRuntimeConfigurationToRuntimeConfigurationTable(runtimeConfiguration *storage.RuntimeFilteringConfiguration) []*storage.RuntimeFilterData {
	runtimeFiltersRows := make([]*storage.RuntimeFilterData, 0)

	runtimeFilters := runtimeConfiguration.RuntimeFilters
	for _, runtimeFilter := range runtimeFilters {
		runtimeFiltersRow := storage.RuntimeFilterData{
			Id:      uuid.NewV4().String(),
			Feature: runtimeFilter.Feature,
			Status:  runtimeFilter.DefaultStatus,
		}
		runtimeFiltersRows = append(runtimeFiltersRows, &runtimeFiltersRow)
		for _, rule := range runtimeFilter.Rules {
			runtimeFiltersRow2 := storage.RuntimeFilterData{
				Id:                   uuid.NewV4().String(),
				Feature:              runtimeFilter.Feature,
				Status:               rule.Status,
				ResourceCollectionId: rule.ResourceCollectionId,
			}
			runtimeFiltersRows = append(runtimeFiltersRows, &runtimeFiltersRow2)
		}

	}

	return runtimeFiltersRows
}

func (ds *datastoreImpl) SetRuntimeConfiguration(ctx context.Context, runtimeConfiguration *storage.RuntimeFilteringConfiguration) error {
	runtimeConfigurationRows := convertRuntimeConfigurationToRuntimeConfigurationTable(runtimeConfiguration)

	log.Infof("Upserting %+v rows", len(runtimeConfigurationRows))
	err := ds.storage.UpsertMany(ctx, runtimeConfigurationRows)
	if err != nil {
		return err
	}
	err = ds.rcStorage.UpsertMany(ctx, runtimeConfiguration.ResourceCollections)

	return err
}
