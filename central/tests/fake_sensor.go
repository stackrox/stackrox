//go:build sql_integration

package tests

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// FakeSensor is a lightweight gRPC client that connects to a test Central.
type FakeSensor struct {
	t         *testing.T
	name      string
	clusterID string
	stream    central.SensorService_CommunicateClient
	conn      *grpc.ClientConn
	cancel    context.CancelFunc
	recvC     chan *central.MsgToSensor
}

// NewFakeSensor creates a FakeSensor with the given cluster name.
func NewFakeSensor(t *testing.T, name string) *FakeSensor {
	return &FakeSensor{
		t:     t,
		name:  name,
		recvC: make(chan *central.MsgToSensor, 64),
	}
}

// ClusterID returns the ID assigned to this sensor's cluster after connecting.
func (s *FakeSensor) ClusterID() string {
	return s.clusterID
}

// ConnectTo pre-creates a cluster, dials the test Central over bufconn, and
// completes the SensorHello/CentralHello handshake.
func (s *FakeSensor) ConnectTo(c *CentralTestServer) {
	// Pre-create the cluster on first connection; reuse on reconnect.
	if s.clusterID == "" {
		clusterID, err := c.ClusterDS.AddCluster(allAccessCtx(), &storage.Cluster{
			Name:      s.name,
			MainImage: "quay.io/stackrox-io/main",
		})
		require.NoError(s.t, err)
		s.clusterID = clusterID
	}

	// Dial over bufconn.
	var err error
	s.conn, err = grpc.NewClient("passthrough:///bufconn",
		grpc.WithContextDialer(bufconnDialer(c.listener)),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(s.t, err)
	s.t.Cleanup(func() { s.Disconnect() })

	// Open Communicate stream with sensor hello metadata + cluster ID.
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel

	md := metadata.Pairs(
		centralsensor.SensorHelloMetadataKey, "true",
		testClusterIDKey, s.clusterID,
	)
	ctx = metadata.NewOutgoingContext(ctx, md)

	client := central.NewSensorServiceClient(s.conn)
	s.stream, err = client.Communicate(ctx)
	require.NoError(s.t, err)

	// Send SensorHello.
	err = s.stream.Send(&central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Hello{
			Hello: &central.SensorHello{
				SensorVersion: "test",
				DeploymentIdentification: &storage.SensorDeploymentIdentification{
					SystemNamespaceId:  s.name,
					DefaultNamespaceId: s.name,
					AppNamespace:       "stackrox",
				},
			},
		},
	})
	require.NoError(s.t, err)

	// Receive CentralHello.
	msg, err := s.stream.Recv()
	require.NoError(s.t, err)
	hello := msg.GetHello()
	require.NotNil(s.t, hello, "expected CentralHello, got %T", msg.GetMsg())

	// Drain initial sync messages (at least ClusterConfig).
	s.recvC = make(chan *central.MsgToSensor, 64)
	s.drainInitialSync()

	// Background receiver.
	go s.recvLoop()
}

func (s *FakeSensor) drainInitialSync() {
	for {
		msg, err := s.stream.Recv()
		if err != nil {
			s.t.Fatalf("error draining initial sync: %v", err)
		}
		// ClusterConfig is always the first (and possibly only) sync message
		// when the sensor advertises no capabilities.
		if msg.GetClusterConfig() != nil {
			return
		}
	}
}

func (s *FakeSensor) recvLoop() {
	for {
		msg, err := s.stream.Recv()
		if err != nil {
			close(s.recvC)
			return
		}
		s.recvC <- msg
	}
}

// Disconnect terminates the current sensor connection. The FakeSensor can be
// reconnected by calling ConnectTo again.
func (s *FakeSensor) Disconnect() {
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
	if s.conn != nil {
		s.conn.Close()
		s.conn = nil
	}
}

// Inject sends MsgFromSensor messages through the full production pipeline.
func (s *FakeSensor) Inject(msgs ...*central.MsgFromSensor) {
	for _, msg := range msgs {
		require.NoError(s.t, s.stream.Send(msg))
	}
}

// NewSensorEvent wraps a SensorEvent in a MsgFromSensor. The caller sets Id,
// Action, and the Resource oneof on the event.
func NewSensorEvent(event *central.SensorEvent) *central.MsgFromSensor {
	return &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: event,
		},
	}
}
