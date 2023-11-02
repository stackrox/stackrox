package compliance

import (
	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/compliance/collection/command"
	"github.com/stackrox/rox/compliance/collection/containerruntimes/crio"
	"github.com/stackrox/rox/compliance/collection/file"
	"github.com/stackrox/rox/compliance/collection/kubernetes/collection/kubelet"
	"github.com/stackrox/rox/generated/internalapi/compliance"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	_ "github.com/stackrox/rox/pkg/compliance/checks" // Make sure all checks are available
	"github.com/stackrox/rox/pkg/compliance/checks/standards"
	"github.com/stackrox/rox/pkg/compliance/data"
	"github.com/stackrox/rox/pkg/compliance/framework"
	"github.com/stackrox/rox/pkg/logging"
)

var log = logging.LoggerForModule()

func runChecks(client sensor.ComplianceService_CommunicateClient,
	scrapeConfig *sensor.MsgToCompliance_ScrapeConfig,
	run *sensor.MsgToCompliance_TriggerRun,
	nodeNameProvider NodeNameProvider,
) error {
	complianceData := gatherData(scrapeConfig, run.GetScrapeId(), nodeNameProvider)
	complianceData.Files = data.FlattenFileMap(complianceData.Files)
	results := getCheckResults(run, scrapeConfig, complianceData)

	return sendResults(results, client, run.GetScrapeId(), nodeNameProvider)
}

func getCheckResults(run *sensor.MsgToCompliance_TriggerRun, _ *sensor.MsgToCompliance_ScrapeConfig, complianceData *standards.ComplianceData) map[string]*compliance.ComplianceStandardResult {
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

func sendResults(results map[string]*compliance.ComplianceStandardResult,
	client sensor.ComplianceService_CommunicateClient, runID string, nodeNameProvider NodeNameProvider) error {
	compressedResults, err := compressResults(results)
	if err != nil {
		return err
	}

	return client.Send(&sensor.MsgFromCompliance{
		Node: nodeNameProvider.GetNodeName(),
		Msg: &sensor.MsgFromCompliance_Return{
			Return: &compliance.ComplianceReturn{
				NodeName: nodeNameProvider.GetNodeName(),
				ScrapeId: runID,
				Time:     types.TimestampNow(),
				Evidence: compressedResults,
			},
		},
	})
}

func gatherData(scrapeConfig *sensor.MsgToCompliance_ScrapeConfig,
	scrapeID string, nodeNameProvider NodeNameProvider) *standards.ComplianceData {
	complianceData := &standards.ComplianceData{
		NodeName: nodeNameProvider.GetNodeName(),
	}

	log.Infof("Running compliance scrape %q for node %q", scrapeID, nodeNameProvider.GetNodeName())

	var err error
	log.Infof("Container runtime is %v", scrapeConfig.GetContainerRuntime())
	if scrapeConfig.GetContainerRuntime() == storage.ContainerRuntime_CRIO_CONTAINER_RUNTIME {
		log.Info("Collecting relevant CRI-O data")
		complianceData.ContainerRuntimeInfo, err = crio.GetContainerRuntimeData()
		if err != nil {
			log.Errorf("Collecting CRI-O data failed: %v", err)
		} else {
			log.Info("Successfully collected relevant CRI-O data")
		}
	} else {
		log.Info("Unknown container runtime, not collecting any data ...")
	}

	log.Info("Starting to collect systemd files")
	complianceData.SystemdFiles, err = file.CollectSystemdFiles()
	if err != nil {
		log.Errorf("Collecting systemd files failed: %v", err)
	}
	log.Info("Successfully collected relevant systemd files")

	log.Info("Starting to collect configuration files")
	complianceData.Files, err = file.CollectFiles()
	if err != nil {
		log.Errorf("Collecting configuration files failed: %v", err)
	}
	log.Info("Successfully collected relevant configuration files")

	log.Info("Starting to collect command lines")
	complianceData.CommandLines, err = command.RetrieveCommands()
	if err != nil {
		log.Errorf("Collecting command lines failed: %v", err)
	}
	log.Info("Successfully collected relevant command lines")

	complianceData.IsMasterNode = scrapeConfig.GetIsMasterNode()

	complianceData.KubeletConfiguration, err = kubelet.GatherKubelet()
	if err != nil {
		log.Errorf("collecting kubelet configuration failed: %v", err)
	}

	complianceData.Time = types.TimestampNow()
	return complianceData
}
