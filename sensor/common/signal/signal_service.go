package signal

import (
	"context"
	"strings"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/internalapi/central"
	sensorAPI "github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	pkgGRPC "github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/grpc/authz/idcheck"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/message"
	"google.golang.org/grpc"
)

const maxBufferSize = 10000

var (
	log = logging.LoggerForModule()
)

// Service is the interface that manages the SignalEvent API from the server side
type Service interface {
	pkgGRPC.APIService
	sensorAPI.SignalServiceServer

	common.SensorComponent
}

type serviceImpl struct {
	sensorAPI.UnimplementedSignalServiceServer

	queue      chan *v1.Signal
	indicators chan *message.ExpiringMessage

	processPipeline Pipeline
	centralReady    concurrency.Signal
}

func (s *serviceImpl) Start() error {
	return nil
}

func (s *serviceImpl) Stop(_ error) {
	s.processPipeline.Shutdown()
}

func (s *serviceImpl) Notify(e common.SensorComponentEvent) {
	log.Debugf("Received notify: %s", e)
	switch e {
	case common.SensorComponentEventCentralReachable:
		s.centralReady.Signal()
	case common.SensorComponentEventOfflineMode:
		s.centralReady.Reset()
	}
	s.processPipeline.Notify(e)
}

func (s *serviceImpl) Capabilities() []centralsensor.SensorCapability {
	return nil
}

func (s *serviceImpl) ProcessMessage(_ *central.MsgToSensor) error {
	return nil
}

func (s *serviceImpl) ResponsesC() <-chan *message.ExpiringMessage {
	return s.indicators
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	sensorAPI.RegisterSignalServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(_ context.Context, _ *runtime.ServeMux, _ *grpc.ClientConn) error {
	// There is no grpc gateway handler for signal service
	return nil
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, idcheck.CollectorOnly().Authorized(ctx, fullMethodName)
}

// PushSignals handles the bidirectional gRPC stream with the collector
func (s *serviceImpl) PushSignals(stream sensorAPI.SignalService_PushSignalsServer) error {
	return s.receiveMessages(stream)
}

// TODO(ROX-3281) this is a workaround for these collector issues
func isProcessSignalValid(signal *storage.ProcessSignal) bool {
	// Example: <NA> or sometimes a truncated variant
	if signal.GetExecFilePath() == "" || signal.GetExecFilePath()[0] == '<' {
		return false
	}
	if signal.GetName() == "" || signal.GetName()[0] == '<' {
		return false
	}
	if strings.HasPrefix(signal.GetExecFilePath(), "/proc/self") {
		return false
	}
	// Example: /var/run/docker/containerd/daemon/io.containerd.runtime.v1.linux/moby/8f79b77ac6785562e875cde2f087c49f1d4e4899f18a26d3739c47155668ec0b/run
	if strings.HasPrefix(signal.GetExecFilePath(), "/var/run/docker") {
		return false
	}
	return true
}

func (s *serviceImpl) receiveMessages(stream sensorAPI.SignalService_PushSignalsServer) error {
	for {
		signalStreamMsg, err := stream.Recv()
		if err != nil {
			log.Error("error dequeueing signalStreamMsg event: ", err)
			return err
		}

		// Ignore the collector register request
		if signalStreamMsg.GetSignal() == nil {
			log.Error("Empty signalStreamMsg")
			continue
		}
		signal := signalStreamMsg.GetSignal()

		switch signal.GetSignal().(type) {
		case *v1.Signal_ProcessSignal:
			processSignal := signal.GetProcessSignal()
			if processSignal == nil {
				log.Error("Empty process signal")
				continue
			}

			processSignal.ExecFilePath = stringutils.OrDefault(processSignal.GetExecFilePath(), processSignal.GetName())
			if !isProcessSignalValid(processSignal) {
				log.Debugf("Invalid process signal: %+v", processSignal)
				continue
			}

			s.processPipeline.Process(processSignal)
		default:
			// Currently eat unhandled signals
			continue
		}
	}
}
