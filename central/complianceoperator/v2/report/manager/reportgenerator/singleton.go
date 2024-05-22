package complianceReportgenerator

import (
	checkResults "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore"
	notifierProcessor "github.com/stackrox/rox/central/notifier/processor"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once     sync.Once
	instance ComplianceReportGenerator
)

// Singleton provides the instance of Manager to use.
func Singleton() ComplianceReportGenerator {
	once.Do(initialize)
	return instance
}

func initialize() {
	instance = New(checkResults.Singleton(), notifierProcessor.Singleton())
}
