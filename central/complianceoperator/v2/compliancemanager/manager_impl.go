package compliancemanager

import (
	"context"
	"fmt"
	"strings"

	"github.com/adhocore/gronx"
	"github.com/pkg/errors"
	clusterDatastore "github.com/stackrox/rox/central/cluster/datastore"
	resultsDatastore "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore"
	compIntegration "github.com/stackrox/rox/central/complianceoperator/v2/integration/datastore"
	profileDatastore "github.com/stackrox/rox/central/complianceoperator/v2/profiles/datastore"
	compScanSetting "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore"
	scansDatastore "github.com/stackrox/rox/central/complianceoperator/v2/scans/datastore"
	ssbDatastore "github.com/stackrox/rox/central/complianceoperator/v2/scansettingbindings/datastore"
	"github.com/stackrox/rox/central/convert/internaltov2storage"
	"github.com/stackrox/rox/central/sensor/service/connection"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/protoconv/schedule"
	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/uuid"
)

var (
	complianceSAC = sac.ForResource(resources.Compliance)

	log = logging.LoggerForModule()
)

type clusterScan struct {
	clusterID string
	scanID    string
}

type managerImpl struct {
	sensorConnMgr connection.Manager
	integrationDS compIntegration.DataStore
	scanSettingDS compScanSetting.DataStore
	ssbDS         ssbDatastore.DataStore
	clusterDS     clusterDatastore.DataStore
	profileDS     profileDatastore.DataStore
	scansDS       scansDatastore.DataStore
	resultsDS     resultsDatastore.DataStore

	// Map used to correlate requests to a sensor with a response.  Each request will generate
	// a unique entry in the map
	runningRequests     map[string]clusterScan
	runningRequestsLock sync.Mutex
}

// New returns on instance of Manager interface that provides functionality to process compliance requests and forward them to Sensor.
func New(sensorConnMgr connection.Manager, integrationDS compIntegration.DataStore, scanSettingDS compScanSetting.DataStore, ssbDS ssbDatastore.DataStore, clusterDS clusterDatastore.DataStore, profileDS profileDatastore.DataStore, scansDS scansDatastore.DataStore, resultsDS resultsDatastore.DataStore) Manager {
	return &managerImpl{
		sensorConnMgr:   sensorConnMgr,
		integrationDS:   integrationDS,
		scanSettingDS:   scanSettingDS,
		ssbDS:           ssbDS,
		runningRequests: make(map[string]clusterScan),
		clusterDS:       clusterDS,
		profileDS:       profileDS,
		scansDS:         scansDS,
		resultsDS:       resultsDS,
	}
}

func (m *managerImpl) Sync(_ context.Context) {
	// TODO (ROX-18711): Sync scan configurations with sensor
}

// ProcessComplianceOperatorInfo processes and stores the compliance operator metadata coming from sensor
func (m *managerImpl) ProcessComplianceOperatorInfo(ctx context.Context, complianceIntegration *storage.ComplianceIntegration) error {
	if !features.ComplianceEnhancements.Enabled() {
		return errors.Errorf("Compliance is disabled. Cannot process request: %s", protoutils.NewWrapper(complianceIntegration))
	}

	// Check and see if we have this info already.
	existingIntegrations, err := m.integrationDS.GetComplianceIntegrations(ctx, search.NewQueryBuilder().
		AddExactMatches(search.ClusterID, complianceIntegration.GetClusterId()).ProtoQuery())
	if err != nil {
		return err
	}
	// TODO (ROX-18101):  Shouldn't happen once ROX-18101 is implemented.  Deferring more thorough handling
	// of this condition to that ticket.
	if len(existingIntegrations) > 1 {
		return errors.Errorf("multiple compliance operators for cluster %q exist.", complianceIntegration.GetClusterId())
	}

	// Not found so an add
	if len(existingIntegrations) == 0 {
		_, err := m.integrationDS.AddComplianceIntegration(ctx, complianceIntegration)
		return err
	}

	// An update, so we need the ID from the one that was returned
	id := existingIntegrations[0].GetId()
	complianceIntegration.Id = id

	return m.integrationDS.UpdateComplianceIntegration(ctx, complianceIntegration)
}

