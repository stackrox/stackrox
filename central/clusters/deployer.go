package clusters

import (
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/helm/charts"
	"github.com/stackrox/rox/pkg/images/defaults"
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/urlfmt"
	"github.com/stackrox/rox/pkg/version"
)

// RenderOptions are options that control the rendering.
type RenderOptions struct {
	CreateUpgraderSA bool
	SlimCollector    bool
	IstioVersion     string

	DisablePodSecurityPolicies bool
}

// FieldsFromClusterAndRenderOpts gets the template values for values.yaml
func FieldsFromClusterAndRenderOpts(c *storage.Cluster, imageFlavor *defaults.ImageFlavor, opts RenderOptions) (*charts.MetaValues, error) {
	mainImage, collectorImage, err := MakeClusterImageNames(imageFlavor, c)
	if err != nil {
		return nil, err
	}

	baseValues := getBaseMetaValues(c, imageFlavor.Versions, imageFlavor.ScannerSlimImageName, imageFlavor.ChartRepo, &opts)
	setMainOverride(mainImage, baseValues)
	deriveScannerRemoteFromMain(mainImage, baseValues)
	baseValues.EnablePodSecurityPolicies = !opts.DisablePodSecurityPolicies

	collectorFull, collectorSlim := determineCollectorImages(mainImage, collectorImage, imageFlavor)
	setCollectorOverrideToMetaValues(collectorFull, collectorSlim, baseValues)

	return baseValues, nil
}

// MakeClusterImageNames creates storage.ImageName objects for provided storage.Cluster main and collector images.
func MakeClusterImageNames(flavor *defaults.ImageFlavor, c *storage.Cluster) (*storage.ImageName, *storage.ImageName, error) {
	mainImage, err := utils.GenerateImageFromStringWithDefaultTag(c.MainImage, flavor.MainImageTag)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "generating main image from cluster value (%s)", c.MainImage)
	}
	mainImageName := mainImage.GetName()

	var collectorImageName *storage.ImageName
	if c.CollectorImage != "" {
		collectorImage, err := utils.GenerateImageFromString(c.CollectorImage)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "generating collector image from cluster value (%s)", c.CollectorImage)
		}
		collectorImageName = collectorImage.GetName()
	}

	return mainImageName, collectorImageName, nil
}

// deriveScannerRemoteFromMain sets scanner-slim image remote, so that it comes from the same location as the main image
func deriveScannerRemoteFromMain(mainImage *storage.ImageName, metaValues *charts.MetaValues) {
	scannerRemoteSlice := strings.Split(mainImage.Remote, "/")
	if len(scannerRemoteSlice) > 0 {
		scannerRemoteSlice[len(scannerRemoteSlice)-1] = metaValues.ScannerSlimImageRemote
		metaValues.ScannerSlimImageRemote = strings.Join(scannerRemoteSlice, "/")
	}
}

// setMainOverride adds main image values to meta values as defined in secured cluster object.
func setMainOverride(mainImage *storage.ImageName, metaValues *charts.MetaValues) {
	metaValues.MainRegistry = mainImage.Registry
	metaValues.ImageRemote = mainImage.Remote
	metaValues.ImageTag = mainImage.Tag
}

// setCollectorOverrideToMetaValues adds collector image values to meta values as defined in the provided *storage.ImageName objects.
func setCollectorOverrideToMetaValues(collectorImage *storage.ImageName, collectorSlimImage *storage.ImageName, metaValues *charts.MetaValues) {
	metaValues.CollectorRegistry = collectorImage.Registry
	metaValues.CollectorFullImageRemote = collectorImage.Remote
	metaValues.CollectorSlimImageRemote = collectorSlimImage.Remote
	metaValues.CollectorFullImageTag = collectorImage.Tag
	metaValues.CollectorSlimImageTag = collectorSlimImage.Tag
}

