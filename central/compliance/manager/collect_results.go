package manager

import (
	"fmt"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/central/compliance/framework"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/internalapi/compliance"
	"github.com/stackrox/rox/generated/storage"
	pkgStandards "github.com/stackrox/rox/pkg/compliance/checks/standards"
	pkgFramework "github.com/stackrox/rox/pkg/compliance/framework"
)

var (
	statusToProtoStatus = map[framework.Status]storage.ComplianceState{
		framework.FailStatus: storage.ComplianceState_COMPLIANCE_STATE_FAILURE,
		framework.PassStatus: storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
		framework.SkipStatus: storage.ComplianceState_COMPLIANCE_STATE_SKIP,
		framework.NoteStatus: storage.ComplianceState_COMPLIANCE_STATE_NOTE,
	}
)

func getDomainProto(domain framework.ComplianceDomain) *storage.ComplianceDomain {
	nodes := framework.Nodes(domain)
	nodeMap := make(map[string]*storage.ComplianceDomain_Node, len(nodes))
	for _, node := range nodes {
		nodeMap[node.GetId()] = convertNode(node)
	}

	deployments := framework.Deployments(domain)
	deploymentMap := make(map[string]*storage.ComplianceDomain_Deployment, len(deployments))
	for _, deployment := range deployments {
		deploymentMap[deployment.GetId()] = convertDeployment(deployment)
	}

	return &storage.ComplianceDomain{
		Id:          domain.ID(),
		Cluster:     convertCluster(domain.Cluster().Cluster()),
		Nodes:       nodeMap,
		Deployments: deploymentMap,
	}
}

func convertCluster(cluster *storage.Cluster) *storage.ComplianceDomain_Cluster {
	return &storage.ComplianceDomain_Cluster{
		Id:   cluster.GetId(),
		Name: cluster.GetName(),
	}
}

func convertDeployment(dep *storage.Deployment) *storage.ComplianceDomain_Deployment {
	return &storage.ComplianceDomain_Deployment{
		Id:          dep.GetId(),
		NamespaceId: dep.GetNamespaceId(),
		Name:        dep.GetName(),
		Type:        dep.GetType(),
		Namespace:   dep.GetNamespace(),
		ClusterId:   dep.GetClusterId(),
		ClusterName: dep.GetClusterName(),
	}
}

func convertNode(node *storage.Node) *storage.ComplianceDomain_Node {
	return &storage.ComplianceDomain_Node{
		Id:          node.GetId(),
		Name:        node.GetName(),
		ClusterId:   node.GetClusterId(),
		ClusterName: node.GetClusterName(),
	}
}

func getEvidenceProto(evidence framework.EvidenceRecord) *storage.ComplianceResultValue_Evidence {
	msg := evidence.Message
	protoStatus, validStatus := statusToProtoStatus[evidence.Status]
	if !validStatus {
		protoStatus = storage.ComplianceState_COMPLIANCE_STATE_ERROR
		msg = fmt.Sprintf("[unknown control status %v] %s", evidence.Status, msg)
	}
	return &storage.ComplianceResultValue_Evidence{
		State:   protoStatus,
		Message: msg,
	}
}

func getResultValueProto(entityResults framework.Results, remoteResults *storage.ComplianceResultValue, errors []error) *storage.ComplianceResultValue {
	var evidenceList []*storage.ComplianceResultValue_Evidence

	if entityResults != nil {
		for _, evidence := range entityResults.Evidence() {
			if evidence.Status == framework.InternalSkipStatus {
				return nil
			}
			evidenceList = append(evidenceList, getEvidenceProto(evidence))
		}
	}

	evidenceList = append(evidenceList, remoteResults.GetEvidence()...)

	for _, err := range errors {
		evidenceList = append(evidenceList, &storage.ComplianceResultValue_Evidence{
			State:   storage.ComplianceState_COMPLIANCE_STATE_ERROR,
			Message: err.Error(),
		})
	}

	overallStatus := storage.ComplianceState_COMPLIANCE_STATE_UNKNOWN
	for _, evidence := range evidenceList {
		if evidence.GetState() > overallStatus {
			overallStatus = evidence.GetState()
		}
	}

	if overallStatus == storage.ComplianceState_COMPLIANCE_STATE_UNKNOWN {
		evidenceList = append(evidenceList, &storage.ComplianceResultValue_Evidence{
			State:   storage.ComplianceState_COMPLIANCE_STATE_ERROR,
			Message: "compliance run reported no results for this entity/control combination",
		})
		overallStatus = storage.ComplianceState_COMPLIANCE_STATE_ERROR
	}

	return &storage.ComplianceResultValue{
		Evidence:     evidenceList,
		OverallState: overallStatus,
	}
}

