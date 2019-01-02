package renderer

import (
	"strings"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/image"
	"github.com/stackrox/rox/pkg/zip"
)

func init() {
	Deployers[storage.ClusterType_SWARM_CLUSTER] = newSwarm()
	Deployers[storage.ClusterType_DOCKER_EE_CLUSTER] = newSwarm()
}

type swarm struct{}

func newSwarm() deployer {
	return &swarm{}
}

func (s *swarm) Render(c Config) ([]*zip.File, error) {
	filenames := []string{
		"central.yaml",
		"central.sh",
		"clairify.yaml",
		"clairify.sh",
	}

	var files []*zip.File
	for k, v := range c.SecretsByteMap {
		files = append(files, zip.NewFile(k, v, zip.Sensitive))
	}

	for _, f := range filenames {
		tmpl, err := image.SwarmBox.FindString(f)
		if err != nil {
			return nil, err
		}
		data, err := executeRawTemplate(tmpl, &c)
		if err != nil {
			return nil, err
		}
		var flags zip.FileFlags
		if strings.HasSuffix(f, ".sh") {
			flags |= zip.Executable
		}
		// Trim the first section off of the path because it defines the orchestrator
		files = append(files, zip.NewFile(f, data, flags))
	}
	files = append(files, dockerAuthFile)
	return wrapFiles(files, &c)
}

func (s *swarm) Instructions(c Config) string {
	return `To deploy:
  1. Unzip the deployment bundle.
  2. Run central.sh.
  3. If you want to run the StackRox Clairify scanner, run clairify.sh.`
}
