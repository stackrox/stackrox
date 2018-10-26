// Package central generates configurations for the Central service.
package central

import (
	"bytes"
	"fmt"
	"io"
	"path"
	"strings"
	"text/template"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/templates"
	"github.com/stackrox/rox/pkg/zip"
)

var (
	log = logging.LoggerForModule()
)

// ExternalPersistence holds the data for a volume that is already created (e.g. docker volume, PV, etc)
type ExternalPersistence struct {
	Name         string
	StorageClass string
}

// HostPathPersistence describes the parameters for a bind mount
type HostPathPersistence struct {
	Name              string
	HostPath          string
	NodeSelectorKey   string
	NodeSelectorValue string
}

// WithNodeSelector is a helper function for the templater that returns if node selectors are used
func (h *HostPathPersistence) WithNodeSelector() bool {
	if h == nil {
		return false
	}
	return h.NodeSelectorKey != ""
}

// CommonConfig contains the common config between orchestrators that cannot be placed at the top level
// Image is an example as it can be parameterized per orchestrator with different defaults so it cannot be placed
// at the top level
type CommonConfig struct {
	MainImage     string
	ClairifyImage string
}

// MonitoringType is the enum for the place monitoring is hosted
type MonitoringType int

// Types of monitoring
const (
	OnPrem = iota
	None
	StackRoxHosted
)

// String returns the string form of the enum
func (m MonitoringType) String() string {
	switch m {
	case OnPrem:
		return "on-prem"
	case None:
		return "none"
	case StackRoxHosted:
		return "stackrox-hosted"
	}
	return "unknown"
}

// OnPrem is true if the monitoring is hosted on prem
func (m MonitoringType) OnPrem() bool {
	return m == OnPrem
}

// StackRoxHosted is true if the monitoring is hosted by StackRox
func (m MonitoringType) StackRoxHosted() bool {
	return m == StackRoxHosted
}

// None returns true if there is no monitoring solution
func (m MonitoringType) None() bool {
	return m == None
}

// K8sConfig contains k8s fields
type K8sConfig struct {
	CommonConfig

	Namespace string
	Registry  string

	// These variables are not prompted for by Cobra, but are set based on
	// provided inputs for use in templating.
	MainImageTag     string
	ClairifyImageTag string

	MonitoringEndpoint string
	MonitoringImage    string
	MonitoringType     MonitoringType
}

// SwarmConfig contains swarm fields
type SwarmConfig struct {
	CommonConfig

	NetworkMode string
	PublicPort  int
}

// Config configures the deployer for the central service.
type Config struct {
	ClusterType v1.ClusterType
	K8sConfig   *K8sConfig
	SwarmConfig *SwarmConfig

	External *ExternalPersistence
	HostPath *HostPathPersistence
	Features []features.FeatureFlag
	Password string
}

type deployer interface {
	Render(Config) ([]*v1.File, error)
	Instructions() string
}

// Deployers contains all implementations for central deployment generators.
var Deployers = make(map[v1.ClusterType]deployer)

func executeTemplate(temp *template.Template, c *Config) (string, error) {
	var b []byte
	buf := bytes.NewBuffer(b)
	err := temp.Execute(buf, c)
	if err != nil {
		log.Errorf("Template execution failed: %s", err)
		return "", err
	}
	return buf.String(), nil
}

func generateMonitoringImage(mainImage string) string {
	img := types.Wrapper{Image: utils.GenerateImageFromString(mainImage)}
	remote := img.Namespace() + "/monitoring"
	// This handles the case where there is no namespace. e.g. stackrox.io/collector:latest
	if img.Repo() == "" {
		remote = "monitoring"
	}
	return fmt.Sprintf("%s/%s:%s", img.GetName().GetRegistry(), remote, img.GetName().GetTag())
}

func renderFilenames(filenames []string, c *Config, staticFilenames ...string) ([]*v1.File, error) {
	var files []*v1.File
	for _, f := range filenames {
		t, err := templates.ReadFileAndTemplate(f)
		if err != nil {
			return nil, err
		}
		d, err := executeTemplate(t, c)
		if err != nil {
			return nil, err
		}

		// Trim the first section off of the path because it defines the orchestrator
		path := f[strings.Index(f, "/")+1:]
		files = append(files, zip.NewFile(path, d, strings.HasSuffix(f, ".sh")))
	}
	for _, staticFilename := range staticFilenames {
		f, err := zip.NewFromFile(staticFilename, path.Base(staticFilename), path.Ext(staticFilename) == ".sh")
		if err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	files = append(files, zip.NewFile("README", standardizeWhitespace(Deployers[c.ClusterType].Instructions()), false))

	if features.HtpasswdAuth.Enabled() {
		htpasswd, err := generateHtpasswd(c)
		if err != nil {
			return nil, err
		}
		files = append(files, zip.NewFile("htpasswd", htpasswd, false))
		files = append(files, zip.NewFile("password", c.Password+"\n", false))
	}
	return files, nil
}

// WriteInstructions writes the instructions for the configured cluster
// to the provided writer.
func (c Config) WriteInstructions(w io.Writer) {
	fmt.Fprint(w, standardizeWhitespace(Deployers[c.ClusterType].Instructions()))
}

func standardizeWhitespace(instructions string) string {
	return strings.TrimSpace(instructions) + "\n"
}
