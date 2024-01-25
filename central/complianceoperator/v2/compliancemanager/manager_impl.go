package compliancemanager

import (
	"context"

	"github.com/adhocore/gronx"
	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
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

	// Map used to correlate requests to a sensor with a response.  Each request will generate
	// a unique entry in the map
	runningRequests     map[string]clusterScan
	runningRequestsLock sync.Mutex
}

// New returns on instance of Manager interface that provides functionality to process compliance requests and forward them to Sensor.
func New(sensorConnMgr connection.Manager, integrationDS compIntegration.DataStore, scanSettingDS compScanSetting.DataStore) Manager {
	return &managerImpl{
		sensorConnMgr:   sensorConnMgr,
		integrationDS:   integrationDS,
		scanSettingDS:   scanSettingDS,
		runningRequests: make(map[string]clusterScan),
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

	err := validateClusterAccess(ctx, clusters)
	if err != nil {
		return nil, err
	}

	var cron string
	if scanRequest.GetSchedule() != nil {
		cron, err = schedule.ConvertToCronTab(scanRequest.GetSchedule())
		if err != nil {
			err = errors.Wrapf(err, "Unable to convert schedule for scan configuration named %q to cron.", scanRequest.GetScanConfigName())
			log.Error(err)
			return nil, err
		}
		cronValidator := gronx.New()
		if !cronValidator.IsValid(cron) {
			err = errors.Errorf("Schedule for scan configuration named %q is invalid.", scanRequest.GetScanConfigName())
			log.Error(err)
			return nil, err
		}
	}

	// Check if scan configuration already exists.
	found, err := m.scanSettingDS.ScanConfigurationExists(ctx, scanRequest.GetScanConfigName())
	if err != nil {
		log.Error(err)
		return nil, errors.Wrapf(err, "Unable to create scan configuration named %q.", scanRequest.GetScanConfigName())
	}
	if found {
		return nil, errors.Errorf("Scan configuration named %q already exists.", scanRequest.GetScanConfigName())
	}

	scanRequest.Id = uuid.NewV4().String()
	scanRequest.CreatedTime = types.TimestampNow()
	err = m.scanSettingDS.UpsertScanConfiguration(ctx, scanRequest)
	if err != nil {
		log.Error(err)
		return nil, errors.Errorf("Unable to save scan configuration named %q.", scanRequest.GetScanConfigName())
	}

	var profiles []string
	for _, profile := range scanRequest.GetProfiles() {
		profiles = append(profiles, profile.GetProfileName())
	}

	for _, clusterID := range clusters {
		// id for the request message to sensor
		sensorRequestID := uuid.NewV4().String()

		sensorMessage := &central.MsgToSensor{
			Msg: &central.MsgToSensor_ComplianceRequest{
				ComplianceRequest: &central.ComplianceRequest{
					Request: &central.ComplianceRequest_ApplyScanConfig{
						ApplyScanConfig: &central.ApplyComplianceScanConfigRequest{
							Id: sensorRequestID,
							ScanRequest: &central.ApplyComplianceScanConfigRequest_ScheduledScan_{
								ScheduledScan: &central.ApplyComplianceScanConfigRequest_ScheduledScan{
									ScanSettings: &central.ApplyComplianceScanConfigRequest_BaseScanSettings{
										ScanName:       scanRequest.GetScanConfigName(),
										StrictNodeScan: true,
										Profiles:       profiles,
									},
									Cron: cron,
								},
							},
						},
					},
				},
			},
		}

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
		err = m.scanSettingDS.UpdateClusterStatus(ctx, scanRequest.GetId(), clusterID, status)
		if err != nil {
			log.Error(err)
			return nil, errors.Errorf("Unable to save scan configuration status for scan named %q.", scanRequest.GetScanConfigName())
		}
	}

	return scanRequest, nil
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

	err := m.scanSettingDS.UpdateClusterStatus(ctx, scanID, clusterID, responsePayload)
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
			err = m.scanSettingDS.UpdateClusterStatus(ctx, scanConfig.GetId(), c.GetClusterId(), err.Error())
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
