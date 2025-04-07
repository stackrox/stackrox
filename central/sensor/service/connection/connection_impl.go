package connection

import (
	"context"
	"fmt"
	"math"

	"github.com/pkg/errors"
	compScanSetting "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore"
	delegatedRegistryConfigConvert "github.com/stackrox/rox/central/delegatedregistryconfig/convert"
	"github.com/stackrox/rox/central/delegatedregistryconfig/util/imageintegration"
	hashManager "github.com/stackrox/rox/central/hash/manager"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/networkpolicies/graph"
	"github.com/stackrox/rox/central/scrape"
	"github.com/stackrox/rox/central/securedclustercertgen"
	"github.com/stackrox/rox/central/sensor/networkentities"
	"github.com/stackrox/rox/central/sensor/networkpolicies"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/connection/messagestream"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/central/sensor/telemetry"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/policyversion"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/dedupingqueue"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/protoconv/schedule"
	"github.com/stackrox/rox/pkg/reflectutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/safe"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/sensor/event"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sliceutils"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/pkg/sync"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	log = logging.LoggerForModule()
)

type sensorConnection struct {
	clusterID           string
	stopSig, stoppedSig concurrency.ErrorSignal

	sendC chan *central.MsgToSensor

	scrapeCtrl          scrape.Controller
	networkPoliciesCtrl networkpolicies.Controller
	networkEntitiesCtrl networkentities.Controller
	telemetryCtrl       telemetry.Controller

	sensorEventHandler *sensorEventHandler

	queues      map[string]*dedupingqueue.DedupingQueue[string]
	queuesMutex sync.Mutex

	eventPipeline pipeline.ClusterPipeline

	clusterMgr                 common.ClusterManager
	networkEntityMgr           common.NetworkEntityManager
	policyMgr                  common.PolicyManager
	baselineMgr                common.ProcessBaselineManager
	networkBaselineMgr         common.NetworkBaselineManager
	delegatedRegistryConfigMgr common.DelegatedRegistryConfigManager
	imageIntegrationMgr        common.ImageIntegrationManager
	complianceOperatorMgr      common.ComplianceOperatorManager
	scanSettingDS              compScanSetting.DataStore

	sensorHello  *central.SensorHello
	capabilities set.Set[centralsensor.SensorCapability]

	hashDeduper hashManager.Deduper
}

func newConnection(ctx context.Context,
	sensorHello *central.SensorHello,
	cluster *storage.Cluster,
	eventPipeline pipeline.ClusterPipeline,
	clusterMgr common.ClusterManager,
	networkEntityMgr common.NetworkEntityManager,
	policyMgr common.PolicyManager,
	baselineMgr common.ProcessBaselineManager,
	networkBaselineMgr common.NetworkBaselineManager,
	delegatedRegistryConfigMgr common.DelegatedRegistryConfigManager,
	imageIntegrationMgr common.ImageIntegrationManager,
	hashMgr hashManager.Manager,
	complianceOperatorMgr common.ComplianceOperatorManager,
	initSyncMgr *initSyncManager,
) *sensorConnection {

	conn := &sensorConnection{
		stopSig:       concurrency.NewErrorSignal(),
		stoppedSig:    concurrency.NewErrorSignal(),
		sendC:         make(chan *central.MsgToSensor),
		eventPipeline: eventPipeline,
		queues:        make(map[string]*dedupingqueue.DedupingQueue[string]),

		clusterID:                  cluster.GetId(),
		clusterMgr:                 clusterMgr,
		policyMgr:                  policyMgr,
		networkEntityMgr:           networkEntityMgr,
		baselineMgr:                baselineMgr,
		networkBaselineMgr:         networkBaselineMgr,
		delegatedRegistryConfigMgr: delegatedRegistryConfigMgr,
		imageIntegrationMgr:        imageIntegrationMgr,
		complianceOperatorMgr:      complianceOperatorMgr,
		scanSettingDS:              compScanSetting.Singleton(),

		sensorHello: sensorHello,
		capabilities: set.NewSet(sliceutils.
			FromStringSlice[centralsensor.SensorCapability](sensorHello.GetCapabilities()...)...),
	}

	// Need a reference to conn for injector
	deduper := hashMgr.GetDeduper(ctx, cluster.GetId())
	deduper.StartSync()

	conn.hashDeduper = deduper
	conn.sensorEventHandler = newSensorEventHandler(cluster, sensorHello.GetSensorVersion(), eventPipeline, conn, &conn.stopSig, deduper, initSyncMgr)
	conn.scrapeCtrl = scrape.NewController(conn, &conn.stopSig)
	conn.networkPoliciesCtrl = networkpolicies.NewController(conn, &conn.stopSig)
	conn.networkEntitiesCtrl = networkentities.NewController(cluster.GetId(), networkEntityMgr, graph.Singleton(), conn, &conn.stopSig)
	conn.telemetryCtrl = telemetry.NewController(conn.capabilities, conn, &conn.stopSig)

	return conn
}

