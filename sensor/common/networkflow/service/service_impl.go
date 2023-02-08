package service

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/grpc/authz/idcheck"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stackrox/rox/sensor/common/metrics"
	"github.com/stackrox/rox/sensor/common/networkflow/manager"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const (
	capMetadataKey         = `rox-collector-capabilities`
	publicIPsUpdateCap     = `public-ips`
	networkGraphExtSrcsCap = `network-graph-external-srcs`
)

// NewService creates a new streaming service with the collector. It should only be called once.
func NewService(networkFlowManager manager.Manager) Service {
	return &serviceImpl{
		manager: networkFlowManager,
	}

}

type serviceImpl struct {
	sensor.UnimplementedNetworkConnectionInfoServiceServer

	manager manager.Manager

	fileOutput chan *sensor.NetworkConnectionInfoMessage
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	sensor.RegisterNetworkConnectionInfoServiceServer(grpcServer, s)
}

// RegisterServiceHandlerFromEndpoint registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	// There is no grpc gateway handler for network connection info service
	return nil
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, idcheck.CollectorOnly().Authorized(ctx, fullMethodName)
}

// PushSignals handles the bidirectional gRPC stream with the collector
func (s *serviceImpl) PushNetworkConnectionInfo(stream sensor.NetworkConnectionInfoService_PushNetworkConnectionInfoServer) error {
	return s.receiveMessages(stream)
}

func (s *serviceImpl) receiveMessages(stream sensor.NetworkConnectionInfoService_PushNetworkConnectionInfoServer) error {
	var hostname string

	incomingMD := metautils.ExtractIncoming(stream.Context())
	hostname = incomingMD.Get("rox-collector-hostname")
	if hostname == "" {
		return errors.New("collector did not transmit a hostname in initial metadata")
	}

	capsStr := incomingMD.Get(capMetadataKey)
	var capsSet set.FrozenStringSet
	if capsStr != "" {
		capsSet = set.NewFrozenStringSet(strings.Split(capsStr, ",")...)
	}

	if err := stream.SendHeader(metadata.MD{}); err != nil {
		return errors.Errorf("error sending initial metadata: %v", err)
	}

	hostConnections, sequenceID := s.manager.RegisterCollector(hostname)
	defer s.manager.UnregisterCollector(hostname, sequenceID)

	recvdMsgC := make(chan *sensor.NetworkConnectionInfoMessage)
	recvErrC := make(chan error, 1)

	go s.runRecv(stream, recvdMsgC, recvErrC)

	s.fileOutput = make(chan *sensor.NetworkConnectionInfoMessage, 1000)
	go s.runRecorder()

	var publicIPsIterator concurrency.ValueStreamIter[*sensor.IPAddressList]
	if capsSet.Contains(publicIPsUpdateCap) {
		publicIPsIterator = s.manager.PublicIPsValueStream().Iterator(false)
		if err := s.sendPublicIPList(stream, publicIPsIterator); err != nil {
			return err
		}
	}
	var externalSrcsIterator concurrency.ValueStreamIter[*sensor.IPNetworkList]
	if capsSet.Contains(networkGraphExtSrcsCap) {
		// Non-strict allows us to skip to the most recent element using `TryNext()` and this is fine since each element in the stream
		// is a full network list that we want to monitor.
		externalSrcsIterator = s.manager.ExternalSrcsValueStream().Iterator(false)
		if err := s.sendExternalSrcsList(stream, externalSrcsIterator); err != nil {
			return err
		}
	}

	for {
		// If the publicIPsIterator is nil (i.e., Sensor does not support receive public IP list updates), leave this
		// as nil, which means the respective select branch will never be taken.
		var publicIPItrDoneC <-chan struct{}
		if publicIPsIterator != nil {
			publicIPItrDoneC = publicIPsIterator.Done()
		}

		var externalSrcsItrDoneC <-chan struct{}
		if externalSrcsIterator != nil {
			externalSrcsItrDoneC = externalSrcsIterator.Done()
		}

		select {
		case <-stream.Context().Done():
			return stream.Context().Err()

		case err := <-recvErrC:
			if err == io.EOF {
				err = nil
			}
			return errors.Wrap(err, "receiving message from collector")

		case msg := <-recvdMsgC:
			s.fileOutput <- msg
			networkInfoMsg := msg.GetInfo()
			networkInfoMsgTimestamp := timestamp.Now()

			if networkInfoMsg == nil {
				return errors.Errorf("received unexpected message type %T from hostname %s", networkInfoMsg, hostname)
			}

			metrics.IncrementTotalNetworkFlowsReceivedCounter(len(msg.GetInfo().GetUpdatedConnections()))
			if err := hostConnections.Process(networkInfoMsg, networkInfoMsgTimestamp, sequenceID); err != nil {
				return errors.Errorf("could not process connections: %v", err)
			}

		case <-publicIPItrDoneC:
			publicIPsIterator = publicIPsIterator.TryNext()
			if err := s.sendPublicIPList(stream, publicIPsIterator); err != nil {
				return err
			}
		case <-externalSrcsItrDoneC:
			externalSrcsIterator = externalSrcsIterator.TryNext()
			if err := s.sendExternalSrcsList(stream, externalSrcsIterator); err != nil {
				return err
			}
		}
	}
}