// determineCollectorImages is used to derive collector slim and full images from provided main and collector values.
// The collector repository defined in the cluster object can be passed from roxctl or as direct
// input in the UI when creating a new secured cluster. If no value is provided, the collector image
// will be derived from the main image. For example:
// main image: "quay.io/rhacs/main" => collector image: "quay.io/rhacs/collector"
// Similarly, slim collector will be derived. However, if a collector registry is specified and
// current image flavor has different image names for collector slim and full: collector slim has to be
// derived from full instead. For example:
// collector full image: "custom.registry.io/collector" => collector slim image: "custom.registry.io/collector-slim"
// returned images are: (collectorFull, collectorSlim)
func determineCollectorImages(clusterMainImage, clusterCollectorImage *storage.ImageName, imageFlavor *defaults.ImageFlavor) (*storage.ImageName, *storage.ImageName) {
	var collectorImageFull *storage.ImageName
	if clusterCollectorImage == nil && imageFlavor.IsImageDefaultMain(clusterMainImage) {
		collectorImageFull = &storage.ImageName{
			Registry: imageFlavor.CollectorRegistry,
			Remote:   imageFlavor.CollectorImageName,
		}
	} else if clusterCollectorImage == nil {
		collectorImageFull = deriveImageWithNewName(clusterMainImage, imageFlavor.CollectorImageName)
	} else {
		collectorImageFull = clusterCollectorImage.Clone()
	}
	collectorImageFull.Tag = imageFlavor.CollectorImageTag
	collectorImageSlim := deriveImageWithNewName(collectorImageFull, imageFlavor.CollectorSlimImageName)
	collectorImageSlim.Tag = imageFlavor.CollectorSlimImageTag
	return collectorImageFull, collectorImageSlim
}

// deriveImageWithNewName returns registry and repository values derived from a base image.
// Slices base image taking into account image namespace and returns values for new image in the same repository as
// base image. For example:
// base image: "quay.io/namespace/main" => another: "quay.io/namespace/another"
func deriveImageWithNewName(baseImage *storage.ImageName, name string) *storage.ImageName {
	// TODO(RS-387): check if this split is still needed. Since we are not consistent in how we split the image, configured image names might have namespaces
	imageNameWithoutNamespace := name[strings.IndexRune(name, '/')+1:]
	baseRemote := baseImage.GetRemote()
	remote := baseRemote[:strings.IndexRune(baseRemote, '/')+1] + imageNameWithoutNamespace
	return &storage.ImageName{
		Registry: baseImage.Registry,
		Remote:   remote,
	}
}

func getBaseMetaValues(c *storage.Cluster, versions version.Versions, scannerSlimImageRemote string, chartRepo defaults.ChartRepo, opts *RenderOptions) *charts.MetaValues {
	envVars := make(map[string]string)
	for _, feature := range features.Flags {
		envVars[feature.EnvVar()] = strconv.FormatBool(feature.Enabled())
	}

	command := "kubectl"
	if c.Type == storage.ClusterType_OPENSHIFT_CLUSTER || c.Type == storage.ClusterType_OPENSHIFT4_CLUSTER {
		command = "oc"
	}

	return &charts.MetaValues{
		ClusterName: c.Name,
		ClusterType: c.Type.String(),

		PublicEndpoint:     urlfmt.FormatURL(c.CentralApiEndpoint, urlfmt.NONE, urlfmt.NoTrailingSlash),
		AdvertisedEndpoint: urlfmt.FormatURL(env.AdvertisedEndpoint.Setting(), urlfmt.NONE, urlfmt.NoTrailingSlash),

		CollectionMethod: c.CollectionMethod.String(),

		ChartRepo: chartRepo,

		TolerationsEnabled: !c.GetTolerationsConfig().GetDisabled(),
		CreateUpgraderSA:   opts.CreateUpgraderSA,

		EnvVars: envVars,

		K8sCommand: command,

		OfflineMode: env.OfflineModeEnv.BooleanSetting(),

		SlimCollector: opts.SlimCollector,

		ScannerImageTag:        versions.ScannerVersion,
		ScannerSlimImageRemote: scannerSlimImageRemote,

		KubectlOutput: true,

		Versions: versions,

		FeatureFlags: make(map[string]interface{}),

		AdmissionController:              c.AdmissionController,
		AdmissionControlListenOnUpdates:  c.GetAdmissionControllerUpdates(),
		AdmissionControlListenOnEvents:   c.GetAdmissionControllerEvents(),
		DisableBypass:                    c.GetDynamicConfig().GetAdmissionControllerConfig().GetDisableBypass(),
		TimeoutSeconds:                   c.GetDynamicConfig().GetAdmissionControllerConfig().GetTimeoutSeconds(),
		ScanInline:                       c.GetDynamicConfig().GetAdmissionControllerConfig().GetScanInline(),
		AdmissionControllerEnabled:       c.GetDynamicConfig().GetAdmissionControllerConfig().GetEnabled(),
		AdmissionControlEnforceOnUpdates: c.GetDynamicConfig().GetAdmissionControllerConfig().GetEnforceOnUpdates(),
		ReleaseBuild:                     buildinfo.ReleaseBuild,

		EnablePodSecurityPolicies: false,
	}
}
