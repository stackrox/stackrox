package datastore

import (
	"context"

	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/reports/common"
	pgStore "github.com/stackrox/rox/central/reports/config/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	log = logging.LoggerForModule()
)

var (
	once sync.Once
	ds   DataStore
)

// Singleton creates a singleton for the report configuration datastore and loads the plugin client config
func Singleton() DataStore {
	once.Do(func() {
		storage := pgStore.New(globaldb.GetPostgres())
		ds = New(storage)
		addViewBasedPlaceholder(ds)
	})
	return ds
}

// addViewBasedPlaceholder creates a placeholder report configuration for view-based reports
// to satisfy the foreign key constraint
func addViewBasedPlaceholder(datastore DataStore) {
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowAllAccessScopeChecker())

	// Check if the placeholder already exists
	_, exists, err := datastore.GetReportConfiguration(ctx, common.ViewBasedReportConfigurationID)
	if err != nil {
		log.Error("Unable to create placeholder config")
	}
	if exists {
		// Placeholder already exists, no need to create it
		return
	}

	// Create the placeholder configuration
	placeholderConfig := &storage.ReportConfiguration{
		Id:          common.ViewBasedReportConfigurationID,
		Name:        "View-Based Reports Placeholder",
		Description: "Internal placeholder configuration for view-based reports",
		Type:        storage.ReportConfiguration_VULNERABILITY,
	}

	err = datastore.UpdateReportConfiguration(ctx, placeholderConfig)
	if err != nil {
		log.Warnf("Failed to create placeholder report configuration for view-based reports: %v", err)
	} else {
		log.Info("Created placeholder report configuration for view-based reports")
	}
}
