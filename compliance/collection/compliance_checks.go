package main

import (
	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/internalapi/compliance"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	_ "github.com/stackrox/rox/pkg/compliance/checks" // Make sure all checks are available
	"github.com/stackrox/rox/pkg/compliance/checks/standards"
)

func runChecks(client sensor.ComplianceService_CommunicateClient, scrapeConfig *sensor.MsgToCompliance_ScrapeConfig, run *sensor.MsgToCompliance_TriggerRun) error {
	complianceData := gatherData(scrapeConfig, run.GetScrapeId())
	results := make(map[string]*compliance.ComplianceStandardResult)
	for _, standardID := range run.GetStandardIds() {
		standard, ok := standards.Standards[standardID]
		if !ok {
			log.Infof("no checks found for standard %s during compliance run %s", standardID, run.GetScrapeId())
			continue
		}
		for checkName, checkAndInterpretation := range standard {
			if checkAndInterpretation.CheckFunc == nil {
				log.Infof("no check function found for check %s in standard %s during compliance run %s", checkName, standardID, run.GetScrapeId())
				continue
			}
			evidence := checkAndInterpretation.CheckFunc(complianceData)
			addCheckResultsToResponse(results, standardID, checkName, evidence)
		}
	}

	return sendResults(results, client, run.GetScrapeId())
}

func addCheckResultsToResponse(results map[string]*compliance.ComplianceStandardResult, standardID, checkName string, evidence []*storage.ComplianceResultValue_Evidence) {
	standardResults, ok := results[standardID]
	if !ok {
		standardResults = &compliance.ComplianceStandardResult{
			CheckResults: make(map[string]*storage.ComplianceControlResult),
		}
		results[standardID] = standardResults
	}

	overallState := storage.ComplianceState_COMPLIANCE_STATE_UNKNOWN
	for _, result := range evidence {
		if result.GetState() > overallState {
			overallState = result.GetState()
		}
	}

	standardResults.CheckResults[checkName] = &storage.ComplianceControlResult{
		ControlId: checkName,
		Value: &storage.ComplianceResultValue{
			Evidence:     evidence,
			OverallState: overallState,
		},
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
