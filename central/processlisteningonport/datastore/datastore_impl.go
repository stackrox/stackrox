package datastore

import (
	"context"
	"fmt"
	"time"

	"github.com/stackrox/rox/central/metrics"
	processIndicatorStore "github.com/stackrox/rox/central/processindicator/datastore"
	"github.com/stackrox/rox/central/processlisteningonport/store/postgres"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
)

type datastoreImpl struct {
	storage            postgres.Store
	indicatorDataStore processIndicatorStore.DataStore
}

var (
	plopSAC = sac.ForResource(resources.DeploymentExtension)
	log     = logging.LoggerForModule()
)

func newDatastoreImpl(
	storage postgres.Store,
	indicatorDataStore processIndicatorStore.DataStore,
) *datastoreImpl {
	return &datastoreImpl{
		storage:            storage,
		indicatorDataStore: indicatorDataStore,
	}
}

func (ds *datastoreImpl) AddProcessListeningOnPort(
	ctx context.Context,
	portProcesses ...*storage.ProcessListeningOnPort,
) error {
	defer metrics.SetDatastoreFunctionDuration(
		time.Now(),
		"ProcessListeningOnPort",
		"AddProcessListeningOnPort",
	)

	var (
		lookups       []*v1.Query
		indicatorsMap = map[string]*storage.ProcessIndicator{}
	)

	if !env.PostgresDatastoreEnabled.BooleanSetting() {
		// PLOP is a Postgres-only feature, do nothing.
		log.Warnf("Tried to add PLOP not on Postgres, ignore: %+v", portProcesses)
		return nil
	}

	if ok, err := plopSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	// First query all needed process indicators in one go
	for _, val := range portProcesses {
		if val.Process == nil {
			log.Warnf("Got PLOP object without Process information, ignore: %+v", val)
			continue
		}

		lookups = append(lookups,
			search.NewQueryBuilder().
				AddExactMatches(search.ContainerName, val.Process.ContainerName).
				AddExactMatches(search.PodID, val.Process.PodId).
				AddExactMatches(search.ProcessName, val.Process.ProcessName).
				AddExactMatches(search.ProcessArguments, val.Process.ProcessArgs).
				AddExactMatches(search.ProcessExecPath, val.Process.ProcessExecFilePath).
				ProtoQuery())
	}

	indicatorsQuery := search.DisjunctionQuery(lookups...)
	log.Debugf("Sending query: %s", indicatorsQuery.String())
	indicators, err := ds.indicatorDataStore.SearchRawProcessIndicators(ctx, indicatorsQuery)
	if err != nil {
		return err
	}

	for _, val := range indicators {
		key := fmt.Sprintf("%s %s %s %s %s",
			val.GetContainerName(),
			val.GetPodId(),
			val.GetSignal().GetName(),
			val.GetSignal().GetArgs(),
			val.GetSignal().GetExecFilePath(),
		)

		// A bit of paranoia is always good
		if old, ok := indicatorsMap[key]; ok {
			log.Warnf("An indicator %s is already present, overwrite with %s",
				old.GetId(), val.GetId())
		}

		indicatorsMap[key] = val
	}

	plopObjects := make([]*storage.ProcessListeningOnPortStorage, len(portProcesses))
	for i, val := range portProcesses {
		indicatorID := ""

		key := fmt.Sprintf("%s %s %s %s %s",
			val.Process.ContainerName,
			val.Process.PodId,
			val.Process.ProcessName,
			val.Process.ProcessArgs,
			val.Process.ProcessExecFilePath,
		)

		if indicator, ok := indicatorsMap[key]; ok {
			indicatorID = indicator.GetId()
			log.Debugf("Got indicator %s: %+v", indicatorID, indicator)
		} else {
			// XXX: Create a metric for this
			log.Warnf("Found no matching indicators for %s", key)
		}

		plopObjects[i] = &storage.ProcessListeningOnPortStorage{
			// XXX, ResignatingFacepalm: Use regular GENERATE ALWAYS AS
			// IDENTITY, which would require changes in store generator
			Id:                 uuid.NewV4().String(),
			Port:               val.Port,
			Protocol:           val.Protocol,
			ProcessIndicatorId: indicatorID,
			CloseTimestamp:     val.CloseTimestamp,
		}
	}

	// Now save actual PLOP objects
	err = ds.storage.UpsertMany(ctx, plopObjects)
	if err != nil {
		return err
	}

	return nil
}

func (ds *datastoreImpl) GetProcessListeningOnPort(
	ctx context.Context,
	opts GetOptions,
) (
	portProcessMap map[string][]*storage.ProcessListeningOnPort, err error,
) {
	if ok, err := plopSAC.ReadAllowed(ctx); err != nil {
		return nil, err
	} else if !ok {
		return nil, sac.ErrResourceAccessDenied
	}

	if opts.Namespace != nil && opts.DeploymentID != nil {
		portProcessMap, err = ds.storage.GetProcessListeningOnPort(ctx,
			*opts.Namespace, *opts.DeploymentID)
	} else if opts.Namespace != nil {
		portProcessMap, err = ds.storage.GetProcessListeningOnPortByNamespace(
			ctx, *opts.Namespace)
	} else {
		log.Warnf("Options for read query are incorrect: %+v", opts)
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return portProcessMap, nil
}
