//go:build sql_integration

package tests

import (
	"context"
	"net"
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
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
	// Pre-create the cluster so Central can look it up by ID.
	clusterID, err := c.ClusterDS.AddCluster(allAccessCtx(), &storage.Cluster{
		Name:      s.name,
		MainImage: "quay.io/stackrox-io/main",
	})
	require.NoError(s.t, err)
	s.clusterID = clusterID

	// Dial over bufconn.
	s.conn, err = grpc.NewClient("passthrough:///bufconn",
		grpc.WithContextDialer(bufconnDialer(c.listener)),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(s.t, err)
	s.t.Cleanup(func() { s.conn.Close() })

	// Open Communicate stream with sensor hello metadata + cluster ID.
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.t.Cleanup(cancel)

	md := metadata.Pairs(
		centralsensor.SensorHelloMetadataKey, "true",
		testClusterIDKey, clusterID,
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

// Inject sends MsgFromSensor messages through the full production pipeline.
func (s *FakeSensor) Inject(msgs ...*central.MsgFromSensor) {
	for _, msg := range msgs {
		require.NoError(s.t, s.stream.Send(msg))
	}
}

func bufconnDialer(lis *bufconn.Listener) func(context.Context, string) (net.Conn, error) {
	return func(ctx context.Context, _ string) (net.Conn, error) {
		return lis.DialContext(ctx)
	}
}
