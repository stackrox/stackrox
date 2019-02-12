package sensor

import (
	"io"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/enforcers"
	complianceLogic "github.com/stackrox/rox/sensor/common/compliance"
)

type centralReceiverImpl struct {
	scrapeCommandHandler complianceLogic.CommandHandler
	enforcer             enforcers.Enforcer

	stopC    concurrency.ErrorSignal
	stoppedC concurrency.ErrorSignal
}

func (s *centralReceiverImpl) Start(stream central.SensorService_CommunicateClient, onStops ...func(error)) {
	go s.receive(stream, onStops...)
}

func (s *centralReceiverImpl) Stop(err error) {
	s.stopC.SignalWithError(err)
}

func (s *centralReceiverImpl) Stopped() concurrency.ReadOnlyErrorSignal {
	return &s.stoppedC
}

// Take in data processed by central, run post processing, then send it to the output channel.
func (s *centralReceiverImpl) receive(stream central.SensorService_CommunicateClient, onStops ...func(error)) {
	defer func() {
		s.stoppedC.SignalWithError(s.stopC.Err())
		runAll(s.stopC.Err(), onStops...)
	}()

	for {
		select {
		case <-s.stopC.Done():
			return

		case <-stream.Context().Done():
			s.stopC.SignalWithError(stream.Context().Err())
			return

		default:
			msg, err := stream.Recv()
			if err == io.EOF {
				s.stopC.Signal()
				return
			}
			if err != nil {
				s.stopC.SignalWithError(err)
				return
			}
			s.processMsg(msg)
		}
	}
}

func (s *centralReceiverImpl) processMsg(msg *central.MsgToSensor) {
	switch msg.Msg.(type) {
	case *central.MsgToSensor_Enforcement:
		enforcementMsg := msg.Msg.(*central.MsgToSensor_Enforcement)
		s.processEnforcement(enforcementMsg.Enforcement)
	case *central.MsgToSensor_ScrapeCommand:
		commandMsg := msg.Msg.(*central.MsgToSensor_ScrapeCommand)
		s.processCommand(commandMsg.ScrapeCommand)
	default:
		log.Errorf("Unsupported message from central of type %T: %+v", msg.Msg, msg.Msg)
	}
}

func (s *centralReceiverImpl) processCommand(command *central.ScrapeCommand) {
	if !s.scrapeCommandHandler.SendCommand(command) {
		log.Errorf("unable to send command: %s", proto.MarshalTextString(command))
	}
}

func (s *centralReceiverImpl) processEnforcement(enforcement *central.SensorEnforcement) {
	if enforcement == nil {
		return
	}

	if enforcement.GetEnforcement() == storage.EnforcementAction_UNSET_ENFORCEMENT {
		log.Errorf("received enforcement with unset action: %s", proto.MarshalTextString(enforcement))
		return
	}

	if !s.enforcer.SendEnforcement(enforcement) {
		log.Errorf("unable to send enforcement: %s", proto.MarshalTextString(enforcement))
	}
}
