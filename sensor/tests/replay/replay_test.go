package replay

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"io"
	"net"
	"os"
	"testing"
	"time"

	"github.com/golang/protobuf/jsonpb"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stackrox/rox/pkg/utils"
	centralDebug "github.com/stackrox/rox/sensor/debugger/central"
	"github.com/stackrox/rox/sensor/debugger/k8s"
	"github.com/stackrox/rox/sensor/debugger/message"
	"github.com/stackrox/rox/sensor/kubernetes/listener/resources"
	"github.com/stackrox/rox/sensor/kubernetes/sensor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

func TestReplayEvents(t *testing.T) {
	suite.Run(t, new(ReplayEventsSuite))
}

type ReplayEventsSuite struct {
	suite.Suite
	fakeClient  *k8s.ClientSet
	fakeCentral *centralDebug.FakeService
	envIsolator *envisolator.EnvIsolator
}

var _ suite.SetupTestSuite = (*ReplayEventsSuite)(nil)
var _ suite.TearDownTestSuite = (*ReplayEventsSuite)(nil)

func (suite *ReplayEventsSuite) SetupTest() {
	suite.fakeClient = k8s.MakeFakeClient()

	suite.envIsolator = envisolator.NewEnvIsolator(suite.T())
	suite.envIsolator.Setenv("ROX_MTLS_CERT_FILE", "../../../tools/local-sensor/certs/cert.pem")
	suite.envIsolator.Setenv("ROX_MTLS_KEY_FILE", "../../../tools/local-sensor/certs/key.pem")
	suite.envIsolator.Setenv("ROX_MTLS_CA_FILE", "../../../tools/local-sensor/certs/caCert.pem")
	suite.envIsolator.Setenv("ROX_MTLS_CA_KEY_FILE", "../../../tools/local-sensor/certs/caKey.pem")

	suite.fakeCentral = centralDebug.MakeFakeCentralWithInitialMessages(
		message.SensorHello("1234"),
		message.ClusterConfig(),
		message.PolicySync([]*storage.Policy{}),
		message.BaselineSync([]*storage.ProcessBaseline{}))
}

func (suite *ReplayEventsSuite) TearDownTest() {
	suite.envIsolator.RestoreAll()
}

func createConnectionAndStartServer(fakeCentral *centralDebug.FakeService) (*grpc.ClientConn, *centralDebug.FakeService, func()) {
	buffer := 1024 * 1024
	listener := bufconn.Listen(buffer)

	server := grpc.NewServer()
	central.RegisterSensorServiceServer(server, fakeCentral)

	go func() {
		utils.IgnoreError(func() error {
			return server.Serve(listener)
		})
	}()

	conn, err := grpc.DialContext(context.Background(), "", grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
		return listener.Dial()
	}), grpc.WithTransportCredentials(insecure.NewCredentials()))

	if err != nil {
		panic(err)
	}

	closeF := func() {
		utils.IgnoreError(listener.Close)
		server.Stop()
	}

	return conn, fakeCentral, closeF
}

var _ io.Writer = (*TraceWriterWithChannel)(nil)

type TraceWriterWithChannel struct {
	destinationChannel chan *central.SensorEvent
	// mu mutex to avoid multiple goroutines writing at the same time
	mu sync.Mutex
}

func (tw *TraceWriterWithChannel) Write(b []byte) (int, error) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	msg := resources.InformerK8sMsg{}
	if err := json.Unmarshal(b, &msg); err != nil {
		return 0, err
	}
	for _, e := range msg.EventsOutput {
		event := &central.SensorEvent{}
		if err := jsonpb.UnmarshalString(e, event); err != nil {
			return 0, err
		}
		tw.destinationChannel <- event
	}
	return 0, nil
}

