package evaluator

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/processbaseline"
	baselinesStore "github.com/stackrox/rox/central/processbaseline/datastore"
	baselineResultsStore "github.com/stackrox/rox/central/processbaselineresults/datastore"
	indicatorsStore "github.com/stackrox/rox/central/processindicator/datastore"
	"github.com/stackrox/rox/generated/storage"
	processBaselinePkg "github.com/stackrox/rox/pkg/processbaseline"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
)

var (
	evaluatorCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Deployment, resources.DeploymentExtension)))
)

type evaluator struct {
	indicators      indicatorsStore.DataStore
	baselines       baselinesStore.DataStore
	baselineResults baselineResultsStore.DataStore
}

func getBaselineStatus(baseline *storage.ProcessBaseline) storage.ContainerNameAndBaselineStatus_BaselineStatus {
	if baseline == nil {
		return storage.ContainerNameAndBaselineStatus_NOT_GENERATED
	}
	if processbaseline.LockedUnderMode(baseline, processbaseline.RoxOrUserLocked) {
		return storage.ContainerNameAndBaselineStatus_LOCKED
	}
	return storage.ContainerNameAndBaselineStatus_UNLOCKED
}

func (e *evaluator) persistResults(ctx context.Context, deployment *storage.Deployment, containerNameToBaselineResults map[string]*storage.ContainerNameAndBaselineStatus) error {
	results := &storage.ProcessBaselineResults{
		DeploymentId: deployment.GetId(),
		ClusterId:    deployment.GetClusterId(),
		Namespace:    deployment.GetNamespace(),
	}

	for _, container := range deployment.GetContainers() {
		if baselineStatus, ok := containerNameToBaselineResults[container.GetName()]; ok {
			results.BaselineStatuses = append(results.BaselineStatuses, baselineStatus)
		}
	}

	return e.baselineResults.UpsertBaselineResults(ctx, results)
}

func (e *evaluator) EvaluateBaselinesAndPersistResult(deployment *storage.Deployment) (violatingProcesses []*storage.ProcessIndicator, err error) {
	containerNameToBaselinedProcesses := make(map[string]*set.StringSet)
	containerNameToBaselineResults := make(map[string]*storage.ContainerNameAndBaselineStatus)

	var hasAtLeastOneLockedBaseline bool
	for _, container := range deployment.GetContainers() {
		baseline, exists, err := e.baselines.GetProcessBaseline(evaluatorCtx, &storage.ProcessBaselineKey{
			DeploymentId:  deployment.GetId(),
			ContainerName: container.GetName(),
			ClusterId:     deployment.GetClusterId(),
			Namespace:     deployment.GetNamespace(),
		})
		if err != nil {
			return nil, errors.Wrapf(err, "fetching process baseline for deployment %s/%s/%s", deployment.GetClusterName(), deployment.GetNamespace(), deployment.GetName())
		}
		baselineStatus := getBaselineStatus(baseline)
		if baselineStatus == storage.ContainerNameAndBaselineStatus_LOCKED {
			hasAtLeastOneLockedBaseline = true
		}
		containerNameToBaselineResults[container.GetName()] = &storage.ContainerNameAndBaselineStatus{
			ContainerName:  container.GetName(),
			BaselineStatus: getBaselineStatus(baseline),
		}
		if !exists {
			continue
		}
		processSet := processbaseline.Processes(baseline, processbaseline.RoxOrUserLocked)
		if processSet != nil {
			containerNameToBaselinedProcesses[container.GetName()] = processSet
		}
	}

	var processes []*storage.ProcessIndicator
	if hasAtLeastOneLockedBaseline {
		processes, err = e.indicators.SearchRawProcessIndicators(evaluatorCtx, search.NewQueryBuilder().AddExactMatches(search.DeploymentID, deployment.GetId()).ProtoQuery())
		if err != nil {
			return nil, errors.Wrapf(err, "searching process indicators for deployment %s/%s/%s", deployment.GetClusterName(), deployment.GetNamespace(), deployment.GetName())
		}
	}
	for _, process := range processes {
		processSet, exists := containerNameToBaselinedProcesses[process.GetContainerName()]
		// If no explicit baseline, then all processes are valid.
		if !exists {
			continue
		}
		baselineItem := processBaselinePkg.BaselineItemFromProcess(process)
		if baselineItem == "" {
			continue
		}
		if processbaseline.IsStartupProcess(process) {
			continue
		}
		if !processSet.Contains(processBaselinePkg.BaselineItemFromProcess(process)) {
			violatingProcesses = append(violatingProcesses, process)
			containerNameToBaselineResults[process.GetContainerName()].AnomalousProcessesExecuted = true
		}
	}

	baselineResults, err := e.baselineResults.GetBaselineResults(evaluatorCtx, deployment.GetId())
	if err != nil {
		return nil, errors.Wrap(err, "error fetching process baseline results")
	}

	var persistenceRequired bool
	if len(baselineResults.GetBaselineStatuses()) != len(containerNameToBaselineResults) {
		persistenceRequired = true
	} else {
		for _, status := range baselineResults.GetBaselineStatuses() {
			newStatus := containerNameToBaselineResults[status.GetContainerName()]
			if newStatus == nil {
				persistenceRequired = true
				break
			}
			if status.GetAnomalousProcessesExecuted() != newStatus.GetAnomalousProcessesExecuted() ||
				status.GetBaselineStatus() != newStatus.GetBaselineStatus() {
				persistenceRequired = true
				break
			}
		}
	}
	if persistenceRequired {
		if err := e.persistResults(evaluatorCtx, deployment, containerNameToBaselineResults); err != nil {
			return nil, errors.Wrap(err, "failed to persist process baseline results")
		}
	}
	return violatingProcesses, nil
}
