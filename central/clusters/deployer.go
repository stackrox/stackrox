package clusters

import (
	"strconv"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/defaultimages"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
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

	deployers = make(map[storage.ClusterType]Deployer)
)

// NewDeployer takes in a cluster and returns the cluster implementation
func NewDeployer(c *storage.Cluster) (Deployer, error) {
	dep, ok := deployers[c.Type]
	if !ok {
		return nil, status.Errorf(codes.Unimplemented, "Cluster type %s is not currently implemented", c.Type.String())
	}
	return dep, nil
}

// RenderOptions are options that control the rendering.
type RenderOptions struct {
	CreateUpgraderSA bool
}

// Deployer is the interface that defines how to get the specific files per orchestrator
// The first parameter is a wrap around the cluster and the second is the CA
type Deployer interface {
	Render(cluster *storage.Cluster, CA []byte, opts RenderOptions) ([]*zip.File, error)
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

func fieldsFromClusterAndRenderOpts(c *storage.Cluster, opts RenderOptions) (map[string]interface{}, error) {
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

	envVars := make(map[string]string)
	for _, feature := range features.Flags {
		envVars[feature.EnvVar()] = strconv.FormatBool(feature.Enabled())
	}

	fields := map[string]interface{}{
		"Image":         mainImageName.GetFullName(),
		"ImageRegistry": mainRegistry,
		"ImageRemote":   mainImageName.GetRemote(),
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

		"TolerationsEnabled":  !c.GetTolerationsConfig().GetDisabled(),
		"AdmissionController": c.AdmissionController,

		"OfflineMode": env.OfflineModeEnv.Setting(),

		"EnvVars": envVars,

		"CreateUpgraderSA": opts.CreateUpgraderSA,
	}

	return fields, nil
}
