package main

import (
	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/internalapi/compliance"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	_ "github.com/stackrox/rox/pkg/compliance/checks" // Make sure all checks are available
	"github.com/stackrox/rox/pkg/compliance/checks/standards"
	"github.com/stackrox/rox/pkg/compliance/data"
	"github.com/stackrox/rox/pkg/compliance/framework"
)

func runChecks(client sensor.ComplianceService_CommunicateClient, scrapeConfig *sensor.MsgToCompliance_ScrapeConfig, run *sensor.MsgToCompliance_TriggerRun) error {
	complianceData := gatherData(scrapeConfig, run.GetScrapeId())
	complianceData.Files = data.FlattenFileMap(complianceData.Files)
	results := getCheckResults(run, complianceData)

	return sendResults(results, client, run.GetScrapeId())
}

func getCheckResults(run *sensor.MsgToCompliance_TriggerRun, complianceData *standards.ComplianceData) map[string]*compliance.ComplianceStandardResult {
	results := make(map[string]*compliance.ComplianceStandardResult)
	for _, standardID := range run.GetStandardIds() {
		standard, ok := standards.NodeChecks[standardID]
		if !ok {
			log.Infof("no checks found for standard %s during compliance run %s", standardID, run.GetScrapeId())
			continue
		}
		for checkName, checkAndMetadata := range standard {
			if checkAndMetadata.CheckFunc == nil {
				log.Infof("no check function found for check %s in standard %s during compliance run %s", checkName, standardID, run.GetScrapeId())
				continue
			}
			evidence := checkAndMetadata.CheckFunc(complianceData)
			addCheckResultsToResponse(results, standardID, checkName, checkAndMetadata.Metadata.TargetKind, evidence)
		}
	}
	return results
}

func addCheckResultsToResponse(results map[string]*compliance.ComplianceStandardResult, standardID, checkName string, target framework.TargetKind, evidence []*storage.ComplianceResultValue_Evidence) {
	standardResults, ok := results[standardID]
	if !ok {
		standardResults = &compliance.ComplianceStandardResult{
			NodeCheckResults:    make(map[string]*storage.ComplianceResultValue),
			ClusterCheckResults: make(map[string]*storage.ComplianceResultValue),
		}
		results[standardID] = standardResults
	}

	overallState := storage.ComplianceState_COMPLIANCE_STATE_UNKNOWN
	for _, result := range evidence {
		if result.GetState() > overallState {
			overallState = result.GetState()
		}
	}

	resultValue := &storage.ComplianceResultValue{
		Evidence:     evidence,
		OverallState: overallState,
	}

	switch target {
	case framework.NodeKind:
		standardResults.NodeCheckResults[checkName] = resultValue
	case framework.ClusterKind:
		standardResults.ClusterCheckResults[checkName] = resultValue
	}
}

func sendResults(results map[string]*compliance.ComplianceStandardResult, client sensor.ComplianceService_CommunicateClient, runID string) error {
	compressedResults, err := compressResults(results)
	if err != nil {
		return err
	}

	return client.Send(&sensor.MsgFromCompliance{
		Node: getNode(),
		Msg: &sensor.MsgFromCompliance_Return{
			Return: &compliance.ComplianceReturn{
				NodeName: getNode(),
				ScrapeId: runID,
				Time:     types.TimestampNow(),
				Evidence: compressedResults,
			},
		},
	})
}