func (c *sensorConnection) Terminate(err error) bool {
	return c.stopSig.SignalWithError(err)
}

func (c *sensorConnection) Stopped() concurrency.ReadOnlyErrorSignal {
	return &c.stoppedSig
}

// multiplexedPush pushes the given message to a dedicated queue for the respective event type.
// The queues parameter, if non-nil, will be used to look up the queue by event type. If the `queues`
// map is nil or does not contain an entry for the respective type, a queue is retrieved from the
// mutex-protected `c.queues` map (and created if exists), and afterward stored in the `queues` map
// if non-nil.
// The envisioned use for this is that a caller invoking `multiplexedPush` repeatedly will maintain
// an exclusively used (i.e., not requiring protection via a mutex) map, that will automatically be
// populated with a subset of the entries from `c.queues`. This avoids mutex lock acquisitions for every
// invocation of `multiplexedPush` with a previously seen (from the perspective of the caller)
// event type.
func (c *sensorConnection) multiplexedPush(ctx context.Context, msg *central.MsgFromSensor, queues map[string]*dedupingqueue.DedupingQueue[string]) {
	if msg.GetMsg() == nil {
		// This is likely because sensor is a newer version than central and is sending a message that this central doesn't know about
		// This is already logged, so it's fine to just ignore it for now
		return
	}

	typ := reflectutils.Type(msg.Msg)
	queue := queues[typ]
	if queue == nil {
		concurrency.WithLock(&c.queuesMutex, func() {
			queue = c.queues[typ]
			if queue == nil {
				queue = dedupingqueue.NewDedupingQueue[string](
					dedupingqueue.WithQueueName[string](stripTypePrefix(typ)),
					dedupingqueue.WithOperationMetricsFunc[string](metrics.IncrementSensorEventQueueCounter))
				go c.handleMessages(ctx, queue)
				c.queues[typ] = queue
			}
		})
		if queues != nil {
			queues[typ] = queue
		}
	}
	queue.Push(msg)
}

func getSensorMessageTypeString(msg *central.MsgFromSensor) string {
	messageType := reflectutils.Type(msg.GetMsg())
	var eventType string
	if msg.GetEvent() != nil {
		eventType = event.GetEventTypeWithoutPrefix(msg.GetEvent().GetResource())
	}
	return fmt.Sprintf("%s_%s", messageType, eventType)
}

func (c *sensorConnection) runRecv(ctx context.Context, grpcServer central.SensorService_CommunicateServer) {
	queues := make(map[string]*dedupingqueue.DedupingQueue[string])
	for !c.stopSig.IsDone() {
		msg, err := grpcServer.Recv()
		if err != nil {
			if errStatus, ok := status.FromError(err); ok {
				if errStatus.Code() == codes.ResourceExhausted {
					log.Debugf("Central received a payload from sensor that was too large: %v", errStatus.Details())
				}
				metrics.RegisterGRPCError(errStatus.Code().String())
			}
			c.stopSig.SignalWithError(errors.Wrap(err, "recv error"))
			return
		}
		metrics.SetGRPCLastMessageSizeReceived(getSensorMessageTypeString(msg), float64(msg.SizeVT()))
		c.multiplexedPush(ctx, msg, queues)
	}
}

func (c *sensorConnection) handleMessages(ctx context.Context, queue *dedupingqueue.DedupingQueue[string]) {
	for msg := queue.PullBlocking(&c.stopSig); msg != nil; msg = queue.PullBlocking(&c.stopSig) {
		msgFromSensor, ok := msg.(*central.MsgFromSensor)
		if !ok {
			log.Error("Invalid sensor message")
			continue
		}
		err := safe.Run(func() {
			if err := c.handleMessage(ctx, msgFromSensor); err != nil {
				log.Errorf("Error handling sensor message: %v", err)
			}
		})
		if err != nil {
			metrics.IncrementPipelinePanics(msgFromSensor)
			log.Errorf("panic in handle message: %v", err)
		}
	}
	c.eventPipeline.OnFinish(c.clusterID)
	c.stoppedSig.SignalWithError(c.stopSig.Err())
}

