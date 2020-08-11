package manager

import (
	"fmt"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/central/compliance/framework"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
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
	nodeMap := make(map[string]*storage.Node, len(nodes))
	for _, node := range nodes {
		nodeMap[node.GetId()] = node
	}

	deployments := framework.Deployments(domain)
	deploymentMap := make(map[string]*storage.ComplianceDeployment, len(deployments))
	for _, deployment := range deployments {
		deploymentMap[deployment.GetId()] = makeComplianceDeployment(deployment)
	}

	return &storage.ComplianceDomain{
		Cluster:     domain.Cluster().Cluster(),
		Nodes:       nodeMap,
		Deployments: deploymentMap,
	}
}

func makeComplianceDeployment(deployment *storage.Deployment) *storage.ComplianceDeployment {
	return &storage.ComplianceDeployment{
		Id:          deployment.GetId(),
		Name:        deployment.GetName(),
		Hash:        deployment.GetHash(),
		Type:        deployment.GetType(),
		Namespace:   deployment.GetNamespace(),
		NamespaceId: deployment.GetNamespaceId(),
		Labels:      deployment.GetLabels(),
		PodLabels:   deployment.GetPodLabels(),
		ClusterId:   deployment.GetClusterId(),
		ClusterName: deployment.GetClusterName(),
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

func getResultValueProto(entityResults framework.Results, errors []error) *storage.ComplianceResultValue {
	var evidenceList []*storage.ComplianceResultValue_Evidence

	if entityResults != nil {
		for _, evidence := range entityResults.Evidence() {
			evidenceList = append(evidenceList, getEvidenceProto(evidence))
		}
	}

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

func collectEntityResults(entity framework.ComplianceTarget, checks []framework.Check, allResults map[string]framework.Results) *storage.ComplianceRunResults_EntityResults {
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

		controlResults[check.ID()] = getResultValueProto(results, errs)
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
	}
}

func (r *runInstance) collectResults(run framework.ComplianceRun) *storage.ComplianceRunResults {
	domainProto := getDomainProto(r.domain)

	allResults := run.GetAllResults()
	checks := run.GetChecks()
	clusterResults := collectEntityResults(r.domain.Cluster(), checks, allResults)

	nodeResults := make(map[string]*storage.ComplianceRunResults_EntityResults)
	for _, node := range r.domain.Nodes() {
		nodeResults[node.ID()] = collectEntityResults(node, checks, allResults)
	}

	deploymentResults := make(map[string]*storage.ComplianceRunResults_EntityResults)
	for _, deployment := range r.domain.Deployments() {
		deploymentResults[deployment.ID()] = collectEntityResults(deployment, checks, allResults)
	}

	runMetadataProto := r.metadataProto(true)
	// need to mark this explicitly
	runMetadataProto.Success = true

	return &storage.ComplianceRunResults{
		Domain:            domainProto,
		RunMetadata:       runMetadataProto,
		ClusterResults:    clusterResults,
		NodeResults:       nodeResults,
		DeploymentResults: deploymentResults,
	}
}