// ProcessScanRequest processes a request to apply a compliance scan configuration to one or more Sensors.
func (m *managerImpl) ProcessScanRequest(ctx context.Context, scanRequest *storage.ComplianceOperatorScanConfigurationV2, clusters []string) (*storage.ComplianceOperatorScanConfigurationV2, error) {
	if !features.ComplianceEnhancements.Enabled() {
		return nil, errors.Errorf("Compliance is disabled. Cannot process scan request: %q", scanRequest.GetScanConfigName())
	}

	if scanRequest.GetId() != "" {
		return nil, errors.Errorf("The scan configuration already exists and cannot be added.  ID %q and name %q", scanRequest.GetId(), scanRequest.GetScanConfigName())
	}

	err := validateClusterAccess(ctx, clusters)
	if err != nil {
		return nil, err
	}

	cron, err := convertSchedule(scanRequest)
	if err != nil {
		return nil, err
	}

	// Check if scan configuration already exists by name.
	scanConfig, err := m.scanSettingDS.GetScanConfigurationByName(ctx, scanRequest.GetScanConfigName())
	if err != nil {
		err = errors.Wrapf(err, "Unable to create scan configuration named %q", scanRequest.GetScanConfigName())
		log.Error(err)
		return nil, err
	}
	if scanConfig != nil {
		return nil, errors.Errorf("Scan configuration named %q already exists.", scanRequest.GetScanConfigName())
	}

	scanRequest.Id = uuid.NewV4().String()
	scanRequest.CreatedTime = protocompat.TimestampNow()
	validatedProfiles, err := m.validateScan(ctx, scanRequest, clusters)
	if err != nil {
		return nil, err
	}

	return m.processRequestToSensor(ctx, scanRequest, cron, clusters, true, validatedProfiles)
}

// UpdateScanRequest processes a request to apply a compliance scan configuration to one or more Sensors.
func (m *managerImpl) UpdateScanRequest(ctx context.Context, scanRequest *storage.ComplianceOperatorScanConfigurationV2, clusters []string) (*storage.ComplianceOperatorScanConfigurationV2, error) {
	if !features.ComplianceEnhancements.Enabled() {
		return nil, errors.Errorf("Compliance is disabled. Cannot process scan request: %q", scanRequest.GetScanConfigName())
	}

	if scanRequest.GetId() == "" {
		return nil, errors.Errorf("Scan Configuration ID is required for an update, %+v", scanRequest)
	}

	err := validateClusterAccess(ctx, clusters)
	if err != nil {
		return nil, err
	}

	cron, err := convertSchedule(scanRequest)
	if err != nil {
		return nil, err
	}

	// Verify the scan configuration ID is valid
	oldScanConfig, found, err := m.scanSettingDS.GetScanConfiguration(ctx, scanRequest.GetId())
	if err != nil {
		err = errors.Wrapf(err, "Unable to find scan configuration with ID %q", scanRequest.GetId())
		log.Error(err)
		return nil, err
	}
	if !found {
		return nil, errors.Errorf("Scan configuration with ID %q does not exist.", scanRequest.GetId())
	}

	// We are using scan schedule name as FK in scan results. Changing name would break relation.
	if oldScanConfig.GetScanConfigName() != scanRequest.GetScanConfigName() {
		return nil, errors.New("Changing the scan schedule name is not allowed.")
	}

	validatedProfiles, err := m.validateScan(ctx, scanRequest, clusters)
	if err != nil {
		return nil, err
	}

	// TODO(ROX-22398): if we restrict cluster deletion, this is where we would do it before any updates are done.
	m.removeObsoleteResultsByClusters(ctx, oldScanConfig, scanRequest)
	m.removeObsoleteResultsByProfiles(ctx, oldScanConfig, scanRequest)

	// Use the created time from the DB
	scanRequest.CreatedTime = oldScanConfig.GetCreatedTime()
	scanRequest, err = m.processRequestToSensor(ctx, scanRequest, cron, clusters, false, validatedProfiles)
	if err != nil {
		return nil, err
	}

	return scanRequest, nil
}