func (c *sensorConnection) runSend(server central.SensorService_CommunicateServer) {

	wrappedStream := messagestream.NewSizingEventStream(server)

	for !c.stopSig.IsDone() {
		select {
		case <-c.stopSig.Done():
			return
		case <-server.Context().Done():
			c.stopSig.SignalWithError(errors.Wrap(server.Context().Err(), "context error"))
			return
		case msg := <-c.sendC:
			if err := wrappedStream.Send(msg); err != nil {
				c.stopSig.SignalWithError(errors.Wrap(err, "send error"))
				return
			}
		}
	}
}

func (c *sensorConnection) Scrapes() scrape.Controller {
	return c.scrapeCtrl
}

func (c *sensorConnection) InjectMessageIntoQueue(msg *central.MsgFromSensor) {
	c.multiplexedPush(sac.WithAllAccess(withConnection(context.Background(), c)), msg, nil)
}

func (c *sensorConnection) NetworkEntities() networkentities.Controller {
	return c.networkEntitiesCtrl
}

func (c *sensorConnection) NetworkPolicies() networkpolicies.Controller {
	return c.networkPoliciesCtrl
}

func (c *sensorConnection) Telemetry() telemetry.Controller {
	return c.telemetryCtrl
}

func (c *sensorConnection) InjectMessage(ctx concurrency.Waitable, msg *central.MsgToSensor) error {
	select {
	case c.sendC <- msg:
		return nil
	case <-ctx.Done():
		return errors.New("context aborted")
	case <-c.stopSig.Done():
		return errors.Wrap(c.stopSig.Err(), "could not send message as sensor connection was stopped")
	}
}

func (c *sensorConnection) handleMessage(ctx context.Context, msg *central.MsgFromSensor) error {
	switch m := msg.Msg.(type) {
	case *central.MsgFromSensor_ScrapeUpdate:
		return c.scrapeCtrl.ProcessScrapeUpdate(m.ScrapeUpdate)
	case *central.MsgFromSensor_NetworkPoliciesResponse:
		return c.networkPoliciesCtrl.ProcessNetworkPoliciesResponse(m.NetworkPoliciesResponse)
	case *central.MsgFromSensor_TelemetryDataResponse:
		return c.telemetryCtrl.ProcessTelemetryDataResponse(ctx, m.TelemetryDataResponse)
	case *central.MsgFromSensor_IssueLocalScannerCertsRequest:
		return c.processIssueLocalScannerCertsRequest(ctx, m.IssueLocalScannerCertsRequest)
	case *central.MsgFromSensor_IssueSecuredClusterCertsRequest:
		return c.processIssueSecuredClusterCertsRequest(ctx, m.IssueSecuredClusterCertsRequest)
	case *central.MsgFromSensor_ComplianceResponse:
		return c.processComplianceResponse(ctx, msg.GetComplianceResponse())
	case *central.MsgFromSensor_Event:
		// Special case the reprocess deployment because its fields are already set
		if msg.GetEvent().GetReprocessDeployment() != nil {
			c.sensorEventHandler.addMultiplexed(ctx, msg)
			return nil
		}
		if shallDedupe(msg) {
			msg.DedupeKey = msg.GetEvent().GetId()
		}
		// Set the hash key for all values
		msg.HashKey = msg.GetEvent().GetId()

		c.sensorEventHandler.addMultiplexed(ctx, msg)
		return nil
	}
	return c.eventPipeline.Run(ctx, msg, c)
}

func shallDedupe(msg *central.MsgFromSensor) bool {
	// Special handling of node inventory and node indexes for Sensor version 4.6 and earlier.
	// NodeInventory and NodeIndex should never be deduped. Despite the packages/images to scan may be the same,
	// the vulnerabilities database in scanner may get updated and new vulnerabilities may affect those packages.
	ev := msg.GetEvent()
	if ev.GetAction() != central.ResourceAction_REMOVE_RESOURCE {
		if ev.GetNodeInventory() != nil || ev.GetIndexReport() != nil {
			return false
		}
	}
	return ev.GetAction() != central.ResourceAction_CREATE_RESOURCE
}

