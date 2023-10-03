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
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/uuid"
)

const (
	// Daily at midnight
	defaultScanSchedule = "0 0 * * *"
)

var (
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
		return nil, errors.Errorf("Compliance is disabled. Cannot process scan request: %q", scanRequest.GetScanName())
	}

	// Convert and validate schedule
	// TODO(ROX-19818):  remove default schedule
	// For MVP a schedule is required.  If one is not present use daily at midnight.  As it evolves
	// it may be as simple as no schedule means a one time scan.
	cron := defaultScanSchedule
	var err error
	if scanRequest.GetSchedule() != nil {
		cron, err = schedule.ConvertToCronTab(scanRequest.GetSchedule())
		if err != nil {
			err = errors.Wrapf(err, "Unable to convert schedule for scan configuration named %q to cron.", scanRequest.GetScanName())
			log.Error(err)
			return nil, err
		}
		cronValidator := gronx.New()
		if !cronValidator.IsValid(cron) {
			err = errors.Errorf("Schedule for scan configuration named %q is invalid.", scanRequest.GetScanName())
			log.Error(err)
			return nil, err
		}
	}

	// Check if scan configuration already exists.
	found, err := m.scanSettingDS.ScanConfigurationExists(ctx, scanRequest.GetScanName())
	if err != nil {
		log.Error(err)
		return nil, errors.Wrapf(err, "Unable to create scan configuration named %q.", scanRequest.GetScanName())
	}
	if found {
		return nil, errors.Errorf("Scan configuration named %q already exists.", scanRequest.GetScanName())
	}

	scanRequest.Id = uuid.NewV4().String()
	scanRequest.CreatedTime = types.TimestampNow()
	err = m.scanSettingDS.UpsertScanConfiguration(ctx, scanRequest)
	if err != nil {
		log.Error(err)
		return nil, errors.Errorf("Unable to save scan configuration named %q.", scanRequest.GetScanName())
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
										ScanName:       scanRequest.GetScanName(),
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
			return nil, errors.Errorf("Unable to save scan configuration status for scan named %q.", scanRequest.GetScanName())
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

func (m *managerImpl) ProcessRescanRequest(_ context.Context, _ interface{}) error {
	// TODO(ROX-18091):
	// 1. Validate config exists in database
	// 2. Push request to Sensor
	panic("implement me")
}

// DeleteScan processes a request to delete an existing compliance scan configuration.
// TODO(ROX-19540)
func (m *managerImpl) DeleteScan(_ context.Context, _ interface{}) error {
	// TODO:
	// 1. Validate config exists in database
	// 2. Lock config so it cannot be edited/updated
	// 3. Push request to Sensor
	panic("implement me")
}
