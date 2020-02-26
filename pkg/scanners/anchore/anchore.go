package anchore

import (
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/antihax/optional"
	"github.com/pkg/errors"
	anchoreClient "github.com/stackrox/anchore-client/client"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/registries"
	scannerTypes "github.com/stackrox/rox/pkg/scanners/types"
	"github.com/stackrox/rox/pkg/urlfmt"
)

const (
	typeString = "anchore"

	defaultTimeout = 10 * time.Second

	// It can take a really long time to retrieve vulnerabilities and components.
	scanRetrievalTimeout = time.Minute

	// This is the interval between polls to see if Anchore finished scanning an image.
	pollInterval    = 5 * time.Second
	maxPollAttempts = 50
)

var (
	log = logging.LoggerForModule()
)

// Creator provides the type an scanners.Creator to add to the scanners Registry.
func Creator(set registries.Set) (string, func(integration *storage.ImageIntegration) (scannerTypes.ImageScanner, error)) {
	return typeString, func(integration *storage.ImageIntegration) (scannerTypes.ImageScanner, error) {
		return newScanner(integration, set)
	}
}

type anchore struct {
	scannerTypes.ScanSemaphore

	client                *anchoreClient.APIClient
	conf                  *storage.AnchoreConfig
	protoImageIntegration *storage.ImageIntegration
	activeRegistries      registries.Set
}

func validateConfig(conf *storage.AnchoreConfig) error {
	errorList := errorhelpers.NewErrorList("Config validation")
	if conf.GetEndpoint() == "" {
		errorList.AddString("Endpoint is required")
	}
	if conf.GetUsername() == "" {
		errorList.AddString("Username is required")
	}
	if conf.GetPassword() == "" {
		errorList.AddString("Password is required")
	}
	return errorList.ToError()
}

func basicAuth(username, password string) string {
	basicStr := fmt.Sprintf("%s:%s", username, password)
	return fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(basicStr)))
}

func newScanner(ii *storage.ImageIntegration, activeRegistries registries.Set) (*anchore, error) {
	anchoreConfig, ok := ii.IntegrationConfig.(*storage.ImageIntegration_Anchore)
	if !ok {
		return nil, errors.New("anchore configuration required")
	}
	conf := anchoreConfig.Anchore
	if err := validateConfig(conf); err != nil {
		return nil, err
	}

	endpoint, err := urlfmt.FormatURL(conf.Endpoint, urlfmt.InsecureHTTP, urlfmt.NoTrailingSlash)
	if err != nil {
		return nil, err
	}

	config := anchoreClient.NewConfiguration()
	config.BasePath = fmt.Sprintf("%s/v1", endpoint)
	config.AddDefaultHeader("Authorization", basicAuth(conf.GetUsername(), conf.GetPassword()))
	client := anchoreClient.NewAPIClient(config)

	scanner := &anchore{
		client:                client,
		conf:                  conf,
		protoImageIntegration: ii,
		activeRegistries:      activeRegistries,

		ScanSemaphore: scannerTypes.NewDefaultSemaphore(),
	}
	return scanner, nil
}

func (a *anchore) getContentTypes(imageID string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	content, _, err := a.client.ImageContentApi.ListImageContent(ctx, imageID, nil)
	return content, err
}

func (a *anchore) getComponentsForType(imageID, cType string) ([]anchoreClient.ContentPackageResponseContent, error) {
	ctx, cancel := context.WithTimeout(context.Background(), scanRetrievalTimeout)
	defer cancel()

	contentResponse, _, err := a.client.ImageContentApi.GetImageContentByType(ctx, imageID, cType, nil)
	if err != nil {
		return nil, err
	}
	return contentResponse.Content, err
}

func (a *anchore) getPackages(imageID string) ([]anchoreClient.ContentPackageResponseContent, error) {
	cTypes, err := a.getContentTypes(imageID)
	if err != nil {
		return nil, err
	}

	var allContents []anchoreClient.ContentPackageResponseContent
	for _, cType := range cTypes {
		// Only report OS components that don't have vulns
		if cType != "os" {
			continue
		}
		contents, err := a.getComponentsForType(imageID, cType)
		if err != nil {
			return nil, err
		}
		allContents = append(allContents, contents...)
	}
	return allContents, nil
}

func (a *anchore) getVulnerabilities(imageID string) ([]anchoreClient.Vulnerability, error) {
	ctx, cancel := context.WithTimeout(context.Background(), scanRetrievalTimeout)
	defer cancel()

	vulnResponse, _, err := a.client.VulnerabilitiesApi.GetImageVulnerabilitiesByType(ctx, imageID, "all", nil)
	if err != nil {
		return nil, err
	}
	return vulnResponse.Vulnerabilities, nil
}