func (c *sensorConnection) processComplianceResponse(ctx context.Context, msg *central.ComplianceResponse) error {
	switch m := msg.Response.(type) {
	case *central.ComplianceResponse_ApplyComplianceScanConfigResponse_:
		return c.complianceOperatorMgr.HandleScanRequestResponse(ctx, m.ApplyComplianceScanConfigResponse.GetId(), c.clusterID, m.ApplyComplianceScanConfigResponse.GetError())
	case *central.ComplianceResponse_DeleteComplianceScanConfigResponse_:
		log.Debugf("received delete compliance scan config error response %v for cluster %v", m.DeleteComplianceScanConfigResponse.GetError(), c.clusterID)
		return nil
	default:
		log.Infof("Unimplemented compliance response  %T", m)
	}
	return errors.Errorf("Unimplemented compliance response  %T", msg.Response)
}

func (c *sensorConnection) processIssueLocalScannerCertsRequest(ctx context.Context, request *central.IssueLocalScannerCertsRequest) error {
	requestID := request.GetRequestId()
	clusterID := c.clusterID
	namespace := c.sensorHello.GetDeploymentIdentification().GetAppNamespace()
	errMsg := fmt.Sprintf("issuing local Scanner certificates for request ID %q, cluster ID %q and namespace %q",
		requestID, clusterID, namespace)
	var (
		err      error
		response *central.IssueLocalScannerCertsResponse
	)
	if requestID == "" {
		err = errors.New("requestID is required to issue the certificates for the local scanner")
	} else {
		var certificates *storage.TypedServiceCertificateSet
		certificates, err = securedclustercertgen.IssueLocalScannerCerts(namespace, clusterID)
		response = &central.IssueLocalScannerCertsResponse{
			RequestId: requestID,
			Response: &central.IssueLocalScannerCertsResponse_Certificates{
				Certificates: certificates,
			},
		}
	}
	if err != nil {
		response = &central.IssueLocalScannerCertsResponse{
			RequestId: requestID,
			Response: &central.IssueLocalScannerCertsResponse_Error{
				Error: &central.LocalScannerCertsIssueError{
					Message: fmt.Sprintf("%s: %s", errMsg, err.Error()),
				},
			},
		}
	}
	err = c.InjectMessage(ctx, &central.MsgToSensor{
		Msg: &central.MsgToSensor_IssueLocalScannerCertsResponse{IssueLocalScannerCertsResponse: response},
	})
	if err != nil {
		return errors.Wrap(err, errMsg)
	}
	return nil
}

func (c *sensorConnection) processIssueSecuredClusterCertsRequest(ctx context.Context, request *central.IssueSecuredClusterCertsRequest) error {
	requestID := request.GetRequestId()
	clusterID := c.clusterID
	namespace := c.sensorHello.GetDeploymentIdentification().GetAppNamespace()
	errMsg := fmt.Sprintf("issuing Secured Cluster certificates for request ID %q, cluster ID %q and namespace %q",
		requestID, clusterID, namespace)
	var (
		err      error
		response *central.IssueSecuredClusterCertsResponse
	)
	if requestID == "" {
		err = errors.New("requestID is required to issue the certificates for a Secured Cluster")
	} else {
		var certificates *storage.TypedServiceCertificateSet
		certificates, err = securedclustercertgen.IssueSecuredClusterCerts(namespace, clusterID)
		response = &central.IssueSecuredClusterCertsResponse{
			RequestId: requestID,
			Response: &central.IssueSecuredClusterCertsResponse_Certificates{
				Certificates: certificates,
			},
		}
	}
	if err != nil {
		response = &central.IssueSecuredClusterCertsResponse{
			RequestId: requestID,
			Response: &central.IssueSecuredClusterCertsResponse_Error{
				Error: &central.SecuredClusterCertsIssueError{
					Message: fmt.Sprintf("%s: %s", errMsg, err.Error()),
				},
			},
		}
	}
	err = c.InjectMessage(ctx, &central.MsgToSensor{
		Msg: &central.MsgToSensor_IssueSecuredClusterCertsResponse{IssueSecuredClusterCertsResponse: response},
	})
	if err != nil {
		return errors.Wrap(err, errMsg)
	}
	return nil
}

// getPolicySyncMsg fetches stored policies and prepares them for delivery to sensor.
func (c *sensorConnection) getPolicySyncMsg(ctx context.Context) (*central.MsgToSensor, error) {
	policies, err := c.policyMgr.GetAllPolicies(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "error getting policies for initial sync")
	}

	return c.getPolicySyncMsgFromPolicies(policies)
}

