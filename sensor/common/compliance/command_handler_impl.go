package compliance

import (
	"sync/atomic"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/internalapi/compliance"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/message"
)

var (
	log = logging.LoggerForModule()
)

type commandHandlerImpl struct {
	commands chan *central.ScrapeCommand
	updates  chan *message.ExpiringMessage

	service Service

	scrapeIDToState map[string]*scrapeState

	stopper          concurrency.Stopper
	centralReachable atomic.Bool
}

func (c *commandHandlerImpl) Capabilities() []centralsensor.SensorCapability {
	return []centralsensor.SensorCapability{centralsensor.ComplianceInNodesCap}
}

func (c *commandHandlerImpl) ResponsesC() <-chan *message.ExpiringMessage {
	return c.updates
}

func (c *commandHandlerImpl) Start() error {
	go c.run()
	return nil
}

func (c *commandHandlerImpl) Stop(_ error) {
	c.stopper.Client().Stop()
}

func (c *commandHandlerImpl) Notify(e common.SensorComponentEvent) {
	switch e {
	case common.SensorComponentEventCentralReachable:
		c.centralReachable.Store(true)
	case common.SensorComponentEventOfflineMode:
		c.centralReachable.Store(false)
	}
}

func (c *commandHandlerImpl) Stopped() concurrency.ReadOnlyErrorSignal {
	return c.stopper.Client().Stopped()
}

func (c *commandHandlerImpl) ProcessMessage(msg *central.MsgToSensor) error {
	command := msg.GetScrapeCommand()
	if command == nil {
		return nil
	}
	select {
	case c.commands <- command:
		return nil
	case <-c.stopper.Flow().StopRequested():
		return errors.Errorf("component is shutting down, unable to send command: %s", proto.MarshalTextString(command))
	}
}

func (c *commandHandlerImpl) run() {
	defer c.stopper.Flow().ReportStopped()

	for {
		select {
		case <-c.stopper.Flow().StopRequested():
			return

		case command, ok := <-c.commands:
			if !ok {
				c.stopper.Flow().StopWithError(errors.New("scrape command input closed"))
				return
			}
			if command.GetScrapeId() == "" {
				log.Errorf("received a command with no id: %s", proto.MarshalTextString(command))
				continue
			}
			if updates := c.runCommand(command); updates != nil {
				c.sendUpdates(updates)
			}

		case result, ok := <-c.service.Output():
			if !ok {
				c.stopper.Flow().StopWithError(errors.New("compliance return input closed"))
				return
			}
			if updates := c.commitResult(result); len(updates) > 0 {
				c.sendUpdates(updates)
			}
		}
	}
}

func (c *commandHandlerImpl) runCommand(command *central.ScrapeCommand) []*central.ScrapeUpdate {
	switch command.Command.(type) {
	case *central.ScrapeCommand_StartScrape:
		return c.startScrape(command.GetScrapeId(), command.GetStartScrape().GetHostnames(), command.GetStartScrape().GetStandards())
	case *central.ScrapeCommand_KillScrape:
		return []*central.ScrapeUpdate{c.killScrape(command.GetScrapeId())}
	default:
		log.Errorf("unrecognized scrape command: %s", proto.MarshalTextString(command))
	}
	return nil
}

func (c *commandHandlerImpl) startScrape(scrapeID string, expectedHosts []string, standards []string) []*central.ScrapeUpdate {
	// Check that the scrape is not already running.
	if _, running := c.scrapeIDToState[scrapeID]; running {
		return nil
	}

	numResults := c.service.RunScrape(&sensor.MsgToCompliance{
		Msg: &sensor.MsgToCompliance_Trigger{
			Trigger: &sensor.MsgToCompliance_TriggerRun{
				ScrapeId:    scrapeID,
				StandardIds: standards,
			},
		},
	})

	// If we succeeded, start tracking the scrape and send a message to central.
	scrapeState := newScrapeState(scrapeID, numResults, expectedHosts)
	c.scrapeIDToState[scrapeID] = scrapeState
	log.Infof("started scrape %q with %d results desired", scrapeID, numResults)

	updates := []*central.ScrapeUpdate{scrapeStarted(scrapeID, "")}
	updates = append(updates, c.checkScrapeCompleted(scrapeID, scrapeState)...)
	return updates
}

