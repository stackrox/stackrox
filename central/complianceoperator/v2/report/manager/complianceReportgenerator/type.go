package complianceReportgenerator

import (
<<<<<<< HEAD
	"context"

=======
>>>>>>> 6faeddcd64 (Added test file)
	"github.com/stackrox/rox/generated/storage"
)

type ComplianceReportRequest struct {
<<<<<<< HEAD
	ScanConfigID   string
	Notifiers      []*storage.NotifierConfiguration
	ClusterIDs     []string
	Profiles       []string
	ScanConfigName string
	Ctx            context.Context
=======
	scanConfigID   string
	notifiers      []*storage.NotifierConfiguration
	clusterIDs     []string
	profiles       []string
	scanConfigName string
>>>>>>> 6faeddcd64 (Added test file)
}