// getPolicySyncMsgFromPolicies prepares given policies for delivery to sensor. If:
//   - sensor's policy version is unknown -> send version 1.1
//   - sensor's policy version is less than the minimum supported by central (see policyversion.MinimumSupportedVersion()), send version 1.1
//   - sensor's policy version is < current version -> downgrade to sensor's version
//   - sensor's policy version is >= current version -> send policies untouched
func (c *sensorConnection) getPolicySyncMsgFromPolicies(policies []*storage.Policy) (*central.MsgToSensor, error) {
	// Older sensors do not broadcast the policy version they support, so if we
	// observe an empty string, we guess the version at version 1.1 and persist it.
	sensorPolicyVersionStr := stringutils.FirstNonEmpty(c.sensorHello.GetPolicyVersion(),
		policyversion.CurrentVersion().String())

	// Forward policies as is if we don't understand sensor's version.
	if sensorPolicyVersion, err := policyversion.FromString(sensorPolicyVersionStr); err != nil {
		log.Errorf("Cannot understand sensor's policy version %q, will assume version %q: %v", sensorPolicyVersionStr,
			policyversion.CurrentVersion().String(), err)
	} else {
		if policyversion.Compare(sensorPolicyVersion, policyversion.MinimumSupportedVersion()) < 0 {
			// If sensor doesn't support a minimum version (which will change as we deprecate older versions),
			// Then just log error and move on.
			log.Errorf("Sensor is running an older version of policy (%q) which has been deprecated and is no longer supported. Upgrade sensor to a version which supports %q",
				sensorPolicyVersionStr, policyversion.CurrentVersion().String())
		} else if policyversion.Compare(policyversion.CurrentVersion(), sensorPolicyVersion) > 0 {
			// Downgrade all policies if necessary. If we can't downgrade one,
			// we likely can't convert any of them, so no need to spam the log.
			log.Infof("Downgrading %d policies from central's version %q to sensor's version %q",
				len(policies), policyversion.CurrentVersion().String(), sensorPolicyVersion.String())

			downgradedPolicies := make([]*storage.Policy, 0, len(policies))

			var downgradeErr error
			for _, p := range policies {
				cloned := p.CloneVT()
				downgradedPolicies = append(downgradedPolicies, cloned)
				err := policyversion.DowngradePolicyTo(cloned, sensorPolicyVersion)
				if downgradeErr == nil && err != nil {
					downgradeErr = err
				}
			}
			if downgradeErr != nil {
				log.Errorf("Policy downgrade failed: %v", downgradeErr)
			}

			policies = downgradedPolicies
		}
		// Otherwise, sensor supports the same version or newer as central
	}

	return &central.MsgToSensor{
		Msg: &central.MsgToSensor_PolicySync{
			PolicySync: &central.PolicySync{
				Policies: policies,
			},
		},
	}, nil
}

func (c *sensorConnection) getNetworkBaselineSyncMsg(ctx context.Context) (*central.MsgToSensor, error) {
	var networkBaselines []*storage.NetworkBaseline
	walkFn := func() error {
		networkBaselines = networkBaselines[:0]
		return c.networkBaselineMgr.Walk(ctx, func(baseline *storage.NetworkBaseline) error {
			if !baseline.GetLocked() {
				// Baseline not locked yet. No need to sync to sensor
				return nil
			}
			if baseline.GetClusterId() != c.clusterID {
				// Not a baseline of the cluster we are talking to
				return nil
			}
			networkBaselines = append(networkBaselines, baseline)
			return nil
		})
	}
	if err := pgutils.RetryIfPostgres(ctx, walkFn); err != nil {
		return nil, errors.Wrap(err, "could not list network baselines for Sensor connection")
	}
	return &central.MsgToSensor{
		Msg: &central.MsgToSensor_NetworkBaselineSync{
			NetworkBaselineSync: &central.NetworkBaselineSync{
				NetworkBaselines: networkBaselines,
			},
		},
	}, nil
}

func (c *sensorConnection) getBaselineSyncMsg(ctx context.Context) (*central.MsgToSensor, error) {
	var baselines []*storage.ProcessBaseline
	walkFn := func() error {
		baselines = baselines[:0]
		return c.baselineMgr.WalkAll(ctx, func(pw *storage.ProcessBaseline) error {
			if pw.GetUserLockedTimestamp() == nil {
				return nil
			}
			if pw.GetKey().GetClusterId() != c.clusterID {
				return nil
			}
			baselines = append(baselines, pw)
			return nil
		})
	}
	if err := pgutils.RetryIfPostgres(ctx, walkFn); err != nil {
		return nil, errors.Wrap(err, "could not list process baselines for Sensor connection")
	}
	return &central.MsgToSensor{
		Msg: &central.MsgToSensor_BaselineSync{
			BaselineSync: &central.BaselineSync{
				Baselines: baselines,
			},
		},
	}, nil
}

