package clusters

import (
	"bytes"
	"path"
	"path/filepath"
	"text/template"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/image"
	"github.com/stackrox/rox/pkg/defaultimages"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/netutil"
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

func generateCollectorImageNameFromString(collectorImage, tag string) (*storage.ImageName, error) {
	image, _, err := utils.GenerateImageNameFromString(collectorImage)
	if err != nil {
		return nil, err
	}
	utils.SetImageTagNoSha(image, tag)
	return image, nil
}

func generateCollectorImageName(mainImageName *storage.ImageName, collectorImage string) (*storage.ImageName, error) {
	collectorVersion := version.GetCollectorVersion()
	var collectorImageName *storage.ImageName
	if collectorImage != "" {
		var err error
		collectorImageName, err = generateCollectorImageNameFromString(collectorImage, collectorVersion)
		if err != nil {
			return nil, err
		}
	} else {
		collectorImageName = defaultimages.GenerateNamedImageFromMainImage(mainImageName, collectorVersion, defaultimages.Collector)
	}
	return collectorImageName, nil
}

func fieldsFromWrap(c Wrap) (map[string]interface{}, error) {
	mainImage, err := utils.GenerateImageFromStringWithDefaultTag(c.MainImage, version.GetMainVersion())
	if err != nil {
		return nil, err
	}
	mainImageName := mainImage.GetName()

	collectorImageName, err := generateCollectorImageName(mainImageName, c.CollectorImage)
	if err != nil {
		return nil, err
	}

	mainRegistry, err := urlfmt.FormatURL(mainImageName.GetRegistry(), urlfmt.HTTPS, urlfmt.NoTrailingSlash)
	if err != nil {
		return nil, err
	}
	collectorRegistry, err := urlfmt.FormatURL(collectorImageName.GetRegistry(), urlfmt.HTTPS, urlfmt.NoTrailingSlash)
	if err != nil {
		return nil, err
	}

	fields := map[string]interface{}{
		"Image":         mainImageName.GetFullName(),
		"ImageRegistry": mainRegistry,
		"ImageTag":      mainImageName.GetTag(),

		"PublicEndpointEnv": env.CentralEndpoint.EnvVar(),
		"PublicEndpoint":    c.CentralApiEndpoint,

		"ClusterIDEnv": env.ClusterID.EnvVar(),
		"ClusterID":    c.Id,
		"ClusterName":  c.Name,

		"AdvertisedEndpointEnv": env.AdvertisedEndpoint.EnvVar(),
		"AdvertisedEndpoint":    env.AdvertisedEndpoint.Setting(),

		"CollectorRegistry":              collectorRegistry,
		"CollectorImage":                 collectorImageName.GetFullName(),
		"CollectorModuleDownloadBaseURL": "https://collector-modules.stackrox.io/612dd2ee06b660e728292de9393e18c81a88f347ec52a39207c5166b5302b656",
		"CollectionMethod":               c.CollectionMethod.String(),

		"MonitoringEndpoint": netutil.WithDefaultPort(c.MonitoringEndpoint, defaultMonitoringPort),
		"ClusterType":        c.Type.String(),

		"AdmissionController": c.AdmissionController,
	}
	return fields, nil
}

func renderFilenames(filenames []string, c map[string]interface{}, staticFilenames ...string) ([]*zip.File, error) {
	var files []*zip.File
	for _, f := range filenames {
		t, err := image.ReadFileAndTemplate(f)
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
		contents, err := image.LoadFileContents(staticFilename)
		if err != nil {
			return nil, err
		}
		f := zip.NewFile(path.Base(staticFilename), []byte(contents), flags)
		files = append(files, f)
	}
	return files, nil
}