func collectEntityResults(entity framework.ComplianceTarget, checks []framework.Check, allResults map[string]framework.Results, allRemoteResults map[string]*storage.ComplianceResultValue) *storage.ComplianceRunResults_EntityResults {
	controlResults := make(map[string]*storage.ComplianceResultValue)
	for _, check := range checks {
		if !check.AppliesToScope(entity.Kind()) {
			continue
		}

		var errs []error
		results := allResults[check.ID()]
		if results != nil && results.Error() != nil {
			errs = append(errs, results.Error())
		}
		if results != nil && entity.Kind() != pkgFramework.ClusterKind {
			results = results.ForChild(entity)
			if results != nil && results.Error() != nil {
				errs = append(errs, results.Error())
			}
		}

		remoteResults := allRemoteResults[check.ID()]

		if result := getResultValueProto(results, remoteResults, errs); result != nil {
			controlResults[check.ID()] = result
		}
	}

	return &storage.ComplianceRunResults_EntityResults{
		ControlResults: controlResults,
	}
}

func (r *runInstance) metadataProto(fixTimestamps bool) *storage.ComplianceRunMetadata {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var startTS, finishTS *types.Timestamp
	var err error
	if !r.startTime.IsZero() {
		startTS, err = types.TimestampProto(r.startTime)
		if err != nil {
			log.Errorf("could not convert compliance run start timestamp to proto: %v", err)
		}
	}

	if !r.finishTime.IsZero() {
		finishTS, err = types.TimestampProto(r.finishTime)
		if err != nil {
			log.Errorf("could not convert compliance run finish timestamp to proto: %v", err)
		}
	}

	if fixTimestamps {
		if startTS == nil {
			startTS = types.TimestampNow()
		}
		if finishTS == nil {
			finishTS = types.TimestampNow()
		}
	}
	var errMsg string
	if r.err != nil {
		errMsg = r.err.Error()
	}

	return &storage.ComplianceRunMetadata{
		RunId:           r.id,
		ClusterId:       r.domain.Cluster().Cluster().GetId(),
		StandardId:      r.standard.Standard.ID,
		StartTimestamp:  startTS,
		FinishTimestamp: finishTS,
		Success:         r.status == v1.ComplianceRun_FINISHED && r.err == nil,
		ErrorMessage:    errMsg,
		DomainId:        r.domain.ID(),
	}
}

func (r *runInstance) collectResults(run framework.ComplianceRun, remoteResults map[string]map[string]*compliance.ComplianceStandardResult) *storage.ComplianceRunResults {
	remoteClusterResults, remoteNodeResults := r.foldRemoteResults(remoteResults)

	allResults := run.GetAllResults()
	checks := run.GetChecks()
	clusterResults := collectEntityResults(r.domain.Cluster(), checks, allResults, remoteClusterResults)

	nodeResults := make(map[string]*storage.ComplianceRunResults_EntityResults)
	for _, node := range r.domain.Nodes() {
		nodeResults[node.ID()] = collectEntityResults(node, checks, allResults, remoteNodeResults[node.ID()])
	}

	deploymentResults := make(map[string]*storage.ComplianceRunResults_EntityResults)
	for _, deployment := range r.domain.Deployments() {
		deploymentResults[deployment.ID()] = collectEntityResults(deployment, checks, allResults, nil)
	}

	machineConfigResults := make(map[string]*storage.ComplianceRunResults_EntityResults)
	for _, mc := range r.domain.MachineConfigs()[r.standard.Name] {
		if results := collectEntityResults(mc, checks, allResults, nil); len(results.GetControlResults()) != 0 {
			machineConfigResults[mc.ID()] = results
		}
	}

	runMetadataProto := r.metadataProto(true)
	// need to mark this explicitly
	runMetadataProto.Success = true

	return &storage.ComplianceRunResults{
		RunMetadata:          runMetadataProto,
		ClusterResults:       clusterResults,
		NodeResults:          nodeResults,
		DeploymentResults:    deploymentResults,
		MachineConfigResults: machineConfigResults,
	}
}

