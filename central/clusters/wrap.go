package clusters

import (
	"bytes"
	"fmt"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/netutil"
	"github.com/stackrox/rox/pkg/templates"
	"github.com/stackrox/rox/pkg/urlfmt"
	"github.com/stackrox/rox/pkg/version"
	"github.com/stackrox/rox/pkg/zip"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	defaultMonitoringPort = 8186
)

var (
	log = logging.LoggerForModule()
)

// Wrap adds additional functionality to a storage.Cluster.
type Wrap storage.Cluster

// NewDeployer takes in a cluster and returns the cluster implementation
func NewDeployer(c *storage.Cluster) (Deployer, error) {
	dep, ok := deployers[c.Type]
	if !ok {
		return nil, status.Errorf(codes.Unimplemented, "Cluster type %s is not currently implemented", c.Type.String())
	}
	return dep, nil
}

// Deployer is the interface that defines how to get the specific files per orchestrator
// The first parameter is a wrap around the cluster and the second is the CA
type Deployer interface {
	Render(wrap Wrap, CA []byte) ([]*zip.File, error)
}

var deployers = make(map[storage.ClusterType]Deployer)

func executeTemplate(temp *template.Template, fields map[string]interface{}) ([]byte, error) {
	var b []byte
	buf := bytes.NewBuffer(b)
	err := temp.Execute(buf, fields)
	if err != nil {
		log.Errorf("template execution: %s", err)
		return nil, err
	}
	return buf.Bytes(), nil
}

func generateCollectorImage(mainImageName *storage.ImageName, tag string) *storage.ImageName {
	// Populate the tag
	collectorName := &storage.ImageName{
		Tag: tag,
	}
	// Populate Registry
	collectorName.Registry = mainImageName.GetRegistry()
	if mainImageName.GetRegistry() == "stackrox.io" {
		collectorName.Registry = "collector.stackrox.io"
	}
	// Populate Remote
	// This handles the case where there is no namespace. e.g. stackrox.io/collector:latest
	if slashIdx := strings.Index(mainImageName.GetRemote(), "/"); slashIdx == -1 {
		collectorName.Remote = "collector"
	} else {
		collectorName.Remote = mainImageName.GetRemote()[:slashIdx] + "/collector"
	}
	// Populate FullName
	collectorName.FullName = fmt.Sprintf("%s/%s:%s",
		collectorName.GetRegistry(), collectorName.GetRemote(), collectorName.GetTag())
	return collectorName
}

func fieldsFromWrap(c Wrap) (map[string]interface{}, error) {
	mainImageName := utils.GenerateImageFromString(c.MainImage).GetName()
	collectorName := generateCollectorImage(mainImageName, version.GetCollectorVersion())

	mainRegistry, err := urlfmt.FormatURL(mainImageName.GetRegistry(), urlfmt.HTTPS, urlfmt.NoTrailingSlash)
	if err != nil {
		return nil, err
	}
	collectorRegistry, err := urlfmt.FormatURL(collectorName.GetRegistry(), urlfmt.HTTPS, urlfmt.NoTrailingSlash)
	if err != nil {
		return nil, err
	}

	fields := map[string]interface{}{
		"Image":         c.MainImage,
		"ImageRegistry": mainRegistry,
		"ImageTag":      mainImageName.GetTag(),

		"PublicEndpointEnv": env.CentralEndpoint.EnvVar(),
		"PublicEndpoint":    c.CentralApiEndpoint,

		"ClusterIDEnv": env.ClusterID.EnvVar(),
		"ClusterID":    c.Id,
		"ClusterName":  c.Name,

		"AdvertisedEndpointEnv": env.AdvertisedEndpoint.EnvVar(),
		"AdvertisedEndpoint":    env.AdvertisedEndpoint.Setting(),

		"RuntimeSupport":                 c.RuntimeSupport,
		"CollectorRegistry":              collectorRegistry,
		"CollectorImage":                 collectorName.GetFullName(),
		"CollectorEbpf":                  features.CollectorEbpf.Enabled(),
		"CollectorModuleDownloadBaseURL": "https://collector-modules.stackrox.io/612dd2ee06b660e728292de9393e18c81a88f347ec52a39207c5166b5302b656",

		"MonitoringEndpoint": netutil.WithDefaultPort(c.MonitoringEndpoint, defaultMonitoringPort),
		"ClusterType":        c.Type.String(),

		"AdmissionController": c.AdmissionController,
	}
	return fields, nil
}

func renderFilenames(filenames []string, c map[string]interface{}, staticFilenames ...string) ([]*zip.File, error) {
	var files []*zip.File
	for _, f := range filenames {
		t, err := templates.ReadFileAndTemplate(f)
		if err != nil {
			return nil, err
		}
		d, err := executeTemplate(t, c)
		if err != nil {
			return nil, err
		}
		var flags zip.FileFlags
		if path.Ext(f) == ".sh" {
			flags |= zip.Executable
		}
		files = append(files, zip.NewFile(filepath.Base(f), d, flags))
	}
	for _, staticFilename := range staticFilenames {
		var flags zip.FileFlags
		if path.Ext(staticFilename) == ".sh" {
			flags |= zip.Executable
		}
		f, err := zip.NewFromFile(staticFilename, path.Base(staticFilename), flags)
		if err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	return files, nil
}
