package replay

import (
	"context"
	"encoding/binary"
	"io"
	"net"
	"os"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
	centralDebug "github.com/stackrox/rox/sensor/debugger/central"
	"github.com/stackrox/rox/sensor/debugger/k8s"
	"github.com/stackrox/rox/sensor/debugger/message"
	"github.com/stackrox/rox/sensor/kubernetes/sensor"
	"github.com/stackrox/rox/sensor/testutils"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

const (
	// Depending on the size of the file the re-sync time may need to be increased.
	// This is because we need to wait for the event outputs of each kubernetes event before sending the next.
	// If we receive re-sync events before we finish processing all the events, we might run into unknown behaviour
	resyncTime = 1 * time.Second
)

// Suite defines the interface to be used with these helper functions
type Suite interface {
	GetFakeClient() *k8s.ClientSet
	SetFakeClient(*k8s.ClientSet)
	GetFakeCentral() *centralDebug.FakeService
	SetFakeCentral(*centralDebug.FakeService)
	GetT() *testing.T
}

// SetupTest sets up the k8s fake client and the central fake service
func SetupTest(suite Suite) {
	suite.SetFakeClient(k8s.MakeFakeClient())

	suite.GetT().Setenv("ROX_MTLS_CERT_FILE", "../../../../tools/local-sensor/certs/cert.pem")
	suite.GetT().Setenv("ROX_MTLS_KEY_FILE", "../../../../tools/local-sensor/certs/key.pem")
	suite.GetT().Setenv("ROX_MTLS_CA_FILE", "../../../../tools/local-sensor/certs/caCert.pem")
	suite.GetT().Setenv("ROX_MTLS_CA_KEY_FILE", "../../../../tools/local-sensor/certs/caKey.pem")

	policies, err := testutils.GetPoliciesFromFile("../data/policies.json")
	if err != nil {
		panic(err)
	}
	suite.SetFakeCentral(centralDebug.MakeFakeCentralWithInitialMessages(
		message.SensorHello("00000000-0000-4000-A000-000000000000"),
		message.ClusterConfig(),
		message.PolicySync(policies),
		message.BaselineSync([]*storage.ProcessBaseline{}),
		message.NetworkBaselineSync([]*storage.NetworkBaseline{})),
	)
}

// StartTest starts a sensor instance for the test
func StartTest(suite Suite) *TraceWriterWithChannel {
	suite.GetT().Setenv("ROX_RESYNC_DISABLED", "true")
	conn, spyCentral, _ := createConnectionAndStartServer(suite.GetFakeCentral())
	fakeConnectionFactory := centralDebug.MakeFakeConnectionFactory(conn)

	ackChannel := make(chan *central.SensorEvent)
	writer := &TraceWriterWithChannel{
		destinationChannel: ackChannel,
		enabled:            true,
	}
	s, err := sensor.CreateSensor(sensor.ConfigWithDefaults().
		WithK8sClient(suite.GetFakeClient()).
		WithLocalSensor(true).
		WithResyncPeriod(resyncTime).
		WithCentralConnectionFactory(fakeConnectionFactory).
		WithTraceWriter(writer))

	if err != nil {
		panic(err)
	}

	go s.Start()

	spyCentral.ConnectionStarted.Wait()

	return writer
}

// RunReplayTest this is the body of the replay test.
// It uses the FakeEventManager to create a bunch of fake k8s events
// and then compares the messages sent to central with a given
// pre-recorded output file.
func RunReplayTest(t *testing.T, suite Suite, writer *TraceWriterWithChannel, k8sEventsFile, sensorOutputFile string) {
	suite.GetFakeCentral().ClearReceivedBuffer()
	eventsReader := &k8s.TraceReader{
		Source: k8sEventsFile,
	}
	err := eventsReader.Init()
	if err != nil {
		panic(err)
	}
	fm := k8s.FakeEventsManager{
		AckChannel: writer.destinationChannel,
		Mode:       k8s.ChannelAck,
		Client:     suite.GetFakeClient(),
		Reader:     eventsReader,
	}
	writer.enable()
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
			case <-writer.destinationChannel:
			case <-ctx.Done():
				return
			}
		}
	}()
	writer.disable()
	// Wait for the re-sync to happen
	time.Sleep(10 * resyncTime)
	allEvents := suite.GetFakeCentral().GetAllMessages()
	// Read the sensorOutputFile containing the expected sensor's output
	expectedEvents, err := readSensorOutputFile(sensorOutputFile)
	if err != nil {
		panic(err)
	}
	expectedAlerts := getAlerts(expectedEvents)
	receivedAlerts := getAlerts(allEvents)
	for id, expectedAlertEvent := range expectedAlerts {
		if receivedAlertEvent, ok := receivedAlerts[id]; !ok {
			t.Error("Deployment Alert Event not found. Expected alert event: ", expectedAlertEvent)
		} else {
			assert.Equal(t, len(expectedAlertEvent), len(receivedAlertEvent))
			for alertID, exp := range expectedAlertEvent {
				if a, ok := receivedAlertEvent[alertID]; !ok {
					t.Error("Alert not found. Expected alert: ", exp)
				} else {
					assert.Equal(t, exp.GetState(), a.GetState())
				}
			}
		}
	}
	expectedDeployments := getLastStateFromDeployments(expectedEvents)
	receivedDeployments := getLastStateFromDeployments(allEvents)
	for id, exp := range expectedDeployments {
		if e, ok := receivedDeployments[id]; !ok {
			t.Error("Deployment not found. Expected Deployment: ", exp)
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
	expectedPods := getLastStateFromPods(expectedEvents)
	receivedPods := getLastStateFromPods(allEvents)
	for id, exp := range expectedPods {
		if e, ok := receivedPods[id]; !ok {
			t.Error("Pod not found. Expected Pod: ", exp)
		} else {
			assert.Equal(t, exp.GetPod().GetDeploymentId(), e.GetPod().GetDeploymentId())
			assert.Equal(t, exp.GetPod().GetName(), e.GetPod().GetName())
			assert.Equal(t, exp.GetPod().GetNamespace(), e.GetPod().GetNamespace())
		}
	}
	cancelFunc()
	if err := fm.DeleteAllResources(); err != nil {
		panic(err)
	}
	time.Sleep(5 * resyncTime)
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

// TraceWriterWithChannel writes sensor-to-central events to a channel
type TraceWriterWithChannel struct {
	destinationChannel chan *central.SensorEvent
	// mu mutex to avoid multiple goroutines writing at the same time
	mu sync.Mutex
	// enabled indicates whether the trace writer needs to write in the channel or not
	enabled bool
}

// Close closes the TraceWriterWithChannel internal channel
func (tw *TraceWriterWithChannel) Close() {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	close(tw.destinationChannel)
}

func (tw *TraceWriterWithChannel) enable() {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	tw.enabled = true
}

func (tw *TraceWriterWithChannel) disable() {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	tw.enabled = false
}

// Write writes the sensor-to-central events in the internal channel
func (tw *TraceWriterWithChannel) Write(_ []byte) (nb int, retErr error) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	if !tw.enabled {
		return 0, nil
	}
	// We just send an empty event to signal that the resource was processed
	event := &central.SensorEvent{}
	tw.destinationChannel <- event
	return 0, nil
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

func getAlerts(messages []*central.MsgFromSensor) map[string]map[string]*storage.Alert {
	events := make(map[string]map[string]*storage.Alert)
	for _, msg := range messages {
		event := msg.GetEvent()
		if event.GetAlertResults() != nil {
			if event.GetAlertResults().GetDeploymentId() != "" {
				alertResults := event.GetAlertResults().GetAlerts()
				alerts := make(map[string]*storage.Alert, len(alertResults))
				for _, a := range alertResults {
					alerts[a.GetPolicy().GetId()] = a
				}
				events[event.GetAlertResults().GetDeploymentId()] = alerts
			}
		}
	}
	return events
}

func getLastStateFromDeployments(messages []*central.MsgFromSensor) map[string]*central.SensorEvent {
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

func getLastStateFromPods(messages []*central.MsgFromSensor) map[string]*central.SensorEvent {
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
