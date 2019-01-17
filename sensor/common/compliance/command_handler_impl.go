package compliance

import (
	"errors"
	"fmt"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/prometheus/common/log"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/internalapi/compliance"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/orchestrators"
	"github.com/stackrox/rox/pkg/retry"
)

const (
	scrapeServiceName    = "stackrox-compliance"
	scrapeServiceAccount = "stackrox-compliance"
	scrapeCommand        = "stackrox/compliance"
	scrapeEnvironment    = "ROX_SCRAPE_ID"
)

var (
	scrapeMounts = []string{
		"/etc:/host/etc:ro",
		"/lib:/host/lib:ro",
		"/usr/bin:/host/usr/bin:ro",
		"/usr/lib:/host/usr/lib:ro",
		"/var/lib:/host/var/lib:ro",
		"/var/log/audit:/host/var/log/audit:ro",
		"/run:/host/run",
		"/var/run/docker.sock:/host/var/run/docker.sock",
	}
)

type commandHandlerImpl struct {
	image        string
	orchestrator orchestrators.Orchestrator

	commands chan *central.ScrapeCommand
	updates  chan *central.ScrapeUpdate

	scrapeIDToState map[string]*scrapeState

	stopC    concurrency.ErrorSignal
	stoppedC concurrency.ErrorSignal
}

func (c *commandHandlerImpl) Start(results <-chan *compliance.ComplianceReturn) {
	go c.run(results)
}

func (c *commandHandlerImpl) Stop(err error) {
	c.stopC.SignalWithError(err)
}

func (c *commandHandlerImpl) Stopped() concurrency.ReadOnlyErrorSignal {
	return &c.stoppedC
}

func (c *commandHandlerImpl) SendCommand(command *central.ScrapeCommand) bool {
	select {
	case c.commands <- command:
		return true
	case <-c.stoppedC.Done():
		return false
	}
}

func (c *commandHandlerImpl) Output() <-chan *central.ScrapeUpdate {
	return c.updates
}

func (c *commandHandlerImpl) run(results <-chan *compliance.ComplianceReturn) {
	defer c.stoppedC.Signal()

	for {
		select {
		case <-c.stopC.Done():
			c.stoppedC.SignalWithError(c.stopC.Err())

		case command, ok := <-c.commands:
			if !ok {
				c.stoppedC.SignalWithError(errors.New("scrape command input closed"))
				return
			}
			if command.GetScrapeId() == "" {
				log.Errorf("received a command with no id: %s", proto.MarshalTextString(command))
				continue
			}
			if update := c.runCommand(command); update != nil {
				c.sendUpdate(update)
			}

		case result, ok := <-results:
			if !ok {
				c.stoppedC.SignalWithError(errors.New("compliance return input closed"))
				return
			}
			if updates := c.commitResult(result); len(updates) > 0 {
				c.sendUpdates(updates)
			}
		}
	}
}

func (c *commandHandlerImpl) runCommand(command *central.ScrapeCommand) *central.ScrapeUpdate {
	switch command.Command.(type) {
	case *central.ScrapeCommand_StartScrape:
		return c.startScrape(command.GetScrapeId(), command.GetStartScrape().GetHostnames())
	case *central.ScrapeCommand_KillScrape:
		return c.killScrape(command.GetScrapeId())
	default:
		log.Errorf("unrecognized scrape command: %s", proto.MarshalTextString(command))
	}
	return nil
}

func (c *commandHandlerImpl) startScrape(scrapeID string, expectedHosts []string) *central.ScrapeUpdate {
	// Check that the scrape is not already running.
	if _, running := c.scrapeIDToState[scrapeID]; running {
		return nil
	}

	// Try to launch the scrape.
	// If we fail, return a message to central that we failed.
	var errStr string
	scrapeName, err := c.orchestrator.Launch(*c.createService(scrapeID))
	if err != nil {
		errStr = fmt.Sprintf("unable to start scrape %s", err)
		log.Error(errStr)
		return scrapeStarted(scrapeID, errStr)
	}

	// If we succeeded, start tracking the scrape and send a message to central.
	c.scrapeIDToState[scrapeID] = newScrapeState(scrapeName, expectedHosts)
	return scrapeStarted(scrapeID, errStr)
}

func (c *commandHandlerImpl) killScrape(scrapeID string) *central.ScrapeUpdate {
	// Check that the scrape has not already been killed.
	scrapeState, running := c.scrapeIDToState[scrapeID]
	if !running {
		return nil
	}

	// Try to kill the scrape.
	// If we fail, return a message to central that we failed.
	var errStr string
	if err := c.killScrapeWithRetry(scrapeState.deploymentName, scrapeID); err != nil {
		errStr = fmt.Sprintf("unable to kill scrape %s", err)
		log.Error(errStr)
		return scrapeKilled(scrapeID, errStr)
	}

	// If killed successfully, remove the scrape from tracking.
	delete(c.scrapeIDToState, scrapeID)
	return scrapeKilled(scrapeID, errStr)
}

// Helper function to kill a scrape with retry.
func (c *commandHandlerImpl) killScrapeWithRetry(name, scrapeID string) error {
	return retry.WithRetry(
		func() error {
			return c.orchestrator.Kill(name)
		},
		retry.Tries(5),
		retry.BetweenAttempts(func() {
			time.Sleep(time.Second)
		}),
		retry.OnFailedAttempts(func(err error) {
			log.Errorf("failed to kill scrape %s: %s", scrapeID, err)
		}),
	)
}

// Helper function that converts a scrape command into a SystemService that can be launched.
func (c *commandHandlerImpl) createService(scrapeID string) *orchestrators.SystemService {
	return &orchestrators.SystemService{
		GenerateName: scrapeServiceName,
		Command:      []string{scrapeCommand},
		Mounts:       scrapeMounts,
		HostPID:      true,
		Envs: []string{
			env.CombineSetting(env.AdvertisedEndpoint),
			env.Combine(scrapeEnvironment, scrapeID),
		},
		Image:          c.image,
		Global:         true,
		ServiceAccount: scrapeServiceAccount,
	}
}

func (c *commandHandlerImpl) commitResult(result *compliance.ComplianceReturn) (ret []*central.ScrapeUpdate) {
	// Check that the scrape has not already been killed.
	scrapeState, running := c.scrapeIDToState[result.GetScrapeId()]
	if !running {
		log.Errorf("received result scrape not tracked: %s", proto.MarshalTextString(result))
		return
	}

	// Check that we have not already received a result for the host.
	if hostNotSeenYet := scrapeState.remainingHosts.Contains(result.GetHostname()); !hostNotSeenYet {
		log.Errorf("received an unexpected result: %s", proto.MarshalTextString(result))
		return
	}

	// Pass the update back to central.
	scrapeState.remainingHosts.Remove(result.GetHostname())
	ret = append(ret, scrapeUpdate(result))

	// If that was the last expected update, kill the scrape.
	if scrapeState.remainingHosts.Cardinality() == 0 {
		if update := c.killScrape(result.GetScrapeId()); update != nil {
			ret = append(ret, update)
		}
	}
	return
}

func (c *commandHandlerImpl) sendUpdates(updates []*central.ScrapeUpdate) {
	if len(updates) > 0 {
		for _, update := range updates {
			c.sendUpdate(update)
		}
	}
}

func (c *commandHandlerImpl) sendUpdate(update *central.ScrapeUpdate) {
	select {
	case <-c.stoppedC.Done():
		log.Errorf("failed to send update: %s", proto.MarshalTextString(update))
		return
	case c.updates <- update:
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
