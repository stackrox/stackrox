package emit

import (
	"fmt"

	"github.com/stackrox/rox/central/sensor/service/streamer"
	"github.com/stackrox/rox/generated/internalapi/central"
)

type emitterImpl struct {
	sensorStreamManager streamer.Manager
}

func (s *emitterImpl) StartScrape(clusterID, scrapeID string, expectedHosts []string) error {
	return s.emitMsg(clusterID, startScrapeCommand(scrapeID, expectedHosts))
}

func (s *emitterImpl) KillScrape(clusterID, scrapeID string) error {
	return s.emitMsg(clusterID, killScrapeCommand(scrapeID))
}

func (s *emitterImpl) emitMsg(clusterID string, msg *central.MsgToSensor) error {
	streamerForCluster := s.sensorStreamManager.GetStreamer(clusterID)
	if streamerForCluster == nil {
		return fmt.Errorf("connection to cluster %s not present", clusterID)
	}
	if !streamerForCluster.InjectMessage(msg) {
		return fmt.Errorf("connection to cluster %s may have closed", clusterID)
	}
	return nil
}

// Helper functions that generate scrape command messges.
/////////////////////////////////////////////////////////

func startScrapeCommand(scrapeID string, expectedHosts []string) *central.MsgToSensor {
	return &central.MsgToSensor{
		Msg: &central.MsgToSensor_ScrapeCommand{
			ScrapeCommand: &central.ScrapeCommand{
				ScrapeId: scrapeID,
				Command: &central.ScrapeCommand_StartScrape{
					StartScrape: &central.StartScrape{
						Hostnames: expectedHosts,
					},
				},
			},
		},
	}
}

func killScrapeCommand(scrapeID string) *central.MsgToSensor {
	return &central.MsgToSensor{
		Msg: &central.MsgToSensor_ScrapeCommand{
			ScrapeCommand: &central.ScrapeCommand{
				ScrapeId: scrapeID,
				Command: &central.ScrapeCommand_KillScrape{
					KillScrape: &central.KillScrape{},
				},
			},
		},
	}
}
