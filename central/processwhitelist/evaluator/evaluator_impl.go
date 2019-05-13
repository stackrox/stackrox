package evaluator

import (
	"context"

	"github.com/pkg/errors"
	indicatorsStore "github.com/stackrox/rox/central/processindicator/datastore"
	"github.com/stackrox/rox/central/processwhitelist"
	whitelistsStore "github.com/stackrox/rox/central/processwhitelist/datastore"
	whitelistResultsStore "github.com/stackrox/rox/central/processwhitelistresults/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
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
	}

	for _, container := range deployment.GetContainers() {
		if whitelistStatus, ok := containerNameToWhitelistResults[container.GetName()]; ok {
			results.WhitelistStatuses = append(results.WhitelistStatuses, whitelistStatus)
		}
	}

	return e.whitelistResults.UpsertWhitelistResults(ctx, results)
}

func (e *evaluator) EvaluateWhitelistsAndPersistResult(deployment *storage.Deployment) (violatingProcesses []*storage.ProcessIndicator, err error) {
	ctx := context.TODO()

	containerNameToWhitelistedProcesses := make(map[string]set.StringSet)
	containerNameToWhitelistResults := make(map[string]*storage.ContainerNameAndWhitelistStatus)
	for _, container := range deployment.GetContainers() {
		whitelist, err := e.whitelists.GetProcessWhitelist(ctx, &storage.ProcessWhitelistKey{
			DeploymentId:  deployment.GetId(),
			ContainerName: container.GetName(),
		})
		if err != nil {
			return nil, errors.Wrapf(err, "fetching process whitelist for deployment %s/%s/%s", deployment.GetClusterName(), deployment.GetNamespace(), deployment.GetName())
		}
		containerNameToWhitelistResults[container.GetName()] = &storage.ContainerNameAndWhitelistStatus{
			ContainerName:   container.GetName(),
			WhitelistStatus: getWhitelistStatus(whitelist),
		}
		if whitelist == nil {
			continue
		}
		processSet := processwhitelist.Processes(whitelist, processwhitelist.RoxOrUserLocked)
		if processSet != nil {
			containerNameToWhitelistedProcesses[container.GetName()] = *processSet
		}

	}

	processes, err := e.indicators.SearchRawProcessIndicators(ctx, search.NewQueryBuilder().AddExactMatches(search.DeploymentID, deployment.GetId()).ProtoQuery())
	if err != nil {
		return nil, errors.Wrapf(err, "searching process indicators for deployment %s/%s/%s", deployment.GetClusterName(), deployment.GetNamespace(), deployment.GetName())
	}

	for _, process := range processes {
		processSet, exists := containerNameToWhitelistedProcesses[process.GetContainerName()]
		// If no explicit whitelist, then all processes are valid.
		if !exists {
			continue
		}
		if !processSet.Contains(processwhitelist.WhitelistItemFromProcess(process)) {
			violatingProcesses = append(violatingProcesses, process)
			containerNameToWhitelistResults[process.GetContainerName()].AnomalousProcessesExecuted = true
		}
	}
	if err := e.persistResults(ctx, deployment, containerNameToWhitelistResults); err != nil {
		return nil, errors.Wrap(err, "failed to persist whitelist results")
	}
	return
}
