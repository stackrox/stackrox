package admissioncontroller

import (
	"context"
	"io"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/concurrency"
	pkgGRPC "github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/grpc/authz/idcheck"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var (
	authorizer = idcheck.AdmissionControlOnly()
)

type managementService struct {
	settingsStream concurrency.ReadOnlyValueStream
}

// NewManagementService retrieves a new admission control management service, that allows pushing config updates out
// to admission control service replicas.
func NewManagementService(mgr SettingsManager) pkgGRPC.APIService {
	return &managementService{
		settingsStream: mgr.SettingsStream(),
	}
}

func (s *managementService) RegisterServiceServer(srv *grpc.Server) {
	sensor.RegisterAdmissionControlManagementServiceServer(srv, s)
}

func (s *managementService) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, cc *grpc.ClientConn) error {
	return nil
}

func (s *managementService) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

func (s *managementService) runRecv(
	stream sensor.AdmissionControlManagementService_CommunicateServer,
	msgC chan<- *sensor.MsgFromAdmissionControl,
	errC chan<- error) {
	for {
		msg, err := stream.Recv()
		if err != nil {
			errC <- err
			return
		}

		select {
		case <-stream.Context().Done():
			return
		case msgC <- msg:
		}
	}
}

func (s *managementService) sendCurrentSettings(stream sensor.AdmissionControlManagementService_CommunicateServer, settingsIt concurrency.ValueStreamIter) error {
	settings, _ := settingsIt.Value().(*sensor.AdmissionControlSettings)
	if settings == nil {
		return nil
	}
	return stream.Send(&sensor.MsgToAdmissionControl{
		Msg: &sensor.MsgToAdmissionControl_SettingsPush{
			SettingsPush: settings,
		},
	})
}

func (s *managementService) Communicate(stream sensor.AdmissionControlManagementService_CommunicateServer) error {
	if err := stream.SendHeader(metadata.MD{}); err != nil {
		return errors.Wrap(err, "sending header metadata")
	}

	settingsIt := s.settingsStream.Iterator(false)

	if err := s.sendCurrentSettings(stream, settingsIt); err != nil {
		return errors.Wrap(err, "sending initial settings")
	}

	recvdMsgC := make(chan *sensor.MsgFromAdmissionControl)
	recvErrC := make(chan error, 1)
	go s.runRecv(stream, recvdMsgC, recvErrC)

	for {
		select {
		case err := <-recvErrC:
			recvErrC = nil // we won't receive anything more on this channel
			if err != nil && err != io.EOF {
				return errors.Wrap(err, "receiving message from admission control service")
			}
		case <-recvdMsgC:
			log.Warn("Received message from admission control service, not sure what to do with it...")

		case <-settingsIt.Done():
			settingsIt = settingsIt.TryNext()
			if err := s.sendCurrentSettings(stream, settingsIt); err != nil {
				return errors.Wrap(err, "sending settings push")
			}

		case <-stream.Context().Done():
			return stream.Context().Err()
		}
	}
}
