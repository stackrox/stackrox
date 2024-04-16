package datastore

import (
	"context"
	// "fmt"
	// "sort"
	//"time"

	// "github.com/jackc/pgx/v5"
	// "github.com/stackrox/rox/central/metrics"
	// countMetrics "github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/runtimeconfiguration/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres"
	// "github.com/stackrox/rox/pkg/postgres/pgutils"
	// "github.com/stackrox/rox/pkg/protocompat"
	// "github.com/stackrox/rox/pkg/sac"
	//"github.com/stackrox/rox/pkg/sac/resources"
	// "github.com/stackrox/rox/pkg/search"
	//"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/uuid"
)

type datastoreImpl struct {
	storage store.Store
	pool    postgres.DB
}

 var (
	//rcSAC = sac.ForResource(resources.Administration)
	log   = logging.LoggerForModule()
)

func newDatastoreImpl(
	storage store.Store,
	pool postgres.DB,
) *datastoreImpl {
	return &datastoreImpl{
		storage: storage,
		pool:    pool,
	}
}

func (ds *datastoreImpl) GetRuntimeConfiguration(ctx context.Context) (*storage.RuntimeFilteringConfiguration, error) {
	runtimeFilters := make([]*storage.RuntimeFilter, 0)
	runtimeFilteringMap := make(map[storage.RuntimeFilterFeatures]storage.RuntimeFilter)

	err := ds.storage.Walk(ctx,
                func(runtimeConfigurationRow *storage.RuntimeFilterData) error {
			log.Infof("Row= %+V", runtimeConfigurationRow)
			if runtimeConfigurationRow.ResourceCollectionId == "" {
				runtimeFilteringMap[runtimeConfigurationRow.Feature] = storage.RuntimeFilter{
					DefaultStatus:	runtimeConfigurationRow.Status,
					Feature:	runtimeConfigurationRow.Feature,
					Rules:		make([]*storage.RuntimeFilter_RuntimeFilterRule, 0),
				}
			} else {
				rule := storage.RuntimeFilter_RuntimeFilterRule{
					ResourceCollectionId:	runtimeConfigurationRow.ResourceCollectionId,
					Status:			runtimeConfigurationRow.Status,
				}
				rules := runtimeFilteringMap[runtimeConfigurationRow.Feature].Rules
				rules = append(rules, &rule)
				runtimeFilter := runtimeFilteringMap[runtimeConfigurationRow.Feature]
				runtimeFilter.Rules = rules
				runtimeFilteringMap[runtimeConfigurationRow.Feature] = runtimeFilter

			}

                        return nil
                })

	for _, runtimeFilter := range runtimeFilteringMap{
		runtimeFilters = append(runtimeFilters, &runtimeFilter)
		log.Infof("runtimeFilter= %+v", runtimeFilter)
	}

	runtimeFilteringConfiguration := storage.RuntimeFilteringConfiguration{
		RuntimeFilters: runtimeFilters,
	}
	log.Infof("Got %+v runtimeFilters", len(runtimeFilters))

	return &runtimeFilteringConfiguration, err
}

func convertRuntimeConfigurationToRuntimeConfigurationTable(runtimeConfiguration *storage.RuntimeFilteringConfiguration) []*storage.RuntimeFilterData {
	runtimeFiltersRows := make([]*storage.RuntimeFilterData, 0)

	runtimeFilters := runtimeConfiguration.RuntimeFilters
	for _, runtimeFilter := range runtimeFilters {
		runtimeFiltersRow := storage.RuntimeFilterData{
			Id:		uuid.NewV4().String(),
			Feature:	runtimeFilter.Feature,
			Status:		runtimeFilter.DefaultStatus,
		}
		runtimeFiltersRows = append(runtimeFiltersRows, &runtimeFiltersRow)
		for _, rule := range runtimeFilter.Rules {
			runtimeFiltersRow2 := storage.RuntimeFilterData{
				Id:			uuid.NewV4().String(),
				Feature:		runtimeFilter.Feature,
				Status:			rule.Status,
				ResourceCollectionId:	rule.ResourceCollectionId,
			}
			runtimeFiltersRows = append(runtimeFiltersRows, &runtimeFiltersRow2)
		}

	}

	return runtimeFiltersRows
}

func (ds *datastoreImpl) SetRuntimeConfiguration(ctx context.Context, runtimeConfiguration *storage.RuntimeFilteringConfiguration) error {
	runtimeConfigurationRows := convertRuntimeConfigurationToRuntimeConfigurationTable(runtimeConfiguration)

	log.Infof("Upserting %+v rows", len(runtimeConfigurationRows))
	ds.storage.UpsertMany(ctx, runtimeConfigurationRows)

	return nil
}