// removeObsoleteResultsByClusters removes existing results related to removed clusters from scheduler configuration
func (m *managerImpl) removeObsoleteResultsByClusters(ctx context.Context, oldScanConfig *storage.ComplianceOperatorScanConfigurationV2, newScanConfig *storage.ComplianceOperatorScanConfigurationV2) {
	oldClusterIDs := set.NewStringSet()
	for _, oldCluster := range oldScanConfig.GetClusters() {
		oldClusterIDs.Add(oldCluster.GetClusterId())
	}

	newClusterIDs := set.NewStringSet()
	for _, newCluster := range newScanConfig.GetClusters() {
		newClusterIDs.Add(newCluster.GetClusterId())
	}

	removedClusterIDs := oldClusterIDs.Difference(newClusterIDs).AsSlice()
	if len(removedClusterIDs) == 0 {
		return
	}

	// Send delete to sensor for any clusters that were deleted
	m.processClusterDelete(ctx, newScanConfig, removedClusterIDs)

	err := m.resultsDS.DeleteResultsByScanConfigAndCluster(ctx, oldScanConfig.GetScanConfigName(), removedClusterIDs)
	if err != nil {
		log.Errorf("removing obsolete scan results for clusters %v: %v", removedClusterIDs, err)
	}
}

// removeObsoleteResultsByProfiles removes existing results related to removed profiles from scheduler configuration
func (m *managerImpl) removeObsoleteResultsByProfiles(ctx context.Context, oldScanConfig *storage.ComplianceOperatorScanConfigurationV2, newScanConfig *storage.ComplianceOperatorScanConfigurationV2) {
	oldProfileNames := set.NewStringSet()
	for _, oldProfile := range oldScanConfig.GetProfiles() {
		oldProfileNames.Add(oldProfile.GetProfileName())
	}

	newProfileNames := set.NewStringSet()
	for _, newProfile := range newScanConfig.GetProfiles() {
		newProfileNames.Add(newProfile.GetProfileName())
	}

	removedProfileNames := oldProfileNames.Difference(newProfileNames).AsSlice()
	if len(removedProfileNames) == 0 {
		return
	}

	oldClusters := oldScanConfig.GetClusters()
	scanRefIds := make([]string, 0)
	for _, profileName := range removedProfileNames {
		for _, oldCluster := range oldClusters {
			query := search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorScanConfigName, oldScanConfig.GetScanConfigName()).AddExactMatches(search.ClusterID, oldCluster.GetClusterId()).AddExactMatches(search.ComplianceOperatorProfileName, profileName).ProtoQuery()
			scans, err := m.scansDS.SearchScans(ctx, query)
			if err != nil {
				log.Error(errors.Wrapf(err, "unable scan for cluster %q and profile %q", oldCluster.GetClusterId(), profileName))
				return
			}

			for _, scan := range scans {
				scanRefIds = append(scanRefIds, internaltov2storage.BuildNameRefID(oldCluster.GetClusterId(), scan.GetScanName()))
			}
		}
	}

	err := m.resultsDS.DeleteResultsByScans(ctx, scanRefIds)
	if err != nil {
		log.Error(errors.Wrapf(err, "removing obsolete scan results for profiles %v", removedProfileNames))
	}
}

func (m *managerImpl) validateScan(ctx context.Context, scanRequest *storage.ComplianceOperatorScanConfigurationV2, clusters []string) ([]*storage.ComplianceOperatorProfileV2, error) {
	var profiles []string
	for _, profile := range scanRequest.GetProfiles() {
		profiles = append(profiles, profile.GetProfileName())
	}

	// Check if there are any existing clusters that have a scan configuration with any of profiles
	// being referenced by the scan request. If so, then we cannot create the scan configuration.
	err := m.scanSettingDS.ScanConfigurationProfileExists(ctx, scanRequest.GetId(), profiles, clusters)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	// Check if any non-ACS-managed ScanSettingBindings already reference the requested profiles.
	if err := m.checkForExternalSSBConflicts(ctx, profiles, clusters); err != nil {
		return nil, err
	}

	// Validate that all profiles exist in the database and have compatible kinds.
	returnedProfiles, err := m.profileDS.SearchProfiles(ctx, search.NewQueryBuilder().
		AddExactMatches(search.ClusterID, clusters[0]).
		AddExactMatches(search.ComplianceOperatorProfileName, profiles...).ProtoQuery())
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to retrieve profiles for scan configuration named %q", scanRequest.GetScanConfigName())
	}

	if len(returnedProfiles) != len(profiles) {
		return nil, errors.Errorf("Unable to find all profiles for scan configuration named %q.", scanRequest.GetScanConfigName())
	}

	// UNSPECIFIED should not appear here because centralToStorageProfileKind normalizes it to
	// PROFILE at ingestion time, but we allow it through defensively — StorageToCentralProfileKind
	// will map it to PROFILE. Truly unknown kinds are rejected.
	for _, p := range returnedProfiles {
		switch p.GetOperatorKind() {
		case storage.ComplianceOperatorProfileV2_PROFILE, storage.ComplianceOperatorProfileV2_TAILORED_PROFILE:
			// valid
		case storage.ComplianceOperatorProfileV2_OPERATOR_KIND_UNSPECIFIED:
			log.Warnf("Profile %q in scan configuration %q has UNSPECIFIED operator kind; treating as PROFILE", p.GetName(), scanRequest.GetScanConfigName())
		default:
			return nil, errors.Errorf("profile %q has unsupported operator kind %v (scan configuration %q)", p.GetName(), p.GetOperatorKind(), scanRequest.GetScanConfigName())
		}
	}

	return returnedProfiles, nil
}