func (suite *ReplayEventsSuite) Test_ReplayEvents() {
	conn, spyCentral, shutdownFakeServer := createConnectionAndStartServer(suite.fakeCentral)
	defer shutdownFakeServer()
	fakeConnectionFactory := centralDebug.MakeFakeConnectionFactory(conn)

	ackChannel := make(chan *central.SensorEvent)
	defer close(ackChannel)
	writer := &TraceWriterWithChannel{
		destinationChannel: ackChannel,
	}
	// Depending on the size of the file the re-sync time may need to be increased.
	// This is because we need to wait for the event outputs of each kubernetes event before sending the next.
	// If we receive re-sync events before we finish processing all the events, we might run into unknown behaviour
	resyncTime := 1 * time.Second
	s, err := sensor.CreateSensor(sensor.ConfigWithDefaults().
		WithK8sClient(suite.fakeClient).
		WithLocalSensor(true).
		WithResyncPeriod(resyncTime).
		WithCentralConnectionFactory(fakeConnectionFactory).
		WithTraceWriter(writer))

	if err != nil {
		panic(err)
	}

	go s.Start()
	defer s.Stop()

	spyCentral.ConnectionStarted.Wait()

	cases := map[string]struct {
		k8sEventsFile    string
		sensorOutputFile string
	}{
		"Safety net test": {
			k8sEventsFile:    "data/trace.jsonl",
			sensorOutputFile: "data/central-out.bin",
		},
	}
	for name, c := range cases {
		suite.T().Run(name, func(t *testing.T) {
			suite.fakeCentral.ClearReceivedBuffer()
			receivedMessagesCh := make(chan *central.MsgFromSensor, 10)
			suite.fakeCentral.OnMessage(func(msg *central.MsgFromSensor) {
				receivedMessagesCh <- msg
			})
			eventsReader := &k8s.TraceReader{
				Source: c.k8sEventsFile,
			}
			err = eventsReader.Init()
			if err != nil {
				panic(err)
			}
			fm := k8s.FakeEventsManager{
				AckChannel: ackChannel,
				Mode:       k8s.ChannelAck,
				Client:     suite.fakeClient,
				Reader:     eventsReader,
			}
			_, errCh := fm.CreateEvents(context.Background())
			// Wait for all the events to be processed
			err = <-errCh
			if err != nil {
				panic(err)
			}
			// We continue to read the ackChannel to avoid blocking
			ctx, cancelFunc := context.WithCancel(context.Background())
			go func() {
				for {
					select {
					case <-ackChannel:
					case <-ctx.Done():
						return
					}
				}
			}()
			// Wait for the re-sync to happen
			time.Sleep(5 * resyncTime)
			allEvents := suite.fakeCentral.GetAllMessages()
			// Read the sensorOutputFile containing the expected sensor's output
			expectedEvents, err := readSensorOutputFile(c.sensorOutputFile)
			if err != nil {
				panic(err)
			}
			expectedDeployments := getAllDeployments(expectedEvents)
			receivedDeployments := getAllDeployments(allEvents)
			assert.Equal(t, len(expectedDeployments), len(receivedDeployments))
			for id, exp := range expectedDeployments {
				if e, ok := receivedDeployments[id]; !ok {
					t.Error("Deployment not found")
				} else {
					assert.Equal(t, exp.GetDeployment().GetServiceAccountPermissionLevel(), e.GetDeployment().GetServiceAccountPermissionLevel())
					assert.Equal(t, exp.GetDeployment().GetPorts(), e.GetDeployment().GetPorts())
					assert.Equal(t, exp.GetDeployment().GetName(), e.GetDeployment().GetName())
					assert.Equal(t, exp.GetDeployment().GetNamespace(), e.GetDeployment().GetNamespace())
					assert.Equal(t, exp.GetDeployment().GetLabels(), e.GetDeployment().GetLabels())
					assert.Equal(t, exp.GetDeployment().GetPodLabels(), e.GetDeployment().GetPodLabels())
					assert.Equal(t, exp.GetDeployment().GetImagePullSecrets(), e.GetDeployment().GetImagePullSecrets())
				}
			}
			expectedPods := getAllPods(expectedEvents)
			receivedPods := getAllPods(allEvents)
			assert.Equal(t, len(expectedPods), len(receivedPods))
			for id, exp := range expectedPods {
				if e, ok := receivedPods[id]; !ok {
					t.Error("Pod not found")
				} else {
					assert.Equal(t, exp.GetPod().GetDeploymentId(), e.GetPod().GetDeploymentId())
					assert.Equal(t, exp.GetPod().GetName(), e.GetPod().GetName())
					assert.Equal(t, exp.GetPod().GetNamespace(), e.GetPod().GetNamespace())
				}
			}
			cancelFunc()
		})
	}
}

func readSensorOutputFile(fname string) ([]*central.MsgFromSensor, error) {
	file, err := os.OpenFile(fname, os.O_RDONLY, 0644)
	if err != nil {
		return nil, err
	}
	var offset int64
	content := make([][]byte, 0)
	for {
		buf := make([]byte, 4)
		_, err = file.ReadAt(buf, offset)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		size := binary.LittleEndian.Uint32(buf)
		offset += 4
		item := make([]byte, size)
		_, err = file.ReadAt(item, offset)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		content = append(content, item)
		offset += int64(size)
	}
	var msgs []*central.MsgFromSensor
	for _, it := range content {
		m := &central.MsgFromSensor{}
		if err = m.Unmarshal(it); err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	return msgs, nil
}

func getAllDeployments(messages []*central.MsgFromSensor) map[string]*central.SensorEvent {
	events := make(map[string]*central.SensorEvent)
	for _, msg := range messages {
		event := msg.GetEvent()
		if event.GetDeployment() != nil {
			if event.GetDeployment().GetId() != "" {
				events[event.GetDeployment().GetId()] = event
			}
		}
	}
	return events
}

func getAllPods(messages []*central.MsgFromSensor) map[string]*central.SensorEvent {
	events := make(map[string]*central.SensorEvent)
	for _, msg := range messages {
		event := msg.GetEvent()
		if event.GetPod() != nil {
			if event.GetPod().GetId() != "" {
				events[event.GetPod().GetId()] = event
			}
		}
	}
	return events
}