func (r *runInstance) foldRemoteResults(remoteResults map[string]map[string]*compliance.ComplianceStandardResult) (map[string]*storage.ComplianceResultValue, map[string]map[string]*storage.ComplianceResultValue) {
	nodeResults := make(map[string]map[string]*storage.ComplianceResultValue)
	clusterResults := make(map[string]*storage.ComplianceResultValue)

	for _, node := range r.domain.Nodes() {
		standardResults := r.getStandardResults(node.Node().GetName(), remoteResults)
		if standardResults == nil {
			continue
		}

		// Merge the cluster-level results into a single map of check ID -> check result
		mergeComplianceResultValue(clusterResults, standardResults.GetClusterCheckResults())

		// Fold in each of the node-level results individually
		nodeResults[node.ID()] = standardResults.NodeCheckResults
	}
	// Add notes for any missing cluster-level checks
	r.noteMissingNodeClusterChecks(clusterResults)

	return clusterResults, nodeResults
}

func mergeComplianceResultValue(destination, source map[string]*storage.ComplianceResultValue) {
	for checkName, sourceComplianceResult := range source {
		destinationComplianceResult, ok := destination[checkName]
		if !ok {
			destination[checkName] = sourceComplianceResult
			continue
		}
		destinationComplianceResult.Evidence = append(destinationComplianceResult.GetEvidence(), sourceComplianceResult.GetEvidence()...)
		if sourceComplianceResult.GetOverallState() > destinationComplianceResult.GetOverallState() {
			destinationComplianceResult.OverallState = sourceComplianceResult.GetOverallState()
		}
	}
}

func (r *runInstance) getStandardResults(nodeName string, nodeResults map[string]map[string]*compliance.ComplianceStandardResult) *compliance.ComplianceStandardResult {
	perStandardNodeResults, ok := nodeResults[nodeName]
	if !ok {
		return nil
	}

	standardResults, ok := perStandardNodeResults[r.standard.ID]
	if !ok {
		log.Infof("no check results received from node %s for compliance standard %s", nodeName, r.standard.ID)
		return nil
	}
	return standardResults
}

func (r *runInstance) noteMissingNodeClusterChecks(clusterResults map[string]*storage.ComplianceResultValue) {
	standard, ok := pkgStandards.NodeChecks[r.standard.ID]
	if !ok {
		return
	}

	for checkName, checkAndMetadata := range standard {
		if checkAndMetadata.Metadata.TargetKind != pkgFramework.ClusterKind {
			continue
		}

		// Only assign a value to a nil clusterResults after we know there is supposed to be evidence
		if clusterResults == nil {
			clusterResults = map[string]*storage.ComplianceResultValue{}
		}

		if evidence, ok := clusterResults[checkName]; !ok || len(evidence.GetEvidence()) == 0 {
			clusterResults[checkName] = &storage.ComplianceResultValue{
				Evidence: []*storage.ComplianceResultValue_Evidence{
					{
						State:   storage.ComplianceState_COMPLIANCE_STATE_NOTE,
						Message: "No evidence was received for this check. This can occur when using a managed Kubernetes service or if the compliance pods are not running on the master nodes.",
					},
				},
				OverallState: storage.ComplianceState_COMPLIANCE_STATE_NOTE,
			}
		}
	}
}
