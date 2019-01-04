package main

import (
	"github.com/stackrox/rox/compliance/collection/command"
	"github.com/stackrox/rox/compliance/collection/docker"
	file2 "github.com/stackrox/rox/compliance/collection/file"
	"github.com/stackrox/rox/generated/internalapi/compliance"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

func main() {
	var msgReturn compliance.ComplianceReturn
	var err error

	msgReturn.DockerData, err = docker.GetDockerData()
	if err != nil {
		log.Error(err)
	}

	msgReturn.Files, err = file2.CollectFiles()
	if err != nil {
		log.Error(err)
	}

	msgReturn.CommandLines, err = command.RetrieveCommands()
	if err != nil {
		log.Error(err)
	}

	log.Infof("%+v", msgReturn)
}
