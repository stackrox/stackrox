package updater

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/activecomponent/converter"
	activeComponent "github.com/stackrox/rox/central/activecomponent/datastore"
	"github.com/stackrox/rox/central/activecomponent/updater/aggregator"
	deploymentStore "github.com/stackrox/rox/central/deployment/datastore"
	imageStore "github.com/stackrox/rox/central/image/datastore"
	processIndicatorStore "github.com/stackrox/rox/central/processindicator/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/simplecache"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	log = logging.LoggerForModule()

	updaterCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Deployment, resources.Image, resources.DeploymentExtension)))
)

type updaterImpl struct {
	acStore         activeComponent.DataStore
	deploymentStore deploymentStore.DataStore
	piStore         processIndicatorStore.DataStore
	imageStore      imageStore.DataStore

	aggregator      aggregator.ProcessAggregator // Aggregator for incoming process indicators
	executableCache simplecache.Cache            // Cache for image scan result
}

type imageExecutable struct {
	execToComponents map[string][]string
	scannerVersion   string
}

func clearExecutables(image *storage.Image) {
	for _, component := range image.GetScan().GetComponents() {
		component.Executables = nil
	}
}

// PopulateExecutableCache extracts executables from image scan and stores them in the executable cache.
// Image executables are cleared on successful return.
func (u *updaterImpl) PopulateExecutableCache(_ context.Context, image *storage.Image) error {
	if !features.ActiveVulnMgmt.Enabled() {
		return nil
	}
	imageID := image.GetId()
	scan := image.GetScan()
	if imageID == "" || scan == nil {
		log.Debugf("no valid scan, skip populating executable cache %s: %s", imageID, image.GetName())
		return nil
	}
	scannerVersion := scan.GetScannerVersion()

	// Check if we should update executable cache
	currRecord, ok := u.executableCache.Get(imageID)
	if ok && currRecord.(*imageExecutable).scannerVersion == scannerVersion {
		// Still clear executables even if cache has been pre-populated as it may be a re-scan
		clearExecutables(image)
		log.Debugf("Skip scan at scan version %s, current scan version (%s) has been populated for image %s: %s", scannerVersion, currRecord.(*imageExecutable).scannerVersion, imageID, image.GetName())
		return nil
	}

	// Create or update executable cache
	execToComponents := u.getExecToComponentsMap(scan)
	u.executableCache.Add(image.GetId(), &imageExecutable{execToComponents: execToComponents, scannerVersion: scannerVersion})

	log.Debugf("Executable cache updated for image %s of scan version %s with %d paths", image.GetId(), scannerVersion, len(execToComponents))

	return nil
}

func (u *updaterImpl) getExecToComponentsMap(imageScan *storage.ImageScan) map[string][]string {
	execToComponents := make(map[string][]string)

	for _, component := range imageScan.GetComponents() {
		// We do not support non-OS level active components at this time.
		if component.GetSource() != storage.SourceType_OS {
			continue
		}
		for _, exec := range component.GetExecutables() {
			execToComponents[exec.GetPath()] = append(execToComponents[exec.GetPath()], exec.GetDependencies()...)
		}
		// Remove the executables to save some memory. The same image won't be processed again.
		component.Executables = nil
	}
	return execToComponents
}

// Update detects active components with most recent process run.
func (u *updaterImpl) Update() {
	if !features.ActiveVulnMgmt.Enabled() {
		return
	}
	ctx := sac.WithAllAccess(context.Background())
	ids, err := u.deploymentStore.GetDeploymentIDs(ctx)
	if err != nil {
		log.Errorf("failed to fetch deployment ids: %v", err)
		return
	}
	deploymentToUpdates := u.aggregator.GetAndPrune(u.imageScanned, set.NewStringSet(ids...))
	if err := u.updateActiveComponents(deploymentToUpdates); err != nil {
		log.Errorf("failed to update active components: %v", err)
	}

	if err := u.pruneExecutableCache(); err != nil {
		log.Errorf("Error pruning active component executable cache: %v", err)
	}
}

func (u *updaterImpl) imageScanned(imageID string) bool {
	_, ok := u.executableCache.Get(imageID)
	return ok
}

func (u *updaterImpl) updateActiveComponents(deploymentToUpdates map[string][]*aggregator.ProcessUpdate) error {
	for deploymentID, updates := range deploymentToUpdates {
		err := u.updateForDeployment(updaterCtx, deploymentID, updates)
		if err != nil {
			return errors.Wrapf(err, "failed to update active components for deployment %s", deploymentID)
		}
	}
	return nil
}

// updateForDeployment detects and updates active components for a deployment
func (u *updaterImpl) updateForDeployment(ctx context.Context, deploymentID string, updates []*aggregator.ProcessUpdate) error {
	idToContainers := make(map[string]map[string]*storage.ActiveComponent_ActiveContext)
	containersToRemove := set.NewStringSet()
	for _, update := range updates {
		if update.ToBeRemoved() {
			containersToRemove.Add(update.ContainerName)
			continue
		}

		if update.FromDatabase() {
			containersToRemove.Add(update.ContainerName)
		}

		result, ok := u.executableCache.Get(update.ImageID)
		if !ok {
			utils.Should(errors.New("cannot find image scan"))
			continue
		}
		execToComponents := result.(*imageExecutable).execToComponents
		execPaths, err := u.getActiveExecPath(deploymentID, update)
		if err != nil {
			return errors.Wrapf(err, "failed to get active executables for deployment %s container %s", deploymentID, update.ContainerName)
		}

		activeContext := &storage.ActiveComponent_ActiveContext{ContainerName: update.ContainerName, ImageId: update.ImageID}
		for _, execPath := range execPaths.AsSlice() {
			componentIDs, ok := execToComponents[execPath]
			if !ok {
				continue
			}
			for _, componentID := range componentIDs {
				id := converter.ComposeID(deploymentID, componentID)
				var containerNameSet map[string]*storage.ActiveComponent_ActiveContext
				if containerNameSet, ok = idToContainers[id]; !ok {
					containerNameSet = make(map[string]*storage.ActiveComponent_ActiveContext)
					idToContainers[id] = containerNameSet
				}
				if _, ok = containerNameSet[update.ContainerName]; !ok {
					containerNameSet[update.ContainerName] = activeContext
				}
			}
		}
	}
	return u.createActiveComponentsAndUpdateDb(ctx, deploymentID, idToContainers, containersToRemove)
}