func (m *managerImpl) processRequestToSensor(ctx context.Context, scanRequest *storage.ComplianceOperatorScanConfigurationV2, cron string, clusters []string, createScanRequest bool, validatedProfiles []*storage.ComplianceOperatorProfileV2) (*storage.ComplianceOperatorScanConfigurationV2, error) {
	profiles := make([]string, 0, len(validatedProfiles))
	for _, p := range validatedProfiles {
		profiles = append(profiles, p.GetName())
	}

	scanRequest.ProfileRefs = internaltov2storage.ProfileV2ToScanConfigRefs(validatedProfiles)

	err := m.scanSettingDS.UpsertScanConfiguration(ctx, scanRequest)
	if err != nil {
		log.Error(err)
		return nil, errors.Errorf("Unable to save scan configuration named %q.", scanRequest.GetScanConfigName())
	}

	profileRefs := internaltov2storage.ScanConfigRefsToCentral(scanRequest.GetProfileRefs())

	for _, clusterID := range clusters {
		// id for the request message to sensor
		sensorRequestID := uuid.NewV4().String()

		sensorMessage := buildScanConfigSensorMsg(sensorRequestID, cron, profiles, profileRefs, scanRequest.GetScanConfigName(), createScanRequest)
		err := m.sensorConnMgr.SendMessage(clusterID, sensorMessage)
		var status string
		if err != nil {
			status = err.Error()
			log.Errorf("error sending compliance scan config to cluster %q: %v", clusterID, err)
		} else {
			// Request was not rejected so add it to map awaiting response
			m.trackSensorRequest(sensorRequestID, clusterID, scanRequest.GetId())
		}

		// Update status in DB
		err = m.updateClusterStatus(ctx, scanRequest.GetId(), clusterID, status)
		if err != nil {
			log.Error(err)
			return nil, errors.Wrapf(err, "Unable to save scan configuration status for scan named %q", scanRequest.GetScanConfigName())
		}
	}

	return scanRequest, nil
}

func (m *managerImpl) processClusterDelete(ctx context.Context, scanRequest *storage.ComplianceOperatorScanConfigurationV2, clusters []string) {
	for _, clusterID := range clusters {
		// id for the request message to sensor
		sensorRequestID := uuid.NewV4().String()

		sensorMessage := &central.MsgToSensor{
			Msg: &central.MsgToSensor_ComplianceRequest{
				ComplianceRequest: &central.ComplianceRequest{
					Request: &central.ComplianceRequest_DeleteScanConfig{
						DeleteScanConfig: &central.DeleteComplianceScanConfigRequest{
							Id:   sensorRequestID,
							Name: scanRequest.GetScanConfigName(),
						},
					},
				},
			},
		}

		err := m.sensorConnMgr.SendMessage(clusterID, sensorMessage)
		if err != nil {
			log.Errorf("error sending deletion of compliance scan config to cluster %q: %v", clusterID, err)
		}

		// Remove the pending request tracker for this cluster and scan config.  If we get any
		// responses for this cluster and scan config after this we will simply swallow the message
		// as it will be obsolete due to the deletion of the scan configuration on the cluster.
		m.removeSensorRequestForCluster(scanRequest.GetId(), clusterID)

		// Remove cluster status
		err = m.scanSettingDS.RemoveClusterStatus(ctx, scanRequest.GetId(), clusterID)
		if err != nil {
			log.Errorf("error removing cluster status for compliance scan config to cluster %q: %v", clusterID, err)
		}
	}
}