func (a *anchore) getImage(image *storage.Image) (*anchoreClient.AnchoreImage, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	var (
		imageList anchoreClient.AnchoreImageList
		resp      *http.Response
		err       error
	)

	if image.GetId() == "" {
		imageList, resp, err = a.client.ImagesApi.ListImages(ctx, &anchoreClient.ListImagesOpts{
			Fulltag: optional.NewString(image.GetName().GetFullName()),
		})
	} else {
		imageList, resp, err = a.client.ImagesApi.GetImage(ctx, image.GetId(), nil)
	}
	if resp != nil && resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if len(imageList) == 0 {
		return nil, fmt.Errorf("expected to get NotFound instead of empty list for image %q", image.GetName().GetFullName())
	}
	return &imageList[0], nil
}

func (a *anchore) Test() error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	_, _, err := a.client.ImagesApi.ListImages(ctx, nil)
	return err
}

func (a *anchore) GetOrTriggerScan(image *storage.Image) (*storage.ImageScan, error) {
	return a.getOrTriggerScan(image)
}

func (a *anchore) GetScan(image *storage.Image) (*storage.ImageScan, error) {
	for attempt := 0; attempt < maxPollAttempts; attempt++ {
		scan, err := a.getOrTriggerScan(image)
		if err != nil {
			return nil, err
		}
		if scan != nil {
			return scan, nil
		}
		time.Sleep(pollInterval)
	}
	return nil, errors.Errorf("timed out waiting for anchore to scan %q", image.GetName().GetFullName())
}

func (a *anchore) getOrTriggerScan(image *storage.Image) (*storage.ImageScan, error) {
	anchoreImg, err := a.getImage(image)
	if err != nil {
		return nil, errors.Wrapf(err, "error getting image %q", image.GetName().GetFullName())
	}
	if anchoreImg == nil {
		err := a.triggerScan(image)
		return nil, err
	}
	if strings.EqualFold(anchoreImg.AnalysisStatus, "analysis_failed") {
		return nil, errors.Errorf("anchore couldn't analyze image %q: analysis failed", image.GetName().GetFullName())
	}
	if !strings.EqualFold(anchoreImg.AnalysisStatus, "analyzed") {
		return nil, nil
	}
	packages, err := a.getPackages(anchoreImg.ImageDigest)
	if err != nil {
		return nil, errors.Wrapf(err, "error retrieving packages for %q", anchoreImg.ImageDigest)
	}
	vulns, err := a.getVulnerabilities(anchoreImg.ImageDigest)
	if err != nil {
		return nil, errors.Wrapf(err, "error retrieve vulnerabilities for %q", anchoreImg.ImageDigest)
	}
	return convertImageScan(anchoreImg, packages, vulns), nil
}

func (a *anchore) registerRegistry(image *storage.Image) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	config := a.activeRegistries.GetRegistryMetadataByImage(image)
	if config == nil {
		return nil
	}
	if config.Username == "" && config.Password == "" {
		return nil
	}

	_, _, err := a.client.RegistriesApi.CreateRegistry(ctx, anchoreClient.RegistryConfigurationRequest{
		Registry:       image.GetName().GetRegistry(),
		RegistryPass:   config.Password,
		RegistryUser:   config.Username,
		RegistryVerify: config.Insecure,
		RegistryType:   "docker",
	}, nil)
	return err
}

func getImageAnalysisRequest(image *storage.Image) (*anchoreClient.ImageAnalysisRequest, error) {
	var iar anchoreClient.ImageAnalysisRequest
	if image.GetId() != "" {
		iar.Digest = image.GetId()
		// This is a strange construct of Anchore, but is required when scanning by tag and digest
		iar.CreatedAt = time.Now().UTC().Format(time.RFC3339)
		iar.Tag = "fake" // Anchore needs a fake tag to be passed, but it is disregarded
	}
	iar.Tag = image.GetName().GetFullName()
	return &iar, nil
}

func (a *anchore) addImage(iar anchoreClient.ImageAnalysisRequest) (anchoreClient.AnchoreImageList, *http.Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	return a.client.ImagesApi.AddImage(ctx, iar, nil)
}

func formatAnchoreError(err error) error {
	if swaggerError, ok := err.(anchoreClient.GenericSwaggerError); ok {
		return fmt.Errorf("%s: %s", swaggerError.Error(), swaggerError.Body())
	}
	return err
}

func getBody(resp *http.Response) string {
	defer func() {
		_ = resp.Body.Close()
	}()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "<no body>"
	}
	return string(body)
}

func (a *anchore) triggerScan(image *storage.Image) error {
	iarPtr, err := getImageAnalysisRequest(image)
	if err != nil {
		return err
	}
	iar := *iarPtr
	_, resp, err := a.addImage(iar)
	if resp != nil && resp.StatusCode == http.StatusBadRequest {
		if err := a.registerRegistry(image); err != nil {
			return errors.Wrap(err, "error registering integration")
		}
		_, resp, err = a.addImage(iar)
	}
	if err != nil {
		return formatAnchoreError(err)
	}
	if resp != nil && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("expected status 200, but received %d: %s", resp.StatusCode, getBody(resp))
	}
	return nil
}

// Match decides if the image is contained within this scanner
func (a *anchore) Match(image *storage.ImageName) bool {
	return a.activeRegistries.Match(image)
}

func (a *anchore) Type() string {
	return typeString
}

func (a *anchore) Name() string {
	return a.protoImageIntegration.GetName()
}
