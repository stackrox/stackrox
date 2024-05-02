package datastore

import (
	"context"
	"fmt"

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

	if len(resourceCollections) == 0 {
		resourceCollections = nil
	}

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
		if len(runtimeFilter.Rules) == 0 {
			runtimeFilter.Rules = nil
		}
		runtimeFilters = append(runtimeFilters, runtimeFilter)
	}

	if len(runtimeFilters) == 0 {
		runtimeFilters = nil
	}

	resourceCollections, err := ds.getResourceCollections(ctx)

	runtimeFilteringConfiguration := storage.RuntimeFilteringConfiguration{
		RuntimeFilters:      runtimeFilters,
		ResourceCollections: resourceCollections,
	}

	return &runtimeFilteringConfiguration, err
}

func getRuntimeConfigId(feature storage.RuntimeFilterFeatures, collection_id string) string {
	idNamespace := uuid.FromStringOrPanic("801fcce1-56d3-48bd-b1ac-c41fdc6c3d94") // Coppied from process/id/id.go probably needs to be changed

        id := uuid.NewV5(idNamespace, fmt.Sprintf("%s %s", feature, collection_id)).String()

        return id
}

func convertRuntimeConfigurationToRuntimeConfigurationTable(runtimeConfiguration *storage.RuntimeFilteringConfiguration) []*storage.RuntimeFilterData {
	runtimeFiltersRows := make([]*storage.RuntimeFilterData, 0)

	if runtimeConfiguration.RuntimeFilters != nil {
		runtimeFilters := runtimeConfiguration.RuntimeFilters
		for _, runtimeFilter := range runtimeFilters {
			id := getRuntimeConfigId(runtimeFilter.Feature, "")
			runtimeFiltersRow := storage.RuntimeFilterData{
				Id:      id,
				Feature: runtimeFilter.Feature,
				Status:  runtimeFilter.DefaultStatus,
			}
			runtimeFiltersRows = append(runtimeFiltersRows, &runtimeFiltersRow)
			for _, rule := range runtimeFilter.Rules {
				id = getRuntimeConfigId(runtimeFilter.Feature, rule.ResourceCollectionId)
				runtimeFiltersRow2 := storage.RuntimeFilterData{
					Id:                   id,
					Feature:              runtimeFilter.Feature,
					Status:               rule.Status,
					ResourceCollectionId: rule.ResourceCollectionId,
				}
				runtimeFiltersRows = append(runtimeFiltersRows, &runtimeFiltersRow2)
			}

		}
	}

	return runtimeFiltersRows
}

func (ds *datastoreImpl) clearTables(ctx context.Context) error {
	ids, err1 := ds.storage.GetIDs(ctx)
	err2 := ds.storage.DeleteMany(ctx, ids)

	ids, err3 := ds.rcStorage.GetIDs(ctx)
	err4 := ds.rcStorage.DeleteMany(ctx, ids)

	if err1 != nil {
		return err1
	}
	if err2 != nil {
		return err2
	}
	if err3 != nil {
		return err3
	}
	if err4 != nil {
		return err4
	}

	return nil
}

func (ds *datastoreImpl) SetRuntimeConfiguration(ctx context.Context, runtimeConfiguration *storage.RuntimeFilteringConfiguration) error {
	runtimeConfigurationRows := convertRuntimeConfigurationToRuntimeConfigurationTable(runtimeConfiguration)

	err := ds.clearTables(ctx)
	if err != nil {
		return err
	}

	log.Infof("Upserting %+v rows", len(runtimeConfigurationRows))
	err = ds.storage.UpsertMany(ctx, runtimeConfigurationRows)
	if err != nil {
		return err
	}
	log.Infof("Upserting %+v collections", len(runtimeConfiguration.ResourceCollections))
	err = ds.rcStorage.UpsertMany(ctx, runtimeConfiguration.ResourceCollections)

	return err
}