func (c *commandHandlerImpl) killScrape(scrapeID string) *central.ScrapeUpdate {
	//// If killed successfully, remove the scrape from tracking.
	delete(c.scrapeIDToState, scrapeID)
	return scrapeKilled(scrapeID, "")
}

func (c *commandHandlerImpl) commitResult(result *compliance.ComplianceReturn) (ret []*central.ScrapeUpdate) {
	// Check that the scrape has not already been killed.
	scrapeState, running := c.scrapeIDToState[result.GetScrapeId()]
	if !running {
		log.Errorf("received result for scrape not tracked: %q", result.GetScrapeId())
		return
	}

	// Check that we have not already received a result for the host.
	if scrapeState.foundNodes.Contains(result.GetNodeName()) {
		log.Errorf("received duplicate result in scrape %s for node %s", result.GetScrapeId(), result.GetNodeName())
		return
	}
	scrapeState.desiredNodes--

	// Check if the node did not exist in the scrape request
	if !scrapeState.remainingNodes.Contains(result.GetNodeName()) {
		log.Errorf("found node %s not requested by Central for scrape %s", result.GetNodeName(), result.GetScrapeId())
		return
	}

	// Pass the update back to central.
	scrapeState.remainingNodes.Remove(result.GetNodeName())
	ret = append(ret, scrapeUpdate(result))
	ret = append(ret, c.checkScrapeCompleted(result.GetScrapeId(), scrapeState)...)

	return
}

func (c *commandHandlerImpl) checkScrapeCompleted(scrapeID string, state *scrapeState) []*central.ScrapeUpdate {
	var updates []*central.ScrapeUpdate
	if state.desiredNodes == 0 || state.remainingNodes.Cardinality() == 0 {
		if state.remainingNodes.Cardinality() != 0 {
			log.Warnf("compliance data for the following nodes was not collected: %s", state.remainingNodes.ElementsString(", "))
		}
		if update := c.killScrape(scrapeID); update != nil {
			updates = append(updates, update)
		}
	}
	return updates
}

func (c *commandHandlerImpl) sendUpdates(updates []*central.ScrapeUpdate) {
	if len(updates) > 0 {
		if c.centralReachable.Load() {
			for _, update := range updates {
				c.sendUpdate(update)
			}
		} else {
			log.Debug("sendUpdate() called while in offline mode, ScrapeUpdate discarded")
		}
	}
}

func (c *commandHandlerImpl) sendUpdate(update *central.ScrapeUpdate) {
	select {
	case <-c.stopper.Flow().StopRequested():
		log.Errorf("component is shutting down, failed to send update: %s", proto.MarshalTextString(update))
		return
	case c.updates <- message.New(&central.MsgFromSensor{
		Msg: &central.MsgFromSensor_ScrapeUpdate{
			ScrapeUpdate: update,
		},
	}):
		return
	}
}

// Helper functions.
///////////////////

func scrapeStarted(scrapeID, err string) *central.ScrapeUpdate {
	return &central.ScrapeUpdate{
		ScrapeId: scrapeID,
		Update: &central.ScrapeUpdate_ScrapeStarted{
			ScrapeStarted: &central.ScrapeStarted{
				ErrorMessage: err,
			},
		},
	}
}

func scrapeKilled(scrapeID, err string) *central.ScrapeUpdate {
	return &central.ScrapeUpdate{
		ScrapeId: scrapeID,
		Update: &central.ScrapeUpdate_ScrapeKilled{
			ScrapeKilled: &central.ScrapeKilled{
				ErrorMessage: err,
			},
		},
	}
}

func scrapeUpdate(result *compliance.ComplianceReturn) *central.ScrapeUpdate {
	return &central.ScrapeUpdate{
		ScrapeId: result.GetScrapeId(),
		Update: &central.ScrapeUpdate_ComplianceReturn{
			ComplianceReturn: result,
		},
	}
}
