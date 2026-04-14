package app

import (
	"github.com/stackrox/rox/central/alert/mappings"
	"github.com/stackrox/rox/central/auth/internaltokens/service"
	csvhandler "github.com/stackrox/rox/central/cve/common/csv"
	debugservice "github.com/stackrox/rox/central/debug/service"
	detectionservice "github.com/stackrox/rox/central/detection/service"
	"github.com/stackrox/rox/central/globaldb/v2backuprestore/formats/postgresv1"
	scannerhandler "github.com/stackrox/rox/central/scannerdefinitions/handler"
	"github.com/stackrox/rox/central/search/options"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	"github.com/stackrox/rox/pkg/logging"
)

var log = logging.LoggerForModule()

// initComponentLogic initializes all central-specific components that were
// previously using init() functions.
func initComponentLogic() {
	// Initialize metrics
	service.RegisterMetrics()

	// Initialize search and alert options
	options.InitCategoryToOptionsSet()
	mappings.InitOptionsMap()

	// Initialize CSV handlers
	csvhandler.InitOptionsMap()

	// Initialize service configurations
	debugservice.InitMainClusterConfig()
	detectionservice.InitWorkloadScheme()
	scannerhandler.InitScannerConfig()

	// Register backup formats
	postgresv1.RegisterFormat()

	// Initialize proxy configuration
	if !proxy.UseWithDefaultTransport() {
		log.Warn("Failed to use proxy transport with default HTTP transport. Some proxy features may not work.")
	}
}
