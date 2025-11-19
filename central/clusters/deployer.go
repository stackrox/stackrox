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
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/urlfmt"
)

// RenderOptions are options that control the rendering.
type RenderOptions struct {
	CreateUpgraderSA bool
	IstioVersion     string

	DisablePodSecurityPolicies bool
}

// FieldsFromClusterAndRenderOpts gets the template values for values.yaml
func FieldsFromClusterAndRenderOpts(c *storage.Cluster, imageFlavor *defaults.ImageFlavor, opts RenderOptions) (*charts.MetaValues, error) {
	mainImage, collectorImage, err := MakeClusterImageNames(imageFlavor, c)
	if err != nil {
		return nil, err
	}

	baseValues := getBaseMetaValues(c, imageFlavor, imageFlavor.ChartRepo, &opts)
	setMainOverride(mainImage, baseValues)
	deriveScannerRemoteFromMain(mainImage, baseValues)
	baseValues.EnablePodSecurityPolicies = !opts.DisablePodSecurityPolicies

	collector := determineCollectorImage(mainImage, collectorImage, imageFlavor)
	setCollectorOverrideToMetaValues(collector, baseValues)

	return baseValues, nil
}

// MakeClusterImageNames creates storage.ImageName objects for provided storage.Cluster main and collector images.
func MakeClusterImageNames(flavor *defaults.ImageFlavor, c *storage.Cluster) (*storage.ImageName, *storage.ImageName, error) {
	mainImage, err := utils.GenerateImageFromStringWithDefaultTag(c.GetMainImage(), flavor.MainImageTag)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "generating main image from cluster value (%s)", c.GetMainImage())
	}
	mainImageName := mainImage.GetName()

	var collectorImageName *storage.ImageName
	if c.GetCollectorImage() != "" {
		collectorImage, err := utils.GenerateImageFromString(c.GetCollectorImage())
		if err != nil {
			return nil, nil, errors.Wrapf(err, "generating collector image from cluster value (%s)", c.GetCollectorImage())
		}
		collectorImageName = collectorImage.GetName()
	}

	return mainImageName, collectorImageName, nil
}

// deriveScannerRemoteFromMain sets scanner-slim image remote, so that it comes from the same location as the main image
func deriveScannerRemoteFromMain(mainImage *storage.ImageName, metaValues *charts.MetaValues) {
	scannerRemoteSlice := strings.Split(mainImage.GetRemote(), "/")
	if len(scannerRemoteSlice) > 0 {
		scannerRemoteSlice[len(scannerRemoteSlice)-1] = metaValues.ScannerSlimImageRemote
		metaValues.ScannerSlimImageRemote = strings.Join(scannerRemoteSlice, "/")
	}
}

// setMainOverride adds main image values to meta values as defined in secured cluster object.
func setMainOverride(mainImage *storage.ImageName, metaValues *charts.MetaValues) {
	metaValues.MainRegistry = mainImage.GetRegistry()
	metaValues.ImageRemote = mainImage.GetRemote()
	metaValues.ImageTag = mainImage.GetTag()
}

// setCollectorOverrideToMetaValues adds collector image values to meta values as defined in the provided *storage.ImageName objects.
func setCollectorOverrideToMetaValues(collectorImage *storage.ImageName, metaValues *charts.MetaValues) {
	metaValues.CollectorRegistry = collectorImage.GetRegistry()
	metaValues.CollectorImageRemote = collectorImage.GetRemote()
	metaValues.CollectorImageTag = collectorImage.GetTag()
}

// determineCollectorImage is used to derive the collector image from provided main and collector values.
// The collector repository defined in the cluster object can be passed from roxctl or as direct
// input in the UI when creating a new secured cluster. If no value is provided, the collector image
// will be derived from the main image. For example:
// main image: "quay.io/rhacs/main" => collector image: "quay.io/rhacs/collector"
func determineCollectorImage(clusterMainImage, clusterCollectorImage *storage.ImageName, imageFlavor *defaults.ImageFlavor) *storage.ImageName {
	var collectorImage *storage.ImageName
	if clusterCollectorImage == nil && imageFlavor.IsImageDefaultMain(clusterMainImage) {
		collectorImage = &storage.ImageName{
			Registry: imageFlavor.CollectorRegistry,
			Remote:   imageFlavor.CollectorImageName,
		}
	} else if clusterCollectorImage == nil {
		collectorImage = deriveImageWithNewName(clusterMainImage, imageFlavor.CollectorImageName)
	} else {
		collectorImage = clusterCollectorImage.CloneVT()
	}
	collectorImage.Tag = imageFlavor.CollectorImageTag
	return collectorImage
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
		Registry: baseImage.GetRegistry(),
		Remote:   remote,
	}
}

