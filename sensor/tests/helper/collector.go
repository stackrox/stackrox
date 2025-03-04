package helper

import (
	"context"
	"time"

	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	net2 "github.com/stackrox/rox/pkg/net"
	"github.com/stackrox/rox/sensor/debugger/collector"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	defaultCollectorMessagesWaitTimeout = 5 * time.Minute
)

// ExpectedSignalMessageFn signature for a function to match SignalStreamMessage.
type ExpectedSignalMessageFn func(*sensor.SignalStreamMessage) bool

// ExpectedNetworkConnectionMessageFn signature for a function to match NetworkConnectionInfoMessage.
type ExpectedNetworkConnectionMessageFn func(*sensor.NetworkConnectionInfoMessage) bool

// WaitToReceiveMessagesFromCollector waits until sensor receives the expected messages from collector with a default timeout.
func WaitToReceiveMessagesFromCollector(ctx context.Context, signal *concurrency.ErrorSignal,
	signalC <-chan *sensor.SignalStreamMessage,
	flowC <-chan *sensor.NetworkConnectionInfoMessage,
	expectedSignalMessages []ExpectedSignalMessageFn,
	expectedNetworkMessages []ExpectedNetworkConnectionMessageFn) {
	WaitToReceiveMessagesFromCollectorWithTimeout(ctx, signal, signalC, flowC, expectedSignalMessages, expectedNetworkMessages, defaultCollectorMessagesWaitTimeout)
}

// WaitToReceiveMessagesFromCollectorWithTimeout waits for given time until sensor receives the expected messages from collector.
func WaitToReceiveMessagesFromCollectorWithTimeout(ctx context.Context, signal *concurrency.ErrorSignal,
	signalC <-chan *sensor.SignalStreamMessage,
	flowC <-chan *sensor.NetworkConnectionInfoMessage,
	expectedSignalMessages []ExpectedSignalMessageFn,
	expectedNetworkMessages []ExpectedNetworkConnectionMessageFn,
	timeoutDuration time.Duration) {
	timeout := time.NewTicker(timeoutDuration)
	for {
		select {
		case <-ctx.Done():
			signal.Signal()
			return
		case <-timeout.C:
			signal.SignalWithError(errors.New("Timeout waiting for collector messages"))
			return
		case msg, ok := <-flowC:
			if !ok {
				signal.SignalWithError(errors.New("NetworkFlows trace channel closed"))
				return
			}
			pos := -1
			for i, fn := range expectedNetworkMessages {
				if fn(msg) {
					pos = i
					break
				}
			}
			if pos != -1 {
				expectedNetworkMessages[pos] = expectedNetworkMessages[len(expectedNetworkMessages)-1]
				expectedNetworkMessages = expectedNetworkMessages[:len(expectedNetworkMessages)-1]
			}
		case msg, ok := <-signalC:
			if !ok {
				signal.SignalWithError(errors.New("Signals trace channel closed"))
				return
			}
			pos := -1
			for i, fn := range expectedSignalMessages {
				if fn(msg) {
					pos = i
					break
				}
			}
			if pos != -1 {
				expectedSignalMessages[pos] = expectedSignalMessages[len(expectedSignalMessages)-1]
				expectedSignalMessages = expectedSignalMessages[:len(expectedSignalMessages)-1]
			}
		}
		if len(expectedSignalMessages) == 0 && len(expectedNetworkMessages) == 0 {
			signal.Signal()
			return
		}
	}
}

// SendSignalMessage uses FakeCollector to send a fake SignalStreamMessage.
func SendSignalMessage(fakeCollector *collector.FakeCollector, containerID string, signalName string) {
	fakeCollector.SendFakeSignal(&sensor.SignalStreamMessage{
		Msg: &sensor.SignalStreamMessage_Signal{
			Signal: &v1.Signal{
				Signal: &v1.Signal_ProcessSignal{
					ProcessSignal: &storage.ProcessSignal{
						ContainerId: containerID,
						Name:        signalName,
					},
				},
			},
		},
	})
}

