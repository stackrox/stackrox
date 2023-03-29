package renderer

import (
	"encoding/base64"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/image"
	"github.com/stackrox/rox/pkg/images/defaults"
	imageUtils "github.com/stackrox/rox/pkg/images/utils"
	kubernetesPkg "github.com/stackrox/rox/pkg/kubernetes"
	"github.com/stackrox/rox/pkg/sliceutils"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/zip"
)

var (
	assetFileNameMap = FileNameMap{}
)

func init() {
	assetFileNameMap.AddWithName("assets/docker-auth.sh", "docker-auth.sh")
}

// mode is the mode we want the renderer to function in.
//
//go:generate stringer -type=mode
type mode int

const (
	// renderAll renders all objects (central+scanner).
	renderAll mode = iota
	// scannerOnly renders only the scanner.
	scannerOnly
	// centralTLSOnly renders only the central tls secret.
	centralTLSOnly
	// scannerTLSOnly renders only the scanner tls secret
	scannerTLSOnly
	// centralDBOnly renders only the central db
	centralDBOnly
)

func postProcessConfig(c *Config, mode mode, imageFlavor defaults.ImageFlavor) error {
	// Ensure that default values are taken from the flavor if not provided explicitly in the parameteres
	if c.K8sConfig.MainImage == "" {
		c.K8sConfig.MainImage = imageFlavor.MainImage()
	}
	if c.K8sConfig.CentralDBImage == "" {
		c.K8sConfig.CentralDBImage = imageFlavor.CentralDBImage()
	}
	if c.K8sConfig.ScannerImage == "" {
		c.K8sConfig.ScannerImage = imageFlavor.ScannerImage()
	}
	if c.K8sConfig.ScannerDBImage == "" {
		c.K8sConfig.ScannerDBImage = imageFlavor.ScannerDBImage()
	}

	// Make all items in SecretsByteMap base64 encoded
	c.SecretsBase64Map = make(map[string]string)
	for k, v := range c.SecretsByteMap {
		c.SecretsBase64Map[k] = base64.StdEncoding.EncodeToString(v)
	}

	if c.HelmImage == nil {
		c.HelmImage = image.GetDefaultImage()
	}

	if mode == centralTLSOnly || mode == scannerTLSOnly {
		return nil
	}
	if c.ClusterType == storage.ClusterType_KUBERNETES_CLUSTER {
		c.K8sConfig.Command = "kubectl"
	} else {
		c.K8sConfig.Command = "oc"
	}

	if mode == centralDBOnly {
		c.K8sConfig.EnableCentralDB = true
	}

	configureImageOverrides(c, imageFlavor)

	var err error
	if mode == renderAll || mode == centralDBOnly {
		c.K8sConfig.Registry, err = kubernetesPkg.GetResolvedRegistry(c.K8sConfig.MainImage)
		if err != nil {
			return err
		}
	}

	c.K8sConfig.ScannerRegistry, err = kubernetesPkg.GetResolvedRegistry(c.K8sConfig.ScannerImage)
	if err != nil {
		return err
	}
	if c.K8sConfig.Registry != c.K8sConfig.ScannerRegistry {
		c.K8sConfig.ScannerSecretName = "stackrox-scanner"
	} else {
		c.K8sConfig.ScannerSecretName = "stackrox"
	}

	if mode == renderAll {
		if err := injectImageTags(c); err != nil {
			return err
		}
	}

	// Currently, when the K8S config is generated through interactive mode, the configuration flags will be called twice.
	// This doesn't affect single value configurations, like booleans and strings, but slices.
	// TODO(ROX-14956):Once the duplication of flag values is removed, this can be removed.
	c.K8sConfig.DeclarativeConfigMounts.ConfigMaps = sliceutils.Unique(c.K8sConfig.DeclarativeConfigMounts.ConfigMaps)
	c.K8sConfig.DeclarativeConfigMounts.Secrets = sliceutils.Unique(c.K8sConfig.DeclarativeConfigMounts.Secrets)
	// Additionally, the default value used by the configuration for empty arrays is "[]", which we will have to remove.
	c.K8sConfig.DeclarativeConfigMounts.ConfigMaps = sliceutils.Without(c.K8sConfig.DeclarativeConfigMounts.ConfigMaps, []string{"[]"})
	c.K8sConfig.DeclarativeConfigMounts.Secrets = sliceutils.Without(c.K8sConfig.DeclarativeConfigMounts.Secrets, []string{"[]"})

	return nil
}

// Render renders a bunch of zip files based on the given config.
func Render(c Config, imageFlavor defaults.ImageFlavor) ([]*zip.File, error) {
	return render(c, renderAll, imageFlavor)
}

// RenderScannerOnly renders the zip files for the scanner based on the given config.
func RenderScannerOnly(c Config, imageFlavor defaults.ImageFlavor) ([]*zip.File, error) {
	return render(c, scannerOnly, imageFlavor)
}

// RenderCentralDBOnly renders the zip files for the Central DB
func RenderCentralDBOnly(c Config, imageFlavor defaults.ImageFlavor) ([]*zip.File, error) {
	return render(c, centralDBOnly, imageFlavor)
}

func renderAndExtractSingleFileContents(c Config, mode mode, imageFlavor defaults.ImageFlavor) ([]byte, error) {
	files, err := render(c, mode, imageFlavor)
	if err != nil {
		return nil, err
	}

	if len(files) != 1 {
		return nil, utils.ShouldErr(errors.Errorf("got unexpected number of files when rendering in mode %s: %d", mode, len(files)))
	}
	return files[0].Content, nil
}

// RenderCentralTLSSecretOnly renders just the file that contains the central-tls secret.
func RenderCentralTLSSecretOnly(c Config, imageFlavor defaults.ImageFlavor) ([]byte, error) {
	return renderAndExtractSingleFileContents(c, centralTLSOnly, imageFlavor)
}

// RenderScannerTLSSecretOnly renders just the file that contains the scanner-tls secret.
func RenderScannerTLSSecretOnly(c Config, imageFlavor defaults.ImageFlavor) ([]byte, error) {
	return renderAndExtractSingleFileContents(c, scannerTLSOnly, imageFlavor)
}

func render(c Config, mode mode, imageFlavor defaults.ImageFlavor) ([]*zip.File, error) {
	err := postProcessConfig(&c, mode, imageFlavor)
	if err != nil {
		return nil, err
	}

	return renderNew(c, mode, imageFlavor)
}

func getTag(imageStr string) (string, error) {
	imageName, err := imageUtils.GenerateImageFromString(imageStr)
	if err != nil {
		return "", err
	}
	return imageName.GetName().GetTag(), nil
}

func injectImageTags(c *Config) error {
	var err error
	c.K8sConfig.MainImageTag, err = getTag(c.K8sConfig.MainImage)
	if err != nil {
		return err
	}
	return nil
}