// removeSensorRequest removes the pending request for a scan configuration or cluster that was deleted.
func (m *managerImpl) removeSensorRequestForCluster(scanConfigID, clusterID string) {
	m.runningRequestsLock.Lock()
	defer m.runningRequestsLock.Unlock()

	for k, v := range m.runningRequests {
		if v.scanID == scanConfigID && v.clusterID == clusterID {
			// The request was found, remove it from the map
			delete(m.runningRequests, k)
		}
	}
}

// trackSensorRequest adds sensor request to a map with cluster and scan config that was sent for correlating responses.
func (m *managerImpl) trackSensorRequest(sensorRequestID, clusterID, scanConfigID string) {
	m.runningRequestsLock.Lock()
	defer m.runningRequestsLock.Unlock()

	// Request was not rejected so add it to map awaiting response
	m.runningRequests[sensorRequestID] = clusterScan{
		clusterID: clusterID,
		scanID:    scanConfigID,
	}
}

// HandleScanRequestResponse processes response of compliance scan configuration from a sensor.
func (m *managerImpl) HandleScanRequestResponse(ctx context.Context, requestID string, clusterID string, responsePayload string) error {
	if !features.ComplianceEnhancements.Enabled() {
		return errors.Errorf("Compliance is disabled. Cannot process request ID: %q", requestID)
	}

	m.runningRequestsLock.Lock()
	defer m.runningRequestsLock.Unlock()

	// TODO(ROX-18711): This mapping will not survive a restart, such cases will be covered by
	// the sync process when implemented
	var scanID string
	clusterScanData, found := m.runningRequests[requestID]
	if !found {
		return errors.Errorf("Unable to find request %q", requestID)
	}

	// The request was found, remove it from the map
	delete(m.runningRequests, requestID)

	if clusterScanData.clusterID != clusterID {
		return errors.Errorf("Cluster mismatch for request %q", requestID)
	}
	scanID = clusterScanData.scanID

	if scanID == "" {
		return errors.Errorf("Unable to map request %q to a scan configuration", requestID)
	}

	err := m.updateClusterStatus(ctx, scanID, clusterID, responsePayload)
	if err != nil {
		return err
	}

	return nil
}

// checkForExternalSSBConflicts checks if any non-ACS-managed ScanSettingBindings in the target
// clusters already reference any of the requested profiles. ACS-managed SSBs are skipped because
// conflicts with those are already enforced by ScanConfigurationProfileExists.
func (m *managerImpl) checkForExternalSSBConflicts(ctx context.Context, profiles []string, clusters []string) error {
	requestedProfiles := set.NewStringSet(profiles...)
	if requestedProfiles.Cardinality() == 0 {
		return nil
	}

	var errList errorhelpers.ErrorList
	for _, clusterID := range clusters {
		ssbs, err := m.ssbDS.GetScanSettingBindingsByCluster(ctx, clusterID)
		if err != nil {
			return errors.Wrapf(err, "getting scan setting bindings for cluster %q", clusterID)
		}

		clusterRef := ""
		for _, ssb := range ssbs {
			if ssb.GetLabels()["app.kubernetes.io/name"] == "stackrox" {
				continue
			}

			var conflicting []string
			for _, profileName := range ssb.GetProfileNames() {
				if requestedProfiles.Contains(profileName) {
					conflicting = append(conflicting, profileName)
				}
			}
			if len(conflicting) > 0 {
				if clusterRef == "" {
					clusterRef = clusterID
					if name, exists, err := m.clusterDS.GetClusterName(ctx, clusterID); err == nil && exists {
						clusterRef = name
					}
				}
				quoted := make([]string, 0, len(conflicting))
				for _, p := range conflicting {
					quoted = append(quoted, fmt.Sprintf("%q", p))
				}
				errList.AddStringf(
					"profiles [%s] conflict with external ScanSettingBinding %q in cluster %q",
					strings.Join(quoted, ", "), ssb.GetName(), clusterRef)
			}
		}
	}

	if err := errList.ToError(); err != nil {
		return fmt.Errorf("%w, remove the external ScanSettingBindings or choose different profiles", err)
	}
	return nil
}

