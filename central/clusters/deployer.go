package clusters

import (
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/devbuild"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/helm/charts"
	"github.com/stackrox/rox/pkg/images/defaults"
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/urlfmt"
	"github.com/stackrox/rox/pkg/version"
)

var (
	log = logging.LoggerForModule()
)

// RenderOptions are options that control the rendering.
type RenderOptions struct {
	CreateUpgraderSA bool
	SlimCollector    bool
	IstioVersion     string
}

// FieldsFromClusterAndRenderOpts gets the template values for values.yaml
func FieldsFromClusterAndRenderOpts(c *storage.Cluster, imageFlavor *defaults.ImageFlavor, opts RenderOptions) (charts.MetaValues, error) {
	mainImage, collectorImage, err := MakeClusterImageNames(imageFlavor, c)
	if err != nil {
		return nil, err
	}

	baseValues := getBaseMetaValues(c, imageFlavor.Versions, &opts)
	setMainOverride(mainImage, baseValues)
	setCollectorOverride(mainImage, collectorImage, imageFlavor, baseValues)

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

// setMainOverride adds main image values to meta values as defined in secured cluster object.
func setMainOverride(mainImage *storage.ImageName, metaValues charts.MetaValues) {
	metaValues["MainRegistry"] = mainImage.Registry
	metaValues["ImageRemote"] = mainImage.Remote
	metaValues["ImageTag"] = mainImage.Tag
}

// setCollectorOverride adds collector full and slim image reference to meta values object.
// The collector repository defined in the cluster object can be passed from roxctl or as direct
// input in the UI when creating a new secured cluster. If no value is provided, the collector image
// will be derived from the main image. For example:
// main image: "quay.io/rhacs/main" => collector image: "quay.io/rhacs/collector"
// Similarly, slim collector will be derived. However, if a collector registry is specified and
// current image flavor has different image names for collector slim and full: collector slim has to be
// derived from full instead. For example:
// collector full image: "custom.registry.io/collector" => collector slim image: "custom.registry.io/collector-slim"
func setCollectorOverride(mainImage, collectorImage *storage.ImageName, imageFlavor *defaults.ImageFlavor, metaValues charts.MetaValues) {
	if collectorImage != nil {
		// Use provided collector image and derive collector slim
		metaValues["CollectorRegistry"] = collectorImage.Registry
		metaValues["CollectorFullImageRemote"] = collectorImage.Remote
		_, derivedName := deriveImageWithNewName(collectorImage, imageFlavor.CollectorSlimImageName)
		log.Infof("Derived collector slim image from collector full as: %s/%s", collectorImage.Registry, derivedName)
		metaValues["CollectorSlimImageRemote"] = derivedName
	} else {
		if imageFlavor.IsImageDefaultMain(mainImage) {
			// Use all defaults from imageFlavor
			metaValues["CollectorRegistry"] = imageFlavor.CollectorRegistry
			metaValues["CollectorFullImageRemote"] = imageFlavor.CollectorImageName
			metaValues["CollectorSlimImageRemote"] = imageFlavor.CollectorSlimImageName
		} else {
			// Derive collector values from main image
			derivedRegistry, derivedName := deriveImageWithNewName(mainImage, imageFlavor.CollectorImageName)
			log.Infof("Derived collector full image from main as: %s/%s", derivedRegistry, derivedName)
			metaValues["CollectorRegistry"] = derivedRegistry
			metaValues["CollectorFullImageRemote"] = derivedName
			_, derivedName = deriveImageWithNewName(mainImage, imageFlavor.CollectorSlimImageName)
			log.Infof("Derived collector slim image from collector full as: %s/%s", derivedRegistry, derivedName)
			metaValues["CollectorSlimImageRemote"] = derivedName
		}
	}
	metaValues["CollectorFullImageTag"] = imageFlavor.CollectorImageTag
	metaValues["CollectorSlimImageTag"] = imageFlavor.CollectorSlimImageTag
}

// deriveImageWithNewName returns registry and repository values derived from a base image.
// Slices base image taking into account image namespace and returns values for new image in the same repository as
// base image. For example:
// base image: "quay.io/namespace/main" => another: "quay.io/namespace/another"
// Return values are split as ("quay.io", "namespace/another")
func deriveImageWithNewName(baseImage *storage.ImageName, name string) (string, string) {
	registry := baseImage.Registry

	// This handles the case where there is no namespace. e.g. stackrox.io/NAME:tag
	var remote string
	if slashIdx := strings.IndexRune(baseImage.GetRemote(), '/'); slashIdx == -1 {
		remote = name
	} else {
		remote = baseImage.GetRemote()[:slashIdx] + "/" + name
	}

	return registry, remote
}

func getBaseMetaValues(c *storage.Cluster, versions version.Versions, opts *RenderOptions) charts.MetaValues {
	envVars := make(map[string]string)
	if devbuild.IsEnabled() {
		for _, feature := range features.Flags {
			envVars[feature.EnvVar()] = strconv.FormatBool(feature.Enabled())
		}
	}

	command := "kubectl"
	if c.Type == storage.ClusterType_OPENSHIFT_CLUSTER || c.Type == storage.ClusterType_OPENSHIFT4_CLUSTER {
		command = "oc"
	}

	return charts.MetaValues{
		"ClusterName": c.Name,
		"ClusterType": c.Type.String(),

		"PublicEndpoint":     urlfmt.FormatURL(c.CentralApiEndpoint, urlfmt.NONE, urlfmt.NoTrailingSlash),
		"AdvertisedEndpoint": urlfmt.FormatURL(env.AdvertisedEndpoint.Setting(), urlfmt.NONE, urlfmt.NoTrailingSlash),

		"CollectionMethod": c.CollectionMethod.String(),

		// Hardcoding RHACS charts repo for now.
		// TODO: fill ChartRepo based on the current image flavor.
		"ChartRepo": defaults.ChartRepo{
			URL: "http://mirror.openshift.com/pub/rhacs/charts",
		},

		"TolerationsEnabled": !c.GetTolerationsConfig().GetDisabled(),
		"CreateUpgraderSA":   opts.CreateUpgraderSA,

		"EnvVars": envVars,

		"K8sCommand": command,

		"OfflineMode": env.OfflineModeEnv.BooleanSetting(),

		"SlimCollector": opts.SlimCollector,

		"KubectlOutput": true,

		"Versions": versions,

		"FeatureFlags": make(map[string]string),

		"AdmissionController":              c.AdmissionController,
		"AdmissionControlListenOnUpdates":  c.GetAdmissionControllerUpdates(),
		"AdmissionControlListenOnEvents":   c.GetAdmissionControllerEvents(),
		"DisableBypass":                    c.GetDynamicConfig().GetAdmissionControllerConfig().GetDisableBypass(),
		"TimeoutSeconds":                   c.GetDynamicConfig().GetAdmissionControllerConfig().GetTimeoutSeconds(),
		"ScanInline":                       c.GetDynamicConfig().GetAdmissionControllerConfig().GetScanInline(),
		"AdmissionControllerEnabled":       c.GetDynamicConfig().GetAdmissionControllerConfig().GetEnabled(),
		"AdmissionControlEnforceOnUpdates": c.GetDynamicConfig().GetAdmissionControllerConfig().GetEnforceOnUpdates(),
	}
}
