package swarm

import (
	"os"
	"time"

	"github.com/stackrox/rox/benchmarks/checks/utils"
	"github.com/stackrox/rox/generated/storage"
)

type caCertificates struct{}

func (c *caCertificates) Definition() utils.Definition {
	return utils.Definition{
		BenchmarkCheckDefinition: storage.BenchmarkCheckDefinition{
			Name:        "CIS Docker v1.1.0 - 7.9",
			Description: "Ensure CA certificates are rotated as appropriate",
		},
	}
}

func (c *caCertificates) Run() (result storage.BenchmarkCheckResult) {
	utils.Note(&result)
	info, err := os.Stat(utils.ContainerPath("/var/lib/docker/swarm/certificates/swarm-root-ca.crt"))
	if err != nil {
		utils.Warn(&result)
		utils.AddNotef(&result, "Could not check age of Swarm Root CA: %+v", err)
		return
	}
	age := int(time.Since(info.ModTime()).Hours() / 24)
	utils.AddNotef(&result, "Check that the swarm root CA is rotated periodically. It was last rotated %v days ago", age)
	return
}

// NewCACertificates implements CIS-7.9
func NewCACertificates() utils.Check {
	return &caCertificates{}
}