func (c *sensorConnection) getClusterConfigMsg(ctx context.Context) (*central.MsgToSensor, error) {
	cluster, exists, err := c.clusterMgr.GetCluster(ctx, c.clusterID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.Errorf("could not pull config for cluster %q because it does not exist", c.clusterID)
	}
	return &central.MsgToSensor{
		Msg: &central.MsgToSensor_ClusterConfig{
			ClusterConfig: &central.ClusterConfig{
				Config: cluster.GetDynamicConfig(),
			},
		},
	}, nil
}

func (c *sensorConnection) getScanConfigurationMsg(ctx context.Context) (*central.MsgToSensor, error) {
	_, exists, err := c.clusterMgr.GetCluster(ctx, c.clusterID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.Errorf("could not pull config for cluster %q because it does not exist", c.clusterID)
	}

	q := search.NewQueryBuilder().AddExactMatches(search.ClusterID, c.clusterID).ProtoQuery()
	scanConfigs, err := c.scanSettingDS.GetScanConfigurations(ctx, q)
	if err != nil {
		return nil, err
	}

	var reformattedConfigs []*central.ApplyComplianceScanConfigRequest
	for _, scanConfig := range scanConfigs {
		var profiles []string
		for _, profile := range scanConfig.GetProfiles() {
			profiles = append(profiles, profile.GetProfileName())
		}
		cron, err := schedule.ConvertToCronTab(scanConfig.GetSchedule())
		if err != nil {
			return nil, err
		}
		scanConfigRequest := central.ApplyComplianceScanConfigRequest{
			ScanRequest: &central.ApplyComplianceScanConfigRequest_UpdateScan{
				UpdateScan: &central.ApplyComplianceScanConfigRequest_UpdateScheduledScan{
					ScanSettings: &central.ApplyComplianceScanConfigRequest_BaseScanSettings{
						ScanName:               scanConfig.GetScanConfigName(),
						Profiles:               profiles,
						StrictNodeScan:         scanConfig.GetStrictNodeScan(),
						AutoApplyRemediations:  scanConfig.GetAutoApplyRemediations(),
						AutoUpdateRemediations: scanConfig.GetAutoUpdateRemediations(),
					},
					Cron: cron,
				},
			},
		}
		reformattedConfigs = append(reformattedConfigs, &scanConfigRequest)
	}

	return &central.MsgToSensor{
		Msg: &central.MsgToSensor_ComplianceRequest{
			ComplianceRequest: &central.ComplianceRequest{
				Request: &central.ComplianceRequest_SyncScanConfigs{
					SyncScanConfigs: &central.SyncComplianceScanConfigRequest{
						ScanConfigs: reformattedConfigs,
					},
				},
			},
		},
	}, nil
}

func (c *sensorConnection) getAuditLogSyncMsg(ctx context.Context) (*central.MsgToSensor, error) {
	cluster, exists, err := c.clusterMgr.GetCluster(ctx, c.clusterID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.Errorf("could not pull config for cluster %q because it does not exist", c.clusterID)
	}

	return &central.MsgToSensor{
		Msg: &central.MsgToSensor_AuditLogSync{
			AuditLogSync: &central.AuditLogSync{
				NodeAuditLogFileStates: cluster.GetAuditLogState(),
			},
		},
	}, nil
}

func (c *sensorConnection) getDelegatedRegistryConfigMsg(ctx context.Context) (*central.MsgToSensor, error) {
	config, exists, err := c.delegatedRegistryConfigMgr.GetConfig(ctx)
	if err != nil {
		return nil, err
	}

	if !exists {
		// Sensor's ProcessMessage handler will ignore a nil config, so send nothing
		log.Debugf("Not sending nil delegated registry config to cluster %q", c.clusterID)
		return nil, nil
	}

	return &central.MsgToSensor{
		Msg: &central.MsgToSensor_DelegatedRegistryConfig{
			DelegatedRegistryConfig: delegatedRegistryConfigConvert.StorageToInternalAPI(config),
		},
	}, nil
}