func (m *managerImpl) ProcessRescanRequest(ctx context.Context, scanID string) error {
	if !features.ComplianceEnhancements.Enabled() {
		return errors.Errorf("Compliance is disabled. Cannot run compliance scan for configuration with ID %s", scanID)
	}

	scanConfig, found, err := m.scanSettingDS.GetScanConfiguration(ctx, scanID)
	if err != nil {
		return errors.Errorf("Encountered error attempting to find scan configuration with ID: %s", scanID)
	} else if !found {
		return errors.Errorf("Failed to find scan configuration by ID: %s", scanID)
	}

	clusters := scanConfig.GetClusters()
	var cs []string
	for _, c := range clusters {
		cs = append(cs, c.GetClusterId())
	}
	err = validateClusterAccess(ctx, cs)
	if err != nil {
		return err
	}

	errList := make([]string, 0)
	for _, c := range clusters {
		msg := &central.MsgToSensor{
			Msg: &central.MsgToSensor_ComplianceRequest{
				ComplianceRequest: &central.ComplianceRequest{
					Request: &central.ComplianceRequest_ApplyScanConfig{
						ApplyScanConfig: &central.ApplyComplianceScanConfigRequest{
							Id: uuid.NewV4().String(),
							ScanRequest: &central.ApplyComplianceScanConfigRequest_RerunScan{
								RerunScan: &central.ApplyComplianceScanConfigRequest_RerunScheduledScan{
									ScanName: scanConfig.GetScanConfigName(),
								},
							},
						},
					},
				},
			},
		}

		errSendMessage := m.sensorConnMgr.SendMessage(c.GetClusterId(), msg)
		if errSendMessage != nil {
			errMsg := fmt.Sprintf("Unable to rescan cluster %s due to message failure: %s", c.GetClusterId(), errSendMessage)
			log.Error(errMsg)
			errList = append(errList, errMsg)
		}
	}

	if len(errList) > 0 {
		return errors.New(strings.Join(errList, "\n"))
	}

	return nil
}

// DeleteScan processes a request to delete an existing compliance scan configuration.
func (m *managerImpl) DeleteScan(ctx context.Context, scanID string) error {
	// Remove the scan configuration from the database
	scanConfigName, err := m.scanSettingDS.DeleteScanConfiguration(ctx, scanID)
	if err != nil {
		return errors.Wrapf(err, "Unable to delete scan configuration ID %q", scanID)
	}

	if scanConfigName == "" {
		return errors.Errorf("Unable to find scan configuration name for ID %q.", scanID)
	}

	// send delete request to sensor
	sensorRequestID := uuid.NewV4().String()
	sensorMessage := &central.MsgToSensor{
		Msg: &central.MsgToSensor_ComplianceRequest{
			ComplianceRequest: &central.ComplianceRequest{
				Request: &central.ComplianceRequest_DeleteScanConfig{
					DeleteScanConfig: &central.DeleteComplianceScanConfigRequest{
						Id:   sensorRequestID,
						Name: scanConfigName,
					},
				},
			},
		},
	}
	m.sensorConnMgr.BroadcastMessage(sensorMessage)

	return nil
}

// validateClusterAccess accepts a context and a slice of cluster strings, and
// returns if the user associated with the context has write permissions on
// each cluster. If not, then a permission error is returned.
func validateClusterAccess(ctx context.Context, clusters []string) error {
	// User MUST have permissions on all clusters being applied.
	clusterScopeKeys := make([][]sac.ScopeKey, 0, len(clusters))
	for _, cluster := range clusters {
		clusterScopeKeys = append(clusterScopeKeys, []sac.ScopeKey{sac.ClusterScopeKey(cluster)})
	}
	if !complianceSAC.ScopeChecker(ctx, storage.Access_READ_WRITE_ACCESS).AllAllowed(clusterScopeKeys) {
		return sac.ErrResourceAccessDenied
	}
	return nil
}

