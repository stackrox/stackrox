package datastore

import (
	"context"

	statusStore "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/scanconfigstatus/store/postgres"
	pgStore "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/store/postgres"
	ssbDatastore "github.com/stackrox/rox/central/complianceoperator/v2/scansettingbindings/datastore"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/uuid"
)

var (
	once sync.Once

	dataStore DataStore
)

var log = logging.LoggerForModule()

func initialize() {
	pool := globaldb.GetPostgres()
	storage := pgStore.New(pool)
	dataStore = New(storage, statusStore.New(pool), pool)

	reconcileDiscoveredOnStartup()
}

// reconcileDiscoveredOnStartup creates scan config records for any SSBs
// that don't already have a corresponding scan config. Covers the case
// where sensor synced SSBs before the reconciler was deployed.
func reconcileDiscoveredOnStartup() {
	ctx := sac.WithAllAccess(context.Background())
	ssbDS := ssbDatastore.Singleton()
	if ssbDS == nil {
		return
	}

	discovered, err := ssbDS.GetDistinctScanConfigs(ctx, search.EmptyQuery())
	if err != nil {
		log.Errorf("startup reconcile: listing SSBs: %v", err)
		return
	}

	for _, dc := range discovered {
		existing, err := dataStore.GetScanConfigurationByName(ctx, dc.Name)
		if err != nil {
			log.Errorf("startup reconcile: looking up %q: %v", dc.Name, err)
			continue
		}
		// Skip configs created by users via the managed path (ProcessScanRequest).
		// Those SSBs carry the stackrox label; the pipeline reconciler handles them.
		if existing != nil && existing.GetModifiedBy().GetId() != "" {
			continue
		}

		clusters := make([]*storage.ComplianceOperatorScanConfigurationV2_Cluster, 0, len(dc.ClusterIDs))
		for _, cid := range dc.ClusterIDs {
			clusters = append(clusters, &storage.ComplianceOperatorScanConfigurationV2_Cluster{ClusterId: cid})
		}
		profiles := make([]*storage.ComplianceOperatorScanConfigurationV2_ProfileName, 0, len(dc.ProfileNames))
		for _, p := range dc.ProfileNames {
			profiles = append(profiles, &storage.ComplianceOperatorScanConfigurationV2_ProfileName{ProfileName: p})
		}

		var scanConfig *storage.ComplianceOperatorScanConfigurationV2
		if existing != nil {
			existing.Clusters = clusters
			existing.Profiles = profiles
			existing.LastUpdatedTime = protocompat.TimestampNow()
			if err := dataStore.UpsertScanConfiguration(ctx, existing); err != nil {
				log.Errorf("startup reconcile: updating config %q: %v", dc.Name, err)
				continue
			}
			scanConfig = existing
		} else {
			scanConfig = &storage.ComplianceOperatorScanConfigurationV2{
				Id:              uuid.NewV4().String(),
				ScanConfigName:  dc.Name,
				Clusters:        clusters,
				Profiles:        profiles,
				CreatedTime:     protocompat.TimestampNow(),
				LastUpdatedTime: protocompat.TimestampNow(),
			}
			if err := dataStore.UpsertScanConfiguration(ctx, scanConfig); err != nil {
				log.Errorf("startup reconcile: creating config for %q: %v", dc.Name, err)
				continue
			}
			log.Infof("startup reconcile: created discovered config %q with ID %s", dc.Name, scanConfig.GetId())
		}

		for _, cid := range dc.ClusterIDs {
			if err := dataStore.UpdateClusterStatus(ctx, scanConfig.GetId(), cid, "", cid); err != nil {
				log.Errorf("startup reconcile: updating cluster status for %q cluster %s: %v", dc.Name, cid, err)
			}
		}
	}
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	if !features.ComplianceEnhancements.Enabled() {
		return nil
	}
	once.Do(initialize)
	return dataStore
}