// getImageIntegrationMsg builds a MsgToSensor containing registry integrations that should
// be sent to sensor. Returns nil if are no eligible integrations.
func (c *sensorConnection) getImageIntegrationMsg(ctx context.Context) (*central.MsgToSensor, error) {
	iis, err := c.imageIntegrationMgr.GetImageIntegrations(ctx, &v1.GetImageIntegrationsRequest{})
	if err != nil {
		return nil, err
	}

	var imageIntegrations []*storage.ImageIntegration
	for _, ii := range iis {
		if !imageintegration.ValidForSync(ii) {
			continue
		}

		imageIntegrations = append(imageIntegrations, ii)
		log.Debugf("Sending registry integration %q (%v) to cluster %q", ii.GetName(), ii.GetId(), c.clusterID)
	}

	if len(imageIntegrations) == 0 {
		return nil, nil
	}

	return &central.MsgToSensor{
		Msg: &central.MsgToSensor_ImageIntegrations{
			ImageIntegrations: &central.ImageIntegrations{
				UpdatedIntegrations: imageIntegrations,
				// On initial/repeat connections to Sensor any previous stored image integrations
				// should be replaced by these (potentially) new ones.
				Refresh: true,
			},
		},
	}, nil
}

func (c *sensorConnection) Run(ctx context.Context, server central.SensorService_CommunicateServer, connectionCapabilities set.Set[centralsensor.SensorCapability]) error {
	// Synchronously send the config to ensure syncing before Sensor marks the connection as Central reachable
	msg, err := c.getClusterConfigMsg(ctx)
	if err != nil {
		return errors.Wrapf(err, "unable to get cluster config for %q", c.clusterID)
	}

	if err := server.Send(msg); err != nil {
		return errors.Wrapf(err, "unable to sync config to cluster %q", c.clusterID)
	}

	if connectionCapabilities.Contains(centralsensor.SensorDetectionCap) {
		msg, err = c.getPolicySyncMsg(ctx)
		if err != nil {
			return errors.Wrapf(err, "unable to get policy sync msg for %q", c.clusterID)
		}
		if err := server.Send(msg); err != nil {
			return errors.Wrapf(err, "unable to sync initial policies to cluster %q", c.clusterID)
		}

		msg, err = c.getBaselineSyncMsg(ctx)
		if err != nil {
			return errors.Wrapf(err, "unable to get process baseline sync msg for %q", c.clusterID)
		}
		if err := server.Send(msg); err != nil {
			return errors.Wrapf(err, "unable to sync initial process baselines to cluster %q", c.clusterID)
		}

		msg, err = c.getNetworkBaselineSyncMsg(ctx)
		if err != nil {
			return errors.Wrapf(err, "unable to get network baseline sync msg for %q", c.clusterID)
		}
		if err := server.Send(msg); err != nil {
			return errors.Wrapf(err, "unable to sync initial network baselines to cluster %q", c.clusterID)
		}

	}

	if features.SensorReconciliationOnReconnect.Enabled() && connectionCapabilities.Contains(centralsensor.SendDeduperStateOnReconnect) {
		// Sensor is capable of doing the reconciliation by itself if receives the hashes from central.
		log.Infof("Sensor (%s) can do client reconciliation: sending deduper state", c.clusterID)

		// Send hashes to sensor
		maxEntries := env.GetMaxDeduperEntriesPerMessage()
		successfulHashes := c.hashDeduper.GetSuccessfulHashes()
		// If there are no hashes we send the empty map
		if len(successfulHashes) == 0 {
			if err := c.sendDeduperState(server, successfulHashes, 1, 1); err != nil {
				return err
			}
		}
		total := int32(math.Ceil(float64(len(successfulHashes)) / float64(maxEntries)))
		log.Debugf("DeduperState total number of hashes %d chunk size %d number of chunks to send %d", len(successfulHashes), maxEntries, total)
		var current int32 = 1
		payload := make(map[string]uint64)
		for k, v := range successfulHashes {
			payload[k] = v
			if len(payload) == int(maxEntries) {
				if err := c.sendDeduperState(server, payload, current, total); err != nil {
					return err
				}
				payload = make(map[string]uint64)
				current++
			}
		}
		if len(payload) > 0 {
			if err := c.sendDeduperState(server, payload, current, total); err != nil {
				return err
			}
		}
		log.Infof("Successfully sent deduper state to sensor (%s)", c.clusterID)
	} else {
		log.Infof("Sensor (%s) cannot receive deduper state", c.clusterID)
	}

	// Any messages after this will be processed by Sensor components and can go in any order.
	// Don't change the order of any messages above!

	if connectionCapabilities.Contains(centralsensor.DelegatedRegistryCap) {
		// Sync delegated registry config.
		msg, err := c.getDelegatedRegistryConfigMsg(ctx)
		if err != nil {
			return errors.Wrapf(err, "unable to get delegated registry config msg for %q", c.clusterID)
		}
		if msg != nil {
			if err := server.Send(msg); err != nil {
				return errors.Wrapf(err, "unable to sync initial delegated registry config to cluster %q", c.clusterID)
			}

			log.Infof("Sent delegated registry config %q to cluster %q", msg.GetDelegatedRegistryConfig(), c.clusterID)
		}

		// Sync integrations.
		msg, err = c.getImageIntegrationMsg(ctx)
		if err != nil {
			return errors.Wrapf(err, "unable to get image integrations msg for %q", c.clusterID)
		}
		if msg != nil {
			if err := server.Send(msg); err != nil {
				return errors.Wrapf(err, "unable to sync initial image integrations to cluster %q", c.clusterID)
			}

			log.Infof("Sent %d image integrations to cluster %q", len(msg.GetImageIntegrations().GetUpdatedIntegrations()), c.clusterID)
		}
	}

	go c.runSend(server)

	// Trigger initial network graph external sources sync. Network graph external sources capability is added to sensor only if the feature is enabled.
	if connectionCapabilities.Contains(centralsensor.NetworkGraphExternalSrcsCap) {
		if err := c.NetworkEntities().SyncNow(ctx); err != nil {
			log.Errorf("Unable to sync initial external network entities to cluster %q: %v", c.clusterID, err)
		}
	}

	if connectionCapabilities.Contains(centralsensor.AuditLogEventsCap) {
		msg, err := c.getAuditLogSyncMsg(ctx)
		if err != nil {
			return errors.Wrapf(err, "unable to get audit log file state sync msg for %q", c.clusterID)
		}

		// Send the audit log state to sensor even if the user has it disabled (that's set in dynamic config). When enabled, sensor will use it correctly
		if err := server.Send(msg); err != nil {
			return errors.Wrapf(err, "unable to sync audit log file state to cluster %q", c.clusterID)
		}
	}

	if features.ComplianceEnhancements.Enabled() && connectionCapabilities.Contains(centralsensor.ComplianceV2ScanConfigSync) {
		log.Infof("Sensor (%s) can sync scan configuration: sending sending scan configuration", c.clusterID)
		scanMsg, err := c.getScanConfigurationMsg(ctx)
		if err != nil {
			return errors.Wrapf(err, "unable to get scan config for %q", c.clusterID)
		}
		log.Infof("sending %d scan configs to sensor", len(scanMsg.GetComplianceRequest().GetSyncScanConfigs().GetScanConfigs()))
		if err := server.Send(scanMsg); err != nil {
			return errors.Wrapf(err, "unable to sync config to cluster %q", c.clusterID)
		}
	}

	metrics.IncrementSensorConnect(c.clusterID, c.sensorHello.GetSensorState().String())

	c.runRecv(ctx, server)
	return c.stopSig.Err()
}

