package main

import (
	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/compliance/collection/command"
	"github.com/stackrox/rox/compliance/collection/containerruntimes/crio"
	"github.com/stackrox/rox/compliance/collection/containerruntimes/docker"
	"github.com/stackrox/rox/compliance/collection/file"
	"github.com/stackrox/rox/generated/internalapi/compliance"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/compliance/checks/standards"
	"github.com/stackrox/rox/pkg/env"
)

func runScrape(client sensor.ComplianceService_CommunicateClient, scrapeConfig *sensor.MsgToCompliance_ScrapeConfig, run *sensor.MsgToCompliance_TriggerRun) error {
	complianceData := gatherData(scrapeConfig, run.GetScrapeId())
	compressedData, err := compressDockerData(complianceData.DockerData)
	if err != nil {
		log.Errorf("Compressing Docker data failed: %v", err)
	}

	msgReturn := compliance.ComplianceReturn{
		NodeName:             getNode(),
		ScrapeId:             run.GetScrapeId(),
		DockerData:           compressedData,
		CommandLines:         complianceData.CommandLines,
		Files:                complianceData.Files,
		SystemdFiles:         complianceData.SystemdFiles,
		ContainerRuntimeInfo: complianceData.ContainerRuntimeInfo,
		Time:                 complianceData.Time,
	}

	log.Info("Trying to push return to sensor")
	err = client.Send(&sensor.MsgFromCompliance{
		Node: getNode(),
		Msg: &sensor.MsgFromCompliance_Return{
			Return: &msgReturn,
		},
	})
	if err != nil {
		log.Errorf("Error posting compliance data to %v: %v", env.AdvertisedEndpoint.Setting(), err)
	} else {
		log.Info("Successfully pushed data to sensor")
	}
	return err
}

func gatherData(scrapeConfig *sensor.MsgToCompliance_ScrapeConfig, scrapeID string) *standards.ComplianceData {
	complianceData := &standards.ComplianceData{
		NodeName: getNode(),
	}

	log.Infof("Running compliance scrape %q for node %q", scrapeID, getNode())

	var err error
	log.Infof("Container runtime is %v", scrapeConfig.GetContainerRuntime())
	if scrapeConfig.GetContainerRuntime() == storage.ContainerRuntime_DOCKER_CONTAINER_RUNTIME {
		log.Info("Starting to collect Docker data")
		complianceData.DockerData, complianceData.ContainerRuntimeInfo, err = docker.GetDockerData()
		if err != nil {
			log.Errorf("Collecting Docker data failed: %v", err)
		} else {
			log.Info("Successfully collected relevant Docker data")
		}
	} else if scrapeConfig.GetContainerRuntime() == storage.ContainerRuntime_CRIO_CONTAINER_RUNTIME {
		log.Info("Collecting relevant CRI-O data")
		complianceData.ContainerRuntimeInfo, err = crio.GetContainerRuntimeData()
		if err != nil {
			log.Errorf("Collecting CRI-O data failed: %v", err)
		} else {
			log.Info("Successfully collected relevant CRI-O data")
		}
	} else {
		log.Info("Unknown container runtime, not collecting any data ...")
	}

	log.Info("Starting to collect systemd files")
	complianceData.SystemdFiles, err = file.CollectSystemdFiles()
	if err != nil {
		log.Errorf("Collecting systemd files failed: %v", err)
	}
	log.Info("Successfully collected relevant systemd files")

	log.Info("Starting to collect configuration files")
	complianceData.Files, err = file.CollectFiles()
	if err != nil {
		log.Errorf("Collecting configuration files failed: %v", err)
	}
	log.Info("Successfully collected relevant configuration files")

	log.Info("Starting to collect command lines")
	complianceData.CommandLines, err = command.RetrieveCommands()
	if err != nil {
		log.Errorf("Collecting command lines failed: %v", err)
	}
	log.Info("Successfully collected relevant command lines")

	complianceData.Time = types.TimestampNow()
	return complianceData
}
