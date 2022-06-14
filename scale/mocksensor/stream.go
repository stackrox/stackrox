package main

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/internalapi/compliance"
	"github.com/stackrox/rox/pkg/sync"
)

type threadSafeStream struct {
	stream central.SensorService_CommunicateClient
	mutex  sync.Mutex
}

func (s *threadSafeStream) SendEvent(event *central.SensorEvent) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.stream.Send(&central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: event,
		},
	})
}

func (s *threadSafeStream) SendNetworkFlows(flows *central.NetworkFlowUpdate) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.stream.Send(&central.MsgFromSensor{
		Msg: &central.MsgFromSensor_NetworkFlowUpdate{
			NetworkFlowUpdate: flows,
		},
	})
}

func (s *threadSafeStream) sendComplianceReturn(scrapeID string, ret *compliance.ComplianceReturn) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	logger.Infof("Sending return for scrape %q and host %q", scrapeID, ret.NodeName)
	return s.stream.Send(&central.MsgFromSensor{
		Msg: &central.MsgFromSensor_ScrapeUpdate{
			ScrapeUpdate: &central.ScrapeUpdate{
				ScrapeId: scrapeID,
				Update: &central.ScrapeUpdate_ComplianceReturn{
					ComplianceReturn: ret,
				},
			},
		},
	})
}

func (s *threadSafeStream) sendComplianceFinished(scrapeID string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.stream.Send(&central.MsgFromSensor{
		Msg: &central.MsgFromSensor_ScrapeUpdate{
			ScrapeUpdate: &central.ScrapeUpdate{
				ScrapeId: scrapeID,
				Update: &central.ScrapeUpdate_ScrapeKilled{
					ScrapeKilled: &central.ScrapeKilled{},
				},
			},
		},
	})
}

func (s *threadSafeStream) StartReceiving() {
	for {
		msg, err := s.stream.Recv()
		if err != nil {
			logger.Fatal(err)
		}
		switch msg.Msg.(type) {
		case *central.MsgToSensor_Enforcement:
		case *central.MsgToSensor_ScrapeCommand:
			logger.Info("Received scrape message from Central")
			commandMsg := msg.Msg.(*central.MsgToSensor_ScrapeCommand)
			switch scrape := commandMsg.ScrapeCommand.Command.(type) {
			case *central.ScrapeCommand_StartScrape:
				scrapeID := commandMsg.ScrapeCommand.ScrapeId
				logger.Infof("Requests to scrape %d hosts", len(scrape.StartScrape.Hostnames))
				for _, hostname := range scrape.StartScrape.Hostnames {
					complianceReturn := getCheckResults(commandMsg.ScrapeCommand.ScrapeId, hostname)
					if err := s.sendComplianceReturn(scrapeID, complianceReturn); err != nil {
						logger.Error(err)
					}
				}
				if err := s.sendComplianceFinished(scrapeID); err != nil {
					logger.Error(err)
				}
			}
		default:
			logger.Errorf("Unsupported message from central of type %T: %+v", msg.Msg, msg.Msg)
		}
	}
}
