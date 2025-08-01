package signal

import (
	"context"
	"io"
	"strings"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	sensorAPI "github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	pkgGRPC "github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/grpc/authz/idcheck"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/common/unimplemented"
	"google.golang.org/grpc"
)

const maxBufferSize = 10000

var (
	log = logging.LoggerForModule()
)

// Option function for the signal service.
type Option func(*serviceImpl)

// WithAuthFuncOverride sets the AuthFuncOverride.
func WithAuthFuncOverride(overrideFn func(context.Context, string) (context.Context, error)) Option {
	return func(srv *serviceImpl) {
		srv.authFuncOverride = overrideFn
	}
}

// WithTraceWriter sets a trace writer that will write the messages received from collector.
func WithTraceWriter(writer io.Writer) Option {
	return func(srv *serviceImpl) {
		srv.writer = writer
	}
}

// Service is the interface that manages the SignalEvent API from the server side
type Service interface {
	pkgGRPC.APIService
	sensorAPI.SignalServiceServer

	common.SensorComponent
}

type serviceImpl struct {
	unimplemented.Receiver

	sensorAPI.UnimplementedSignalServiceServer

	queue      chan *v1.Signal
	indicators chan *message.ExpiringMessage

	processPipeline  Pipeline
	writer           io.Writer
	authFuncOverride func(context.Context, string) (context.Context, error)
}

func (s *serviceImpl) Name() string {
	return "signal.serviceImpl"
}

func authFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	err := idcheck.CollectorOnly().Authorized(ctx, fullMethodName)
	return ctx, errors.Wrap(err, "collector authorization")
}

func (s *serviceImpl) Start() error {
	return nil
}

func (s *serviceImpl) Stop() {
	s.processPipeline.Shutdown()
}

func (s *serviceImpl) Notify(e common.SensorComponentEvent) {
	log.Info(common.LogSensorComponentEvent(e))
	s.processPipeline.Notify(e)
}

func (s *serviceImpl) Capabilities() []centralsensor.SensorCapability {
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
	return s.authFuncOverride(ctx, fullMethodName)
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
			return errors.Wrap(err, "receiving signal stream message")
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
			if s.writer != nil {
				if data, err := signalStreamMsg.MarshalVT(); err == nil {
					if _, err := s.writer.Write(data); err != nil {
						log.Warnf("Error writing msg: %v", err)
					}
				} else {
					log.Warnf("Error marshalling  msg: %v", err)
				}
			}

			s.processPipeline.Process(processSignal)
		default:
			// Currently eat unhandled signals
			continue
		}
	}
}