func SendCloseFlowMessage(fakeCollector *collector.FakeCollector,
	socketFamily sensor.SocketFamily,
	protocol storage.L4Protocol,
	fromID string,
	toID string,
	fromIP string,
	toIP string,
	port uint32,
	closeTimestampSeconds int64) {
	fakeCollector.SendFakeNetworkFlow(&sensor.NetworkConnectionInfoMessage{
		Msg: &sensor.NetworkConnectionInfoMessage_Info{
			Info: &sensor.NetworkConnectionInfo{
				Time: &timestamppb.Timestamp{
					Seconds: closeTimestampSeconds,
					Nanos:   0,
				},
				UpdatedConnections: []*sensor.NetworkConnection{
					{
						SocketFamily: socketFamily,
						Protocol:     protocol,
						Role:         sensor.ClientServerRole_ROLE_CLIENT,
						ContainerId:  fromID,
						LocalAddress: &sensor.NetworkAddress{
							AddressData: nil,
							IpNetwork:   nil,
							Port:        0,
						},
						RemoteAddress: &sensor.NetworkAddress{
							AddressData: net2.ParseIP(toIP).AsNetIP(),
							IpNetwork:   net2.ParseIP(toIP).AsNetIP(),
							Port:        port,
						},
						CloseTimestamp: &timestamppb.Timestamp{
							Seconds: closeTimestampSeconds,
							Nanos:   0,
						},
					},
					{
						SocketFamily: socketFamily,
						Protocol:     protocol,
						Role:         sensor.ClientServerRole_ROLE_SERVER,
						ContainerId:  toID,
						LocalAddress: &sensor.NetworkAddress{
							AddressData: nil,
							IpNetwork:   nil,
							Port:        port,
						},
						RemoteAddress: &sensor.NetworkAddress{
							AddressData: net2.ParseIP(fromIP).AsNetIP(),
							IpNetwork:   net2.ParseIP(fromIP).AsNetIP(),
							Port:        0,
						},
						CloseTimestamp: &timestamppb.Timestamp{
							Seconds: closeTimestampSeconds,
							Nanos:   0,
						},
					},
				},
				UpdatedEndpoints: []*sensor.NetworkEndpoint{
					{
						SocketFamily: socketFamily,
						Protocol:     protocol,
						ListenAddress: &sensor.NetworkAddress{
							AddressData: net2.ParseIP(toIP).AsNetIP(),
							IpNetwork:   net2.ParseIP(toIP).AsNetIP(),
							Port:        port,
						},
						ContainerId: toID,
						Originator: &storage.NetworkProcessUniqueKey{
							ProcessName:         "nginx",
							ProcessExecFilePath: "/path/nginx",
						},
						CloseTimestamp: &timestamppb.Timestamp{
							Seconds: closeTimestampSeconds,
							Nanos:   0,
						},
					},
				},
			},
		},
	})
}

// SendFlowMessage uses FakeCollector to send a fake NetworkConnectionInfoMessage.
func SendFlowMessage(fakeCollector *collector.FakeCollector,
	socketFamily sensor.SocketFamily,
	protocol storage.L4Protocol,
	fromID string,
	toID string,
	fromIP string,
	toIP string,
	port uint32,
	timestampSeconds int64) {
	fakeCollector.SendFakeNetworkFlow(&sensor.NetworkConnectionInfoMessage{
		Msg: &sensor.NetworkConnectionInfoMessage_Info{
			Info: &sensor.NetworkConnectionInfo{
				Time: &timestamppb.Timestamp{
					Seconds: timestampSeconds,
					Nanos:   0,
				},
				UpdatedConnections: []*sensor.NetworkConnection{
					{
						SocketFamily: socketFamily,
						Protocol:     protocol,
						Role:         sensor.ClientServerRole_ROLE_CLIENT,
						ContainerId:  fromID,
						LocalAddress: &sensor.NetworkAddress{
							AddressData: nil,
							IpNetwork:   nil,
							Port:        0,
						},
						RemoteAddress: &sensor.NetworkAddress{
							AddressData: net2.ParseIP(toIP).AsNetIP(),
							IpNetwork:   net2.ParseIP(toIP).AsNetIP(),
							Port:        port,
						},
					},
					{
						SocketFamily: socketFamily,
						Protocol:     protocol,
						Role:         sensor.ClientServerRole_ROLE_SERVER,
						ContainerId:  toID,
						LocalAddress: &sensor.NetworkAddress{
							AddressData: nil,
							IpNetwork:   nil,
							Port:        port,
						},
						RemoteAddress: &sensor.NetworkAddress{
							AddressData: net2.ParseIP(fromIP).AsNetIP(),
							IpNetwork:   net2.ParseIP(fromIP).AsNetIP(),
							Port:        0,
						},
					},
				},
				UpdatedEndpoints: []*sensor.NetworkEndpoint{
					{
						SocketFamily: socketFamily,
						Protocol:     protocol,
						ListenAddress: &sensor.NetworkAddress{
							AddressData: net2.ParseIP(toIP).AsNetIP(),
							IpNetwork:   net2.ParseIP(toIP).AsNetIP(),
							Port:        port,
						},
						ContainerId: toID,
						Originator: &storage.NetworkProcessUniqueKey{
							ProcessName:         "nginx",
							ProcessExecFilePath: "/path/nginx",
						},
					},
				},
			},
		},
	})
}
