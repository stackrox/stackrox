package evaluator

import (
	"context"

	"github.com/pkg/errors"
	indicatorsStore "github.com/stackrox/rox/central/processindicator/datastore"
	"github.com/stackrox/rox/central/processwhitelist"
	whitelistsStore "github.com/stackrox/rox/central/processwhitelist/datastore"
	whitelistResultsStore "github.com/stackrox/rox/central/processwhitelistresults/datastore"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	processWhitelistPkg "github.com/stackrox/rox/pkg/processwhitelist"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
)

var (
	evaluatorCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Deployment, resources.ProcessWhitelist, resources.Indicator)))
)

type evaluator struct {
	indicators       indicatorsStore.DataStore
	whitelists       whitelistsStore.DataStore
	whitelistResults whitelistResultsStore.DataStore
}

func getWhitelistStatus(whitelist *storage.ProcessWhitelist) storage.ContainerNameAndWhitelistStatus_WhitelistStatus {
	if whitelist == nil {
		return storage.ContainerNameAndWhitelistStatus_NOT_GENERATED
	}
	if processwhitelist.LockedUnderMode(whitelist, processwhitelist.RoxOrUserLocked) {
		return storage.ContainerNameAndWhitelistStatus_LOCKED
	}
	return storage.ContainerNameAndWhitelistStatus_UNLOCKED
}

func (e *evaluator) persistResults(ctx context.Context, deployment *storage.Deployment, containerNameToWhitelistResults map[string]*storage.ContainerNameAndWhitelistStatus) error {
	results := &storage.ProcessWhitelistResults{
		DeploymentId: deployment.GetId(),
		ClusterId:    deployment.GetClusterId(),
		Namespace:    deployment.GetNamespace(),
	}

	for _, container := range deployment.GetContainers() {
		if whitelistStatus, ok := containerNameToWhitelistResults[container.GetName()]; ok {
			results.WhitelistStatuses = append(results.WhitelistStatuses, whitelistStatus)
		}
	}

	return e.whitelistResults.UpsertWhitelistResults(ctx, results)
}

func (e *evaluator) EvaluateWhitelistsAndPersistResult(deployment *storage.Deployment) (violatingProcesses []*storage.ProcessIndicator, err error) {
	containerNameToWhitelistedProcesses := make(map[string]*set.StringSet)
	containerNameToWhitelistResults := make(map[string]*storage.ContainerNameAndWhitelistStatus)
	for _, container := range deployment.GetContainers() {
		whitelist, exists, err := e.whitelists.GetProcessWhitelist(evaluatorCtx, &storage.ProcessWhitelistKey{
			DeploymentId:  deployment.GetId(),
			ContainerName: container.GetName(),
			ClusterId:     deployment.GetClusterId(),
			Namespace:     deployment.GetNamespace(),
		})
		if err != nil {
			return nil, errors.Wrapf(err, "fetching process whitelist for deployment %s/%s/%s", deployment.GetClusterName(), deployment.GetNamespace(), deployment.GetName())
		}
		containerNameToWhitelistResults[container.GetName()] = &storage.ContainerNameAndWhitelistStatus{
			ContainerName:   container.GetName(),
			WhitelistStatus: getWhitelistStatus(whitelist),
		}
		if !exists {
			continue
		}
		processSet := processwhitelist.Processes(whitelist, processwhitelist.RoxOrUserLocked)
		if processSet != nil {
			containerNameToWhitelistedProcesses[container.GetName()] = processSet
		}

	}

	processes, err := e.indicators.SearchRawProcessIndicators(evaluatorCtx, search.NewQueryBuilder().AddExactMatches(search.DeploymentID, deployment.GetId()).ProtoQuery())
	if err != nil {
		return nil, errors.Wrapf(err, "searching process indicators for deployment %s/%s/%s", deployment.GetClusterName(), deployment.GetNamespace(), deployment.GetName())
	}

	for _, process := range processes {
		processSet, exists := containerNameToWhitelistedProcesses[process.GetContainerName()]
		// If no explicit whitelist, then all processes are valid.
		if !exists {
			continue
		}
		whitelistItem := processWhitelistPkg.WhitelistItemFromProcess(process)
		if whitelistItem == "" {
			continue
		}
		if processwhitelist.IsStartupProcess(process) {
			continue
		}
		if !processSet.Contains(processWhitelistPkg.WhitelistItemFromProcess(process)) {
			violatingProcesses = append(violatingProcesses, process)
			containerNameToWhitelistResults[process.GetContainerName()].AnomalousProcessesExecuted = true
		}
	}

	whitelistResults, err := e.whitelistResults.GetWhitelistResults(evaluatorCtx, deployment.GetId())
	if err != nil {
		return nil, errors.Wrap(err, "error fetching whitelist results")
	}

	var persistenceRequired bool
	if len(whitelistResults.GetWhitelistStatuses()) != len(containerNameToWhitelistResults) {
		persistenceRequired = true
	} else {
		for _, status := range whitelistResults.GetWhitelistStatuses() {
			newStatus := containerNameToWhitelistResults[status.GetContainerName()]
			if newStatus == nil {
				persistenceRequired = true
				break
			}
			if status.GetAnomalousProcessesExecuted() != newStatus.GetAnomalousProcessesExecuted() ||
				status.GetWhitelistStatus() != newStatus.GetWhitelistStatus() {
				persistenceRequired = true
				break
			}
		}
	}
	if persistenceRequired {
		if err := e.persistResults(evaluatorCtx, deployment, containerNameToWhitelistResults); err != nil {
			return nil, errors.Wrap(err, "failed to persist whitelist results")
		}
	}
	return
}
