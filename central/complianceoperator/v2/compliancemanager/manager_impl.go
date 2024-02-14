package compliancemanager

import (
	"context"

	"github.com/adhocore/gronx"
	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	clusterDatastore "github.com/stackrox/rox/central/cluster/datastore"
	compIntegration "github.com/stackrox/rox/central/complianceoperator/v2/integration/datastore"
	compScanSetting "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore"
	"github.com/stackrox/rox/central/sensor/service/connection"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protoconv/schedule"
	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/sliceutils"
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
	clusterDS     clusterDatastore.DataStore

	// Map used to correlate requests to a sensor with a response.  Each request will generate
	// a unique entry in the map
	runningRequests     map[string]clusterScan
	runningRequestsLock sync.Mutex
}

// New returns on instance of Manager interface that provides functionality to process compliance requests and forward them to Sensor.
func New(sensorConnMgr connection.Manager, integrationDS compIntegration.DataStore, scanSettingDS compScanSetting.DataStore, clusterDS clusterDatastore.DataStore) Manager {
	return &managerImpl{
		sensorConnMgr:   sensorConnMgr,
		integrationDS:   integrationDS,
		scanSettingDS:   scanSettingDS,
		runningRequests: make(map[string]clusterScan),
		clusterDS:       clusterDS,
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
		err = errors.Wrapf(err, "Unable to create scan configuration named %q.", scanRequest.GetScanConfigName())
		log.Error(err)
		return nil, err
	}
	if scanConfig != nil {
		return nil, errors.Errorf("Scan configuration named %q already exists.", scanRequest.GetScanConfigName())
	}

	scanRequest.Id = uuid.NewV4().String()
	scanRequest.CreatedTime = types.TimestampNow()

	return m.processRequestToSensor(ctx, scanRequest, cron, clusters, true)
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
		err = errors.Wrapf(err, "Unable to find scan configuration with ID %q.", scanRequest.GetId())
		log.Error(err)
		return nil, err
	}
	if !found {
		return nil, errors.Errorf("Scan configuration with ID %q does not exist.", scanRequest.GetId())
	}

	// Use the old config to determine which clusters were deleted from the configuration
	// TODO(ROX-22398): if we restrict cluster deletion, this is where we would do it before any updates are done.
	var deletedClusters []string
	for _, oldCluster := range oldScanConfig.GetClusters() {
		if sliceutils.Find(clusters, oldCluster.GetClusterId()) == -1 {
			deletedClusters = append(deletedClusters, oldCluster.ClusterId)
		}
	}

	// Use the created time from the DB
	scanRequest.CreatedTime = oldScanConfig.GetCreatedTime()

	scanRequest, err = m.processRequestToSensor(ctx, scanRequest, cron, clusters, false)
	if err != nil {
		return nil, err
	}

	// Send delete to sensor for any clusters that were deleted
	m.processClusterDelete(ctx, scanRequest, deletedClusters)

	return scanRequest, nil
}

func (m *managerImpl) processRequestToSensor(ctx context.Context, scanRequest *storage.ComplianceOperatorScanConfigurationV2, cron string, clusters []string, createScanRequest bool) (*storage.ComplianceOperatorScanConfigurationV2, error) {
	var profiles []string
	for _, profile := range scanRequest.GetProfiles() {
		profiles = append(profiles, profile.GetProfileName())
	}

	// Check if any existing cluster that has the scan configuration with any of profiles being referenced by the scan request.
	// If so, then we cannot create the scan configuration.
	err := m.scanSettingDS.ScanConfigurationProfileExists(ctx, scanRequest.GetId(), profiles, clusters)
	if err != nil {
		log.Error(err)
		return nil, errors.Wrapf(err, "Unable to create scan configuration named %q.", scanRequest.GetScanConfigName())
	}

	err = m.scanSettingDS.UpsertScanConfiguration(ctx, scanRequest)
	if err != nil {
		log.Error(err)
		return nil, errors.Errorf("Unable to save scan configuration named %q.", scanRequest.GetScanConfigName())
	}

	for _, clusterID := range clusters {
		// id for the request message to sensor
		sensorRequestID := uuid.NewV4().String()

		sensorMessage := buildScanConfigSensorMsg(sensorRequestID, cron, profiles, scanRequest.GetScanConfigName(), createScanRequest)
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
			return nil, errors.Errorf("Unable to save scan configuration status for scan named %q.", scanRequest.GetScanConfigName())
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

		// Remove cluster status
		err = m.scanSettingDS.RemoveClusterStatus(ctx, scanRequest.GetId(), clusterID)
		if err != nil {
			log.Errorf("error removing cluster status for compliance scan config to cluster %q: %v", clusterID, err)
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
		err := m.sensorConnMgr.SendMessage(c.GetClusterId(), msg)
		if err != nil {
			log.Errorf("Unable to rescan cluster %s due to message failure: %s", c.GetClusterId(), err)
			// Update status in DB
			err = m.updateClusterStatus(ctx, scanConfig.GetId(), c.GetClusterId(), err.Error())
			if err != nil {
				log.Error(err)
				return errors.Errorf("Unable to save scan configuration status for scan configuration %q.", scanConfig.GetScanConfigName())
			}
		}
	}

	return nil
}

// DeleteScan processes a request to delete an existing compliance scan configuration.
func (m *managerImpl) DeleteScan(ctx context.Context, scanID string) error {
	// Remove the scan configuration from the database
	scanConfigName, err := m.scanSettingDS.DeleteScanConfiguration(ctx, scanID)
	if err != nil {
		return errors.Wrapf(err, "Unable to delete scan configuration ID %q.", scanID)
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

func convertSchedule(scanRequest *storage.ComplianceOperatorScanConfigurationV2) (string, error) {
	var cron string
	var err error
	if scanRequest.GetSchedule() != nil {
		cron, err = schedule.ConvertToCronTab(scanRequest.GetSchedule())
		if err != nil {
			err = errors.Wrapf(err, "Unable to convert schedule for scan configuration named %q to cron.", scanRequest.GetScanConfigName())
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
