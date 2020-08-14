package connection

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/scrape"
	"github.com/stackrox/rox/central/sensor/networkpolicies"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/central/sensor/telemetry"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/reflectutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sync"
	"google.golang.org/grpc/metadata"
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
	telemetryCtrl       telemetry.Controller

	sensorEventHandler *sensorEventHandler

	queues      map[string]*dedupingQueue
	queuesMutex sync.Mutex

	eventPipeline pipeline.ClusterPipeline

	clusterMgr   ClusterManager
	policyMgr    PolicyManager
	whitelistMgr WhitelistManager

	capabilities centralsensor.SensorCapabilitySet
}

func newConnection(ctx context.Context, clusterID string, eventPipeline pipeline.ClusterPipeline, clusterMgr ClusterManager, policyMgr PolicyManager, whitelistMgr WhitelistManager) *sensorConnection {
	conn := &sensorConnection{
		stopSig:       concurrency.NewErrorSignal(),
		stoppedSig:    concurrency.NewErrorSignal(),
		sendC:         make(chan *central.MsgToSensor),
		eventPipeline: eventPipeline,
		queues:        make(map[string]*dedupingQueue),

		clusterID:    clusterID,
		clusterMgr:   clusterMgr,
		policyMgr:    policyMgr,
		whitelistMgr: whitelistMgr,

		capabilities: centralsensor.ExtractCapsFromContext(ctx),
	}

	// Need a reference to conn for injector
	conn.sensorEventHandler = newSensorEventHandler(eventPipeline, conn, &conn.stopSig)
	conn.scrapeCtrl = scrape.NewController(conn, &conn.stopSig)
	conn.networkPoliciesCtrl = networkpolicies.NewController(conn, &conn.stopSig)
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
// mutex-protected `c.queues` map (and created if exists), and afterwards stored in the `queues` map
// if non-nil.
// The envisioned use for this is that a caller invoking `multiplexedPush` repeatedly will maintain
// an exclusively used (i.e., not requiring protection via a mutex) map, that will automatically be
// populated with a subset of the entries from `c.queues`. This avoids mutex lock acquisitions for every
// invocation of `multiplexedPush` with a previously seen (from the perspective of the caller)
// event type.
func (c *sensorConnection) multiplexedPush(ctx context.Context, msg *central.MsgFromSensor, queues map[string]*dedupingQueue) {
	typ := reflectutils.Type(msg.Msg)
	queue := queues[typ]
	if queue == nil {
		concurrency.WithLock(&c.queuesMutex, func() {
			queue = c.queues[typ]
			if queue == nil {
				queue = newDedupingQueue(stripTypePrefix(typ))
				go c.handleMessages(ctx, queue)
				c.queues[typ] = queue
			}
		})
		if queues != nil {
			queues[typ] = queue
		}
	}
	queue.push(msg)
}

func (c *sensorConnection) runRecv(ctx context.Context, grpcServer central.SensorService_CommunicateServer) {
	queues := make(map[string]*dedupingQueue)
	for !c.stopSig.IsDone() {
		msg, err := grpcServer.Recv()
		if err != nil {
			c.stopSig.SignalWithError(errors.Wrap(err, "recv error"))
			return
		}

		c.multiplexedPush(ctx, msg, queues)
	}
}

func (c *sensorConnection) handleMessages(ctx context.Context, queue *dedupingQueue) {
	for msg := queue.pullBlocking(&c.stopSig); msg != nil; msg = queue.pullBlocking(&c.stopSig) {
		if err := c.handleMessage(ctx, msg); err != nil {
			log.Errorf("Error handling sensor message: %v", err)
		}
	}
	c.eventPipeline.OnFinish(c.clusterID)
	c.stoppedSig.SignalWithError(c.stopSig.Err())
}

func (c *sensorConnection) runSend(server central.SensorService_CommunicateServer) {
	for !c.stopSig.IsDone() {
		select {
		case <-c.stopSig.Done():
			return
		case <-server.Context().Done():
			c.stopSig.SignalWithError(errors.Wrap(server.Context().Err(), "context error"))
			return
		case msg := <-c.sendC:
			if err := server.Send(msg); err != nil {
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
	c.multiplexedPush(sac.WithAllAccess(context.Background()), msg, nil)
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
		return c.telemetryCtrl.ProcessTelemetryDataResponse(m.TelemetryDataResponse)
	case *central.MsgFromSensor_Event:
		// Special case the reprocess deployment because its fields are already set
		if msg.GetEvent().GetReprocessDeployment() != nil {
			c.sensorEventHandler.addMultiplexed(ctx, msg)
			return nil
		}
		// Only dedupe on non-creates
		if msg.GetEvent().GetAction() != central.ResourceAction_CREATE_RESOURCE {
			msg.DedupeKey = msg.GetEvent().GetId()
		}
		// Set the hash key for all values
		msg.HashKey = msg.GetEvent().GetId()

		c.sensorEventHandler.addMultiplexed(ctx, msg)
		return nil
	}
	return c.eventPipeline.Run(ctx, msg, c)
}

func (c *sensorConnection) getPolicySyncMsg(ctx context.Context) (*central.MsgToSensor, error) {
	policies, err := c.policyMgr.GetAllPolicies(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "error getting policies for initial sync")
	}
	return &central.MsgToSensor{
		Msg: &central.MsgToSensor_PolicySync{
			PolicySync: &central.PolicySync{
				Policies: policies,
			},
		},
	}, nil
}

func (c *sensorConnection) getWhitelistSyncMsg(ctx context.Context) (*central.MsgToSensor, error) {
	var whitelists []*storage.ProcessWhitelist
	err := c.whitelistMgr.WalkAll(ctx, func(pw *storage.ProcessWhitelist) error {
		if pw.GetUserLockedTimestamp() == nil {
			return nil
		}
		if pw.GetKey().GetClusterId() != c.clusterID {
			return nil
		}
		whitelists = append(whitelists, pw)
		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "could not list process baselines for Sensor connection")
	}
	return &central.MsgToSensor{
		Msg: &central.MsgToSensor_WhitelistSync{
			WhitelistSync: &central.WhitelistSync{
				Whitelists: whitelists,
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

func (c *sensorConnection) Run(ctx context.Context, server central.SensorService_CommunicateServer, connectionCapabilities centralsensor.SensorCapabilitySet) error {
	if err := server.SendHeader(metadata.MD{}); err != nil {
		return errors.Wrap(err, "sending initial metadata")
	}

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

		msg, err = c.getWhitelistSyncMsg(ctx)
		if err != nil {
			return errors.Wrapf(err, "unable to get process baseline sync msg for %q", c.clusterID)
		}
		if err := server.Send(msg); err != nil {
			return errors.Wrapf(err, "unable to sync initial process baselines to cluster %q", c.clusterID)
		}
	}

	go c.runSend(server)

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