func (u *updaterImpl) createActiveComponentsAndUpdateDb(ctx context.Context, deploymentID string, acToContexts map[string]map[string]*storage.ActiveComponent_ActiveContext, contextsToRemove set.StringSet) error {
	var err error
	var existingAcs []*storage.ActiveComponent
	if contextsToRemove.Cardinality() == 0 {
		var ids []string
		for id := range acToContexts {
			ids = append(ids, id)
		}
		existingAcs, err = u.acStore.GetBatch(ctx, ids)
	} else {
		// Need to check all active components in case there are containers to remove
		query := search.NewQueryBuilder().AddExactMatches(search.DeploymentID, deploymentID).ProtoQuery()
		existingAcs, err = u.acStore.SearchRawActiveComponents(ctx, query)
	}
	if err != nil {
		return err
	}

	var acToRemove []string
	var activeComponents []*storage.ActiveComponent
	for _, ac := range existingAcs {
		updateAc, shouldRemove := merge(ac, contextsToRemove, acToContexts[ac.GetId()])
		if updateAc != nil {
			activeComponents = append(activeComponents, updateAc)
		}
		if shouldRemove {
			acToRemove = append(acToRemove, ac.GetId())
		}
		delete(acToContexts, ac.GetId())
	}
	for id, activeContexts := range acToContexts {
		_, componentID, err := converter.DecomposeID(id)
		if err != nil {
			utils.Should(err)
			continue
		}
		newAc := &storage.ActiveComponent{
			Id:                  id,
			DeploymentId:        deploymentID,
			ComponentId:         componentID,
			ActiveContextsSlice: converter.ConvertActiveContextsMapToSlice(activeContexts),
		}
		activeComponents = append(activeComponents, newAc)
	}
	log.Debugf("Upserting %d active components and deleting %d for deployment %s", len(activeComponents), len(acToRemove), deploymentID)
	if len(activeComponents) > 0 {
		err = u.acStore.UpsertBatch(ctx, activeComponents)
		if err != nil {
			return errors.Wrapf(err, "failed to upsert %d activeComponents", len(activeComponents))
		}
	}
	if len(acToRemove) > 0 {
		err = u.acStore.DeleteBatch(ctx, acToRemove...)
	}
	return err
}

// merge existing active component with new contexts, addend could be nil
func merge(base *storage.ActiveComponent, subtrahend set.StringSet, addend map[string]*storage.ActiveComponent_ActiveContext) (*storage.ActiveComponent, bool) {
	// Only remove the containers that won't be added back.
	toRemove := set.NewStringSet()
	for sub := range subtrahend {
		if _, ok := addend[sub]; !ok {
			toRemove.Add(sub)
		}
	}

	contexts := make(map[string]*storage.ActiveComponent_ActiveContext)
	for _, activeContext := range base.GetActiveContextsSlice() {
		contexts[activeContext.ContainerName] = activeContext
	}

	var changed bool
	for activeContext := range contexts {
		if toRemove.Contains(activeContext) {
			delete(contexts, activeContext)
			changed = true
		}
	}

	for containerName, activeContext := range addend {
		if baseContext, ok := contexts[containerName]; !ok || baseContext.ImageId != activeContext.ImageId {
			contexts[containerName] = activeContext
			changed = true
		}
	}

	base.ActiveContextsSlice = converter.ConvertActiveContextsMapToSlice(contexts)

	if len(contexts) == 0 {
		return nil, true
	}
	if !changed {
		return nil, false
	}
	return base, false
}

func (u *updaterImpl) getActiveExecPath(deploymentID string, update *aggregator.ProcessUpdate) (set.StringSet, error) {
	if update.FromCache() {
		return update.NewPaths, nil
	}
	containerName := update.ContainerName
	query := search.NewQueryBuilder().AddExactMatches(search.DeploymentID, deploymentID).AddExactMatches(search.ContainerName, containerName).ProtoQuery()
	pis, err := u.piStore.SearchRawProcessIndicators(updaterCtx, query)
	if err != nil {
		return nil, err
	}
	execSet := set.NewStringSet()
	for _, pi := range pis {
		if update.ImageID != pi.GetImageId() {
			continue
		}
		execSet.Add(pi.GetSignal().GetExecFilePath())
	}
	log.Debugf("Active executables for %s:%s: %v", deploymentID, containerName, execSet)
	return execSet, nil
}

func (u *updaterImpl) pruneExecutableCache() error {
	results, err := u.imageStore.Search(updaterCtx, search.EmptyQuery())
	if err != nil {
		return err
	}
	imageIDs := search.ResultsToIDSet(results)

	for _, entry := range u.executableCache.Keys() {
		imageID := entry.(string)
		if !imageIDs.Contains(imageID) {
			u.executableCache.Remove(imageID)
		}
	}
	return nil
}