func getBaseMetaValues(c *storage.Cluster, imageFlavor *defaults.ImageFlavor, chartRepo defaults.ChartRepo, opts *RenderOptions) *charts.MetaValues {
	versions := imageFlavor.Versions

	command := "kubectl"
	if c.GetType() == storage.ClusterType_OPENSHIFT_CLUSTER || c.GetType() == storage.ClusterType_OPENSHIFT4_CLUSTER {
		command = "oc"
	}

	return &charts.MetaValues{
		ClusterName: c.GetName(),
		ClusterType: c.GetType().String(),

		PublicEndpoint:     urlfmt.FormatURL(c.GetCentralApiEndpoint(), urlfmt.NONE, urlfmt.NoTrailingSlash),
		AdvertisedEndpoint: urlfmt.FormatURL(env.AdvertisedEndpoint.Setting(), urlfmt.NONE, urlfmt.NoTrailingSlash),

		CollectionMethod: c.GetCollectionMethod().String(),

		ChartRepo: chartRepo,

		TolerationsEnabled: !c.GetTolerationsConfig().GetDisabled(),
		CreateUpgraderSA:   opts.CreateUpgraderSA,

		EnvVars: getFeatureFlagsAsManifestBundleEnv(),

		K8sCommand: command,

		OfflineMode: env.OfflineModeEnv.BooleanSetting(),

		FactImageTag:    versions.FactVersion,
		FactImageRemote: imageFlavor.FactImageName,

		ScannerImageTag:        versions.ScannerVersion,
		ScannerSlimImageRemote: imageFlavor.ScannerSlimImageName,

		KubectlOutput: true,

		Versions: versions,

		FeatureFlags: features.GetFeatureFlagsAsGenericMap(),

		AdmissionController:              c.GetAdmissionController(),
		AdmissionControlListenOnUpdates:  c.GetAdmissionControllerUpdates(),
		AdmissionControlListenOnEvents:   c.GetAdmissionControllerEvents(),
		DisableBypass:                    c.GetDynamicConfig().GetAdmissionControllerConfig().GetDisableBypass(),
		TimeoutSeconds:                   c.GetDynamicConfig().GetAdmissionControllerConfig().GetTimeoutSeconds(),
		ScanInline:                       c.GetDynamicConfig().GetAdmissionControllerConfig().GetScanInline(),
		AdmissionControllerEnabled:       c.GetDynamicConfig().GetAdmissionControllerConfig().GetEnabled(),
		AdmissionControlEnforceOnUpdates: c.GetDynamicConfig().GetAdmissionControllerConfig().GetEnforceOnUpdates(),
		AdmissionControllerFailOnError:   c.GetAdmissionControllerFailOnError(),
		AutoLockProcessBaselines:         c.GetDynamicConfig().GetAutoLockProcessBaselinesConfig().GetEnabled(),
		ReleaseBuild:                     buildinfo.ReleaseBuild,

		EnablePodSecurityPolicies: false,
	}
}

func getFeatureFlagsAsManifestBundleEnv() map[string]string {
	// For the environment variables we need to filter out ROX_SCANNER_V4, because it would
	// wrongly enable Scanner V4 delegated scanning on secured clusters which are set up
	// using manifest bundles. But delegated scanning is not supported for manifest bundle
	// installed secured clusters.
	skipFeatureFlags := set.NewFrozenStringSet("ROX_SCANNER_V4")
	featureFlagVals := make(map[string]string)
	for _, feature := range features.Flags {
		envVar := feature.EnvVar()
		if skipFeatureFlags.Contains(envVar) {
			continue
		}
		featureFlagVals[envVar] = strconv.FormatBool(feature.Enabled())
	}
	return featureFlagVals
}