// updateClusterStatus updates cluster status
func (m *managerImpl) updateClusterStatus(ctx context.Context, scanConfigID string, clusterID string, clusterStatus string) error {
	clusterName, exists, errCluster := m.clusterDS.GetClusterName(ctx, clusterID)
	if errCluster != nil {
		return errCluster
	}
	if !exists {
		return errors.Errorf("could not pull config for cluster %q because it does not exist", clusterID)
	}

	return m.scanSettingDS.UpdateClusterStatus(ctx, scanConfigID, clusterID, clusterStatus, clusterName)
}

func (m *managerImpl) ReconcileDiscoveredConfig(_ context.Context, ssbName string) error {
	if !features.ComplianceEnhancements.Enabled() {
		return nil
	}

	ctx := sac.WithAllAccess(context.Background())
	existing, err := m.scanSettingDS.GetScanConfigurationByName(ctx, ssbName)
	if err != nil {
		return errors.Wrapf(err, "looking up scan config %q", ssbName)
	}

	query := search.NewQueryBuilder().
		AddExactMatches(search.ComplianceOperatorScanSettingBindingName, ssbName).
		ProtoQuery()
	discovered, err := m.ssbDS.GetDistinctScanConfigs(ctx, query)
	if err != nil {
		return errors.Wrapf(err, "aggregating SSBs for %q", ssbName)
	}

	var dc *ssbDatastore.DiscoveredScanConfig
	for _, d := range discovered {
		if d.Name == ssbName {
			dc = d
			break
		}
	}

	if dc == nil {
		if existing != nil {
			_, err = m.scanSettingDS.DeleteScanConfiguration(ctx, existing.GetId())
			if err != nil {
				log.Errorf("deleting orphaned discovered config %q: %v", ssbName, err)
			}
		}
		return nil
	}

	clusters := make([]*storage.ComplianceOperatorScanConfigurationV2_Cluster, 0, len(dc.ClusterIDs))
	for _, cid := range dc.ClusterIDs {
		clusters = append(clusters, &storage.ComplianceOperatorScanConfigurationV2_Cluster{ClusterId: cid})
	}
	profiles := make([]*storage.ComplianceOperatorScanConfigurationV2_ProfileName, 0, len(dc.ProfileNames))
	for _, p := range dc.ProfileNames {
		profiles = append(profiles, &storage.ComplianceOperatorScanConfigurationV2_ProfileName{ProfileName: p})
	}

	var scanConfig *storage.ComplianceOperatorScanConfigurationV2
	if existing != nil {
		existing.Clusters = clusters
		existing.Profiles = profiles
		existing.LastUpdatedTime = protocompat.TimestampNow()
		if err := m.scanSettingDS.UpsertScanConfiguration(ctx, existing); err != nil {
			return err
		}
		scanConfig = existing
	} else {
		scanConfig = &storage.ComplianceOperatorScanConfigurationV2{
			Id:              uuid.NewV4().String(),
			ScanConfigName:  ssbName,
			Clusters:        clusters,
			Profiles:        profiles,
			CreatedTime:     protocompat.TimestampNow(),
			LastUpdatedTime: protocompat.TimestampNow(),
		}
		if err := m.scanSettingDS.UpsertScanConfiguration(ctx, scanConfig); err != nil {
			return err
		}
	}

	for _, cid := range dc.ClusterIDs {
		if err := m.updateClusterStatus(ctx, scanConfig.GetId(), cid, ""); err != nil {
			log.Errorf("updating cluster status for discovered config %q cluster %s: %v", ssbName, cid, err)
		}
	}
	return nil
}

func convertSchedule(scanRequest *storage.ComplianceOperatorScanConfigurationV2) (string, error) {
	var cron string
	var err error
	if scanRequest.GetSchedule() != nil {
		cron, err = schedule.ConvertToCronTab(scanRequest.GetSchedule())
		if err != nil {
			err = errors.Wrapf(err, "Unable to convert schedule for scan configuration named %q to cron", scanRequest.GetScanConfigName())
			log.Error(err)
			return "", err
		}
		cronValidator := gronx.New()
		if !cronValidator.IsValid(cron) {
			err = errors.Errorf("Schedule for scan configuration named %q is invalid.", scanRequest.GetScanConfigName())
			log.Error(err)
			return "", err
		}
	}

	return cron, nil
}