func (c *sensorConnection) ClusterID() string {
	return c.clusterID
}

func (c *sensorConnection) HasCapability(capability centralsensor.SensorCapability) bool {
	return c.capabilities.Contains(capability)
}

func (c *sensorConnection) ObjectsDeletedByReconciliation() (map[string]int, bool) {
	return c.sensorEventHandler.reconciliationMap.DeletedElementsByType()
}

func (c *sensorConnection) CheckAutoUpgradeSupport() error {
	if c.sensorHello.GetHelmManagedConfigInit() != nil {
		switch c.sensorHello.GetHelmManagedConfigInit().GetManagedBy() {
		case storage.ManagerType_MANAGER_TYPE_HELM_CHART:
			return errors.New("Helm controls the secured cluster version.")
		case storage.ManagerType_MANAGER_TYPE_KUBERNETES_OPERATOR:
			return errors.New("Operator controls the secured cluster version.")
		}
	}
	return nil
}

func (c *sensorConnection) SensorVersion() string {
	return c.sensorHello.GetSensorVersion()
}

func (c *sensorConnection) sendDeduperState(server central.SensorService_CommunicateServer, payload map[string]uint64, current, total int32) error {
	deduperMessage := &central.MsgToSensor{Msg: &central.MsgToSensor_DeduperState{
		DeduperState: &central.DeduperState{
			ResourceHashes: payload,
			Current:        current,
			Total:          total,
		},
	}}

	log.Infof("Sending %d hashes (Size=%d), current chunk: %d, total: %d", len(payload), deduperMessage.SizeVT(), current, total)

	err := server.Send(deduperMessage)
	if err != nil {
		log.Errorf("Central wasn't able to send deduper state to sensor (%s): %s", c.clusterID, err)
		return errors.Wrap(err, "unable to sync deduper state")
	}
	return nil
}