func (s *serviceImpl) runRecorder() {
	bytesWrittenCurrFile := 0
	filesWrittenCount := 1

	fileNameString := fmt.Sprintf("collector_output_%d", filesWrittenCount)
	fileNameBytes := fmt.Sprintf("collector_output_bytes_%d", filesWrittenCount)

	fullPathString := path.Join("/var/log/stackrox", fileNameString)
	stringFile, err := os.OpenFile(fullPathString, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Warnf("failed to open file %s", fullPathString)
		return
	}

	fullPathBytes := path.Join("/var/log/stackrox", fileNameBytes)
	byteFile, err := os.OpenFile(fullPathBytes, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Warnf("failed to open file %s", fullPathBytes)
		return
	}

	for {
		content, more := <-s.fileOutput
		if !more {
			return
		}
		_, err := stringFile.WriteString(content.String() + "\n")
		if err != nil {
			log.Warnf("failed to write string to collector recorder: %s", err)
		}

		byteContent, err := content.Marshal()
		if err != nil {
			log.Warnf("failed to marshal message into bytes; %s", err)
			continue
		}

		byteContent = append(byteContent, 0xFA, 0xFB, 0xFC, 0xFD)
		written, err := byteFile.Write(byteContent)
		if err != nil {
			log.Warnf("failed to write string to collector recorder: %s", err)
		}

		bytesWrittenCurrFile += written

		if written >= 10_000_000 {
			bytesWrittenCurrFile = 0
			filesWrittenCount++
			err := stringFile.Close()
			if err != nil {
				log.Warnf("failed to close file %s: %s", fullPathString, err)
				return
			}

			err = byteFile.Close()
			if err != nil {
				log.Warnf("failed to close file %s: %s", fullPathBytes, err)
				return
			}

			fileNameString = fmt.Sprintf("collector_output_%d", filesWrittenCount)
			fileNameBytes = fmt.Sprintf("collector_output_bytes_%d", filesWrittenCount)

			fullPathString = path.Join("/var/log/stackrox", fileNameString)
			stringFile, err = os.OpenFile(fullPathString, os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				log.Warnf("failed to open file %s", fullPathString)
				return
			}

			fullPathBytes = path.Join("/var/log/stackrox", fileNameBytes)
			byteFile, err = os.OpenFile(fullPathBytes, os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				log.Warnf("failed to open file %s", fullPathBytes)
				return
			}

		}
	}
}

func (s *serviceImpl) runRecv(stream sensor.NetworkConnectionInfoService_PushNetworkConnectionInfoServer, msgC chan<- *sensor.NetworkConnectionInfoMessage, errC chan<- error) {
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

func (s *serviceImpl) sendPublicIPList(stream sensor.NetworkConnectionInfoService_PushNetworkConnectionInfoServer, iter concurrency.ValueStreamIter[*sensor.IPAddressList]) error {
	listProto := iter.Value()
	if listProto == nil {
		return nil
	}

	controlMsg := &sensor.NetworkFlowsControlMessage{
		PublicIpAddresses: listProto,
	}

	if err := stream.Send(controlMsg); err != nil {
		return errors.Wrap(err, "sending public IPs list")
	}
	return nil
}

func (s *serviceImpl) sendExternalSrcsList(stream sensor.NetworkConnectionInfoService_PushNetworkConnectionInfoServer, iter concurrency.ValueStreamIter[*sensor.IPNetworkList]) error {
	listProto := iter.Value()
	if listProto == nil {
		return nil
	}

	controlMsg := &sensor.NetworkFlowsControlMessage{
		IpNetworks: listProto,
	}

	if err := stream.Send(controlMsg); err != nil {
		return errors.Wrap(err, "sending external sources (IP Network) list")
	}
	return nil
}
